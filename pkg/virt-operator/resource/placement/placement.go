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
func InjectPlacementMetadata(componentConfig *v1.ComponentConfig, podSpec *corev1.PodSpec, nodePlacementOption DefaultInfraComponentsNodePlacement) {
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
			log.Log.Errorf("Unknown nodePlacementOption %d provided to InjectPlacementMetadata. Falling back to the AnyNode option", nodePlacementOption)
			componentConfig = &v1.ComponentConfig{NodePlacement: &v1.NodePlacement{}}
		}
	}

	nodePlacement := componentConfig.NodePlacement
	if len(nodePlacement.NodeSelector) == 0 {
		nodePlacement.NodeSelector = make(map[string]string)
	}
	if _, ok := nodePlacement.NodeSelector[KubernetesOSLabel]; !ok {
		nodePlacement.NodeSelector[KubernetesOSLabel] = KubernetesOSLinux
	}
	if len(podSpec.NodeSelector) == 0 {
		podSpec.NodeSelector = make(map[string]string, len(nodePlacement.NodeSelector))
	}
	// podSpec.NodeSelector
	for nsKey, nsVal := range nodePlacement.NodeSelector {
		// Favor podSpec over NodePlacement. This prevents cluster admin from clobbering
		// node selectors that KubeVirt intentionally set.
		if _, ok := podSpec.NodeSelector[nsKey]; !ok {
			podSpec.NodeSelector[nsKey] = nsVal
		}
	}

	// podSpec.Affinity
	if nodePlacement.Affinity != nil {
		if podSpec.Affinity == nil {
			podSpec.Affinity = nodePlacement.Affinity.DeepCopy()
		} else {
			// podSpec.Affinity.NodeAffinity
			if nodePlacement.Affinity.NodeAffinity != nil {
				if podSpec.Affinity.NodeAffinity == nil {
					podSpec.Affinity.NodeAffinity = nodePlacement.Affinity.NodeAffinity.DeepCopy()
				} else {
					// need to copy all affinity terms one by one
					if nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
						if podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
							podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.DeepCopy()
						} else {
							// merge the list of terms from NodePlacement into podSpec
							for _, term := range nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
								podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, term)
							}
						}
					}

					//PreferredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
					}

				}
			}
			// podSpec.Affinity.PodAffinity
			if nodePlacement.Affinity.PodAffinity != nil {
				if podSpec.Affinity.PodAffinity == nil {
					podSpec.Affinity.PodAffinity = nodePlacement.Affinity.PodAffinity.DeepCopy()
				} else {
					//RequiredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, term)
					}
					//PreferredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
					}
				}
			}
			// podSpec.Affinity.PodAntiAffinity
			if nodePlacement.Affinity.PodAntiAffinity != nil {
				if podSpec.Affinity.PodAntiAffinity == nil {
					podSpec.Affinity.PodAntiAffinity = nodePlacement.Affinity.PodAntiAffinity.DeepCopy()
				} else {
					//RequiredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, term)
					}
					//PreferredDuringSchedulingIgnoredDuringExecution
					for _, term := range nodePlacement.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
						podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
					}
				}
			}
		}
	}

	//podSpec.Tolerations
	if len(nodePlacement.Tolerations) != 0 {
		if len(podSpec.Tolerations) == 0 {
			podSpec.Tolerations = []corev1.Toleration{}
		}
		for _, toleration := range nodePlacement.Tolerations {
			podSpec.Tolerations = append(podSpec.Tolerations, toleration)
		}
	}
}
