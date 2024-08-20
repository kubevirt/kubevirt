package downwardmetrics

import (
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/sysfs"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	metricspkg "kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/metrics"
)

type hostMetricsCollector struct {
	procPath string
	sysPath  string
	pageSize int
}

func (h *hostMetricsCollector) hostCPUMetrics() []api.Metric {
	metrics := make([]api.Metric, 0, 2)
	metrics = append(metrics, h.hostPhysicalCpuCount()...)
	metrics = append(metrics, h.hostCpuTime()...)

	return metrics
}

func (h *hostMetricsCollector) hostPhysicalCpuCount() []api.Metric {
	sysFS, err := sysfs.NewFS(h.sysPath)
	if err != nil {
		log.Log.Reason(err).Info("failed to access /sys")
		return nil
	}

	cpus, err := sysFS.CPUs()
	if err != nil {
		log.Log.Reason(err).Info("failed to collect cpus on the node")
		return nil
	}

	uniquePhysicalIDs := map[string]struct{}{}
	for _, cpu := range cpus {
		topology, err := cpu.Topology()
		if err != nil {
			log.Log.Reason(err).Info("failed to read cpu topology")
			return nil
		}
		uniquePhysicalIDs[topology.PhysicalPackageID] = struct{}{}
	}

	return []api.Metric{
		metricspkg.MustToUnitlessHostMetric(len(uniquePhysicalIDs), "NumberOfPhysicalCPUs"),
	}
}

func (h *hostMetricsCollector) hostCpuTime() []api.Metric {
	procFS, err := procfs.NewFS(h.procPath)
	if err != nil {
		log.Log.Reason(err).Info("failed to access /proc")
		return nil
	}

	stat, err := procFS.Stat()
	if err != nil {
		log.Log.Reason(err).Info("failed to collect cputime on the node")
		return nil
	}

	cpuTime := stat.CPUTotal.User + stat.CPUTotal.Nice + stat.CPUTotal.System
	return []api.Metric{
		metricspkg.MustToHostMetric(cpuTime, "TotalCPUTime", "s"),
	}
}

func (h *hostMetricsCollector) hostMemoryMetrics() []api.Metric {
	fs, err := procfs.NewFS(h.procPath)
	if err != nil {
		log.Log.Reason(err).Info("failed to access /proc")
		return nil
	}

	var metrics []api.Metric
	if memInfo, err := fs.Meminfo(); err == nil {
		memFree := derefOrZero(memInfo.MemFree)
		swapFree := derefOrZero(memInfo.SwapFree)
		memTotal := derefOrZero(memInfo.MemTotal)
		swapTotal := derefOrZero(memInfo.SwapTotal)
		buffers := derefOrZero(memInfo.Buffers)
		cached := derefOrZero(memInfo.Cached)
		swapCached := derefOrZero(memInfo.SwapCached)

		metrics = append(metrics,
			metricspkg.MustToHostMetric(memFree, "FreePhysicalMemory", "KiB"),
			metricspkg.MustToHostMetric(memFree+swapFree, "FreeVirtualMemory", "KiB"),
			metricspkg.MustToHostMetric(memTotal-memFree-buffers-cached, "MemoryAllocatedToVirtualServers", "KiB"),
			metricspkg.MustToHostMetric(memTotal+swapTotal-memFree-cached-buffers-swapCached, "UsedVirtualMemory", "KiB"),
		)
	} else {
		log.Log.Reason(err).Info("failed to collect meminfo on the node")
	}

	if vmstat, err := readVMStat(filepath.Join(h.procPath, "vmstat")); err == nil {
		metrics = append(metrics,
			metricspkg.MustToHostMetric(vmstat.pswpin*uint64(h.pageSize)/1024, "PagedInMemory", "KiB"),
			metricspkg.MustToHostMetric(vmstat.pswpout*uint64(h.pageSize)/1024, "PagedOutMemory", "KiB"),
		)
	} else {
		log.Log.Reason(err).Info("failed to collect vmstat on the node")
	}

	return metrics
}

func (h *hostMetricsCollector) Collect() (metrics []api.Metric) {
	metrics = append(metrics, h.hostCPUMetrics()...)
	metrics = append(metrics, h.hostMemoryMetrics()...)
	metrics = append(metrics,
		metricspkg.MustToHostMetric(time.Now().Unix(), "Time", "s"),
	)
	return
}

func defaultHostMetricsCollector() *hostMetricsCollector {
	return &hostMetricsCollector{
		procPath: "/proc",
		sysPath:  "/sys",
		pageSize: os.Getpagesize(),
	}
}

func derefOrZero(val *uint64) uint64 {
	if val == nil {
		return 0
	}
	return *val
}
