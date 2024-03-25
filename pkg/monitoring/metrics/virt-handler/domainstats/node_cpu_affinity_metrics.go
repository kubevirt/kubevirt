package domainstats

import "github.com/machadovilaca/operator-observability/pkg/operatormetrics"

var (
	nodeCpuAffinity = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_node_cpu_affinity",
			Help: "Number of VMI CPU affinities to node physical cores.",
		},
	)
)

type cpuAffinityMetrics struct{}

func (cpuAffinityMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{nodeCpuAffinity}
}

func (cpuAffinityMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	if vmiReport.vmiStats.DomainStats == nil || !vmiReport.vmiStats.DomainStats.CPUMapSet {
		return []operatormetrics.CollectorResult{}
	}

	affinityCount := 0.0

	for vidx := 0; vidx < len(vmiReport.vmiStats.DomainStats.CPUMap); vidx++ {
		for cidx := 0; cidx < len(vmiReport.vmiStats.DomainStats.CPUMap[vidx]); cidx++ {
			if vmiReport.vmiStats.DomainStats.CPUMap[vidx][cidx] {
				affinityCount++
			}
		}
	}

	return []operatormetrics.CollectorResult{
		vmiReport.newCollectorResult(nodeCpuAffinity, affinityCount),
	}
}
