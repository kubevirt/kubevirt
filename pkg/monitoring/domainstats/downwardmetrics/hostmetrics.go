package downwardmetrics

import (
	"os"
	"time"

	"github.com/c9s/goprocinfo/linux"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	metricspkg "kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/metrics"
)

type hostMetricsCollector struct {
	procCPUInfo string
	procStat    string
	procMemInfo string
	procVMStat  string
	pageSize    int
}

func (h *hostMetricsCollector) hostCPUMetrics() (metrics []api.Metric) {
	if cpuinfo, err := linux.ReadCPUInfo(h.procCPUInfo); err == nil {
		metrics = append(metrics,
			metricspkg.MustToUnitlessHostMetric(cpuinfo.NumPhysicalCPU(), "NumberOfPhysicalCPUs"),
		)
	} else {
		log.Log.Reason(err).Info("failed to collect cpuinfo on the node")
	}

	if stat, err := linux.ReadStat(h.procStat); err == nil {
		// CLK_TCK is a constant on Linux, see e.g.
		// https://github.com/containerd/cgroups/pull/12
		var clk_tck float64 = 100
		cpuTime := float64(stat.CPUStatAll.User+stat.CPUStatAll.Nice+stat.CPUStatAll.System) / clk_tck
		metrics = append(metrics,
			metricspkg.MustToHostMetric(cpuTime, "TotalCPUTime", "s"),
		)
	} else {
		log.Log.Reason(err).Info("failed to collect cputime on the node")
	}
	return
}

func (h *hostMetricsCollector) hostMemoryMetrics() (metrics []api.Metric) {
	if memInfo, err := linux.ReadMemInfo(h.procMemInfo); err == nil {
		metrics = append(metrics,
			metricspkg.MustToHostMetric(memInfo.MemFree, "FreePhysicalMemory", "KiB"),
			metricspkg.MustToHostMetric(memInfo.MemFree+memInfo.SwapFree, "FreeVirtualMemory", "KiB"),
			metricspkg.MustToHostMetric(memInfo.MemTotal-memInfo.MemFree-memInfo.Buffers-memInfo.Cached, "MemoryAllocatedToVirtualServers", "KiB"),
			metricspkg.MustToHostMetric(memInfo.MemTotal+memInfo.SwapTotal-memInfo.MemFree-memInfo.Cached-memInfo.Buffers-memInfo.SwapCached, "UsedVirtualMemory", "KiB"),
		)
	} else {
		log.Log.Reason(err).Info("failed to collect meminfo on the node")
	}

	if vmstat, err := linux.ReadVMStat(h.procVMStat); err == nil {
		metrics = append(metrics,
			metricspkg.MustToHostMetric(vmstat.PageSwapin*uint64(h.pageSize)/1024, "PagedInMemory", "KiB"),
			metricspkg.MustToHostMetric(vmstat.PageSwapout*uint64(h.pageSize)/1024, "PagedOutMemory", "KiB"),
		)
	} else {
		log.Log.Reason(err).Info("failed to collect vmstat on the node")
	}
	return
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
		procCPUInfo: "/proc/cpuinfo",
		procStat:    "/proc/stat",
		procMemInfo: "/proc/meminfo",
		procVMStat:  "/proc/vmstat",
		pageSize:    os.Getpagesize(),
	}
}
