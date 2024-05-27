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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package apply

import (
	"bufio"
	"bytes"

	installstrategy "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	marshalutil "kubevirt.io/kubevirt/tools/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

type MockStore struct {
	get interface{}
}

func (m *MockStore) Add(_ interface{}) error    { return nil }
func (m *MockStore) Update(_ interface{}) error { return nil }
func (m *MockStore) Delete(_ interface{}) error { return nil }
func (m *MockStore) List() []interface{}        { return nil }
func (m *MockStore) ListKeys() []string         { return nil }
func (m *MockStore) Get(_ interface{}) (item interface{}, exists bool, err error) {
	item = m.get
	if m.get != nil {
		exists = true
	}
	return
}
func (m *MockStore) GetByKey(_ string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}
func (m *MockStore) Replace([]interface{}, string) error { return nil }
func (m *MockStore) Resync() error                       { return nil }

const (
	Namespace = "ns"
	Version   = "1.0"
	Registry  = "rep"
	Id        = "42"
)

func getConfig(registry, version string) *util.KubeVirtDeploymentConfig {
	return util.GetTargetConfigFromKV(&v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
		},
		Spec: v1.KubeVirtSpec{
			ImageRegistry: registry,
			ImageTag:      version,
		},
	}, nil)
}

func loadTargetStrategy(resource interface{}, config *util.KubeVirtDeploymentConfig, stores util.Stores) *install.Strategy {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	marshalutil.MarshallObject(resource, writer)
	writer.Flush()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubevirt-install-strategy-",
			Namespace:    config.GetNamespace(),
			Labels: map[string]string{
				v1.ManagedByLabel:       v1.ManagedByLabelOperatorValue,
				v1.InstallStrategyLabel: "",
			},
			Annotations: map[string]string{
				v1.InstallStrategyVersionAnnotation:    config.GetKubeVirtVersion(),
				v1.InstallStrategyRegistryAnnotation:   config.GetImageRegistry(),
				v1.InstallStrategyIdentifierAnnotation: config.GetDeploymentID(),
			},
		},
		Data: map[string]string{
			"manifests": string(b.Bytes()),
		},
	}

	stores.InstallStrategyConfigMapCache.Add(configMap)
	targetStrategy, err := installstrategy.LoadInstallStrategyFromCache(stores, config)
	Expect(err).ToNot(HaveOccurred())

	return targetStrategy
}

