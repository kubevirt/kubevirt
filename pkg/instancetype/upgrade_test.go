//nolint:lll
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
 * Copyright 2023 Red Hat, Inc.
 *
 */
package instancetype_test

import (
	"context"
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	. "kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("instancetype and preference Upgrades", func() {
	var (
		upgrader *Upgrader

		vm *virtv1.VirtualMachine

		virtClient  *kubecli.MockKubevirtClient
		vmInterface *kubecli.MockVirtualMachineInterface
		k8sClient   *k8sfake.Clientset

		vmInformer cache.SharedIndexInformer
	)

	syncCaches := func(stop chan struct{}) {
		go vmInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})

		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

		k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		stop := make(chan struct{})
		syncCaches(stop)

		vm = kubecli.NewMinimalVM("testvm")
		vm.Namespace = k8sv1.NamespaceDefault
		Expect(vmInformer.GetStore().Add(vm)).To(Succeed())

		upgrader = NewUpgrader(virtClient, vmInformer)
	})

	Context("ControllerRevisionUpgrade should", func() {
		DescribeTable("skip upgrade of ControllerRevision labelled with latest object version of", func(createControllerRevision func() (*appsv1.ControllerRevision, error)) {
			cr, err := createControllerRevision()
			Expect(err).ToNot(HaveOccurred())

			// Assert that the original is the latest just to ensure this test
			// is updated when a new version is introduced
			Expect(IsObjectLatestVersion(cr)).To(BeTrue())

			newCR, err := upgrader.Upgrade(cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(newCR).To(Equal(cr))
		},
			Entry("VirtualMachineInstancetype",
				func() (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachineInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "instancetype",
							},
							Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1beta1.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1beta1.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
				},
			),
			Entry("VirtualMachineClusterInstancetype",
				func() (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachineClusterInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterinstancetype",
							},
							Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1beta1.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1beta1.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
				},
			),
			Entry("VirtualMachinePreference",
				func() (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1beta1.PreferSockets
					return CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachinePreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "preference",
							},
							Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1beta1.CPUPreferences{
									PreferredCPUTopology: &cpuPreference,
								},
							},
						},
					)
				},
			),
			Entry("VirtualMachineClusterPreference",
				func() (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1beta1.PreferSockets
					return CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachineClusterPreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterpreference",
							},
							Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1beta1.CPUPreferences{
									PreferredCPUTopology: &cpuPreference,
								},
							},
						},
					)
				},
			),
		)

		expectControllerRevisionCreation := func() {
			k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				createdObj := created.GetObject()
				createdCR, ok := createdObj.(*appsv1.ControllerRevision)
				Expect(ok).To(BeTrue())

				Expect(IsObjectLatestVersion(createdCR)).To(BeTrue())

				return true, createdObj, nil
			})
		}

		expectVirtualMachineRevisionNamePatch := func() {
			vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &metav1.PatchOptions{})
		}

		expectControllerRevisionDeletion := func(expectedCRName string) {
			k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				deleted, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())
				Expect(deleted.GetName()).To(Equal(expectedCRName))
				return true, nil, nil
			})
		}

		updateInstancetypeMatcher := func(revisionName string) {
			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				RevisionName: revisionName,
			}
		}

		updatePreferenceMatcher := func(revisionName string) {
			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				RevisionName: revisionName,
			}
		}

		DescribeTable("upgrade ControllerRevision containing", func(createOriginalCR func(*virtv1.VirtualMachine) (*appsv1.ControllerRevision, error), updateMatcher func(string)) {
			originalCR, err := createOriginalCR(vm)
			Expect(err).ToNot(HaveOccurred())

			updateMatcher(originalCR.Name)

			if originalCR.Data.Object != nil {
				originalCR.Data.Raw, err = json.Marshal(originalCR.Data.Object)
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(k8sClient.Tracker().Add(originalCR)).To(Succeed())

			expectControllerRevisionCreation()
			expectVirtualMachineRevisionNamePatch()
			expectControllerRevisionDeletion(originalCR.Name)

			newCR, err := upgrader.Upgrade(originalCR)
			Expect(err).ToNot(HaveOccurred())

			Expect(IsObjectLatestVersion(newCR)).To(BeTrue())
		},
			Entry("VirtualMachineInstancetype v1beta1 without object version labels",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					cr, err := CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachineInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "instancetype",
							},
							Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1beta1.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1beta1.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(cr.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
					delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
					return cr, nil
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachineInstancetype v1alpha2",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha2.VirtualMachineInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "instancetype",
							},
							Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha2.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1alpha2.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachineInstancetype v1alpha1",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha1.VirtualMachineInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "instancetype",
							},
							Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha1.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1alpha1.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachineInstancetypeSpecRevision v1alpha1",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1alpha1.CPUInstancetype{
							Guest: uint32(1),
						},
						Memory: instancetypev1alpha1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					}
					specBytes, err := json.Marshal(&instancetypeSpec)
					Expect(err).ToNot(HaveOccurred())

					specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
						APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
						Spec:       specBytes,
					}
					specRevisionBytes, err := json.Marshal(specRevision)
					Expect(err).ToNot(HaveOccurred())

					return &appsv1.ControllerRevision{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "VirtualMachineInstancetypeSpecRevision",
							Namespace:       vm.Namespace,
							OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
						},
						Data: runtime.RawExtension{
							Raw: specRevisionBytes,
						},
					}, nil
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachineClusterInstancetype v1beta1 without object version labels",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					cr, err := CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachineClusterInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterinstancetype",
							},
							Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1beta1.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1beta1.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(cr.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
					delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
					return cr, nil
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachineClusterInstancetype v1alpha2",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha2.VirtualMachineClusterInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterinstancetype",
							},
							Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha2.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1alpha2.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachineClusterInstancetype v1alpha1",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha1.VirtualMachineClusterInstancetype{
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterinstancetype",
							},
							Spec: instancetypev1alpha1.VirtualMachineInstancetypeSpec{
								CPU: instancetypev1alpha1.CPUInstancetype{
									Guest: uint32(1),
								},
								Memory: instancetypev1alpha1.MemoryInstancetype{
									Guest: resource.MustParse("128Mi"),
								},
							},
						},
					)
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachinePreference v1beta1 without object version labels",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1beta1.PreferSockets
					cr, err := CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachinePreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "preference",
							},
							Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1beta1.CPUPreferences{
									PreferredCPUTopology: &cpuPreference,
								},
							},
						},
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(cr.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
					delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
					return cr, nil
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachinePreference v1alpha2",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha2.VirtualMachinePreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "preference",
							},
							Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha2.CPUPreferences{
									PreferredCPUTopology: instancetypev1alpha2.PreferSockets,
								},
							},
						},
					)
				},
				updatePreferenceMatcher,
			),
			Entry("VirtualMachinePreference v1alpha1",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha1.VirtualMachinePreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "preference",
							},
							Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha1.CPUPreferences{
									PreferredCPUTopology: instancetypev1alpha1.PreferSockets,
								},
							},
						},
					)
				},
				updatePreferenceMatcher,
			),
			Entry("VirtualMachinePreferenceSpecRevision v1alpha1",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1alpha1.CPUPreferences{
							PreferredCPUTopology: instancetypev1alpha1.PreferSockets,
						},
					}
					specBytes, err := json.Marshal(&preferenceSpec)
					Expect(err).ToNot(HaveOccurred())

					specRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
						APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
						Spec:       specBytes,
					}
					specRevisionBytes, err := json.Marshal(specRevision)
					Expect(err).ToNot(HaveOccurred())

					return &appsv1.ControllerRevision{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "VirtualMachinePreferenceSpecRevision",
							Namespace:       vm.Namespace,
							OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
						},
						Data: runtime.RawExtension{
							Raw: specRevisionBytes,
						},
					}, nil
				},
				updatePreferenceMatcher,
			),
			Entry("VirtualMachineClusterPreference v1beta1 without object version labels",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1beta1.PreferSockets
					cr, err := CreateControllerRevision(vm,
						&instancetypev1beta1.VirtualMachineClusterPreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "preference",
							},
							Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1beta1.CPUPreferences{
									PreferredCPUTopology: &cpuPreference,
								},
							},
						},
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(cr.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
					delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
					return cr, nil
				},
				updateInstancetypeMatcher,
			),
			Entry("VirtualMachineClusterPreference v1alpha2",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha2.VirtualMachineClusterPreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterpreference",
							},
							Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha2.CPUPreferences{
									PreferredCPUTopology: instancetypev1alpha2.PreferSockets,
								},
							},
						},
					)
				},
				updatePreferenceMatcher,
			),
			Entry("VirtualMachineClusterPreference v1alpha1",
				func(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
					return CreateControllerRevision(vm,
						&instancetypev1alpha1.VirtualMachineClusterPreference{
							ObjectMeta: metav1.ObjectMeta{
								Name: "clusterpreference",
							},
							Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha1.CPUPreferences{
									PreferredCPUTopology: instancetypev1alpha1.PreferSockets,
								},
							},
						},
					)
				},
				updatePreferenceMatcher,
			),
		)
	})
})
