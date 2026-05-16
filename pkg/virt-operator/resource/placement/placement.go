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
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

type DefaultInfraComponentsNodePlacement int

const (
	AnyNode DefaultInfraComponentsNodePlacement = iota
	RequireControlPlanePreferNonWorker
)

const (
	KubernetesOSLabel = corev1.LabelOSStable
	KubernetesOSLinux = "linux"
)

// InjectPlacementMetadata merges all Tolerations, Affinity and NodeSelectors from NodePlacement into pod spec
func InjectPlacementMetadata(
	componentConfig *v1.ComponentConfig,
	podSpec *corev1.PodSpec,
	nodePlacementOption DefaultInfraComponentsNodePlacement,
) {
	if podSpec == nil {
		podSpec = &corev1.PodSpec{}
	}

	if componentConfig == nil || componentConfig.NodePlacement == nil {
		switch nodePlacementOption {
		case AnyNode:
			componentConfig = &v1.ComponentConfig{NodePlacement: &v1.NodePlacement{}}

		case RequireControlPlanePreferNonWorker:
			componentConfig = &v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/control-plane",
												Operator: corev1.NodeSelectorOpExists,
											},
										},
									},
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/master",
												Operator: corev1.NodeSelectorOpExists,
											},
										},
									},
								},
							},
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 100,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/worker",
												Operator: corev1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/control-plane",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
				},
			}

		default:
			log.Log.Errorf(
				"Unknown nodePlacementOption %d provided to InjectPlacementMetadata. Falling back to the AnyNode option",
				nodePlacementOption,
			)
			componentConfig = &v1.ComponentConfig{NodePlacement: &v1.NodePlacement{}}
		}
	}

	nodePlacement := componentConfig.NodePlacement
	setDefaultNodeSelector(nodePlacement)
	mergeNodeSelector(podSpec, nodePlacement)
	mergeAffinity(podSpec, nodePlacement)
	mergeTolerations(podSpec, nodePlacement)
}

func setDefaultNodeSelector(nodePlacement *v1.NodePlacement) {
	if len(nodePlacement.NodeSelector) == 0 {
		nodePlacement.NodeSelector = make(map[string]string)
	}

	if _, ok := nodePlacement.NodeSelector[KubernetesOSLabel]; !ok {
		nodePlacement.NodeSelector[KubernetesOSLabel] = KubernetesOSLinux
	}
}

func mergeNodeSelector(podSpec *corev1.PodSpec, nodePlacement *v1.NodePlacement) {
	if len(podSpec.NodeSelector) == 0 {
		podSpec.NodeSelector = make(map[string]string, len(nodePlacement.NodeSelector))
	}
	for nsKey, nsVal := range nodePlacement.NodeSelector {
		// Favor podSpec over NodePlacement. This prevents cluster admin from clobbering
		// node selectors that KubeVirt intentionally set.
		if _, ok := podSpec.NodeSelector[nsKey]; !ok {
			podSpec.NodeSelector[nsKey] = nsVal
		}
	}
}

func mergeAffinity(podSpec *corev1.PodSpec, nodePlacement *v1.NodePlacement) {
	if nodePlacement.Affinity == nil {
		return
	}
	if podSpec.Affinity == nil {
		podSpec.Affinity = nodePlacement.Affinity.DeepCopy()
		return
	}
	mergeNodeAffinity(podSpec, nodePlacement.Affinity.NodeAffinity)
	mergePodAffinity(podSpec, nodePlacement.Affinity.PodAffinity)
	mergePodAntiAffinity(podSpec, nodePlacement.Affinity.PodAntiAffinity)
}

func mergeNodeAffinity(podSpec *corev1.PodSpec, nodeAffinity *corev1.NodeAffinity) {
	if nodeAffinity == nil {
		return
	}
	if podSpec.Affinity.NodeAffinity == nil {
		podSpec.Affinity.NodeAffinity = nodeAffinity.DeepCopy()
		return
	}
	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		if podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			req := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.DeepCopy()
			podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = req
		} else {
			podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
				podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
				nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms...,
			)
		}
	}
	podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
		nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution...,
	)
}

func mergePodAffinity(podSpec *corev1.PodSpec, podAffinity *corev1.PodAffinity) {
	if podAffinity == nil {
		return
	}
	if podSpec.Affinity.PodAffinity == nil {
		podSpec.Affinity.PodAffinity = podAffinity.DeepCopy()
		return
	}
	podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
		podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		podAffinity.RequiredDuringSchedulingIgnoredDuringExecution...,
	)
	podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
		podAffinity.PreferredDuringSchedulingIgnoredDuringExecution...,
	)
}

func mergePodAntiAffinity(podSpec *corev1.PodSpec, podAntiAffinity *corev1.PodAntiAffinity) {
	if podAntiAffinity == nil {
		return
	}
	if podSpec.Affinity.PodAntiAffinity == nil {
		podSpec.Affinity.PodAntiAffinity = podAntiAffinity.DeepCopy()
		return
	}
	podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
		podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution...,
	)
	podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
		podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution...,
	)
}

func mergeTolerations(podSpec *corev1.PodSpec, nodePlacement *v1.NodePlacement) {
	if len(nodePlacement.Tolerations) == 0 {
		return
	}

	podSpec.Tolerations = append(podSpec.Tolerations, nodePlacement.Tolerations...)
}
