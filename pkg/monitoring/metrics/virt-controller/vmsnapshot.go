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
 * Copyright the KubeVirt Authors.
 *
 */

package virt_controller

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"

	io_prometheus_client "github.com/prometheus/client_model/go"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
)

var (
	vmSnapshotMetrics = []operatormetrics.Metric{
		VmSnapshotSucceededTimestamp,
	}

	VmSnapshotSucceededTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_succeeded_timestamp_seconds",
			Help: "Returns the timestamp of successful virtual machine snapshot.",
		},
		[]string{"name", "snapshot_name", "namespace"},
	)
)

func HandleSucceededVMSnapshot(snapshot *snapshotv1.VirtualMachineSnapshot) {
	if snapshot.Status.Phase == snapshotv1.Succeeded {
		VmSnapshotSucceededTimestamp.WithLabelValues(
			snapshot.Spec.Source.Name,
			snapshot.Name,
			snapshot.Namespace,
		).Set(float64(snapshot.Status.CreationTime.Unix()))
	}
}

func GetVmSnapshotSucceededTimestamp(vm, snapshot, namespace string) (float64, error) {
	dto := &io_prometheus_client.Metric{}
	if err := VmSnapshotSucceededTimestamp.WithLabelValues(vm, snapshot, namespace).Write(dto); err != nil {
		return 0, err
	}
	return *dto.Gauge.Value, nil
}
