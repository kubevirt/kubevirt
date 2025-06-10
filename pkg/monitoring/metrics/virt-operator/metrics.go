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

package virt_operator

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
