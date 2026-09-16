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
	"slices"
	"strconv"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	corev1 "k8s.io/api/core/v1"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
)

var (
	vmRestoreStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmRestoreInfo,
		},
		CollectCallback: vmRestoreStatsCollectorCallback,
	}

	vmRestoreInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmrestore_info",
			Help: "Information about VirtualMachineRestores.",
		},
		[]string{"namespace", "name", "uid", "vm", "snapshot_name", "complete", "failure"},
	)
)

func vmRestoreStatsCollectorCallback() []operatormetrics.CollectorResult {
	if stores == nil || stores.VMRestore == nil {
		return []operatormetrics.CollectorResult{}
	}

	cachedObjs := stores.VMRestore.List()
	restores := make([]*snapshotv1.VirtualMachineRestore, len(cachedObjs))
	for i, obj := range cachedObjs {
		restores[i] = obj.(*snapshotv1.VirtualMachineRestore)
	}

	return reportVMRestoreStats(restores)
}

func reportVMRestoreStats(restores []*snapshotv1.VirtualMachineRestore) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult
	for _, restore := range restores {
		results = append(results, collectVMRestoreInfo(restore))
	}
	return results
}

func collectVMRestoreInfo(restore *snapshotv1.VirtualMachineRestore) operatormetrics.CollectorResult {
	return operatormetrics.CollectorResult{
		Metric: vmRestoreInfo,
		Value:  1,
		Labels: []string{
			restore.Namespace,
			restore.Name,
			string(restore.UID),
			restore.Spec.Target.Name,
			restore.Spec.VirtualMachineSnapshotName,
			strconv.FormatBool(isVMRestoreComplete(restore)),
			strconv.FormatBool(isVMRestoreFailed(restore)),
		},
	}
}

func isVMRestoreComplete(restore *snapshotv1.VirtualMachineRestore) bool {
	return restore.Status != nil && restore.Status.Complete != nil && *restore.Status.Complete
}

func isVMRestoreFailed(restore *snapshotv1.VirtualMachineRestore) bool {
	if restore.Status == nil {
		return false
	}
	return slices.ContainsFunc(restore.Status.Conditions, func(condition snapshotv1.Condition) bool {
		return condition.Type == snapshotv1.ConditionFailure && condition.Status == corev1.ConditionTrue
	})
}
