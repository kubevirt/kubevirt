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
 * Copyright The KubeVirt Authors.
 *
 */
package upgrade_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	fakeclientset "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/instancetype/upgrade"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

func deepCopyList(objects []interface{}) []interface{} {
	for i := range objects {
		objects[i] = objects[i].(runtime.Object).DeepCopyObject()
	}
	return objects
}

var _ = Describe("ControllerRevision upgrades", func() {
	type upgrader interface {
		Upgrade(vm *virtv1.VirtualMachine) error
	}

	var (
		vm *virtv1.VirtualMachine

		virtClient *kubecli.MockKubevirtClient

		upgradeHandler                  upgrader
		controllerrevisionInformerStore cache.Store
	)

	sanityUpgrade := func(vm *virtv1.VirtualMachine) error {
		stores := []cache.Store{controllerrevisionInformerStore}
		var listOfObjects [][]interface{}

		for _, store := range stores {
			listOfObjects = append(listOfObjects, deepCopyList(store.List()))
		}

		err := upgradeHandler.Upgrade(vm)

		for i, objects := range listOfObjects {
			ExpectWithOffset(1, stores[i].List()).To(ConsistOf(objects...))
		}
		return err
	}

	BeforeEach(func() {
		controllerrevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		controllerrevisionInformerStore = controllerrevisionInformer.GetStore()

		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().AppsV1().Return(k8sfake.NewSimpleClientset().AppsV1()).AnyTimes()

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			fakeclientset.NewSimpleClientset().KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		vm = libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault)),
			libvmi.WithInstancetype("foo"),
			libvmi.WithPreference("bar"),
		)

		upgradeHandler = upgrade.New(controllerrevisionInformerStore, virtClient)
	})

	createControllerRevisionFromObject := func(obj runtime.Object) *appsv1.ControllerRevision {
		originalCR, err := revision.CreateControllerRevision(vm, obj)
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
		Expect(controllerrevisionInformerStore.Add(originalInstancetypeCR)).To(Succeed())
		vm.Status.InstancetypeRef = &virtv1.InstancetypeStatusRef{
			ControllerRevisionRef: &virtv1.ControllerRevisionRef{
				Name: originalInstancetypeCR.Name,
			},
		}

		originalPreferenceCR := createPreferenceCR()
		Expect(controllerrevisionInformerStore.Add(originalPreferenceCR)).To(Succeed())
		vm.Status.PreferenceRef = &virtv1.InstancetypeStatusRef{
			ControllerRevisionRef: &virtv1.ControllerRevisionRef{
				Name: originalPreferenceCR.Name,
			},
		}

		var err error
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(sanityUpgrade(vm)).To(Succeed())

		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).ToNot(Equal(originalInstancetypeCR.Name))

		_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
			context.Background(), originalInstancetypeCR.Name, metav1.GetOptions{})
		Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))

		newInstancetypeCR, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
			context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(upgrade.IsObjectLatestVersion(newInstancetypeCR)).To(BeTrue())
		if originalKindLabel, hasLabel := originalInstancetypeCR.Labels[instancetypeapi.ControllerRevisionObjectKindLabel]; hasLabel {
			Expect(newInstancetypeCR.Labels).To(HaveKeyWithValue(instancetypeapi.ControllerRevisionObjectKindLabel, originalKindLabel))
		}

		Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).ToNot(Equal(originalPreferenceCR.Name))

		_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
			context.Background(), originalPreferenceCR.Name, metav1.GetOptions{})
		Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))

		newPreferenceCR, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
			context.Background(), vm.Status.PreferenceRef.ControllerRevisionRef.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(upgrade.IsObjectLatestVersion(newPreferenceCR)).To(BeTrue())
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
								PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets),
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
								PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets),
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
		Expect(controllerrevisionInformerStore.Add(originalInstancetypeCR)).To(Succeed())
		vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
			RevisionName: originalInstancetypeCR.Name,
		}

		originalPreferenceCR := createPreferenceCR()
		Expect(controllerrevisionInformerStore.Add(originalPreferenceCR)).To(Succeed())
		vm.Spec.Preference = &virtv1.PreferenceMatcher{
			RevisionName: originalPreferenceCR.Name,
		}

		var err error
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(sanityUpgrade(vm)).To(Succeed())

		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(vm.Spec.Instancetype.RevisionName).To(Equal(originalInstancetypeCR.Name))
		Expect(vm.Spec.Preference.RevisionName).To(Equal(originalPreferenceCR.Name))

		// Repeat the Upgrade call to show it is idempotent
		Expect(sanityUpgrade(vm)).To(Succeed())

		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

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
								PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets),
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
								PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets),
							},
						},
					},
				)
			},
		),
	)
})
