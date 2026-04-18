/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtcontroller

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	io_prometheus_client "github.com/prometheus/client_model/go"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
)

var (
	vmSnapshotMetrics = []operatormetrics.Metric{
		VMSnapshotSucceededTimestamp,
	}

	VMSnapshotSucceededTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmsnapshot_succeeded_timestamp_seconds",
			Help: "Returns the timestamp of successful virtual machine snapshot.",
		},
		[]string{"name", "snapshot_name", "namespace"},
	)
)

func HandleSucceededVMSnapshot(snapshot *snapshotv1.VirtualMachineSnapshot) {
	if snapshot.Status.Phase == snapshotv1.Succeeded {
		VMSnapshotSucceededTimestamp.WithLabelValues(
			snapshot.Spec.Source.Name,
			snapshot.Name,
			snapshot.Namespace,
		).Set(float64(snapshot.Status.CreationTime.Unix()))
	}
}

func GetVMSnapshotSucceededTimestamp(vm, snapshot, namespace string) (float64, error) {
	dto := &io_prometheus_client.Metric{}
	if err := VMSnapshotSucceededTimestamp.WithLabelValues(vm, snapshot, namespace).Write(dto); err != nil {
		return 0, err
	}
	return *dto.Gauge.Value, nil
}
