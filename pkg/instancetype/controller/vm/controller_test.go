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

//nolint:dupl
package vm_test

import (
	"context"
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	instancetypecontroller "kubevirt.io/kubevirt/pkg/instancetype/controller/vm"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

var _ = Describe("Instance type and Preference VirtualMachine Controller", func() {
	const (
		resourceUID        types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
		resourceGeneration int64     = 1
	)

	const (
		instancetypeName = "instancetype"
		preferenceName   = "preference"
	)

	type instancetypeVMController interface {
		Sync(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
		ApplyToVM(*virtv1.VirtualMachine) error
		ApplyToVMI(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error
		ApplyAutoAttachPreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error
	}

	var (
		vm  *virtv1.VirtualMachine
		vmi *virtv1.VirtualMachineInstance

		instancetypeController instancetypeVMController

		instancetypeObj *v1beta1.VirtualMachineInstancetype
		preference      *v1beta1.VirtualMachinePreference

		config   *virtconfig.ClusterConfig
		recorder *record.FakeRecorder

		virtClient *kubecli.MockKubevirtClient

		kvStore                          cache.Store
		instancetypeInformerStore        cache.Store
		clusterInstancetypeInformerStore cache.Store
		preferenceInformerStore          cache.Store
		clusterPreferenceInformerStore   cache.Store
		controllerrevisionInformerStore  cache.Store
	)

	BeforeEach(func() {
		vmi = libvmi.New(libvmi.WithNamespace(k8sv1.NamespaceDefault))
		vm = libvmi.NewVirtualMachine(vmi)

		// We need to clear the domainSpec here to ensure the instancetype doesn't conflict
		vm.Spec.Template.Spec.Domain = virtv1.DomainSpec{}

		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			fake.NewSimpleClientset().KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			fake.NewSimpleClientset().KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

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

		controllerrevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		controllerrevisionInformerStore = controllerrevisionInformer.GetStore()

		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		instancetypeController = instancetypecontroller.New(
			instancetypeInformerStore,
			clusterInstancetypeInformerStore,
			preferenceInformerStore,
			clusterPreferenceInformerStore,
			controllerrevisionInformerStore,
			virtClient,
			config,
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
			Spec: v1beta1.VirtualMachinePreferenceSpec{
				Firmware: &v1beta1.FirmwarePreferences{
					DeprecatedPreferredUseEfi: pointer.P(true),
				},
				Devices: &v1beta1.DevicePreferences{
					PreferredDiskBus:        virtv1.DiskBusVirtio,
					PreferredInterfaceModel: "virtio",
					PreferredInputBus:       virtv1.InputBusUSB,
					PreferredInputType:      virtv1.InputTypeTablet,
				},
			},
		}
		_, err = virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(preferenceInformerStore.Add(preference)).To(Succeed())
	})

	deepCopyList := func(objects []interface{}) []interface{} {
		for i := range objects {
			objects[i] = objects[i].(runtime.Object).DeepCopyObject()
		}
		return objects
	}

	sanitySync := func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
		stores := []cache.Store{
			instancetypeInformerStore,
			clusterInstancetypeInformerStore,
			preferenceInformerStore,
			clusterPreferenceInformerStore,
			controllerrevisionInformerStore,
		}
		listOfObjects := [][]interface{}{}

		for _, store := range stores {
			listOfObjects = append(listOfObjects, deepCopyList(store.List()))
		}

		_, err := instancetypeController.Sync(vm, vmi)
		Expect(err).ToNot(HaveOccurred())

		for i, objects := range listOfObjects {
			ExpectWithOffset(1, stores[i].List()).To(ConsistOf(objects...))
		}
	}

	Context("instancetype", func() {
		var clusterInstancetypeObj *v1beta1.VirtualMachineClusterInstancetype

		BeforeEach(func() {
			clusterInstancetypeObj = &v1beta1.VirtualMachineClusterInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "clusterInstancetype",
					UID:        resourceUID,
					Generation: resourceGeneration,
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta1.SchemeGroupVersion.String(),
					Kind:       "VirtualMachineClusterInstancetype",
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
			_, err := virtClient.VirtualMachineClusterInstancetype().Create(
				context.Background(), clusterInstancetypeObj, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			err = clusterInstancetypeInformerStore.Add(clusterInstancetypeObj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should store VirtualMachineInstancetype as ControllerRevision on Sync", func() {
			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name: instancetypeObj.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			var err error
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedRevision, err := revision.CreateControllerRevision(vm, instancetypeObj)
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(expectedRevision.Name))

			revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
				context.Background(), expectedRevision.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			revisionInstancetype, ok := revision.Data.Object.(*v1beta1.VirtualMachineInstancetype)
			Expect(ok).To(BeTrue(), "Expected Instancetype in ControllerRevision")

			Expect(revisionInstancetype.Spec).To(Equal(instancetypeObj.Spec))
		})

		DescribeTable("should apply VirtualMachineInstancetype from ControllerRevision to VirtualMachineInstance",
			func(getRevisionData func() []byte) {
				instancetypeRevision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crName",
					},
					Data: runtime.RawExtension{
						Raw: getRevisionData(),
					},
				}

				instancetypeRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), instancetypeRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
					Name:         instancetypeObj.Name,
					Kind:         instancetypeapi.SingularResourceName,
					RevisionName: instancetypeRevision.Name,
				}
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(instancetypeController.ApplyToVMI(vm, vmi)).To(Succeed())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
				Expect(vmi.Spec.Domain.Memory.Guest).To(HaveValue(Equal(instancetypeObj.Spec.Memory.Guest)))
				Expect(vmi.Annotations).To(HaveKeyWithValue(virtv1.InstancetypeAnnotation, instancetypeObj.Name))
				Expect(vmi.Annotations).ToNot(HaveKey(virtv1.PreferenceAnnotation))
				Expect(vmi.Annotations).ToNot(HaveKey(virtv1.ClusterInstancetypeAnnotation))
				Expect(vmi.Annotations).ToNot(HaveKey(virtv1.ClusterPreferenceAnnotation))
			},
			Entry("using v1beta1", func() []byte {
				instancetypeBytes, err := json.Marshal(instancetypeObj)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeBytes
			}),
		)

		It("should sync correctly if an existing ControllerRevision is present but not referenced by InstancetypeMatcher", func() {
			instancetypeRevision, err := revision.CreateControllerRevision(vm, instancetypeObj)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), instancetypeRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name: instancetypeObj.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(instancetypeRevision.Name))
		})

		It("should store VirtualMachineClusterInstancetype as ControllerRevision on Sync", func() {
			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name: clusterInstancetypeObj.Name,
				Kind: instancetypeapi.ClusterSingularResourceName,
			}

			var err error
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedRevisionName := revision.GenerateName(
				vm.Name, clusterInstancetypeObj.Name, clusterInstancetypeObj.GroupVersionKind().Version,
				clusterInstancetypeObj.UID, clusterInstancetypeObj.Generation)
			expectedRevision, err := revision.CreateControllerRevision(vm, clusterInstancetypeObj)
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(expectedRevision.Name))

			revision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
				context.Background(), expectedRevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			revisionInstancetype, ok := revision.Data.Object.(*v1beta1.VirtualMachineClusterInstancetype)
			Expect(ok).To(BeTrue(), "Expected Instancetype in ControllerRevision")

			Expect(revisionInstancetype.Spec).To(Equal(clusterInstancetypeObj.Spec))
		})

		It("should apply VirtualMachineClusterInstancetype from ControllerRevision to VirtualMachineInstance", func() {
			instancetypeRevision, err := revision.CreateControllerRevision(vm, clusterInstancetypeObj)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), instancetypeRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name:         clusterInstancetypeObj.Name,
				Kind:         instancetypeapi.ClusterSingularResourceName,
				RevisionName: instancetypeRevision.Name,
			}

			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(instancetypeController.ApplyToVMI(vm, vmi)).To(Succeed())

			Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(clusterInstancetypeObj.Spec.CPU.Guest))
			Expect(vmi.Spec.Domain.Memory.Guest).To(HaveValue(Equal(clusterInstancetypeObj.Spec.Memory.Guest)))
			Expect(vmi.Annotations).To(HaveKeyWithValue(virtv1.ClusterInstancetypeAnnotation, clusterInstancetypeObj.Name))
			Expect(vmi.Annotations).ToNot(HaveKey(virtv1.PreferenceAnnotation))
			Expect(vmi.Annotations).ToNot(HaveKey(virtv1.InstancetypeAnnotation))
			Expect(vmi.Annotations).ToNot(HaveKey(virtv1.ClusterPreferenceAnnotation))
		})

		DescribeTable("should fail to sync with FailedFindInstancetype reason",
			func(matcher *virtv1.InstancetypeMatcher) {
				vm.Spec.Instancetype = matcher
				syncVM, err := instancetypeController.Sync(vm, vmi)
				Expect(syncVM).To(Equal(vm))
				Expect(err).To(HaveOccurred())

				var syncErr common.SyncError
				Expect(errors.As(err, &syncErr)).To(BeTrue())
				Expect(syncErr.Reason()).To(Equal("FailedFindInstancetype"))
			},
			Entry("if an invalid InstancetypeMatcher Kind is provided",
				&virtv1.InstancetypeMatcher{
					Name: instancetypeName,
					Kind: "foobar",
				},
			),
			Entry("if a VirtualMachineInstancetype cannot be found",
				&virtv1.InstancetypeMatcher{
					Name: "foobar",
					Kind: instancetypeapi.SingularResourceName,
				},
			),
			Entry("if a VirtualMachineClusterInstancetype cannot be found",
				&virtv1.InstancetypeMatcher{
					Name: "foobar",
					Kind: instancetypeapi.ClusterSingularResourceName,
				},
			),
		)

		It("should fail to sync if the VirtualMachineInstancetype conflicts with the VirtualMachineInstance", func() {
			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name: instancetypeObj.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{
				Sockets: uint32(1),
				Cores:   uint32(4),
				Threads: uint32(1),
			}

			_, err := instancetypeController.Sync(vm, vmi)
			Expect(err).To(MatchError(ContainSubstring("conflicts with selected instance type")))
			testutils.ExpectEvents(recorder, common.FailedCreateVirtualMachineReason)
		})

		It("should fail to sync if an existing ControllerRevision is found with unexpected VirtualMachineInstancetypeSpec data", func() {
			unexpectedInstancetype := instancetypeObj.DeepCopy()
			unexpectedInstancetype.Spec.CPU.Guest = 15

			instancetypeRevision, err := revision.CreateControllerRevision(vm, unexpectedInstancetype)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), instancetypeRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name: instancetypeObj.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			_, err = instancetypeController.Sync(vm, vmi)
			Expect(err).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			testutils.ExpectEvents(recorder, common.FailedCreateVirtualMachineReason)
		})
	})

	Context("preference", func() {
		var clusterPreference *v1beta1.VirtualMachineClusterPreference

		BeforeEach(func() {
			clusterPreference = &v1beta1.VirtualMachineClusterPreference{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "clusterPreference",
					UID:        resourceUID,
					Generation: resourceGeneration,
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta1.SchemeGroupVersion.String(),
					Kind:       "VirtualMachineClusterPreference",
				},
				Spec: v1beta1.VirtualMachinePreferenceSpec{
					Firmware: &v1beta1.FirmwarePreferences{
						DeprecatedPreferredUseEfi: pointer.P(true),
					},
					Devices: &v1beta1.DevicePreferences{
						PreferredDiskBus:        virtv1.DiskBusVirtio,
						PreferredInterfaceModel: "virtio",
						PreferredInputBus:       virtv1.InputBusUSB,
						PreferredInputType:      virtv1.InputTypeTablet,
					},
				},
			}
			_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			err = clusterPreferenceInformerStore.Add(clusterPreference)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should store VirtualMachinePreference as ControllerRevision on Sync", func() {
			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			var err error
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedPreferenceRevisionName := revision.GenerateName(
				vm.Name, preference.Name, preference.GroupVersionKind().Version, preference.UID, preference.Generation)
			expectedPreferenceRevision, err := revision.CreateControllerRevision(vm, preference)
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))

			preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
				context.Background(), expectedPreferenceRevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			preferenceRevisionObj, ok := preferenceRevision.Data.Object.(*v1beta1.VirtualMachinePreference)
			Expect(ok).To(BeTrue(), "Expected Preference in ControllerRevision")
			Expect(preferenceRevisionObj.Spec).To(Equal(preference.Spec))
		})

		DescribeTable("should apply VirtualMachinePreference from ControllerRevision to VirtualMachineInstance",
			func(getRevisionData func() []byte) {
				preferenceRevision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crName",
					},
					Data: runtime.RawExtension{
						Raw: getRevisionData(),
					},
				}

				preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), preferenceRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &virtv1.PreferenceMatcher{
					Name:         preference.Name,
					Kind:         instancetypeapi.SingularPreferenceResourceName,
					RevisionName: preferenceRevision.Name,
				}

				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(instancetypeController.ApplyToVMI(vm, vmi)).To(Succeed())

				Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
				Expect(vmi.Annotations).ToNot(HaveKey(virtv1.InstancetypeAnnotation))
				Expect(vmi.Annotations).To(HaveKeyWithValue(virtv1.PreferenceAnnotation, preference.Name))
				Expect(vmi.Annotations).ToNot(HaveKey(virtv1.ClusterInstancetypeAnnotation))
				Expect(vmi.Annotations).ToNot(HaveKey(virtv1.ClusterPreferenceAnnotation))
			},
			Entry("using v1beta1", func() []byte {
				preferenceBytes, err := json.Marshal(preference)
				Expect(err).ToNot(HaveOccurred())

				return preferenceBytes
			}),
		)

		It("should sync corrrectly if an existing ControllerRevision is present but not referenced by PreferenceMatcher", func() {
			preferenceRevision, err := revision.CreateControllerRevision(vm, preference)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), preferenceRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(preferenceRevision.Name))
		})

		It("should store VirtualMachineClusterPreference as ControllerRevision on sync", func() {
			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Name: clusterPreference.Name,
				Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
			}

			var err error
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			expectedPreferenceRevision, err := revision.CreateControllerRevision(vm, clusterPreference)
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(expectedPreferenceRevision.Name))

			preferenceRevision, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
				context.Background(), expectedPreferenceRevision.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			preferenceRevisionObj, ok := preferenceRevision.Data.Object.(*v1beta1.VirtualMachineClusterPreference)
			Expect(ok).To(BeTrue(), "Expected Preference in ControllerRevision")
			Expect(preferenceRevisionObj.Spec).To(Equal(clusterPreference.Spec))
		})

		It("should apply VirtualMachineClusterPreference from ControllerRevision to VirtualMachineInstance", func() {
			preferenceRevision, err := revision.CreateControllerRevision(vm, clusterPreference)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), preferenceRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Name:         clusterPreference.Name,
				Kind:         instancetypeapi.ClusterSingularPreferenceResourceName,
				RevisionName: preferenceRevision.Name,
			}

			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(instancetypeController.ApplyToVMI(vm, vmi)).To(Succeed())

			Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
			Expect(vmi.Annotations).ToNot(HaveKey(virtv1.InstancetypeAnnotation))
			Expect(vmi.Annotations).To(HaveKeyWithValue(virtv1.ClusterPreferenceAnnotation, clusterPreference.Name))
			Expect(vmi.Annotations).ToNot(HaveKey(virtv1.ClusterInstancetypeAnnotation))
			Expect(vmi.Annotations).ToNot(HaveKey(virtv1.PreferenceAnnotation))
		})

		DescribeTable("should fail to sync with FailedFindPreference reason",
			func(matcher *virtv1.PreferenceMatcher) {
				vm.Spec.Preference = matcher
				syncVM, err := instancetypeController.Sync(vm, vmi)
				Expect(syncVM).To(Equal(vm))
				Expect(err).To(HaveOccurred())

				var syncErr common.SyncError
				Expect(errors.As(err, &syncErr)).To(BeTrue())
				Expect(syncErr.Reason()).To(Equal("FailedFindPreference"))
			},
			Entry("if an invalid InstancetypeMatcher Kind is provided",
				&virtv1.PreferenceMatcher{
					Name: preferenceName,
					Kind: "foobar",
				},
			),
			Entry("if a VirtualMachinePreference cannot be found",
				&virtv1.PreferenceMatcher{
					Name: "foobar",
					Kind: instancetypeapi.SingularPreferenceResourceName,
				},
			),
			Entry("if a VirtualMachineClusterPreference cannot be found",
				&virtv1.PreferenceMatcher{
					Name: "foobar",
					Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
				},
			),
		)

		It("should fail to sync if an existing ControllerRevision is found with unexpected VirtualMachinePreferenceSpec data", func() {
			unexpectedPreference := preference.DeepCopy()
			unexpectedPreference.Spec.Firmware = &v1beta1.FirmwarePreferences{
				PreferredUseBios: pointer.P(true),
			}

			preferenceRevision, err := revision.CreateControllerRevision(vm, unexpectedPreference)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), preferenceRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			_, err = instancetypeController.Sync(vm, vmi)
			Expect(err).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			testutils.ExpectEvents(recorder, common.FailedCreateVirtualMachineReason)
		})
	})

	Context("InstancetypeReferencePolicy", func() {
		addRevisionsToVMFunc := func() {
			instancetypeRevision, err := revision.CreateControllerRevision(vm, instancetypeObj)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), instancetypeRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preferenceRevision, err := revision.CreateControllerRevision(vm, preference)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), preferenceRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Status.InstancetypeRef = &virtv1.InstancetypeStatusRef{
				ControllerRevisionRef: &virtv1.ControllerRevisionRef{
					Name: instancetypeRevision.Name,
				},
			}
			vm.Status.PreferenceRef = &virtv1.InstancetypeStatusRef{
				ControllerRevisionRef: &virtv1.ControllerRevisionRef{
					Name: preferenceRevision.Name,
				},
			}
		}

		kvWithReferencePolicyExpand := &virtv1.KubeVirt{
			Spec: virtv1.KubeVirtSpec{
				Configuration: virtv1.KubeVirtConfiguration{
					Instancetype: &virtv1.InstancetypeConfiguration{
						ReferencePolicy: pointer.P(virtv1.Expand),
					},
				},
			},
		}

		kvWithReferencePolicyExpandAll := &virtv1.KubeVirt{
			Spec: virtv1.KubeVirtSpec{
				Configuration: virtv1.KubeVirtConfiguration{
					Instancetype: &virtv1.InstancetypeConfiguration{
						ReferencePolicy: pointer.P(virtv1.ExpandAll),
					},
				},
			},
		}

		BeforeEach(func() {
			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name: instancetypeObj.Name,
				Kind: instancetypeapi.SingularResourceName,
			}
			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}
		})

		DescribeTable("should not expand and update VM ", func(kv *virtv1.KubeVirt, updateVMFunc func()) {
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
			updateVMFunc()

			var err error
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(
				context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(
				context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			Expect(vm.Spec.Instancetype.Name).To(Equal(instancetypeObj.Name))
			Expect(vm.Spec.Instancetype.RevisionName).To(BeEmpty())
			Expect(revision.HasControllerRevisionRef(vm.Status.InstancetypeRef)).To(BeTrue())
			Expect(vm.Spec.Preference).ToNot(BeNil())
			Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))
			Expect(vm.Spec.Preference.RevisionName).To(BeEmpty())
			Expect(revision.HasControllerRevisionRef(vm.Status.PreferenceRef)).To(BeTrue())
		},
			Entry("default referencePolicy",
				&virtv1.KubeVirt{Spec: virtv1.KubeVirtSpec{Configuration: virtv1.KubeVirtConfiguration{}}}, func() {}),
			Entry("referencePolicy reference",
				&virtv1.KubeVirt{
					Spec: virtv1.KubeVirtSpec{
						Configuration: virtv1.KubeVirtConfiguration{
							Instancetype: &virtv1.InstancetypeConfiguration{
								ReferencePolicy: pointer.P(virtv1.Reference),
							},
						},
					},
				}, func() {}),
			Entry("referencePolicy expand and revisionNames already captured",
				kvWithReferencePolicyExpand, addRevisionsToVMFunc,
			),
		)

		DescribeTable("should expand and update VM", func(kv *virtv1.KubeVirt, updateVMFunc func()) {
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
			updateVMFunc()

			// Stash a copy of the original VM to assert ControllerRevision removal later
			originalVM := vm.DeepCopy()

			var err error
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(
				context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			sanitySync(vm, vmi)

			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(
				context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Instancetype).To(BeNil())
			Expect(vm.Status.InstancetypeRef).To(BeNil())
			Expect(vm.Spec.Preference).To(BeNil())
			Expect(vm.Status.PreferenceRef).To(BeNil())
			Expect(vm.Spec.Template.Spec.Domain.CPU.Sockets).To(Equal(instancetypeObj.Spec.CPU.Guest))
			Expect(vm.Spec.Template.Spec.Domain.Memory.Guest.Value()).To(Equal(instancetypeObj.Spec.Memory.Guest.Value()))

			// Assert that the original ControllerRevisions have been cleaned up
			if originalVM.Spec.Instancetype.RevisionName != "" {
				_, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
					context.TODO(), originalVM.Spec.Instancetype.RevisionName, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			}
			if originalVM.Spec.Preference.RevisionName != "" {
				_, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
					context.TODO(), originalVM.Spec.Preference.RevisionName, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			}
		},
			Entry("referencePolicy expand", kvWithReferencePolicyExpand, func() {}),
			Entry("referencePolicy expandAll", kvWithReferencePolicyExpandAll, func() {}),
			Entry("referencePolicy expandAll and revisionNames already captured",
				kvWithReferencePolicyExpandAll, addRevisionsToVMFunc),
		)
	})
})
