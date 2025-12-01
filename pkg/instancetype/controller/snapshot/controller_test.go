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
 */

package snapshot_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	snapshotcontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/snapshot"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Instancetype Snapshot Controller", func() {
	const (
		resourceUID        types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
		resourceGeneration int64     = 1
		snapshotUID        types.UID = "snapshot-9160e5de-2540-476a-86d9-af0081aee68a"
	)

	const (
		instancetypeName = "instancetype"
		preferenceName   = "preference"
		snapshotName     = "test-snapshot"
	)

	var (
		vm       *virtv1.VirtualMachine
		snapshot *snapshotv1.VirtualMachineSnapshot

		snapshotController snapshotcontroller.Controller

		instancetypeObj *v1beta1.VirtualMachineInstancetype
		preference      *v1beta1.VirtualMachinePreference

		recorder *record.FakeRecorder

		virtClient *kubecli.MockKubevirtClient

		instancetypeInformerStore        cache.Store
		clusterInstancetypeInformerStore cache.Store
		preferenceInformerStore          cache.Store
		clusterPreferenceInformerStore   cache.Store
	)

	BeforeEach(func() {
		vmi := libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
		vm = libvmi.NewVirtualMachine(vmi)
		vm.Spec.Template.Spec.Domain = virtv1.DomainSpec{}

		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			fake.NewSimpleClientset().KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineInstancetype(k8sv1.NamespaceDefault).Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineInstancetypes(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()

		virtClient.EXPECT().VirtualMachinePreference(k8sv1.NamespaceDefault).Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterPreference().Return(
			fake.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

		virtClient.EXPECT().AppsV1().Return(k8sfake.NewSimpleClientset().AppsV1()).AnyTimes()

		instancetypeInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachineInstancetype{})
		instancetypeInformerStore = instancetypeInformer.GetStore()

		clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachineClusterInstancetype{})
		clusterInstancetypeInformerStore = clusterInstancetypeInformer.GetStore()

		preferenceInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachinePreference{})
		preferenceInformerStore = preferenceInformer.GetStore()

		clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachineClusterPreference{})
		clusterPreferenceInformerStore = clusterPreferenceInformer.GetStore()

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		snapshotController = snapshotcontroller.New(
			instancetypeInformerStore,
			clusterInstancetypeInformerStore,
			preferenceInformerStore,
			clusterPreferenceInformerStore,
			virtClient,
			recorder,
		)

		instancetypeObj = &v1beta1.VirtualMachineInstancetype{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "VirtualMachineInstancetype",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:       instancetypeName,
				Namespace:  vm.Namespace,
				UID:        resourceUID,
				Generation: resourceGeneration,
			},
			Spec: v1beta1.VirtualMachineInstancetypeSpec{
				CPU: v1beta1.CPUInstancetype{
					Guest: uint32(2),
				},
				Memory: v1beta1.MemoryInstancetype{
					Guest: resource.MustParse("128M"),
				},
			},
		}
		_, err := virtClient.VirtualMachineInstancetype(vm.Namespace).Create(context.Background(), instancetypeObj, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(instancetypeInformerStore.Add(instancetypeObj)).To(Succeed())

		preference = &v1beta1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:       preferenceName,
				Namespace:  vm.Namespace,
				UID:        resourceUID,
				Generation: resourceGeneration,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "VirtualMachinePreference",
			},
			Spec: v1beta1.VirtualMachinePreferenceSpec{},
		}
		_, err = virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(preferenceInformerStore.Add(preference)).To(Succeed())

		snapshot = &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      snapshotName,
				Namespace: vm.Namespace,
				UID:       snapshotUID,
			},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &virtv1.SchemeGroupVersion.Group,
					Kind:     "VirtualMachine",
					Name:     vm.Name,
				},
			},
		}
	})

	It("should update instancetype controller revision ownerref", func() {
		vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
			Name: instancetypeObj.Name,
			Kind: instancetypeapi.SingularResourceName,
		}

		createdVM, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		instancetypeRevision, err := revision.CreateControllerRevision(createdVM, instancetypeObj)
		Expect(err).NotTo(HaveOccurred())

		instancetypeRevision, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
			context.Background(), instancetypeRevision, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		createdVM.Status.InstancetypeRef = &virtv1.InstancetypeStatusRef{
			ControllerRevisionRef: &virtv1.ControllerRevisionRef{
				Name: instancetypeRevision.Name,
			},
		}

		createdVM, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), createdVM, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		err = snapshotController.Sync(snapshot)
		Expect(err).NotTo(HaveOccurred())

		updatedRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
			context.Background(), instancetypeRevision.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedRevision.OwnerReferences).To(HaveLen(1))
		Expect(updatedRevision.OwnerReferences[0].Kind).To(Equal("VirtualMachineSnapshot"))
		Expect(updatedRevision.OwnerReferences[0].Name).To(Equal(snapshot.Name))
		Expect(updatedRevision.OwnerReferences[0].UID).To(Equal(snapshot.UID))
		Expect(*updatedRevision.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
	})

	It("should update preference controller revision ownerref", func() {
		vm.Spec.Preference = &virtv1.PreferenceMatcher{
			Name: preference.Name,
			Kind: instancetypeapi.SingularPreferenceResourceName,
		}

		createdVM, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		preferenceRevision, err := revision.CreateControllerRevision(createdVM, preference)
		Expect(err).NotTo(HaveOccurred())

		preferenceRevision, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
			context.Background(), preferenceRevision, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		createdVM.Status.PreferenceRef = &virtv1.InstancetypeStatusRef{
			ControllerRevisionRef: &virtv1.ControllerRevisionRef{
				Name: preferenceRevision.Name,
			},
		}

		createdVM, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), createdVM, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		err = snapshotController.Sync(snapshot)
		Expect(err).NotTo(HaveOccurred())

		updatedRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
			context.Background(), preferenceRevision.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(updatedRevision.OwnerReferences).To(HaveLen(1))
		Expect(updatedRevision.OwnerReferences[0].Kind).To(Equal("VirtualMachineSnapshot"))
		Expect(updatedRevision.OwnerReferences[0].Name).To(Equal(snapshot.Name))
		Expect(updatedRevision.OwnerReferences[0].UID).To(Equal(snapshot.UID))
		Expect(*updatedRevision.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
	})
})
