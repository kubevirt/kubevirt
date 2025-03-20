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
 */

package vmi

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ Collector = ConditionsCollector{}

type ConditionsCollector struct{}

var (
	conditions = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_conditions",
			Help: "The conditions of the VMI. Values are 1 if the condition is true, 0 otherwise.",
		},
		[]string{"namespace", "name", "type", "reason", "message"},
	)
)

func (c ConditionsCollector) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		conditions,
	}
}

func (c ConditionsCollector) Collect(vmi *k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult

	for _, condition := range vmi.Status.Conditions {
		value := 0.0
		if condition.Status == k8sv1.ConditionTrue {
			value = 1.0
		}

		results = append(results, operatormetrics.CollectorResult{
			Metric: conditions,
			Labels: []string{
				vmi.Namespace,
				vmi.Name,
				string(condition.Type),
				condition.Reason,
				condition.Message,
			},
			Value: value,
		})
	}
	return results
}
