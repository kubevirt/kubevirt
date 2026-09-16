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

package virtcontroller

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	k6tv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
)

var (
	vmExportStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmExportInfo,
			vmExportTTLExpirationTimestamp,
		},
		CollectCallback: vmExportStatsCollectorCallback,
	}

	vmExportInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmexport_info",
			Help: "Information about VirtualMachineExports.",
		},
		[]string{"namespace", "name", "uid", "vm", "source", "source_kind", "phase"},
	)

	vmExportTTLExpirationTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmexport_ttl_expiration_timestamp_seconds",
			Help: "Time at which a VirtualMachineExport will be deleted according to " +
				"its TTL (status.ttlExpirationTime).",
		},
		[]string{"name", "namespace"},
	)
)

func vmExportStatsCollectorCallback() []operatormetrics.CollectorResult {
	if stores == nil {
		return []operatormetrics.CollectorResult{}
	}
	return reportVMExportStats(listStoreObjects[exportv1.VirtualMachineExport](stores.VMExport))
}

func reportVMExportStats(exports []*exportv1.VirtualMachineExport) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult
	for _, vmExport := range exports {
		results = append(results, collectVMExportInfo(vmExport))
		results = append(results, collectVMExportTTLExpiration(vmExport)...)
	}
	return results
}

func collectVMExportInfo(vmExport *exportv1.VirtualMachineExport) operatormetrics.CollectorResult {
	phase := inventoryPhaseUnset
	if vmExport.Status != nil {
		phase = resourcePhaseLabel(string(vmExport.Status.Phase))
	}

	return operatormetrics.CollectorResult{
		Metric: vmExportInfo,
		Value:  1,
		Labels: []string{
			vmExport.Namespace,
			vmExport.Name,
			string(vmExport.UID),
			exportSourceVM(vmExport),
			vmExport.Spec.Source.Name,
			vmExport.Spec.Source.Kind,
			phase,
		},
	}
}

func exportSourceVM(vmExport *exportv1.VirtualMachineExport) string {
	if vmExport.Status != nil {
		if name := optionalStringLabel(vmExport.Status.VirtualMachineName); name != none {
			return name
		}
	}
	if vmExport.Spec.Source.Kind == k6tv1.VirtualMachineGroupVersionKind.Kind {
		return vmExport.Spec.Source.Name
	}
	return none
}

func collectVMExportTTLExpiration(vmExport *exportv1.VirtualMachineExport) []operatormetrics.CollectorResult {
	if vmExport.Status == nil {
		return nil
	}
	return collectOptionalUnixTimestamp(
		vmExportTTLExpirationTimestamp,
		vmExport.Status.TTLExpirationTime,
		[]string{vmExport.Name, vmExport.Namespace},
	)
}
