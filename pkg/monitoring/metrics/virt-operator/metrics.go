/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtoperator

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/client"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/workqueue"
)

func SetupMetrics() error {
	if err := client.SetupMetrics(); err != nil {
		return err
	}

	if err := workqueue.SetupMetrics(); err != nil {
		return err
	}

	return operatormetrics.RegisterMetrics(
		operatorMetrics,
	)
}

func RegisterLeaderMetrics() error {
	if err := operatormetrics.RegisterMetrics(leaderMetrics); err != nil {
		return err
	}

	return nil
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
