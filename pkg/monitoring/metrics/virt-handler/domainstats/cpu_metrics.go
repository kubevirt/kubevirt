package domainstats

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"kubevirt.io/client-go/log"
)

var (
	cpuUsageSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_cpu_usage_seconds_total",
			Help: "Total CPU time spent in all modes (sum of both vcpu and hypervisor usage).",
		},
	)

	cpuUserUsageSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_cpu_user_usage_seconds_total",
			Help: "Total CPU time spent in user mode.",
		},
	)

	cpuSystemUsageSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_cpu_system_usage_seconds_total",
			Help: "Total CPU time spent in system mode.",
		},
	)
)

type cpuMetrics struct{}

func (cpuMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		cpuUsageSeconds,
		cpuUserUsageSeconds,
		cpuSystemUsageSeconds,
	}
}

func (cpuMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil || vmiReport.vmiStats.DomainStats.Cpu == nil {
		return crs
	}

	cpu := vmiReport.vmiStats.DomainStats.Cpu

	if !cpu.TimeSet && !cpu.UserSet && !cpu.SystemSet {
		log.Log.Warningf("No domain CPU stats is set for %s VMI.", vmiReport.vmi.Name)
	}

	if cpu.TimeSet {
		crs = append(crs, vmiReport.newCollectorResult(cpuUsageSeconds, nanosecondsToSeconds(cpu.Time)))
	}

	if cpu.UserSet {
		crs = append(crs, vmiReport.newCollectorResult(cpuUserUsageSeconds, nanosecondsToSeconds(cpu.User)))
	}

	if cpu.SystemSet {
		crs = append(crs, vmiReport.newCollectorResult(cpuSystemUsageSeconds, nanosecondsToSeconds(cpu.System)))
	}

	return crs
}

func nanosecondsToSeconds(ns uint64) float64 {
	return float64(ns) / 1000000000
}
