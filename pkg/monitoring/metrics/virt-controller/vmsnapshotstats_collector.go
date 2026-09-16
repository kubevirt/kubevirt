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
	"strconv"
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
)

var (
	vmSnapshotStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmSnapshotInfo,
			vmSnapshotCreationTimestamp,
		},
		CollectCallback: vmSnapshotStatsCollectorCallback,
	}

	vmSnapshotInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_info",
			Help: "Information about VirtualMachineSnapshots.",
		},
		[]string{"namespace", "name", "uid", "vm", "phase", "ready_to_use"},
	)

	vmSnapshotCreationTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_create_date_timestamp_seconds",
			Help: "Virtual Machine Snapshot creation timestamp.",
		},
		[]string{"name", "namespace"},
	)
)

func vmSnapshotStatsCollectorCallback() []operatormetrics.CollectorResult {
	if stores == nil || stores.VMSnapshot == nil {
		return []operatormetrics.CollectorResult{}
	}

	cachedObjs := stores.VMSnapshot.List()
	snapshots := make([]*snapshotv1.VirtualMachineSnapshot, len(cachedObjs))
	for i, obj := range cachedObjs {
		snapshots[i] = obj.(*snapshotv1.VirtualMachineSnapshot)
	}

	return reportVMSnapshotStats(snapshots)
}

func reportVMSnapshotStats(snapshots []*snapshotv1.VirtualMachineSnapshot) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult
	for _, snapshot := range snapshots {
		results = append(results, collectVMSnapshotInfo(snapshot))
		results = append(results, collectVMSnapshotCreationTimestamp(snapshot)...)
	}
	return results
}

func collectVMSnapshotInfo(snapshot *snapshotv1.VirtualMachineSnapshot) operatormetrics.CollectorResult {
	return operatormetrics.CollectorResult{
		Metric: vmSnapshotInfo,
		Value:  1,
		Labels: []string{
			snapshot.Namespace,
			snapshot.Name,
			string(snapshot.UID),
			snapshot.Spec.Source.Name,
			vmSnapshotPhase(snapshot),
			strconv.FormatBool(isVMSnapshotReadyToUse(snapshot)),
		},
	}
}

func collectVMSnapshotCreationTimestamp(snapshot *snapshotv1.VirtualMachineSnapshot) []operatormetrics.CollectorResult {
	if snapshot.CreationTimestamp.IsZero() {
		return nil
	}
	return []operatormetrics.CollectorResult{{
		Metric: vmSnapshotCreationTimestamp,
		Value:  float64(snapshot.CreationTimestamp.Unix()),
		Labels: []string{snapshot.Name, snapshot.Namespace},
	}}
}

func vmSnapshotPhase(snapshot *snapshotv1.VirtualMachineSnapshot) string {
	if snapshot.Status == nil {
		return none
	}
	return strings.ToLower(string(snapshot.Status.Phase))
}

func isVMSnapshotReadyToUse(snapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse
}
