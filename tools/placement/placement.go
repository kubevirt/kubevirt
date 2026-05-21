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
package placement

import (
	corev1 "k8s.io/api/core/v1"
)

const KubeVirtControlPlaneLabel = "node-role.kubevirt.io/control-plane"

// InjectKubeVirtControlPlanePlacement appends a node-role.kubevirt.io/control-plane
// NodeSelectorTerm and matching toleration to the given pod spec. The affinity
// chain is initialized if nil so the NodeSelectorTerm is always added.
func InjectKubeVirtControlPlanePlacement(podSpec *corev1.PodSpec) {
	if podSpec.Affinity == nil {
		podSpec.Affinity = &corev1.Affinity{}
	}
	if podSpec.Affinity.NodeAffinity == nil {
		podSpec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}
	podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
		podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
		corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{
				{
					Key:      KubeVirtControlPlaneLabel,
					Operator: corev1.NodeSelectorOpExists,
				},
			},
		},
	)
	podSpec.Tolerations = append(podSpec.Tolerations, corev1.Toleration{
		Key:      KubeVirtControlPlaneLabel,
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	})
}
