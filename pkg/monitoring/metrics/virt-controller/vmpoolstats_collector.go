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
	"slices"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"

	poolv1 "kubevirt.io/api/pool/v1beta1"
)

const defaultPoolReplicas = 1

var (
	vmPoolStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmPoolInfo,
			vmPoolDesiredReplicas,
			vmPoolReplicas,
			vmPoolReadyReplicas,
			vmPoolPaused,
			vmPoolReplicaFailure,
		},
		CollectCallback: vmPoolStatsCollectorCallback,
	}

	vmPoolInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_info",
			Help: "Information about VirtualMachinePools.",
		},
		[]string{"namespace", "name", "uid"},
	)

	vmPoolDesiredReplicas = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_desired_replicas",
			Help: "Desired number of VirtualMachine replicas in a VirtualMachinePool " +
				"(spec.replicas, default 1).",
		},
		[]string{"namespace", "name"},
	)

	vmPoolReplicas = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_replicas",
			Help: "Current number of VirtualMachine replicas in a VirtualMachinePool " +
				"(status.replicas).",
		},
		[]string{"namespace", "name"},
	)

	vmPoolReadyReplicas = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_ready_replicas",
			Help: "Number of ready VirtualMachine replicas in a VirtualMachinePool " +
				"(status.readyReplicas).",
		},
		[]string{"namespace", "name"},
	)

	vmPoolPaused = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_paused",
			Help: "Whether a VirtualMachinePool is paused (spec.paused). 1 if paused, " +
				"0 otherwise.",
		},
		[]string{"namespace", "name"},
	)

	vmPoolReplicaFailure = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmpool_replica_failure",
			Help: "Whether a VirtualMachinePool has a ReplicaFailure condition set to " +
				"True. 1 if failing, 0 otherwise.",
		},
		[]string{"namespace", "name"},
	)
)

func vmPoolStatsCollectorCallback() []operatormetrics.CollectorResult {
	if stores == nil {
		return []operatormetrics.CollectorResult{}
	}
	return reportVMPoolStats(listStoreObjects[poolv1.VirtualMachinePool](stores.VMPool))
}

func reportVMPoolStats(pools []*poolv1.VirtualMachinePool) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult
	for _, pool := range pools {
		results = append(results, collectVMPoolStats(pool)...)
	}
	return results
}

func collectVMPoolStats(pool *poolv1.VirtualMachinePool) []operatormetrics.CollectorResult {
	identity := []string{pool.Namespace, pool.Name}
	return []operatormetrics.CollectorResult{
		{
			Metric: vmPoolInfo,
			Value:  1,
			Labels: []string{pool.Namespace, pool.Name, string(pool.UID)},
		},
		{
			Metric: vmPoolDesiredReplicas,
			Value:  poolDesiredReplicas(pool),
			Labels: identity,
		},
		{
			Metric: vmPoolReplicas,
			Value:  float64(pool.Status.Replicas),
			Labels: identity,
		},
		{
			Metric: vmPoolReadyReplicas,
			Value:  float64(pool.Status.ReadyReplicas),
			Labels: identity,
		},
		{
			Metric: vmPoolPaused,
			Value:  boolGaugeValue(pool.Spec.Paused),
			Labels: identity,
		},
		{
			Metric: vmPoolReplicaFailure,
			Value:  boolGaugeValue(poolHasReplicaFailure(pool)),
			Labels: identity,
		},
	}
}

func poolDesiredReplicas(pool *poolv1.VirtualMachinePool) float64 {
	if pool.Spec.Replicas == nil {
		return defaultPoolReplicas
	}
	return float64(*pool.Spec.Replicas)
}

func poolHasReplicaFailure(pool *poolv1.VirtualMachinePool) bool {
	return slices.ContainsFunc(pool.Status.Conditions, func(condition poolv1.VirtualMachinePoolCondition) bool {
		return condition.Type == poolv1.VirtualMachinePoolReplicaFailure &&
			condition.Status == k8sv1.ConditionTrue
	})
}
