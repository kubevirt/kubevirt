/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtoperator

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	operatorMetrics = []operatormetrics.Metric{
		leaderGauge,
		readyGauge,
	}

	leaderGauge = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_virt_operator_leading_status",
			Help: "Indication for an operating virt-operator.",
		},
	)

	readyGauge = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_virt_operator_ready_status",
			Help: "Indication for a virt-operator that is ready to take the lead.",
		},
	)
)

func SetLeader(isLeader bool) {
	leaderGauge.Set(boolToFloat64(isLeader))
}

func SetReady(isReady bool) {
	readyGauge.Set(boolToFloat64(isReady))
}
