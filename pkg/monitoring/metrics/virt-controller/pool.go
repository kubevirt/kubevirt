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
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	vmPoolMetrics = []operatormetrics.Metric{
		vmPoolVMsStartedTotal,
		vmPoolVMsStoppedTotal,
		vmPoolAutoHealingOperationsTotal,
	}

	vmPoolVMsStartedTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_vms_started_total",
			Help: "Total number of VMs started per pool.",
		},
		[]string{"pool_name", "namespace"},
	)

	vmPoolVMsStoppedTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_vms_stopped_total",
			Help: "Total number of VMs stopped per pool.",
		},
		[]string{"pool_name", "namespace"},
	)

	vmPoolAutoHealingOperationsTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_auto_healing_operations_total",
			Help: "Total number of successful auto healing operations performed by the VMPool.",
		},
		[]string{"pool_name", "namespace"},
	)
)

func RecordVMPoolVMStarted(poolName, namespace string) {
	vmPoolVMsStartedTotal.WithLabelValues(poolName, namespace).Inc()
}

func RecordVMPoolVMStopped(poolName, namespace string) {
	vmPoolVMsStoppedTotal.WithLabelValues(poolName, namespace).Inc()
}

func RecordVMPoolAutoHealingOperation(poolName, namespace string) {
	vmPoolAutoHealingOperationsTotal.WithLabelValues(poolName, namespace).Inc()
}
