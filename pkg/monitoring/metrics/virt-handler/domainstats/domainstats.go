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
	"encoding/json"
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

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
	networkNames  map[string]string
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

	// Fill networkNames.
	vmiReport.parseNetworkNames()

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

func (vmiReport *VirtualMachineInstanceReport) networkNameByIface(iface string) string {
	if netName, ok := vmiReport.networkNames[iface]; ok {
		return netName
	}
	return iface
}

type d8NetworkSpec struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	IfName string `json:"ifName"`
}

// parseNetworkNames parse annotation with network specs and makes index interfaceName -> networkName.
//
// Annotation example:
//
//	network.deckhouse.io/networks-spec: '[{"type":"ClusterNetwork","name":"cnet1","ifName":"veth_cne8f3b5d3"},{"type":"ClusterNetwork","name":"cn-1003-for-e2e-test","ifName":"veth_cnce02ff17"}]'
func (vmiReport *VirtualMachineInstanceReport) parseNetworkNames() {
	networksSpecsRaw := vmiReport.vmi.GetAnnotations()["network.deckhouse.io/networks-spec"]
	if networksSpecsRaw == "" {
		log.Log.Warningf("no d8 networks specs: annotations=%+v on VM %s", vmiReport.vmi.GetAnnotations(), vmiReport.vmi.Name)
		return
	}
	var networksSpecs []d8NetworkSpec
	err := json.Unmarshal([]byte(networksSpecsRaw), &networksSpecs)
	if err != nil {
		log.Log.Warningf("invalid d8 networks specs for network labels on VM %s: %v", vmiReport.vmi.Name, err)
		return
	}

	vmiReport.networkNames = make(map[string]string)
	for _, n := range networksSpecs {
		vmiReport.networkNames[n.IfName] = n.Name
	}
}
