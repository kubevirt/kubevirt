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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	secv1 "github.com/openshift/api/security/v1"
	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/controller"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply Apps", func() {

	Context("on calling syncPodDisruptionBudgetForDeployment", func() {

		var deployment *appsv1.Deployment
		var err error
		var clientset *kubecli.MockKubevirtClient
		var kv *v1.KubeVirt
		var expectations *util.Expectations
		var stores util.Stores
		var mockPodDisruptionBudgetCacheStore *MockStore
		var pdbClient *fake.Clientset
		var cachedPodDisruptionBudget *v1beta1.PodDisruptionBudget
		var patched bool
		var shouldPatchFail bool
		var created bool
		var shouldCreateFail bool
		var ctrl *gomock.Controller

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

			patched = false
			shouldPatchFail = false
			created = false
			shouldCreateFail = false

			pdbClient = fake.NewSimpleClientset()

			pdbClient.Fake.PrependReactor("patch", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				_, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				if shouldPatchFail {
					return true, nil, fmt.Errorf("Patch failed!")
				}
				patched = true
				return true, &v1beta1.PodDisruptionBudget{}, nil
			})

			pdbClient.Fake.PrependReactor("create", "poddisruptionbudgets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				if shouldCreateFail {
					return true, nil, fmt.Errorf("Create failed!")
				}
				created = true
				return true, update.GetObject(), nil
			})

			stores = util.Stores{}
			mockPodDisruptionBudgetCacheStore = &MockStore{}
			stores.PodDisruptionBudgetCache = mockPodDisruptionBudgetCacheStore

			expectations = &util.Expectations{}
			expectations.PodDisruptionBudget = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("PodDisruptionBudgets"))

			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
			clientset.EXPECT().PolicyV1beta1().Return(pdbClient.PolicyV1beta1()).AnyTimes()
			kv = &v1.KubeVirt{}

			deployment, err = components.NewApiServerDeployment(Namespace, Registry, "", Version, "", "", "", corev1.PullIfNotPresent, "verbosity", map[string]string{})
			Expect(err).ToNot(HaveOccurred())

			cachedPodDisruptionBudget = components.NewPodDisruptionBudgetForDeployment(deployment)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should not fail creation", func() {
			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: expectations,
				stores:       stores,
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)

			Expect(created).To(BeTrue())
			Expect(patched).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not fail patching", func() {
			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget
			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: expectations,
				stores:       stores,
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)

			Expect(patched).To(BeTrue())
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should skip patching of same version", func() {
			kv.Status.TargetKubeVirtRegistry = Registry
			kv.Status.TargetKubeVirtVersion = Version
			kv.Status.TargetDeploymentID = Id

			SetGeneration(&kv.Status.Generations, cachedPodDisruptionBudget)
			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget
			injectOperatorMetadata(kv, &cachedPodDisruptionBudget.ObjectMeta, Version, Registry, Id, true)
			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: expectations,
				stores:       stores,
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)

			Expect(created).To(BeFalse())
			Expect(patched).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return create error", func() {
			shouldCreateFail = true
			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: expectations,
				stores:       stores,
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)

			Expect(err).To(HaveOccurred())
			Expect(created).To(BeFalse())
			Expect(patched).To(BeFalse())
		})

		It("should return patch error", func() {
			shouldPatchFail = true
			mockPodDisruptionBudgetCacheStore.get = cachedPodDisruptionBudget
			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: expectations,
				stores:       stores,
			}
			err = r.syncPodDisruptionBudgetForDeployment(deployment)

			Expect(err).To(HaveOccurred())
			Expect(created).To(BeFalse())
			Expect(patched).To(BeFalse())
		})
	})

	Context("setting virt-handler maxDevices flag ", func() {

		var daemonSet *appsv1.DaemonSet
		var err error
		var clientset *kubecli.MockKubevirtClient
		var kv *v1.KubeVirt
		var expectations *util.Expectations
		var stores util.Stores
		var mockDSCacheStore *MockStore
		var dsClient *fake.Clientset

		var ctrl *gomock.Controller

		vmiPerNode := 10

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

			dsClient = fake.NewSimpleClientset()

			stores = util.Stores{}
			mockDSCacheStore = &MockStore{}
			stores.DaemonSetCache = mockDSCacheStore

			expectations = &util.Expectations{}
			expectations.DaemonSet = controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectationsWithName("DaemonSet"))

			clientset = kubecli.NewMockKubevirtClient(ctrl)
			clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
			clientset.EXPECT().AppsV1().Return(dsClient.AppsV1()).AnyTimes()
			kv = &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: Namespace,
				},
			}

			daemonSet, err = components.NewHandlerDaemonSet(Namespace, Registry, "", Version, "", "", "", "", corev1.PullIfNotPresent, "verbosity", map[string]string{})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should create with maxDevices Set", func() {
			kv.Spec.Configuration.VirtualMachineInstancesPerNode = &vmiPerNode
			created := false
			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: expectations,
				stores:       stores,
			}

			dsClient.Fake.PrependReactor("create", "daemonsets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				created = true

				ds := update.GetObject().(*appsv1.DaemonSet)

				command := ds.Spec.Template.Spec.Containers[0].Command
				Expect(strings.Join(command, " ")).To(ContainSubstring("--maxDevices 10"))

				return true, update.GetObject(), nil
			})

			err = r.syncDaemonSet(daemonSet)

			Expect(err).ToNot(HaveOccurred())
			Expect(created).To(BeTrue())
		})

		It("should patch DS with maxDevices and then remove it", func() {
			mockDSCacheStore.get = daemonSet
			SetGeneration(&kv.Status.Generations, daemonSet)
			patched := false
			containMaxDeviceFlag := false

			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: expectations,
				stores:       stores,
			}

			// add VirtualMachineInstancesPerNode configuration
			kv.Spec.Configuration.VirtualMachineInstancesPerNode = &vmiPerNode
			containMaxDeviceFlag = true
			kv.SetGeneration(2)

			dsClient.Fake.PrependReactor("patch", "daemonsets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				a, ok := action.(testing.PatchAction)
				Expect(ok).To(BeTrue())
				patched = true

				patches := []utiltypes.PatchOperation{}
				json.Unmarshal(a.GetPatch(), &patches)

				var dsSpec *appsv1.DaemonSetSpec
				for _, v := range patches {
					if v.Path == "/spec" && v.Op == "replace" {
						dsSpec = &appsv1.DaemonSetSpec{}
						template, err := json.Marshal(v.Value)
						Expect(err).ToNot(HaveOccurred())
						json.Unmarshal(template, dsSpec)
					}
				}

				Expect(dsSpec).ToNot(BeNil())

				command := dsSpec.Template.Spec.Containers[0].Command
				if containMaxDeviceFlag {
					Expect(strings.Join(command, " ")).To(ContainSubstring("--maxDevices 10"))
				} else {
					Expect(strings.Join(command, " ")).ToNot(ContainSubstring("--maxDevices 10"))
				}

				return true, &appsv1.DaemonSet{}, nil
			})

			err = r.syncDaemonSet(daemonSet)

			Expect(patched).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			// remove VirtualMachineInstancesPerNode configuration
			patched = false
			kv.Spec.Configuration.VirtualMachineInstancesPerNode = nil
			containMaxDeviceFlag = false
			kv.SetGeneration(3)

			err = r.syncDaemonSet(daemonSet)

			Expect(patched).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})
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
			Expect(managedBy).To(Equal("kubevirt-operator"))

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

	Context("on calling injectPlacementMetadata", func() {
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
			injectPlacementMetadata(nil, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should succeed if nodePlacement is nil", func() {
			componentConfig.NodePlacement = nil
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should succeed if podSpec is nil", func() {
			orig := componentConfig.DeepCopy()
			orig.NodePlacement.NodeSelector = map[string]string{kubernetesOSLabel: kubernetesOSLinux}
			injectPlacementMetadata(componentConfig, nil)
			Expect(reflect.DeepEqual(orig, componentConfig)).To(BeTrue())
		})

		It("should copy NodeSelectors when podSpec is empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should merge NodeSelectors when podSpec is not empty", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["existing"] = "value"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(3))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("bar"))
			Expect(podSpec.NodeSelector["existing"]).To(Equal("value"))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should favor podSpec if NodeSelectors collide", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector["foo"] = "bar"
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(2))
			Expect(podSpec.NodeSelector["foo"]).To(Equal("from-podspec"))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should set OS label if not defined", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal(kubernetesOSLinux))
		})

		It("should favor NodeSelector OS label if present", func() {
			nodePlacement.NodeSelector = make(map[string]string)
			nodePlacement.NodeSelector[kubernetesOSLabel] = "linux-custom"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal("linux-custom"))
		})

		It("should favor podSpec OS label if present", func() {
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector[kubernetesOSLabel] = "linux-custom"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(1))
			Expect(podSpec.NodeSelector[kubernetesOSLabel]).To(Equal("linux-custom"))
		})

		It("should preserve NodeSelectors if nodePlacement has none", func() {
			podSpec.NodeSelector = make(map[string]string)
			podSpec.NodeSelector["foo"] = "from-podspec"
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.NodeSelector)).To(Equal(2))
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
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Tolerations)).To(Equal(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should preserve tolerations when nodePlacement is empty", func() {
			podSpec.Tolerations = []corev1.Toleration{toleration}
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Tolerations)).To(Equal(1))
			Expect(podSpec.Tolerations[0].Key).To(Equal("test-taint"))
		})

		It("should merge tolerations when both are defined", func() {
			nodePlacement.Tolerations = []corev1.Toleration{toleration}
			podSpec.Tolerations = []corev1.Toleration{toleration2}
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Tolerations)).To(Equal(2))
		})

		It("It should copy NodePlacement if podSpec Affinity is empty", func() {
			nodePlacement.Affinity = affinity
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(reflect.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should copy NodePlacement if Node, Pod and Anti affinities are empty", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = &corev1.Affinity{}
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(reflect.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeTrue())

		})

		It("It should merge NodePlacement and podSpec affinity terms", func() {
			nodePlacement.Affinity = affinity
			podSpec.Affinity = affinity2
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)).To(Equal(2))
			Expect(len(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(len(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
			Expect(reflect.DeepEqual(nodePlacement.Affinity, podSpec.Affinity)).To(BeFalse())
		})

		It("It should copy Required NodeAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			nodePlacement.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.DeepCopy()
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)).To(Equal(1))
		})

		It("It should copy Preferred NodeAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			nodePlacement.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Required PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Preferred PodAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAffinity = &corev1.PodAffinity{}
			nodePlacement.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Required PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})

		It("It should copy Preferred PodAntiAffinity", func() {
			nodePlacement.Affinity = &corev1.Affinity{}
			nodePlacement.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			nodePlacement.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
			injectPlacementMetadata(componentConfig, podSpec)
			Expect(len(podSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
		})
	})

	Context("Manage users in Security Context Constraints", func() {
		var stop chan struct{}
		var ctrl *gomock.Controller
		var stores util.Stores
		var informers util.Informers
		var virtClient *kubecli.MockKubevirtClient
		var secClient *secv1fake.FakeSecurityV1
		var err error

		namespace := "kubevirt-test"

		generateSCC := func(sccName string, usersList []string) *secv1.SecurityContextConstraints {
			return &secv1.SecurityContextConstraints{
				ObjectMeta: v12.ObjectMeta{
					Name: sccName,
				},
				Users: usersList,
			}
		}

		setupPrependReactor := func(sccName string, expectedPatch []byte) {
			secClient.Fake.PrependReactor("patch", "securitycontextconstraints",
				func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					patch, ok := action.(testing.PatchAction)
					Expect(ok).To(BeTrue())
					Expect(patch.GetName()).To(Equal(sccName), "Patch object name should match SCC name")
					Expect(patch.GetPatch()).To(Equal(expectedPatch))
					return true, nil, nil
				})
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			informers.SCC, _ = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
			stores.SCCCache = informers.SCC.GetStore()
			secClient = &secv1fake.FakeSecurityV1{
				Fake: &fake.NewSimpleClientset().Fake,
			}
			virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
		})

		executeTest := func(scc *secv1.SecurityContextConstraints, expectedPatch string) {
			setupPrependReactor(scc.ObjectMeta.Name, []byte(expectedPatch))
			stores.SCCCache.Add(scc)

			r := &Reconciler{
				clientset: virtClient,
				stores:    stores,
			}

			err = r.removeKvServiceAccountsFromDefaultSCC(namespace)
			Expect(err).ToNot(HaveOccurred(), "Should successfully remove only the kubevirt service accounts")
		}

		AfterEach(func() {
			close(stop)
			ctrl.Finish()
		})

		table.DescribeTable("Should remove Kubevirt service accounts from the default privileged SCC", func(additionalUserlist []string) {
			var expectedJsonPatch string
			var serviceAccounts []string
			saMap := rbac.GetKubevirtComponentsServiceAccounts(namespace)
			for key := range saMap {
				serviceAccounts = append(serviceAccounts, key)
			}
			serviceAccounts = append(serviceAccounts, additionalUserlist...)
			scc := generateSCC("privileged", serviceAccounts)
			if len(additionalUserlist) != 0 {
				expectedJsonPatch = fmt.Sprintf(`[ { "op": "test", "path": "/users", "value": ["%s"] }, { "op": "replace", "path": "/users", "value": ["%s"] } ]`, strings.Join(serviceAccounts, `","`), strings.Join(additionalUserlist, `","`))
			} else {
				expectedJsonPatch = fmt.Sprintf(`[ { "op": "test", "path": "/users", "value": ["%s"] }, { "op": "replace", "path": "/users", "value": null } ]`, strings.Join(serviceAccounts, `","`))
			}
			executeTest(scc, expectedJsonPatch)
		},
			table.Entry("Without custom users", []string{}),
			table.Entry("With custom users", []string{"someuser"}),
		)
	})
})
