/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

import (
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var (
	// Formatter used to sanitize k8s metadata into metric labels
	labelFormatter = strings.NewReplacer(".", "_", "/", "_", "-", "_")

	// Preffixes used when transforming K8s metadata into metric labels
	labelPrefix = "kubernetes_vmi_label_"
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

func newVirtualMachineInstanceReport(
	vmi *k6tv1.VirtualMachineInstance, vmiStats *VirtualMachineInstanceStats,
) *VirtualMachineInstanceReport {
	vmiReport := &VirtualMachineInstanceReport{
		vmi:      vmi,
		vmiStats: vmiStats,
	}
	vmiReport.buildRuntimeLabels()

	return vmiReport
}

func (vmiReport *VirtualMachineInstanceReport) GetVmiStats() VirtualMachineInstanceStats {
	return *vmiReport.vmiStats
}

func (vmiReport *VirtualMachineInstanceReport) buildRuntimeLabels() {
	vmiReport.runtimeLabels = map[string]string{}

	for label, val := range vmiReport.vmi.Labels {
		key := labelPrefix + labelFormatter.Replace(label)
		vmiReport.runtimeLabels[key] = val
	}
}

func (vmiReport *VirtualMachineInstanceReport) newCollectorResult(
	metric operatormetrics.Metric, value float64,
) operatormetrics.CollectorResult {
	return vmiReport.newCollectorResultWithLabels(metric, value, nil)
}

func (vmiReport *VirtualMachineInstanceReport) newCollectorResultWithLabels(
	metric operatormetrics.Metric, value float64, additionalLabels map[string]string,
) operatormetrics.CollectorResult {
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
