/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
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

func newVirtualMachineInstanceReport(vmi *k6tv1.VirtualMachineInstance, vmiStats *VirtualMachineInstanceStats) *VirtualMachineInstanceReport {
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
