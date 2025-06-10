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
 */

package virt_operator

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
