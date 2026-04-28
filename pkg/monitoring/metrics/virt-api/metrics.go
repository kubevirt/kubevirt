/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtapi

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
		connectionMetrics,
		vmMetrics,
	)
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
