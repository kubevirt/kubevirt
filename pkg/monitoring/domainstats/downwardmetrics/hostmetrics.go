package downwardmetrics

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	var metrics []api.Metric
	metrics = append(metrics, h.hostPhysicalCpuCount()...)

	procFS, err := procfs.NewFS(h.procPath)
	if err != nil {
		log.Log.Reason(err).Info("failed to access /proc")
		return nil
	}
	if stat, err := procFS.Stat(); err == nil {
		cpuTime := stat.CPUTotal.User + stat.CPUTotal.Nice + stat.CPUTotal.System
		metrics = append(metrics,
			metricspkg.MustToHostMetric(cpuTime, "TotalCPUTime", "s"),
		)
	} else {
		log.Log.Reason(err).Info("failed to collect cputime on the node")
	}

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

type vmStat struct {
	pswpin  uint64
	pswpout uint64
}

// readVMStat reads specific fields from the /proc/vmstat file.
// We implement it here, because it is not implemented in "github.com/prometheus/procfs"
// library.
func readVMStat(path string) (*vmStat, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := &vmStat{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 2 {
			return nil, fmt.Errorf("malformed line: %q", s.Text())
		}

		var resultField *uint64
		switch fields[0] {
		case "pswpin":
			resultField = &(result.pswpin)
		case "pswpout":
			resultField = &(result.pswpout)
		default:
			continue
		}

		value, err := strconv.ParseUint(fields[1], 0, 64)
		if err != nil {
			return nil, err
		}

		*resultField = value
	}

	return result, nil
}
