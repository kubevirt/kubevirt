//nolint:dupl
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
 * Copyright 2024 Red Hat, Inc.
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

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	. "kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("ControllerRevision upgrades", func() {
	var (
		methods *InstancetypeMethods

		vm *virtv1.VirtualMachine

		virtClient  *kubecli.MockKubevirtClient
		vmInterface *kubecli.MockVirtualMachineInterface
		k8sClient   *k8sfake.Clientset
	)

	BeforeEach(func() {
		controllerrevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		controllerrevisionInformerStore := controllerrevisionInformer.GetStore()

		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

		methods = &InstancetypeMethods{
			ControllerRevisionStore: controllerrevisionInformerStore,
			Clientset:               virtClient,
		}

		vm = kubecli.NewMinimalVM("testvm")
		vm.Namespace = k8sv1.NamespaceDefault
	})

	expectControllerRevisionCreation := func() {
		k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			created, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			createdObj := created.GetObject()
			createdCR, ok := createdObj.(*appsv1.ControllerRevision)
			Expect(ok).To(BeTrue())

			Expect(IsObjectLatestVersion(createdCR)).To(BeTrue())

			Expect(methods.ControllerRevisionStore.Add(createdCR)).To(Succeed())
			return true, createdObj, nil
		})
	}

	expectVirtualMachineRevisionNamePatch := func() {
		vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), metav1.PatchOptions{})
	}

	crKeyFunc := func(namespace, name string) string {
		return types.NamespacedName{Namespace: namespace, Name: name}.String()
	}

	expectControllerRevisionDeletion := func() {
		k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			deleted, ok := action.(testing.DeleteAction)
			Expect(ok).To(BeTrue())

			deletedObj, exists, err := methods.ControllerRevisionStore.GetByKey(crKeyFunc(deleted.GetNamespace(), deleted.GetName()))
			Expect(exists).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			Expect(methods.ControllerRevisionStore.Delete(deletedObj)).To(Succeed())
			return true, nil, nil
		})
	}

	createControllerRevisionFromObject := func(obj runtime.Object) *appsv1.ControllerRevision {
		originalCR, err := CreateControllerRevision(vm, obj)
		Expect(err).ToNot(HaveOccurred())

		originalCR.Data.Raw, err = json.Marshal(originalCR.Data.Object)
		Expect(err).ToNot(HaveOccurred())

		return originalCR
	}

	DescribeTable("should upgrade ControllerRevisions containing", func(
		createInstancetypeCR func() *appsv1.ControllerRevision,
		createPreferenceCR func() *appsv1.ControllerRevision,
	) {
		originalInstancetypeCR := createInstancetypeCR()
		Expect(methods.ControllerRevisionStore.Add(originalInstancetypeCR)).To(Succeed())
		vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
			RevisionName: originalInstancetypeCR.Name,
		}

		originalPreferenceCR := createPreferenceCR()
		Expect(methods.ControllerRevisionStore.Add(originalPreferenceCR)).To(Succeed())
		vm.Spec.Preference = &virtv1.PreferenceMatcher{
			RevisionName: originalPreferenceCR.Name,
		}

		expectControllerRevisionCreation()
		expectControllerRevisionCreation()

		expectVirtualMachineRevisionNamePatch()

		expectControllerRevisionDeletion()
		expectControllerRevisionDeletion()

		Expect(methods.Upgrade(vm)).To(Succeed())

		Expect(methods.ControllerRevisionStore.List()).To(HaveLen(2))

		Expect(vm.Spec.Instancetype.Name).ToNot(Equal(originalInstancetypeCR.Name))
		Expect(vm.Spec.Preference.Name).ToNot(Equal(originalPreferenceCR.Name))

		newObj, exists, err := methods.ControllerRevisionStore.GetByKey(crKeyFunc(vm.Namespace, vm.Spec.Instancetype.RevisionName))
		Expect(exists).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		newInstancetypeCR, ok := newObj.(*appsv1.ControllerRevision)
		Expect(ok).To(BeTrue())

		Expect(IsObjectLatestVersion(newInstancetypeCR)).To(BeTrue())
		if originalKindLabel, hasLabel := originalInstancetypeCR.Labels[instancetypeapi.ControllerRevisionObjectKindLabel]; hasLabel {
			Expect(newInstancetypeCR.Labels).To(HaveKeyWithValue(instancetypeapi.ControllerRevisionObjectKindLabel, originalKindLabel))
		}

		newObj, exists, err = methods.ControllerRevisionStore.GetByKey(crKeyFunc(vm.Namespace, vm.Spec.Preference.RevisionName))
		Expect(exists).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())

		newPreferenceCR, ok := newObj.(*appsv1.ControllerRevision)
		Expect(ok).To(BeTrue())

		Expect(IsObjectLatestVersion(newPreferenceCR)).To(BeTrue())
		if originalKindLabel, hasLabel := originalPreferenceCR.Labels[instancetypeapi.ControllerRevisionObjectKindLabel]; hasLabel {
			Expect(newPreferenceCR.Labels).To(HaveKeyWithValue(instancetypeapi.ControllerRevisionObjectKindLabel, originalKindLabel))
		}
	},
		Entry("v1alpha1 VirtualMachinePreferenceSpec & VirtualMachineInstancetypeSpec within VirtualMachineInstancetypeSpecRevision",
			func() *appsv1.ControllerRevision {
				instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha1.CPUInstancetype{
						Guest: uint32(1),
					},
					Memory: instancetypev1alpha1.MemoryInstancetype{
						Guest: resource.MustParse("128Mi"),
					},
				}
				instancetypeSpecBytes, err := json.Marshal(instancetypeSpec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
					APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
					Spec:       instancetypeSpecBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				return &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "legacy-instancetype-cr",
						Namespace: vm.Namespace,
					},
					Data: runtime.RawExtension{
						Raw: specRevisionBytes,
					},
				}
			},
			func() *appsv1.ControllerRevision {
				preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1alpha1.CPUPreferences{
						PreferredCPUTopology: instancetypev1alpha1.PreferSockets,
					},
				}
				preferenceSpecBytes, err := json.Marshal(preferenceSpec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
					APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
					Spec:       preferenceSpecBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				return &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "legacy-preference-cr",
						Namespace: vm.Namespace,
					},
					Data: runtime.RawExtension{
						Raw: specRevisionBytes,
					},
				}
			},
		),
		Entry("v1alpha1 VirtualMachineClusterPreference & VirtualMachineClusterInstancetype",
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha1.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha1.PreferSockets,
							},
						},
					},
				)
			},
		),
		Entry("v1alpha1 VirtualMachinePreference & VirtualMachineInstancetype",
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha1.VirtualMachineInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1alpha1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha1.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha1.PreferSockets,
							},
						},
					},
				)
			},
		),
		Entry("v1alpha2 VirtualMachineClusterPreference & VirtualMachineClusterInstancetype",
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha2.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha2.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha2.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha2.PreferSockets,
							},
						},
					},
				)
			},
		),
		Entry("VirtualMachinePreference & VirtualMachineInstancetype v1alpha2",
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha2.VirtualMachineInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1alpha2.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1alpha2.CPUPreferences{
								PreferredCPUTopology: instancetypev1alpha2.PreferSockets,
							},
						},
					},
				)
			},
		),
		Entry("v1beta1 VirtualMachineClusterPreference & VirtualMachineClusterInstancetype without version label",
			func() *appsv1.ControllerRevision {
				cr := createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
				cr.Name = "legacy-clusterinstancetype-cr-name"
				delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
				return cr
			},
			func() *appsv1.ControllerRevision {
				cr := createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{
								PreferredCPUTopology: pointer.P(instancetypev1beta1.PreferSockets),
							},
						},
					},
				)
				cr.Name = "legacy-clusterpreference-cr-name"
				delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
				return cr
			},
		),
		Entry("v1beta1 VirtualMachinePreference & VirtualMachineInstancetype without version label",
			func() *appsv1.ControllerRevision {
				cr := createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachineInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
				cr.Name = "legacy-instancetype-cr-name"
				delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
				return cr
			},
			func() *appsv1.ControllerRevision {
				cr := createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{
								PreferredCPUTopology: pointer.P(instancetypev1beta1.PreferSockets),
							},
						},
					},
				)
				cr.Name = "legacy-preference-cr-name"
				delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)
				return cr
			},
		),
	)

	DescribeTable("should not upgrade ControllerRevisions containing", func(
		createInstancetypeCR func() *appsv1.ControllerRevision,
		createPreferenceCR func() *appsv1.ControllerRevision,
	) {
		originalInstancetypeCR := createInstancetypeCR()
		Expect(methods.ControllerRevisionStore.Add(originalInstancetypeCR)).To(Succeed())
		vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
			RevisionName: originalInstancetypeCR.Name,
		}

		originalPreferenceCR := createPreferenceCR()
		Expect(methods.ControllerRevisionStore.Add(originalPreferenceCR)).To(Succeed())
		vm.Spec.Preference = &virtv1.PreferenceMatcher{
			RevisionName: originalPreferenceCR.Name,
		}

		Expect(methods.Upgrade(vm)).To(Succeed())

		Expect(vm.Spec.Instancetype.RevisionName).To(Equal(originalInstancetypeCR.Name))
		Expect(vm.Spec.Preference.RevisionName).To(Equal(originalPreferenceCR.Name))

		// Repeat the Upgrade call to show it is idempotent
		Expect(methods.Upgrade(vm)).To(Succeed())

		Expect(vm.Spec.Instancetype.RevisionName).To(Equal(originalInstancetypeCR.Name))
		Expect(vm.Spec.Preference.RevisionName).To(Equal(originalPreferenceCR.Name))
	},
		Entry("v1beta1 VirtualMachineClusterPreference & VirtualMachineClusterInstancetype",
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{
								PreferredCPUTopology: pointer.P(instancetypev1beta1.PreferSockets),
							},
						},
					},
				)
			},
		),
		Entry("v1beta1 VirtualMachinePreference & VirtualMachineInstancetype",
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachineInstancetype{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instancetype",
							Namespace: vm.Namespace,
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
			func() *appsv1.ControllerRevision {
				return createControllerRevisionFromObject(
					&instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "preference",
							Namespace: vm.Namespace,
						},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{
								PreferredCPUTopology: pointer.P(instancetypev1beta1.PreferSockets),
							},
						},
					},
				)
			},
		),
	)
})
