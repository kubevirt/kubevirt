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

package backendstorage

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	k8sfake "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Backend Storage", func() {
	var backendStorage *BackendStorage
	var config *virtconfig.ClusterConfig
	var kvStore cache.Store
	var storageClassStore cache.Store
	var storageProfileStore cache.Store
	var pvcStore cache.Store
	var virtClient *kubecli.MockKubevirtClient

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubevirtFakeConfig := &virtv1.KubeVirtConfiguration{}
		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(kubevirtFakeConfig)
		storageClassInformer, _ := testutils.NewFakeInformerFor(&storagev1.StorageClass{})
		storageProfileInformer, _ := testutils.NewFakeInformerFor(&cdiv1.StorageProfile{})
		storageClassStore = storageClassInformer.GetStore()
		storageProfileStore = storageProfileInformer.GetStore()
		pvcInformer, _ := testutils.NewFakeInformerFor(&v1.PersistentVolumeClaim{})
		pvcStore = pvcInformer.GetStore()

		backendStorage = NewBackendStorage(virtClient, config, storageClassStore, storageProfileStore, pvcStore)
	})

	Context("Storage class", func() {
		It("Should return VMStateStorageClass and RWX when set", func() {
			By("Setting a VM state storage class in the CR")
			kvCR := testutils.GetFakeKubeVirtClusterConfig(kvStore)
			kvCR.Spec.Configuration.VMStateStorageClass = "myfave"
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvCR)

			By("Expecting getStorageClass() to return that one")
			sc, err := backendStorage.getStorageClass()
			Expect(err).NotTo(HaveOccurred())
			Expect(sc).To(Equal("myfave"))

			By("Expecting getAccessMode() to return RWX")
			accessMode := backendStorage.getAccessMode(sc, v1.PersistentVolumeFilesystem)
			Expect(accessMode).To(Equal(v1.ReadWriteMany))
		})

		It("Should return the default storage class when VMStateStorageClass is not set", func() {
			By("Creating 5 storage classes with one default")
			for i := 0; i < 5; i++ {
				sc := storagev1.StorageClass{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name: fmt.Sprintf("sc%d", i),
					},
				}
				if i == 3 {
					sc.Annotations = map[string]string{"storageclass.kubernetes.io/is-default-class": "true"}
				}
				err := storageClassStore.Add(&sc)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Expecting getStorageClass() to return the default one")
			sc, err := backendStorage.getStorageClass()
			Expect(err).NotTo(HaveOccurred())
			Expect(sc).To(Equal("sc3"))

			By("Expecting getAccessMode() to return RWO")
			accessMode := backendStorage.getAccessMode(sc, v1.PersistentVolumeFilesystem)
			Expect(accessMode).To(Equal(v1.ReadWriteOnce))
		})
	})

	Context("Access mode", func() {
		BeforeEach(func() {
			By("Creating a storage profile with no access/volume mode")
			sp := &cdiv1.StorageProfile{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "nomode",
				},
				Spec: cdiv1.StorageProfileSpec{},
				Status: cdiv1.StorageProfileStatus{
					ClaimPropertySets: []cdiv1.ClaimPropertySet{},
				},
			}
			err := storageProfileStore.Add(sp)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a storage profile with RWO FS as its only mode")
			sp = sp.DeepCopy()
			sp.Name = "onlyrwo"
			sp.Status.ClaimPropertySets = []cdiv1.ClaimPropertySet{{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				VolumeMode:  pointer.P(v1.PersistentVolumeFilesystem),
			}}
			err = storageProfileStore.Add(sp)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a storage profile that supports FS in both RWO and RWX")
			sp = sp.DeepCopy()
			sp.Name = "both"
			sp.Status.ClaimPropertySets = []cdiv1.ClaimPropertySet{{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany, v1.ReadWriteOnce},
				VolumeMode:  pointer.P(v1.PersistentVolumeFilesystem),
			}}
			err = storageProfileStore.Add(sp)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should default to RWO when no storage profile is defined", func() {
			accessMode := backendStorage.getAccessMode("doesntexist", v1.PersistentVolumeFilesystem)
			Expect(accessMode).To(Equal(v1.ReadWriteOnce))
		})

		It("Should default to RWO when the storage profile doesn't have any access mode", func() {
			accessMode := backendStorage.getAccessMode("nomode", v1.PersistentVolumeFilesystem)
			Expect(accessMode).To(Equal(v1.ReadWriteOnce))
		})

		It("Should pick RWX when both RWX and RWO are available", func() {
			accessMode := backendStorage.getAccessMode("both", v1.PersistentVolumeFilesystem)
			Expect(accessMode).To(Equal(v1.ReadWriteMany))
		})

		It("Should pick RWO when RWX isn't possible", func() {
			accessMode := backendStorage.getAccessMode("onlyrwo", v1.PersistentVolumeFilesystem)
			Expect(accessMode).To(Equal(v1.ReadWriteOnce), fmt.Sprintf("%#v", storageProfileStore.ListKeys()))
		})
	})

	Context("Migration", func() {
		var k8sClient *k8sfake.Clientset
		var migration *virtv1.VirtualMachineInstanceMigration
		const (
			nsName        = "testns"
			vmiName       = "testvmi"
			sourcePVCName = "sourcepvc"
			targetPVCName = "targetpvc"
			migrationName = "migration"
		)

		BeforeEach(func() {
			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
			sourcePVC := &v1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:   sourcePVCName,
					Labels: map[string]string{"persistent-state-for": vmiName},
				},
			}
			targetPVC := &v1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:   targetPVCName,
					Labels: map[string]string{"kubevirt.io/migrationName": migrationName},
				},
			}
			pvc, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Create(context.TODO(), sourcePVC, k8smetav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			err = pvcStore.Add(pvc)
			Expect(err).NotTo(HaveOccurred())
			pvc, err = k8sClient.CoreV1().PersistentVolumeClaims(nsName).Create(context.TODO(), targetPVC, k8smetav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			err = pvcStore.Add(pvc)
			Expect(err).NotTo(HaveOccurred())
			migration = &virtv1.VirtualMachineInstanceMigration{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      migrationName,
					Namespace: nsName,
				},
				Spec: virtv1.VirtualMachineInstanceMigrationSpec{
					VMIName: vmiName,
				},
				Status: virtv1.VirtualMachineInstanceMigrationStatus{
					MigrationState: &virtv1.VirtualMachineInstanceMigrationState{
						SourcePersistentStatePVCName: sourcePVCName,
						TargetPersistentStatePVCName: targetPVCName,
					},
				},
			}
		})
		It("Should label the target PVC and remove the source PVC on migration success", func() {
			err := MigrationHandoff(virtClient, pvcStore, migration)
			Expect(err).NotTo(HaveOccurred())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), sourcePVCName, k8smetav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
			targetPVC, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), targetPVCName, k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(targetPVC.Labels).To(HaveKeyWithValue("persistent-state-for", vmiName))
		})
		It("Should remove the target PVC on migration failure", func() {
			err := MigrationAbort(virtClient, migration)
			Expect(err).NotTo(HaveOccurred())
			sourcePVC, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), sourcePVCName, k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(sourcePVC.Labels).To(HaveKeyWithValue("persistent-state-for", vmiName))
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), targetPVCName, k8smetav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})
		It("Should keep the shared PVC on migration success", func() {
			migration.Status.MigrationState.TargetPersistentStatePVCName = sourcePVCName
			err := MigrationHandoff(virtClient, pvcStore, migration)
			Expect(err).NotTo(HaveOccurred())
			sourcePVC, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), sourcePVCName, k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(sourcePVC.Labels).To(HaveKeyWithValue("persistent-state-for", vmiName))
		})
		It("Should keep the shared PVC on migration failure", func() {
			migration.Status.MigrationState.TargetPersistentStatePVCName = sourcePVCName
			err := MigrationAbort(virtClient, migration)
			Expect(err).NotTo(HaveOccurred())
			sourcePVC, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), sourcePVCName, k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(sourcePVC.Labels).To(HaveKeyWithValue("persistent-state-for", vmiName))
		})
		It("Should build a recovery job without RunAsUser, RunAsGroup, or FSGroup", func() {
			job := buildRecoveryJob("test-recovery", "test-image", migration)

			psc := job.Spec.Template.Spec.SecurityContext
			Expect(psc).NotTo(BeNil())
			Expect(psc.RunAsNonRoot).To(HaveValue(BeTrue()))
			Expect(psc.RunAsUser).To(BeNil())
			Expect(psc.RunAsGroup).To(BeNil())
			Expect(psc.FSGroup).To(BeNil())
		})
	})

	Context("createPVC", func() {
		var (
			k8sClient *k8sfake.Clientset
			sc        storagev1.StorageClass
		)
		const (
			nsName  = "testns"
			vmiName = "testvmi"
			pvcName = "persistent-state-for-" + vmiName
		)

		BeforeEach(func() {
			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
			sc = storagev1.StorageClass{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:        "sc",
					Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"},
				},
			}
			err := storageClassStore.Add(&sc)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create a PVC with the correct labels", func() {
			vmi := &virtv1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      vmiName,
					Namespace: nsName,
				},
			}

			pvc, err := backendStorage.createPVC(vmi, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc).NotTo(BeNil())
			Expect(pvc.Labels).To(HaveKeyWithValue(storagetypes.LabelApplyStorageProfile, "true"))
		})

		It("Should create a PVC with overhead if CBT is enabled for the VM", func() {
			vmi := &virtv1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      vmiName,
					Namespace: nsName,
				},
				Status: virtv1.VirtualMachineInstanceStatus{
					ChangedBlockTracking: &virtv1.ChangedBlockTrackingStatus{
						State: virtv1.ChangedBlockTrackingInitializing,
					},
				},
			}

			pvc, err := backendStorage.createPVC(vmi, map[string]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc).NotTo(BeNil())
			pvcSize := resource.MustParse(PVCSize)
			pvcSize.Add(resource.MustParse(cbt.CBTBackendStateOverhead))
			Expect(pvc.Spec.Resources.Requests.Storage().Cmp(pvcSize)).To(Equal(0))
		})
	})

	Context("Legacy PVCs", func() {
		var k8sClient *k8sfake.Clientset

		const (
			nsName  = "testns"
			vmiName = "testvmi"
			pvcName = "persistent-state-for-" + vmiName
		)

		BeforeEach(func() {
			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
			legacyPVC := &v1.PersistentVolumeClaim{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      pvcName,
					Namespace: nsName,
				},
			}
			pvc, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Create(context.TODO(), legacyPVC, k8smetav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			err = pvcStore.Add(pvc)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should get labelled by CreatePVCForVMI when called with a KubeVirt client", func() {
			vmi := &virtv1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      vmiName,
					Namespace: nsName,
				},
			}
			pvc, err := backendStorage.CreatePVCForVMI(vmi)
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc).NotTo(BeNil())
			pvc, err = k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), pvcName, k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc.Labels).To(HaveKeyWithValue(PVCPrefix, vmiName))
		})
	})
	Context("IsBackendStorageNeeded", func() {
		var vm *virtv1.VirtualMachine
		var vmi *virtv1.VirtualMachineInstance
		var snapshotVM *snapshotv1.VirtualMachine

		BeforeEach(func() {
			vmi = libvmi.New(
				libvmi.WithUefi(false),
				libvmi.WithTPM(false),
			)
			vm = libvmi.NewVirtualMachine(vmi)
			snapshotVM = &snapshotv1.VirtualMachine{
				Spec: vm.Spec,
			}
		})

		DescribeTable("should always", func(expected bool, alter func(spec *virtv1.VirtualMachineInstanceSpec)) {
			alter(&vm.Spec.Template.Spec)
			alter(&snapshotVM.Spec.Template.Spec)
			alter(&vmi.Spec)
			Expect(IsBackendStorageNeeded(vm)).To(Equal(expected))
			Expect(IsBackendStorageNeeded(snapshotVM)).To(Equal(expected))
			Expect(IsBackendStorageNeeded(vmi)).To(Equal(expected))
		},
			Entry("be false when no cbt and no persistent feature is set", false, func(_ *virtv1.VirtualMachineInstanceSpec) {
			}),
			Entry("be true when persistent TPM is set", true, func(spec *virtv1.VirtualMachineInstanceSpec) {
				spec.Domain.Devices.TPM.Persistent = pointer.P(true)
			}),
			Entry("be true when persistent EFI is set", true, func(spec *virtv1.VirtualMachineInstanceSpec) {
				spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(true)
			}),
			Entry("be true when persistent TPM and EFI are set no cbt", true, func(spec *virtv1.VirtualMachineInstanceSpec) {
				spec.Domain.Devices.TPM.Persistent = pointer.P(true)
				spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(true)
			}),
			Entry("be true when virtualMachineState is set", true, func(spec *virtv1.VirtualMachineInstanceSpec) {
				spec.VirtualMachineState = &virtv1.VirtualMachineStateSpec{}
			}),
		)

		DescribeTable("should with VM and VMI", func(expected bool, alter func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance)) {
			alter(vm, vmi)
			Expect(IsBackendStorageNeeded(vm)).To(Equal(expected))
			Expect(IsBackendStorageNeeded(vmi)).To(Equal(expected))
		},
			Entry("be true when cbt is Initializing", true, func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingInitializing)
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingInitializing)
			}),
			Entry("be true when cbt is Enabled", true, func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
			}),
			Entry("be false when cbt is Disabled", false, func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingDisabled)
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingDisabled)
			}),
			Entry("be true when persistent TPM and CBT are set", true, func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				vm.Spec.Template.Spec.Domain.Devices.TPM.Persistent = pointer.P(true)
				vmi.Spec.Domain.Devices.TPM.Persistent = pointer.P(true)
			}),
			Entry("be true when persistent EFI and CBT are set", true, func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				vm.Spec.Template.Spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(true)
				vmi.Spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(true)
			}),
			Entry("be true when persistent TPM, EFI and CBT are set", true, func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
				cbt.SetCBTState(&vm.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				vm.Spec.Template.Spec.Domain.Devices.TPM.Persistent = pointer.P(true)
				vmi.Spec.Domain.Devices.TPM.Persistent = pointer.P(true)
				vm.Spec.Template.Spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(true)
				vmi.Spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(true)
			}),
		)
		DescribeTable("should with VMSnapshot", func(expected bool, alter func(snapshotVM *snapshotv1.VirtualMachine)) {
			alter(snapshotVM)
			Expect(IsBackendStorageNeeded(snapshotVM)).To(Equal(expected))
		},
			Entry("be false when cbt only is defined", false, func(snapshotVM *snapshotv1.VirtualMachine) {
				cbt.SetCBTState(&snapshotVM.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingInitializing)
			}),
			Entry("be true when persistent TPM and CBT are set", true, func(snapshotVM *snapshotv1.VirtualMachine) {
				cbt.SetCBTState(&snapshotVM.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				snapshotVM.Spec.Template.Spec.Domain.Devices.TPM.Persistent = pointer.P(true)
			}),
			Entry("be true when persistent EFI and CBT are set", true, func(snapshotVM *snapshotv1.VirtualMachine) {
				cbt.SetCBTState(&snapshotVM.Status.ChangedBlockTracking, virtv1.ChangedBlockTrackingEnabled)
				snapshotVM.Spec.Template.Spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(true)
			}),
		)
	})

	Context("Declarative VMState", func() {
		var k8sClient *k8sfake.Clientset

		const (
			nsName  = "testns"
			vmiName = "testvmi"
			vmiUID  = "vmi-uid"
		)

		newDeclarativeVMI := func() *virtv1.VirtualMachineInstance {
			return &virtv1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      vmiName,
					Namespace: nsName,
					UID:       vmiUID,
				},
				Spec: virtv1.VirtualMachineInstanceSpec{
					VirtualMachineState: &virtv1.VirtualMachineStateSpec{},
				},
			}
		}

		BeforeEach(func() {
			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		})

		DescribeTable("HasDeclarativeVMState", func(state *virtv1.VirtualMachineStateSpec, expected bool) {
			spec := &virtv1.VirtualMachineInstanceSpec{VirtualMachineState: state}
			Expect(HasDeclarativeVMState(spec)).To(Equal(expected))
		},
			Entry("is true when virtualMachineState is set", &virtv1.VirtualMachineStateSpec{}, true),
			Entry("is false when virtualMachineState is nil", nil, false),
		)

		It("ownerUIDForVMI returns the controller owner UID when present", func() {
			vmi := newDeclarativeVMI()
			vmi.OwnerReferences = []k8smetav1.OwnerReference{{
				Controller: pointer.P(true),
				UID:        "vm-uid",
				Name:       "vm",
				Kind:       "VirtualMachine",
			}}
			Expect(ownerUIDForVMI(vmi)).To(Equal("vm-uid"))
		})

		It("ownerUIDForVMI falls back to the VMI's own UID", func() {
			Expect(ownerUIDForVMI(newDeclarativeVMI())).To(Equal(vmiUID))
		})

		DescribeTable("HasPersistentEFI", func(declarative bool, persistent *bool, expected bool) {
			spec := &virtv1.VirtualMachineInstanceSpec{
				Domain: virtv1.DomainSpec{
					Firmware: &virtv1.Firmware{
						Bootloader: &virtv1.Bootloader{
							EFI: &virtv1.EFI{Persistent: persistent},
						},
					},
				},
			}
			if declarative {
				spec.VirtualMachineState = &virtv1.VirtualMachineStateSpec{}
			}
			Expect(HasPersistentEFI(spec)).To(Equal(expected))
		},
			Entry("declarative: unset persistent implies persistent EFI", true, nil, true),
			Entry("declarative: persistent:false opts out", true, pointer.P(false), false),
			Entry("declarative: persistent:true keeps persistent EFI", true, pointer.P(true), true),
			Entry("non-declarative: unset persistent is not persistent", false, nil, false),
		)

		It("declarativeOwnerReferences returns the controller ref when the VMI has one", func() {
			vmi := newDeclarativeVMI()
			vmi.OwnerReferences = []k8smetav1.OwnerReference{{
				Controller: pointer.P(true),
				UID:        "vm-uid",
				Name:       "vm",
				Kind:       "VirtualMachine",
			}}
			refs := declarativeOwnerReferences(vmi)
			Expect(refs).To(HaveLen(1))
			Expect(string(refs[0].UID)).To(Equal("vm-uid"))
			Expect(refs[0].Name).To(Equal("vm"))
		})

		It("declarativeOwnerReferences falls back to a controller ref to the VMI", func() {
			refs := declarativeOwnerReferences(newDeclarativeVMI())
			Expect(refs).To(HaveLen(1))
			Expect(string(refs[0].UID)).To(Equal(vmiUID))
			Expect(refs[0].Name).To(Equal(vmiName))
		})

		Context("declarativePVCForVMI resolution", func() {
			addPVC := func(pvc *v1.PersistentVolumeClaim) {
				Expect(pvcStore.Add(pvc)).To(Succeed())
			}

			It("resolves by the status volume claim name", func() {
				vmi := newDeclarativeVMI()
				vmi.Status.VirtualMachineStateVolume = &virtv1.VolumeStatus{
					PersistentVolumeClaimInfo: &virtv1.PersistentVolumeClaimInfo{ClaimName: "status-pvc"},
				}
				addPVC(&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{Name: "status-pvc", Namespace: nsName}})

				pvc := declarativePVCForVMI(pvcStore, vmi)
				Expect(pvc).NotTo(BeNil())
				Expect(pvc.Name).To(Equal("status-pvc"))
			})

			It("falls back to the owner-UID labelled PVC", func() {
				addPVC(&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "owned-pvc",
					Namespace: nsName,
					Labels:    map[string]string{VMStateOwnerLabel: vmiUID},
				}})

				pvc := declarativePVCForVMI(pvcStore, newDeclarativeVMI())
				Expect(pvc).NotTo(BeNil())
				Expect(pvc.Name).To(Equal("owned-pvc"))
			})

			It("falls back to the source PVC name", func() {
				vmi := newDeclarativeVMI()
				vmi.Spec.VirtualMachineState.Source = &virtv1.VirtualMachineStateSource{Name: "source-pvc"}
				addPVC(&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{Name: "source-pvc", Namespace: nsName}})

				pvc := declarativePVCForVMI(pvcStore, vmi)
				Expect(pvc).NotTo(BeNil())
				Expect(pvc.Name).To(Equal("source-pvc"))
			})

			It("skips a PVC being deleted", func() {
				vmi := newDeclarativeVMI()
				vmi.Spec.VirtualMachineState.Source = &virtv1.VirtualMachineStateSource{Name: "source-pvc"}
				addPVC(&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{
					Name:              "source-pvc",
					Namespace:         nsName,
					DeletionTimestamp: pointer.P(k8smetav1.Now()),
				}})

				Expect(declarativePVCForVMI(pvcStore, vmi)).To(BeNil())
			})

			It("returns nil when nothing matches", func() {
				Expect(declarativePVCForVMI(pvcStore, newDeclarativeVMI())).To(BeNil())
			})

			It("deterministically prefers the newest PVC when several share the owner label", func() {
				now := time.Now()
				addPVC(&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{
					Name:              "older-pvc",
					Namespace:         nsName,
					Labels:            map[string]string{VMStateOwnerLabel: vmiUID},
					CreationTimestamp: k8smetav1.NewTime(now.Add(-time.Hour)),
				}})
				addPVC(&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{
					Name:              "newer-pvc",
					Namespace:         nsName,
					Labels:            map[string]string{VMStateOwnerLabel: vmiUID},
					CreationTimestamp: k8smetav1.NewTime(now),
				}})

				pvc := declarativePVCForVMI(pvcStore, newDeclarativeVMI())
				Expect(pvc).NotTo(BeNil())
				Expect(pvc.Name).To(Equal("newer-pvc"))
			})
		})

		Context("createOrAdoptDeclarativePVC via CreatePVCForVMI", func() {
			It("adopts an existing resolvable PVC and stamps the owner label", func() {
				vmi := newDeclarativeVMI()
				vmi.Spec.VirtualMachineState.Source = &virtv1.VirtualMachineStateSource{Name: "existing-pvc"}
				existing, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Create(context.TODO(),
					&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{Name: "existing-pvc", Namespace: nsName}},
					k8smetav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(pvcStore.Add(existing)).To(Succeed())

				pvc, err := backendStorage.CreatePVCForVMI(vmi)
				Expect(err).NotTo(HaveOccurred())
				Expect(pvc.Labels).To(HaveKeyWithValue(VMStateOwnerLabel, vmiUID))

				got, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), "existing-pvc", k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Labels).To(HaveKeyWithValue(VMStateOwnerLabel, vmiUID))
			})

			It("returns ErrVMStatePVCNotFound when the source PVC is missing", func() {
				vmi := newDeclarativeVMI()
				vmi.Spec.VirtualMachineState.Source = &virtv1.VirtualMachineStateSource{Name: "missing-pvc"}
				_, err := backendStorage.CreatePVCForVMI(vmi)
				Expect(err).To(MatchError(ErrVMStatePVCNotFound))
			})

			It("creates a PVC from the volumeClaimTemplate when none exists", func() {
				vmi := newDeclarativeVMI()
				vmi.Spec.VirtualMachineState.VolumeClaimTemplate = &v1.PersistentVolumeClaimTemplate{
					Spec: v1.PersistentVolumeClaimSpec{StorageClassName: pointer.P("sc")},
				}

				pvc, err := backendStorage.CreatePVCForVMI(vmi)
				Expect(err).NotTo(HaveOccurred())
				Expect(pvc.GenerateName).To(Equal(PVCPrefix + "-" + vmiName + "-"))
				Expect(pvc.Labels).To(HaveKeyWithValue(VMStateOwnerLabel, vmiUID))
			})

			It("createPVCFromTemplate defaults omitted fields and merges labels with controller precedence", func() {
				Expect(storageClassStore.Add(&storagev1.StorageClass{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name:        "sc",
						Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"},
					},
				})).To(Succeed())

				vmi := newDeclarativeVMI()
				vmi.Spec.VirtualMachineState.VolumeClaimTemplate = &v1.PersistentVolumeClaimTemplate{
					ObjectMeta: k8smetav1.ObjectMeta{
						Labels: map[string]string{
							"custom":          "value",
							VMStateOwnerLabel: "should-not-win",
						},
					},
				}

				pvc, err := backendStorage.CreatePVCForVMI(vmi)
				Expect(err).NotTo(HaveOccurred())
				Expect(pvc.Spec.VolumeMode).To(HaveValue(Equal(v1.PersistentVolumeFilesystem)))
				Expect(pvc.Spec.Resources.Requests.Storage().Cmp(resource.MustParse(PVCSize))).To(Equal(0))
				Expect(pvc.Spec.AccessModes).NotTo(BeEmpty())
				Expect(pvc.Labels).To(HaveKeyWithValue("custom", "value"))
				Expect(pvc.Labels).To(HaveKeyWithValue(VMStateOwnerLabel, vmiUID))
			})
		})

		Context("ensureOwnerLabel", func() {
			DescribeTable("adds the owner label when absent", func(initial map[string]string) {
				vmi := newDeclarativeVMI()
				created, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Create(context.TODO(),
					&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{Name: "pvc", Namespace: nsName, Labels: initial}},
					k8smetav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				updated, err := backendStorage.ensureOwnerLabel(vmi, created)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Labels).To(HaveKeyWithValue(VMStateOwnerLabel, vmiUID))
			},
				Entry("PVC with no labels", nil),
				Entry("PVC with an unrelated label", map[string]string{"foo": "bar"}),
			)

			It("is a no-op when the PVC is already owned", func() {
				pvc := &v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "pvc",
					Namespace: nsName,
					Labels:    map[string]string{VMStateOwnerLabel: vmiUID},
				}}
				updated, err := backendStorage.ensureOwnerLabel(newDeclarativeVMI(), pvc)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(Equal(pvc))
			})
		})

		Context("AcquireVMStateLock", func() {
			DescribeTable("records the holder UID", func(initial map[string]string) {
				created, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Create(context.TODO(),
					&v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{Name: "lock-pvc", Namespace: nsName, Labels: initial}},
					k8smetav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(backendStorage.AcquireVMStateLock(created, "holder-uid")).To(Succeed())

				got, err := k8sClient.CoreV1().PersistentVolumeClaims(nsName).Get(context.TODO(), "lock-pvc", k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Labels).To(HaveKeyWithValue(VMStateInUseByLabel, "holder-uid"))
			},
				Entry("PVC with no labels", nil),
				Entry("PVC with existing labels", map[string]string{"foo": "bar"}),
			)

			It("is a no-op when already held by the same UID", func() {
				pvc := &v1.PersistentVolumeClaim{ObjectMeta: k8smetav1.ObjectMeta{
					Name:      "lock-pvc",
					Namespace: nsName,
					Labels:    map[string]string{VMStateInUseByLabel: "holder-uid"},
				}}
				Expect(backendStorage.AcquireVMStateLock(pvc, "holder-uid")).To(Succeed())
			})
		})
	})
})
