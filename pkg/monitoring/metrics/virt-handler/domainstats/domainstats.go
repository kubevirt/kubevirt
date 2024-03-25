package domainstats

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

type VirtualMachineInstanceReport struct {
	vmi           *k6tv1.VirtualMachineInstance
	vmiStats      *VirtualMachineInstanceStats
	runtimeLabels map[string]string
}

type VirtualMachineInstanceStats struct {
	DomainStats *stats.DomainStats
	FsStats     k6tv1.VirtualMachineInstanceFileSystemList
}

func newVirtualMachineInstanceReport(vmi *k6tv1.VirtualMachineInstance, vmiStats *VirtualMachineInstanceStats) *VirtualMachineInstanceReport {
	vmiReport := &VirtualMachineInstanceReport{
		vmi:      vmi,
		vmiStats: vmiStats,
	}
	vmiReport.buildRuntimeLabels()

	return vmiReport
}

func (vmiReport *VirtualMachineInstanceReport) buildRuntimeLabels() map[string]string {
	var runtimeLabels = map[string]string{}

	for label, val := range vmiReport.vmi.Labels {
		runtimeLabels[label] = val
	}

	return runtimeLabels
}

func (vmiReport *VirtualMachineInstanceReport) newCollectorResult(metric operatormetrics.Metric, value float64) operatormetrics.CollectorResult {
	return vmiReport.newCollectorResultWithLabels(metric, value, nil)
}

func (vmiReport *VirtualMachineInstanceReport) newCollectorResultWithLabels(metric operatormetrics.Metric, value float64, additionalLabels map[string]string) operatormetrics.CollectorResult {
	vmiLabels := map[string]string{
		"node":      vmiReport.vmi.Status.NodeName,
		"namespace": vmiReport.vmi.Namespace,
		"name":      vmiReport.vmi.Name,
	}

	for k, v := range vmiReport.runtimeLabels {
		vmiLabels[k] = v
	}

	for k, v := range additionalLabels {
		vmiLabels[k] = v
	}

	return operatormetrics.CollectorResult{
		Metric:      metric,
		ConstLabels: vmiLabels,
		Value:       value,
	}
}
