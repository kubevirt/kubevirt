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

	clonev1 "kubevirt.io/api/clone/v1beta1"
)

var (
	vmCloneStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmCloneInfo,
			vmCloneCreationTimestamp,
		},
		CollectCallback: vmCloneStatsCollectorCallback,
	}

	vmCloneInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmclone_info",
			Help: "Information about VirtualMachineClones.",
		},
		[]string{
			"namespace", "name", "uid", "source", "source_kind", "target_vm",
			"snapshot_name", "restore_name", "phase",
		},
	)

	vmCloneCreationTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmclone_create_date_timestamp_seconds",
			Help: "Virtual Machine Clone creation timestamp.",
		},
		[]string{"name", "namespace"},
	)
)

func vmCloneStatsCollectorCallback() []operatormetrics.CollectorResult {
	if stores == nil {
		return []operatormetrics.CollectorResult{}
	}
	return reportVMCloneStats(listStoreObjects[clonev1.VirtualMachineClone](stores.VMClone))
}

func reportVMCloneStats(clones []*clonev1.VirtualMachineClone) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult
	for _, vmClone := range clones {
		results = append(results, collectVMCloneInfo(vmClone))
		results = append(results, collectVMCloneCreationTimestamp(vmClone)...)
	}
	return results
}

func collectVMCloneInfo(vmClone *clonev1.VirtualMachineClone) operatormetrics.CollectorResult {
	return operatormetrics.CollectorResult{
		Metric: vmCloneInfo,
		Value:  1,
		Labels: []string{
			vmClone.Namespace,
			vmClone.Name,
			string(vmClone.UID),
			typedLocalObjectName(vmClone.Spec.Source),
			typedLocalObjectKind(vmClone.Spec.Source),
			cloneTargetVM(vmClone),
			optionalStringLabel(vmClone.Status.SnapshotName),
			optionalStringLabel(vmClone.Status.RestoreName),
			resourcePhaseLabel(string(vmClone.Status.Phase)),
		},
	}
}

func cloneTargetVM(vmClone *clonev1.VirtualMachineClone) string {
	if name := typedLocalObjectName(vmClone.Spec.Target); name != none {
		return name
	}
	return optionalStringLabel(vmClone.Status.TargetName)
}

func collectVMCloneCreationTimestamp(vmClone *clonev1.VirtualMachineClone) []operatormetrics.CollectorResult {
	return collectUnixTimestamp(
		vmCloneCreationTimestamp,
		vmClone.CreationTimestamp,
		[]string{vmClone.Name, vmClone.Namespace},
	)
}
