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

package placement_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/placement"
)

var _ = Describe("Placement", func() {
	Context("on calling InjectPlacementMetadata", func() {
		var componentConfig *v1.ComponentConfig
		var nodePlacement *v1.NodePlacement
		var podSpec *corev1.PodSpec
		var toleration corev1.Toleration
		var toleration2 corev1.Toleration
		var affinity *corev1.Affinity
		var affinity2 *corev1.Affinity

		BeforeEach(func() {
			componentConfig = &v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{},
			}
			nodePlacement = componentConfig.NodePlacement
			podSpec = &corev1.PodSpec{}

			toleration = corev1.Toleration{
				Key:      "test-taint",
				Operator: "Exists",
				Effect:   "NoSchedule",
			}
			toleration2 = corev1.Toleration{
				Key:      "test-taint2",
				Operator: "Exists",
				Effect:   "NoSchedule",
			}

			affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "required",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "preferred",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
				},
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"required": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"preferred": "term"},
								},
							},
						},
					},
				},
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"anti-required": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"anti-preferred": "term"},
								},
							},
						},
					},
				},
			}

			affinity2 = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "required2",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "preferred2",
										Operator: "in",
										Values:   []string{"test"},
									},
								},
							},
						},
					},
				},
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"required2": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"preferred2": "term"},
								},
							},
						},
					},
				},
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"anti-required2": "term"},
							},
						},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"anti-preferred2": "term"},
								},
							},
						},
					},
				},
			}

		})

		// Node Selectors
		It("should succeed if componentConfig is nil", func() {
			// if componentConfig is nil
			placement.InjectPlacementMetadata(nil, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal(placement.KubernetesOSLinux))
		})

		It("should succeed if nodePlacement is nil", func() {
			componentConfig.NodePlacement = nil
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal(placement.KubernetesOSLinux))
		})

		It("should succeed if podSpec is nil", func() {
			orig := componentConfig.DeepCopy()
			orig.NodePlacement.NodeSelector = map[string]string{placement.KubernetesOSLabel: placement.KubernetesOSLinux}
			placement.InjectPlacementMetadata(componentConfig, nil, placement.AnyNode)
			Expect(equality.Semantic.DeepEqual(orig, componentConfig)).To(BeTrue())
		})

		It("should copy NodeSelectors when podSpec is empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal(placement.KubernetesOSLinux))
		})

		It("should merge NodeSelectors when podSpec is not empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["existing"] = "value"
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(3))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
			Expect(podSpec.NodeSelector["existing"]).To(Equal("value"))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal(placement.KubernetesOSLinux))
		})

		It("should favor podSpec if NodeSelectors collide", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("from-podspec"))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal(placement.KubernetesOSLinux))
		})

		It("should set OS label if not defined", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal(placement.KubernetesOSLinux))
		})

		It("should favor NodeSelector OS label if present", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector[placement.KubernetesOSLabel] = "linux-custom"
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal("linux-custom"))
		})

		It("should favor podSpec OS label if present", func() {
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector[placement.KubernetesOSLabel] = "linux-custom"
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal("linux-custom"))
		})

		It("should preserve NodeSelectors if nodePlacement has none", func() {
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("from-podspec"))
			Expect(podSpec.NodeSelector[placement.KubernetesOSLabel]).To(Equal(placement.KubernetesOSLinux))
		})

		// tolerations
		It("should copy tolerations when podSpec is empty", func() {
			toleration := corev1.Toleration{
				Key:      "test-taint",
				Operator: "Exists",
				Effect:   "NoSchedule",
			}
			nodePlacement.Tolerations = []corev1.Toleration{toleration}
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Tolerations).To(HaveLen(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should preserve tolerations when nodePlacement is empty", func() {
			podSpec.Tolerations = []corev1.Toleration{toleration}
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Tolerations).To(HaveLen(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should merge tolerations when both are defined", func() {
			nodePlacement.Tolerations = []corev1.Toleration{toleration}
			podSpec.Tolerations = []corev1.Toleration{toleration2}
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Tolerations).To(HaveLen(2))
		})

		It("It should copy NodePlacement if podSpec Affinity is empty", func() {
			nodePlacement.Affinity = affinity
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(equality.Semantic.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should copy NodePlacement if Node, Pod and Anti affinities are empty", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = &corev1.Affinity{}
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(equality.Semantic.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should merge NodePlacement and podSpec affinity terms", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = affinity2
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(2))
			Expect(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(2))
			Expect(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(2))
			Expect(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(2))
			Expect(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(2))
			Expect(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(2))
			Expect(equality.Semantic.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeFalse())
		})

		It("It should copy Required NodeAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.DeepCopy()
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))
		})

		It("It should copy Preferred NodeAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			nodePlacement.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Required PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Preferred PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Required PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Preferred PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			placement.InjectPlacementMetadata(componentConfig, podSpec, placement.AnyNode)
			Expect(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		Context("default scheduling to CP nodes", func() {
			DescribeTable("should require scheduling to control-plane nodes and avoid worker nodes", func(config *v1.ComponentConfig) {
				const (
					masterLabel       = "node-role.kubernetes.io/master"
					controlPlaceLabel = "node-role.kubernetes.io/control-plane"
					workerLabel       = "node-role.kubernetes.io/worker"
				)

				placement.InjectPlacementMetadata(config, podSpec, placement.RequireControlPlanePreferNonWorker)

				Expect(podSpec.Affinity).ToNot(BeNil())
				Expect(podSpec.Affinity.NodeAffinity).ToNot(BeNil())

				By("Testing scheduling requirements")
				Expect(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).ToNot(BeNil())
				Expect(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(2))

				requirementSelectorTerm := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
				Expect(requirementSelectorTerm.MatchFields).To(BeEmpty())
				Expect(requirementSelectorTerm.MatchExpressions).To(HaveLen(1))

				requirementMatchExpression := requirementSelectorTerm.MatchExpressions[0]
				Expect(requirementMatchExpression.Key).To(Equal(controlPlaceLabel))
				Expect(requirementMatchExpression.Operator).To(Equal(corev1.NodeSelectorOpExists))

				requirementSelectorTerm = podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[1]
				Expect(requirementSelectorTerm.MatchFields).To(BeEmpty())
				Expect(requirementSelectorTerm.MatchExpressions).To(HaveLen(1))

				requirementMatchExpression = requirementSelectorTerm.MatchExpressions[0]
				Expect(requirementMatchExpression.Key).To(Equal(masterLabel))
				Expect(requirementMatchExpression.Operator).To(Equal(corev1.NodeSelectorOpExists))

				By("Testing scheduling preferences")
				Expect(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
				preferredSchedulingTerm := podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0]
				Expect(preferredSchedulingTerm.Weight).To(Equal(int32(100)))
				Expect(preferredSchedulingTerm.Preference.MatchFields).To(BeEmpty())
				Expect(preferredSchedulingTerm.Preference.MatchExpressions).To(HaveLen(1))

				matchExpression := preferredSchedulingTerm.Preference.MatchExpressions[0]
				Expect(matchExpression.Key).To(Equal(workerLabel))
				Expect(matchExpression.Operator).To(Equal(corev1.NodeSelectorOpDoesNotExist))

				Expect(podSpec.Tolerations).ToNot(BeEmpty())
				Expect(podSpec.Tolerations).To(ContainElements(
					corev1.Toleration{
						Key:      "node-role.kubernetes.io/control-plane",
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					},
					corev1.Toleration{
						Key:      "node-role.kubernetes.io/master",
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					},
				))
			},
				Entry("nil component config", nil),
				Entry("nil node placement", &v1.ComponentConfig{}),
			)

			Context("should not add requirement scheduling to control-plane nodes", func() {
				It("if the user already provided a node placement", func() {
					componentConfig = &v1.ComponentConfig{
						NodePlacement: &v1.NodePlacement{
							Tolerations: []corev1.Toleration{},
						},
					}
					placement.InjectPlacementMetadata(componentConfig, podSpec, placement.RequireControlPlanePreferNonWorker)

					Expect(podSpec.Affinity).To(BeNil())
				})

				It("should not add requirement scheduling to control-plane nodes if the corresponding parameter is set to false", func() {
					placement.InjectPlacementMetadata(nil, podSpec, placement.AnyNode)
					Expect(podSpec.Affinity).To(BeNil())
				})
			})

		})
	})
})