var _ = Describe("Apply", func() {

	Context("should calculate", func() {

		DescribeTable("update path based on semver", func(target string, current string, expected bool) {
			takeUpdatePath := shouldTakeUpdatePath(target, current)

			Expect(takeUpdatePath).To(Equal(expected))
		},
			Entry("with increasing semver", "v0.15.0", "v0.14.0", true),
			Entry("with decreasing semver", "v0.14.0", "v0.15.0", false),
			Entry("with identical semver", "v0.15.0", "v0.15.0", false),
			Entry("with invalid semver", "devel", "v0.14.0", true),
			Entry("with increasing semver no prefix", "0.15.0", "0.14.0", true),
			Entry("with decreasing semver no prefix", "0.14.0", "0.15.0", false),
			Entry("with identical semver no prefix", "0.15.0", "0.15.0", false),
			Entry("with invalid semver no prefix", "devel", "0.14.0", true),
			Entry("with no current no prefix", "devel", "", false),
		)
	})

	Context("Injecting Metadata", func() {

		It("should set expected values", func() {

			kv := &v1.KubeVirt{}
			kv.Status.TargetKubeVirtRegistry = Registry
			kv.Status.TargetKubeVirtVersion = Version
			kv.Status.TargetDeploymentID = Id

			deployment := appsv1.Deployment{}
			injectOperatorMetadata(kv, &deployment.ObjectMeta, "fakeversion", "fakeregistry", "fakeid", false)

			// NOTE we are purposfully not using the defined constant values
			// in types.go here. This test is explicitly verifying that those
			// values in types.go that we depend on for virt-operator updates
			// do not change. This is meant to preserve backwards and forwards
			// compatibility

			managedBy, ok := deployment.Labels["app.kubernetes.io/managed-by"]

			Expect(ok).To(BeTrue())
			Expect(managedBy).To(Equal("virt-operator"))

			version, ok := deployment.Annotations["kubevirt.io/install-strategy-version"]
			Expect(ok).To(BeTrue())
			Expect(version).To(Equal("fakeversion"))

			registry, ok := deployment.Annotations["kubevirt.io/install-strategy-registry"]
			Expect(ok).To(BeTrue())
			Expect(registry).To(Equal("fakeregistry"))

			id, ok := deployment.Annotations["kubevirt.io/install-strategy-identifier"]
			Expect(ok).To(BeTrue())
			Expect(id).To(Equal("fakeid"))
		})
	})

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
			InjectPlacementMetadata(nil, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should succeed if nodePlacement is nil", func() {
			componentConfig.NodePlacement = nil
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should succeed if podSpec is nil", func() {
			orig := componentConfig.DeepCopy()
			orig.NodePlacement.NodeSelector = map[string]string{kubernetesOSLabel: kubernetesOSLinux}
			InjectPlacementMetadata(componentConfig, nil, AnyNode)
			Expect(equality.Semantic.DeepEqual(orig, componentConfig)).To(BeTrue())
		})

		It("should copy NodeSelectors when podSpec is empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should merge NodeSelectors when podSpec is not empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["existing"] = "value"
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(3))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
			Expect(podSpec.NodeSelector["existing"]).To(Equal("value"))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should favor podSpec if NodeSelectors collide", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("from-podspec"))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should set OS label if not defined", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should favor NodeSelector OS label if present", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector[kubernetesOSLabel] = "linux-custom"
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal("linux-custom"))
		})

		It("should favor podSpec OS label if present", func() {
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector[kubernetesOSLabel] = "linux-custom"
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal("linux-custom"))
		})

		It("should preserve NodeSelectors if nodePlacement has none", func() {
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.NodeSelector).To(HaveLen(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("from-podspec"))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		// tolerations
		It("should copy tolerations when podSpec is empty", func() {
			toleration := corev1.Toleration{
				Key:      "test-taint",
				Operator: "Exists",
				Effect:   "NoSchedule",
			}
			nodePlacement.Tolerations = []corev1.Toleration{toleration}
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Tolerations).To(HaveLen(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should preserve tolerations when nodePlacement is empty", func() {
			podSpec.Tolerations = []corev1.Toleration{toleration}
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Tolerations).To(HaveLen(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should merge tolerations when both are defined", func() {
			nodePlacement.Tolerations = []corev1.Toleration{toleration}
			podSpec.Tolerations = []corev1.Toleration{toleration2}
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Tolerations).To(HaveLen(2))
		})

		It("It should copy NodePlacement if podSpec Affinity is empty", func() {
			nodePlacement.Affinity = affinity
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(equality.Semantic.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should copy NodePlacement if Node, Pod and Anti affinities are empty", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = &corev1.Affinity{}
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(equality.Semantic.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should merge NodePlacement and podSpec affinity terms", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = affinity2
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
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
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))
		})

		It("It should copy Preferred NodeAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			nodePlacement.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Required PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Preferred PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Required PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		It("It should copy Preferred PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			InjectPlacementMetadata(componentConfig, podSpec, AnyNode)
			Expect(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
		})

		Context("default scheduling to CP nodes", func() {
			DescribeTable("should require scheduling to control-plane nodes and avoid worker nodes", func(config *v1.ComponentConfig) {
				const (
					masterLabel       = "node-role.kubernetes.io/master"
					controlPlaceLabel = "node-role.kubernetes.io/control-plane"
					workerLabel       = "node-role.kubernetes.io/worker"
				)

				InjectPlacementMetadata(config, podSpec, RequireControlPlanePreferNonWorker)

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
					InjectPlacementMetadata(componentConfig, podSpec, RequireControlPlanePreferNonWorker)

					Expect(podSpec.Affinity).To(BeNil())
				})

				It("should not add requirement scheduling to control-plane nodes if the corresponding parameter is set to false", func() {
					InjectPlacementMetadata(nil, podSpec, AnyNode)
					Expect(podSpec.Affinity).To(BeNil())
				})
			})

		})
	})
})
