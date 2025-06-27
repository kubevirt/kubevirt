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

package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	secv1 "github.com/openshift/api/security/v1"
	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
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
		var cachedPodDisruptionBudget *policyv1.PodDisruptionBudget
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
				return true, &policyv1.PodDisruptionBudget{}, nil
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
			clientset.EXPECT().PolicyV1().Return(pdbClient.PolicyV1()).AnyTimes()
			kv = &v1.KubeVirt{}

			virtApiConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
			}
			deployment = components.NewApiServerDeployment(
				Namespace,
				virtApiConfig.GetImageRegistry(),
				virtApiConfig.GetImagePrefix(),
				virtApiConfig.GetApiVersion(),
				"",
				"",
				"",
				virtApiConfig.VirtApiImage,
				virtApiConfig.GetImagePullPolicy(),
				virtApiConfig.GetImagePullSecrets(),
				virtApiConfig.GetVerbosity(),
				virtApiConfig.GetExtraEnv())

			cachedPodDisruptionBudget = components.NewPodDisruptionBudgetForDeployment(deployment)
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

	Describe("on updating/creating virt-handler", func() {
		var daemonSet *appsv1.DaemonSet
		var err error
		var clientset *kubecli.MockKubevirtClient
		var kv *v1.KubeVirt
		var expectations *util.Expectations
		var stores util.Stores
		var mockDSCacheStore *MockStore
		var mockPodCacheStore *cache.FakeCustomStore
		var dsClient *fake.Clientset

		var ctrl *gomock.Controller

		createDaemonSetPod := func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet, phase corev1.PodPhase, ready bool) *corev1.Pod {
			version := kv.Status.TargetKubeVirtVersion
			registry := kv.Status.TargetKubeVirtRegistry
			id := kv.Status.TargetDeploymentID

			boolTrue := true
			pod := &corev1.Pod{
				ObjectMeta: v12.ObjectMeta{
					OwnerReferences: []v12.OwnerReference{
						{Name: daemonSet.Name, Controller: &boolTrue, UID: daemonSet.UID},
					},
					Annotations: map[string]string{
						v1.InstallStrategyVersionAnnotation:    version,
						v1.InstallStrategyRegistryAnnotation:   registry,
						v1.InstallStrategyIdentifierAnnotation: id,
					},
				},
				Status: corev1.PodStatus{
					Phase: phase,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: ready},
					},
				},
			}
			return pod
		}

		createCrashedCanaryPod := func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) *corev1.Pod {
			pod := createDaemonSetPod(kv, daemonSet, corev1.PodRunning, false)
			pod.Status.ContainerStatuses[0].RestartCount = 1
			return pod
		}

		markHandlerReady := func(deamonSet *appsv1.DaemonSet) {
			daemonSet.Status.DesiredNumberScheduled = 1
			daemonSet.Status.NumberReady = 1
			pod := createDaemonSetPod(kv, daemonSet, corev1.PodRunning, true)
			mockPodCacheStore.ListFunc = func() []interface{} {
				return []interface{}{pod}
			}
		}

		markHandlerNotReady := func(deamonSet *appsv1.DaemonSet) {
			daemonSet.Status.DesiredNumberScheduled = 1
			daemonSet.Status.NumberReady = 0
		}

		markHandlerCrashed := func(deamonSet *appsv1.DaemonSet) {
			daemonSet.Status.DesiredNumberScheduled = 1
			daemonSet.Status.NumberReady = 0
			pod := createCrashedCanaryPod(kv, daemonSet)
			mockPodCacheStore.ListFunc = func() []interface{} {
				return []interface{}{pod}
			}
		}

		markHandlerCanaryReady := func(deamonSet *appsv1.DaemonSet) {
			daemonSet.Status.DesiredNumberScheduled = 2
			daemonSet.Status.NumberReady = 1
			pod := createDaemonSetPod(kv, daemonSet, corev1.PodRunning, true)
			mockPodCacheStore.ListFunc = func() []interface{} {
				return []interface{}{pod}
			}
		}

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)

			dsClient = fake.NewSimpleClientset()

			stores = util.Stores{}
			mockDSCacheStore = &MockStore{}
			stores.DaemonSetCache = mockDSCacheStore
			mockPodCacheStore = &cache.FakeCustomStore{}
			stores.InfrastructurePodCache = mockPodCacheStore

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

			virtHandlerConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
			}
			daemonSet = components.NewHandlerDaemonSet(
				Namespace,
				virtHandlerConfig.GetImageRegistry(),
				virtHandlerConfig.GetImagePrefix(),
				virtHandlerConfig.GetHandlerVersion(),
				"",
				"",
				"",
				"",
				virtHandlerConfig.GetLauncherVersion(),
				virtHandlerConfig.GetPrHelperVersion(),
				virtHandlerConfig.VirtHandlerImage,
				virtHandlerConfig.VirtLauncherImage,
				virtHandlerConfig.PrHelperImage,
				virtHandlerConfig.SidecarShimImage,
				virtHandlerConfig.GetImagePullPolicy(),
				virtHandlerConfig.GetImagePullSecrets(),
				nil,
				virtHandlerConfig.GetVerbosity(),
				virtHandlerConfig.GetExtraEnv(),
				virtHandlerConfig.GetSpecificHostPath(),
				false)
			markHandlerReady(daemonSet)
			daemonSet.UID = "random-id"
		})

		Context("setting virt-handler maxDevices flag", func() {
			vmiPerNode := 10

			It("should create with maxDevices Set", func() {
				kv.Spec.Configuration.VirtualMachineInstancesPerNode = &vmiPerNode
				created := false
				r := &Reconciler{
					clientset:    clientset,
					kv:           kv,
					expectations: expectations,
					stores:       stores,
					recorder:     record.NewFakeRecorder(100),
				}

				dsClient.Fake.PrependReactor("create", "daemonsets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					update, ok := action.(testing.CreateAction)
					Expect(ok).To(BeTrue())
					created = true

					ds := update.GetObject().(*appsv1.DaemonSet)

					command := ds.Spec.Template.Spec.Containers[0].Command
					Expect(strings.Join(command, " ")).To(ContainSubstring("--max-devices 10"))

					return true, update.GetObject(), nil
				})

				_, err = r.syncDaemonSet(daemonSet)

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
					recorder:     record.NewFakeRecorder(100),
				}

				// add VirtualMachineInstancesPerNode configuration
				kv.Spec.Configuration.VirtualMachineInstancesPerNode = &vmiPerNode
				containMaxDeviceFlag = true
				kv.SetGeneration(2)

				dsClient.Fake.PrependReactor("patch", "daemonsets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					a, ok := action.(testing.PatchAction)
					Expect(ok).To(BeTrue())
					patched = true

					patches, err := patch.UnmarshalPatch(a.GetPatch())
					Expect(err).ToNot(HaveOccurred())

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
						Expect(strings.Join(command, " ")).To(ContainSubstring("--max-devices 10"))
					} else {
						Expect(strings.Join(command, " ")).ToNot(ContainSubstring("--max-devices 10"))
					}

					return true, &appsv1.DaemonSet{}, nil
				})

				_, err = r.syncDaemonSet(daemonSet)

				Expect(patched).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())

				// remove VirtualMachineInstancesPerNode configuration
				patched = false
				kv.Spec.Configuration.VirtualMachineInstancesPerNode = nil
				containMaxDeviceFlag = false
				kv.SetGeneration(3)

				_, err = r.syncDaemonSet(daemonSet)

				Expect(patched).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("updating virt-handler", func() {

			addCustomTargetDeployment := func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {
				version := "custom.version"
				registry := "custom.registry"
				id := "custom.id"
				kv.Status.TargetKubeVirtVersion = version
				kv.Status.TargetKubeVirtRegistry = registry
				kv.Status.TargetDeploymentID = id

				daemonSet.ObjectMeta.Annotations = map[string]string{
					v1.InstallStrategyVersionAnnotation:    version,
					v1.InstallStrategyRegistryAnnotation:   registry,
					v1.InstallStrategyIdentifierAnnotation: id,
				}
			}

			It("should start canary upgrade if updating virt-handler", func() {
				mockDSCacheStore.get = daemonSet
				SetGeneration(&kv.Status.Generations, daemonSet)
				patched := false

				r := &Reconciler{
					clientset:    clientset,
					kv:           kv,
					expectations: expectations,
					stores:       stores,
					recorder:     record.NewFakeRecorder(100),
				}

				dsClient.Fake.PrependReactor("patch", "daemonsets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					a, ok := action.(testing.PatchAction)
					Expect(ok).To(BeTrue())
					patched = true

					patches := []patch.PatchOperation{}
					json.Unmarshal(a.GetPatch(), &patches)

					var annotations map[string]string
					for _, v := range patches {
						if v.Path == "/metadata/annotations" {
							template, err := json.Marshal(v.Value)
							Expect(err).ToNot(HaveOccurred())
							json.Unmarshal(template, &annotations)
						}
					}

					patchedDs := &appsv1.DaemonSet{
						ObjectMeta: v12.ObjectMeta{
							Annotations: annotations,
						},
					}
					Expect(util.DaemonSetIsUpToDate(kv, patchedDs)).To(BeTrue())
					return true, patchedDs, nil
				})

				newDs := daemonSet.DeepCopy()
				addCustomTargetDeployment(kv, newDs)
				done, err := r.syncDaemonSet(newDs)

				Expect(patched).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
				Expect(done).To(BeFalse())
			})

			type daemonSetBuilder func(*v1.KubeVirt, *appsv1.DaemonSet) (current *appsv1.DaemonSet,
				target *appsv1.DaemonSet)
			type daemonSetPatchChecker func(*v1.KubeVirt, *appsv1.DaemonSet)

			DescribeTable("process canary upgrade",
				func(dsBuild daemonSetBuilder,
					dsCheck daemonSetPatchChecker,
					expectedStatus CanaryUpgradeStatus,
					expectedDone bool,
					expectingError bool,
					expectingPatch bool) {
					patched := false

					r := &Reconciler{
						clientset:    clientset,
						kv:           kv,
						expectations: expectations,
						stores:       stores,
						recorder:     record.NewFakeRecorder(100),
					}

					currentDs, newDs := dsBuild(kv, daemonSet)
					mockDSCacheStore.get = daemonSet
					SetGeneration(&kv.Status.Generations, currentDs)

					dsClient.Fake.PrependReactor("patch", "daemonsets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
						a, ok := action.(testing.PatchAction)
						Expect(ok).To(BeTrue())
						patched = true

						patches := []patch.PatchOperation{}
						json.Unmarshal(a.GetPatch(), &patches)

						var annotations map[string]string
						var dsSpec appsv1.DaemonSetSpec
						for _, v := range patches {
							if v.Path == "/spec" {
								template, err := json.Marshal(v.Value)
								Expect(err).ToNot(HaveOccurred())
								json.Unmarshal(template, &dsSpec)
							}
							if v.Path == "/metadata/annotations" {
								template, err := json.Marshal(v.Value)
								Expect(err).ToNot(HaveOccurred())
								json.Unmarshal(template, &annotations)
							}
						}

						patchedDs := &appsv1.DaemonSet{
							ObjectMeta: v12.ObjectMeta{
								Annotations: annotations,
							},
							Spec: dsSpec,
						}

						dsCheck(kv, patchedDs)
						return true, patchedDs, nil
					})

					done, err, status := r.processCanaryUpgrade(currentDs, newDs, false)

					Expect(patched).To(Equal(expectingPatch))
					Expect(done).To(Equal(expectedDone))
					Expect(status).To(Equal(expectedStatus))
					if expectingError {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).ToNot(HaveOccurred())
					}
				},
				Entry("should start canary upgrade with MaxUnavailable 1",
					func(kv *v1.KubeVirt, currentDs *appsv1.DaemonSet) (*appsv1.DaemonSet, *appsv1.DaemonSet) {
						newDs := daemonSet.DeepCopy()
						addCustomTargetDeployment(kv, newDs)
						return currentDs, newDs
					},
					func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {
						Expect(util.DaemonSetIsUpToDate(kv, daemonSet)).To(BeTrue())
						rollingUpdate := daemonSet.Spec.UpdateStrategy.RollingUpdate
						Expect(rollingUpdate).ToNot(BeNil())
						Expect(rollingUpdate.MaxUnavailable).ToNot(BeNil())
						Expect(rollingUpdate.MaxUnavailable.IntValue()).To(Equal(1))
					},
					CanaryUpgradeStatusStarted, false, false, true,
				),
				Entry("should wait for canary pod to be created",
					func(kv *v1.KubeVirt, currentDs *appsv1.DaemonSet) (*appsv1.DaemonSet, *appsv1.DaemonSet) {
						newDs := daemonSet.DeepCopy()
						addCustomTargetDeployment(kv, newDs)
						addCustomTargetDeployment(kv, currentDs)
						markHandlerNotReady(daemonSet)
						return currentDs, newDs
					},
					func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {},
					CanaryUpgradeStatusStarted, false, false, false,
				),
				Entry("should wait for canary pod to be ready",
					func(kv *v1.KubeVirt, currentDs *appsv1.DaemonSet) (*appsv1.DaemonSet, *appsv1.DaemonSet) {
						newDs := daemonSet.DeepCopy()
						addCustomTargetDeployment(kv, newDs)
						addCustomTargetDeployment(kv, currentDs)
						markHandlerNotReady(daemonSet)
						return currentDs, newDs
					},
					func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {},
					CanaryUpgradeStatusStarted, false, false, false,
				),
				Entry("should restart daemonset rollout with MaxUnavailable 10%",
					func(kv *v1.KubeVirt, currentDs *appsv1.DaemonSet) (*appsv1.DaemonSet, *appsv1.DaemonSet) {
						newDs := daemonSet.DeepCopy()
						addCustomTargetDeployment(kv, newDs)
						addCustomTargetDeployment(kv, currentDs)
						markHandlerCanaryReady(daemonSet)
						currentDs.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
							MaxUnavailable: nil,
						}
						return currentDs, newDs
					},
					func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {
						Expect(util.DaemonSetIsUpToDate(kv, daemonSet)).To(BeTrue())
						rollingUpdate := daemonSet.Spec.UpdateStrategy.RollingUpdate
						Expect(rollingUpdate).ToNot(BeNil())
						Expect(rollingUpdate.MaxUnavailable).ToNot(BeNil())
						Expect(rollingUpdate.MaxUnavailable.String()).To(Equal("10%"))
					},
					CanaryUpgradeStatusUpgradingDaemonSet, false, false, true,
				),
				Entry("should report an error when canary pod fails",
					func(kv *v1.KubeVirt, currentDs *appsv1.DaemonSet) (*appsv1.DaemonSet, *appsv1.DaemonSet) {
						newDs := daemonSet.DeepCopy()
						addCustomTargetDeployment(kv, newDs)
						addCustomTargetDeployment(kv, currentDs)
						markHandlerCrashed(daemonSet)
						return currentDs, newDs
					},
					func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {},
					CanaryUpgradeStatusFailed, false, true, false,
				),
				Entry("should wait for new daemonset rollout",
					func(kv *v1.KubeVirt, currentDs *appsv1.DaemonSet) (*appsv1.DaemonSet, *appsv1.DaemonSet) {
						maxUnavailable := intstr.FromString("10%")
						newDs := daemonSet.DeepCopy()
						addCustomTargetDeployment(kv, newDs)
						addCustomTargetDeployment(kv, currentDs)
						markHandlerCanaryReady(daemonSet)
						currentDs.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
							MaxUnavailable: &maxUnavailable,
						}
						return currentDs, newDs
					},
					func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {},
					CanaryUpgradeStatusWaitingDaemonSetRollout, false, false, false,
				),
				Entry("should complete rollout",
					func(kv *v1.KubeVirt, currentDs *appsv1.DaemonSet) (*appsv1.DaemonSet, *appsv1.DaemonSet) {
						maxUnavailable := intstr.FromString("10%")
						newDs := daemonSet.DeepCopy()
						addCustomTargetDeployment(kv, newDs)
						addCustomTargetDeployment(kv, currentDs)
						markHandlerReady(daemonSet)
						currentDs.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
							MaxUnavailable: &maxUnavailable,
						}
						return currentDs, newDs
					},
					func(kv *v1.KubeVirt, daemonSet *appsv1.DaemonSet) {
						Expect(util.DaemonSetIsUpToDate(kv, daemonSet)).To(BeTrue())
						rollingUpdate := daemonSet.Spec.UpdateStrategy.RollingUpdate
						Expect(rollingUpdate).ToNot(BeNil())
						Expect(rollingUpdate.MaxUnavailable).ToNot(BeNil())
						Expect(rollingUpdate.MaxUnavailable.IntValue()).To(Equal(1))
					},
					CanaryUpgradeStatusSuccessful, true, false, true,
				),
			)
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
		})

		DescribeTable("Should remove Kubevirt service accounts from the default privileged SCC", func(additionalUserlist []string) {
			var serviceAccounts []string
			saMap := rbac.GetKubevirtComponentsServiceAccounts(namespace)
			for key := range saMap {
				serviceAccounts = append(serviceAccounts, key)
			}
			serviceAccounts = append(serviceAccounts, additionalUserlist...)
			scc := generateSCC("privileged", serviceAccounts)
			patchSet := patch.New()
			const usersPath = "/users"
			if len(additionalUserlist) != 0 {
				patchSet.AddOption(
					patch.WithTest(usersPath, serviceAccounts),
					patch.WithReplace(usersPath, additionalUserlist),
				)
			} else {
				patchSet.AddOption(
					patch.WithTest(usersPath, serviceAccounts),
					patch.WithReplace(usersPath, nil),
				)
			}
			patches, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred(), "Failed to generate patch payload")
			executeTest(scc, string(patches))
		},
			Entry("Without custom users", []string{}),
			Entry("With custom users", []string{"someuser"}),
		)
	})

	Context("on calling syncDeployment", func() {
		var cachedDeployment *appsv1.Deployment
		var strategyDeployment *appsv1.Deployment
		var clientset *kubecli.MockKubevirtClient
		var kv *v1.KubeVirt
		var stores util.Stores
		var ctrl *gomock.Controller
		const revisionAnnotation = "deployment.kubernetes.io/revision"
		const fakeAnnotation = "fakeAnnotation.io/fake"
		var virtAPIDeployment *appsv1.Deployment
		var dpClient *fake.Clientset

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			kvInterface := kubecli.NewMockKubeVirtInterface(ctrl)
			clientset = kubecli.NewMockKubevirtClient(ctrl)
			dpClient = fake.NewSimpleClientset()
			clientset.EXPECT().KubeVirt(Namespace).Return(kvInterface).AnyTimes()
			clientset.EXPECT().AppsV1().Return(dpClient.AppsV1()).AnyTimes()
			clientset.EXPECT().CoreV1().Return(dpClient.CoreV1()).AnyTimes()

			kv = &v1.KubeVirt{ObjectMeta: v12.ObjectMeta{Namespace: Namespace}}
			virtControllerConfig := &util.KubeVirtDeploymentConfig{
				Registry:        Registry,
				KubeVirtVersion: Version,
			}
			var err error
			strategyDeployment = components.NewControllerDeployment(
				Namespace,
				virtControllerConfig.GetImageRegistry(),
				virtControllerConfig.GetImagePrefix(),
				virtControllerConfig.GetControllerVersion(),
				virtControllerConfig.GetLauncherVersion(),
				virtControllerConfig.GetExportServerVersion(),
				"",
				"",
				"",
				"",
				virtControllerConfig.VirtControllerImage,
				virtControllerConfig.VirtLauncherImage,
				virtControllerConfig.VirtExportServerImage,
				virtControllerConfig.SidecarShimImage,
				virtControllerConfig.GetImagePullPolicy(),
				virtControllerConfig.GetImagePullSecrets(),
				virtControllerConfig.GetVerbosity(),
				virtControllerConfig.GetExtraEnv())

			cachedDeployment = strategyDeployment.DeepCopy()
			cachedDeployment.Generation = 2
			cachedDeployment.Annotations = map[string]string{
				revisionAnnotation: "4",
				fakeAnnotation:     "fake",
			}
			_, err = clientset.AppsV1().Deployments(Namespace).Create(context.TODO(), cachedDeployment, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			stores = util.Stores{DeploymentCache: &MockStore{get: cachedDeployment}}

			virtAPIDeployment = components.NewApiServerDeployment(
				Namespace,
				virtControllerConfig.GetImageRegistry(),
				virtControllerConfig.GetImagePrefix(),
				virtControllerConfig.GetApiVersion(),
				"",
				"",
				"",
				virtControllerConfig.VirtApiImage,
				virtControllerConfig.GetImagePullPolicy(),
				virtControllerConfig.GetImagePullSecrets(),
				virtControllerConfig.GetVerbosity(),
				virtControllerConfig.GetExtraEnv())

			virtAPIDeployment.Generation = 2
			virtAPIDeployment.Annotations = map[string]string{
				revisionAnnotation: "4",
				fakeAnnotation:     "fake",
			}

			_, err = dpClient.AppsV1().Deployments(Namespace).Create(context.TODO(), virtAPIDeployment, metav1.CreateOptions{})
		})

		It("should not remove revision annotation", func() {
			kv.Status.Generations = []v1.GenerationStatus{{
				Group:     "apps",
				Resource:  "deployments",
				Namespace: strategyDeployment.Namespace,
				Name:      strategyDeployment.Name,
				//Generation is not up-to-date with cachedDeployment
				//therefore Operator need to update the deployment
				LastGeneration: cachedDeployment.Generation - 1,
			}}
			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: &util.Expectations{},
				stores:       stores,
			}
			updatedDeploy, err := r.syncDeployment(strategyDeployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedDeploy.Annotations).ToNot(BeNil())
			Expect(updatedDeploy.Annotations).To(HaveKeyWithValue(revisionAnnotation, "4"))
			Expect(updatedDeploy.Annotations).ToNot(HaveKey(fakeAnnotation))
		})

		DescribeTable("should calculate correct replicas for deployments based on node count", func(nodesCount int, expectedReplicas int) {
			createFakeNodes(dpClient, nodesCount)

			r := &Reconciler{
				clientset:    clientset,
				kv:           kv,
				expectations: &util.Expectations{},
				stores:       stores,
			}

			updatedDeployment, err := r.syncDeployment(virtAPIDeployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(*updatedDeployment.Spec.Replicas).To(BeEquivalentTo(expectedReplicas))
		},
			Entry("Single-node cluster", 1, 1),
			Entry("Small cluster with 5 nodes", 5, 2),
			Entry("Medium cluster with 50 nodes", 50, 5),
		)
	})
})

func createFakeNodes(client *fake.Clientset, count int) {
	for i := range count {
		_, err := client.CoreV1().Nodes().Create(context.TODO(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("node-%d", i),
			},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
}
