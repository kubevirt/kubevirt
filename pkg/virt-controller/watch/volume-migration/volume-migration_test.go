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
 * Copyright The KubeVirt Authors
 *
 */

package volumemigration_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	volumemigration "kubevirt.io/kubevirt/pkg/virt-controller/watch/volume-migration"
)

var _ = Describe("Volume Migration", func() {
	Context("ValidateVolumes", func() {
		var (
			dataVolumeStore cache.Store
			pvcStore        cache.Store

			dvCSI    *cdiv1.DataVolume
			dvNoSCSI *cdiv1.DataVolume
		)
		const (
			noCSIDVName = "nocsi-dv"
			csiDVName   = "csi-dv"
			ns          = "test"
			popAnn      = "cdi.kubevirt.io/storage.usePopulator"
		)
		BeforeEach(func() {
			dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})

			dataVolumeStore = dataVolumeInformer.GetStore()
			pvcStore = pvcInformer.GetStore()

			dvCSI = libdv.NewDataVolume(libdv.WithNamespace(ns), libdv.WithName(csiDVName), libdv.WithAnnotation(popAnn, "true"))
			dvNoSCSI = libdv.NewDataVolume(libdv.WithNamespace(ns), libdv.WithName(noCSIDVName), libdv.WithAnnotation(popAnn, "true"))
			pvcCSI := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      csiDVName,
					Namespace: ns,
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					DataSourceRef: pointer.P(k8sv1.TypedObjectReference{}),
				},
			}
			pvcNOCSI := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      noCSIDVName,
					Namespace: ns,
				},
			}

			Expect(dataVolumeStore.Add(dvCSI)).To(Succeed())
			Expect(dataVolumeStore.Add(dvNoSCSI)).To(Succeed())
			Expect(pvcStore.Add(pvcCSI)).To(Succeed())
			Expect(pvcStore.Add(pvcNOCSI)).To(Succeed())
		})

		DescribeTable("should validate the migrated volumes", func(vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine, expectError error) {
			err := volumemigration.ValidateVolumes(vmi, vm, dataVolumeStore, pvcStore)
			if expectError != nil {
				Expect(err).To(Equal(expectError))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
			Entry("with empty VMI", nil, &v1.VirtualMachine{}, fmt.Errorf("cannot validate the migrated volumes for an empty VMI")),
			Entry("with empty VM", &v1.VirtualMachineInstance{}, nil, fmt.Errorf("cannot validate the migrated volumes for an empty VM")),
			Entry("without any migrated volumes", libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), libvmi.WithPersistentVolumeClaim("disk1", "vol1"),
			), libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), libvmi.WithPersistentVolumeClaim("disk1", "vol1"),
			)), nil),
			Entry("with valid volumes", libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), libvmi.WithPersistentVolumeClaim("disk1", "vol1"),
			), libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol2"), libvmi.WithPersistentVolumeClaim("disk1", "vol3"),
			)), nil),
			Entry("with an invalid lun volume", libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), libvmi.WithPersistentVolumeClaimLun("disk1", "vol1", false),
			), libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol2"), libvmi.WithPersistentVolumeClaimLun("disk1", "vol4", false),
			)), fmt.Errorf("invalid volumes to update with migration: luns: [disk1]")),
			Entry("with an invalid shareable volume", libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), withShareableVolume("disk1", "vol1"),
			), libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol2"), withShareableVolume("disk1", "vol4"),
			)), fmt.Errorf("invalid volumes to update with migration: shareable: [disk1]")),
			Entry("with an invalid filesystem volume", libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), withFilesystemVolume("disk1", "vol1"),
			), libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol2"), withFilesystemVolume("disk1", "vol4"),
			)), fmt.Errorf("invalid volumes to update with migration: filesystems: [disk1]")),
			Entry("with valid hotplugged volume", libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), withHotpluggedVolume("disk1", "vol1"),
			), libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol2"), withHotpluggedVolume("disk1", "vol4"),
			)), nil),
			Entry("with a DV with a csi storageclass", libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", "vol0")),
				libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", csiDVName))), nil),
			Entry("with a DV with a no-csi storageclass", libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", "vol0")),
				libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", noCSIDVName))),
				fmt.Errorf("invalid volumes to update with migration: DV storage class isn't a CSI or not using volume populators: [disk0]")),
		)

		It("should return an error if the DV doesn't exist", func() {
			const dvName = "testdv"
			vmi := libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", "vol0"))
			vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", dvName)))
			err := volumemigration.ValidateVolumes(vmi, vm, dataVolumeStore, pvcStore)
			Expect(err).To(MatchError(fmt.Sprintf("the datavolume %s doesn't exist", dvName)))
		})

		It("should return an error if the PVC doesn't exist", func() {
			const dvName = "testdv"
			dv := libdv.NewDataVolume(libdv.WithNamespace(ns), libdv.WithName(dvName))
			Expect(dataVolumeStore.Add(dv)).To(Succeed())
			vmi := libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", "vol0"))
			vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", dvName)))
			err := volumemigration.ValidateVolumes(vmi, vm, dataVolumeStore, pvcStore)
			Expect(err).To(MatchError(fmt.Sprintf("the pvc %s doesn't exist", dvName)))
		})

		It("should validate the migrated volume with a DV in succeeded phase", func() {
			const dvName = "testdv"
			dv := libdv.NewDataVolume(libdv.WithNamespace(ns), libdv.WithName(dvName))
			dv.Status.Phase = cdiv1.Succeeded
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dvName,
					Namespace: ns,
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					DataSourceRef: pointer.P(k8sv1.TypedObjectReference{}),
				},
			}
			Expect(dataVolumeStore.Add(dv)).To(Succeed())
			Expect(pvcStore.Add(pvc)).To(Succeed())
			vmi := libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", "vol0"))
			vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(ns), libvmi.WithDataVolume("disk0", dvName)))
			err := volumemigration.ValidateVolumes(vmi, vm, dataVolumeStore, pvcStore)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VolumeMigrationCancel", func() {
		var (
			ctrl          *gomock.Controller
			virtClient    *kubecli.MockKubevirtClient
			fakeClientset *fake.Clientset
		)
		const ns = k8sv1.NamespaceDefault
		type migVolumes struct {
			volName string
			src     string
			dst     string
		}

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			fakeClientset = fake.NewSimpleClientset()
			virtClient.EXPECT().VirtualMachineInstance(ns).Return(fakeClientset.KubevirtV1().VirtualMachineInstances(ns)).AnyTimes()
		})

		DescribeTable("should evaluate the volume migration cancellation", func(vmiVols, vmVols []string, migVols []migVolumes, expectRes bool, expectErr error, expectCancellation bool) {
			vmi := libvmi.New(append(addVMIOptionsForVolumes(vmiVols), libvmi.WithNamespace(ns))...)
			vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
				Type: v1.VirtualMachineInstanceVolumesChange, Status: k8sv1.ConditionTrue,
			})
			for _, v := range migVols {
				vmi.Status.MigratedVolumes = append(vmi.Status.MigratedVolumes, v1.StorageMigratedVolumeInfo{
					VolumeName:         v.volName,
					SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: v.src},
					DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: v.dst},
				})
			}
			vm := libvmi.NewVirtualMachine(libvmi.New(append(addVMIOptionsForVolumes(vmVols), libvmi.WithNamespace(ns))...))

			_, err := fakeClientset.KubevirtV1().VirtualMachineInstances(ns).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = fakeClientset.KubevirtV1().VirtualMachines(ns).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Clear actions to easy check the length later
			fakeClientset.ClearActions()
			res, err := volumemigration.VolumeMigrationCancel(virtClient, vmi, vm)

			if expectCancellation {
				updatedVMI, err := fakeClientset.KubevirtV1().VirtualMachineInstances(ns).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVMI.Status.MigratedVolumes).To(BeEmpty())
				Expect(updatedVMI.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Type":   Equal(v1.VirtualMachineInstanceVolumesChange),
						"Status": Equal(k8sv1.ConditionFalse),
					},
				)))
				Expect(fakeClientset.Actions()).To(WithTransform(func(actions []testing.Action) []testing.Action {
					var patchOperations []testing.Action
					for _, action := range actions {
						if action.GetVerb() == "patch" && action.GetResource().Resource == "virtualmachineinstances" {
							patchOperations = append(patchOperations, action)
						}
					}
					return patchOperations
				}, HaveLen(2)))
			}
			if expectErr != nil {
				Expect(err).To(Equal(expectErr))
			} else {
				Expect(err).ShouldNot(HaveOccurred())
			}
			Expect(res).To(Equal(expectRes))
		},
			Entry("without any updates", []string{"dst0"}, []string{"dst0"}, []migVolumes{{generateDiskNameFromIndex(0), "src0", "dst0"}}, false, nil, false),
			Entry("with the migrated volumes reversion to the source volumes", []string{"dst0"}, []string{"src0"},
				[]migVolumes{{generateDiskNameFromIndex(0), "src0", "dst0"}}, true, nil, true),
			Entry("with invalid update", []string{"dst0"}, []string{"other"}, []migVolumes{{generateDiskNameFromIndex(0), "src0", "dst0"}}, true,
				fmt.Errorf(volumemigration.InvalidUpdateErrMsg), false),
			Entry("with invalid partial update", []string{"dst0", "dst1", "dst2"}, []string{"src0", "dst1", "dst2"}, []migVolumes{
				{generateDiskNameFromIndex(0), "src0", "dst0"}, {generateDiskNameFromIndex(1), "src1", "dst1"}, {generateDiskNameFromIndex(2), "src2", "dst2"},
			},
				true, fmt.Errorf(volumemigration.InvalidUpdateErrMsg), false),
		)
	})

	Context("IsVolumeMigrating", func() {
		DescribeTable("should detect the volume update condition", func(cond *v1.VirtualMachineInstanceCondition, expectRes bool) {
			vmi := libvmi.New()
			if cond != nil {
				vmi.Status.Conditions = append(vmi.Status.Conditions, *cond)
			}
			Expect(volumemigration.IsVolumeMigrating(vmi)).To(Equal(expectRes))
		},
			Entry("without the condition", nil, false),
			Entry("with true condition", &v1.VirtualMachineInstanceCondition{
				Type: v1.VirtualMachineInstanceVolumesChange, Status: k8sv1.ConditionTrue,
			}, true),
			Entry("without false condition", &v1.VirtualMachineInstanceCondition{
				Type: v1.VirtualMachineInstanceVolumesChange, Status: k8sv1.ConditionFalse,
			}, false),
		)
	})

	Context("PatchVMIStatusWithMigratedVolumes", func() {
		var (
			ctrl          *gomock.Controller
			virtClient    *kubecli.MockKubevirtClient
			fakeClientset *fake.Clientset
			pvcStore      cache.Store
		)
		const ns = k8sv1.NamespaceDefault
		type migVolumes struct {
			src string
			dst string
		}

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			fakeClientset = fake.NewSimpleClientset()
			virtClient.EXPECT().VirtualMachineInstance(ns).Return(fakeClientset.KubevirtV1().VirtualMachineInstances(ns)).AnyTimes()
			pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
			pvcStore = pvcInformer.GetStore()
		})
		shouldAddPVCsIntoTheStore := func(vmiVols, vmVols []string) {
			alreadyAddedVols := make(map[string]bool)
			for _, v := range vmiVols {
				pvcStore.Add(&k8sv1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: v, Namespace: ns},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						VolumeMode: virtpointer.P(k8sv1.PersistentVolumeFilesystem),
					},
				})
				alreadyAddedVols[v] = true
			}
			for _, v := range vmVols {
				if _, ok := alreadyAddedVols[v]; ok {
					continue
				}
				pvcStore.Add(&k8sv1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: v, Namespace: ns},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						VolumeMode: virtpointer.P(k8sv1.PersistentVolumeFilesystem),
					},
				})
			}
		}
		DescribeTable("should update the migrated volumes in the vmi", func(vmiVols, vmVols []string, expectedMigVols map[string]migVolumes) {
			shouldAddPVCsIntoTheStore(vmiVols, vmVols)
			vmi := libvmi.New(append(addVMIOptionsForVolumes(vmiVols), libvmi.WithNamespace(ns))...)
			vm := libvmi.NewVirtualMachine(libvmi.New(append(addVMIOptionsForVolumes(vmVols), libvmi.WithNamespace(ns))...))
			_, err := fakeClientset.KubevirtV1().VirtualMachineInstances(ns).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = fakeClientset.KubevirtV1().VirtualMachines(ns).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			volMig, err := volumemigration.GenerateMigratedVolumes(pvcStore, vmi, vm)
			Expect(err).ToNot(HaveOccurred())

			Expect(volumemigration.PatchVMIStatusWithMigratedVolumes(virtClient, volMig, vmi)).ToNot(HaveOccurred())

			if len(expectedMigVols) > 0 {
				updatedVMI, err := fakeClientset.KubevirtV1().VirtualMachineInstances(ns).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, migVol := range updatedVMI.Status.MigratedVolumes {
					v, ok := expectedMigVols[migVol.VolumeName]
					Expect(ok).To(BeTrue())
					Expect(migVol.SourcePVCInfo).ToNot(BeNil())
					Expect(migVol.DestinationPVCInfo).ToNot(BeNil())
					Expect(migVol.SourcePVCInfo.ClaimName).To(Equal(v.src))
					Expect(migVol.DestinationPVCInfo.ClaimName).To(Equal(v.dst))
					Expect(migVol.SourcePVCInfo.VolumeMode).To(HaveValue(Equal(k8sv1.PersistentVolumeFilesystem)))
					Expect(migVol.DestinationPVCInfo.VolumeMode).To(HaveValue(Equal(k8sv1.PersistentVolumeFilesystem)))
				}
			}
		},
			Entry("with an update of a volume", []string{"src0"}, []string{"dst0"},
				map[string]migVolumes{generateDiskNameFromIndex(0): {src: "src0", dst: "dst0"}}),
			Entry("with an update of multiple volumes", []string{"src0", "src1"}, []string{"dst0", "dst1"},
				map[string]migVolumes{
					generateDiskNameFromIndex(0): {src: "src0", dst: "dst0"},
					generateDiskNameFromIndex(1): {src: "src1", dst: "dst1"},
				}),
			Entry("without any update", []string{"vol0"}, []string{"vol0"}, map[string]migVolumes{}),
		)
	})

	Context("ValidateVolumesUpdateMigration", func() {
		DescribeTable("should validate if the VMI can be migrate due to a volume update", func(vmi *v1.VirtualMachineInstance, exectedRes error) {
			var err error
			if vmi == nil {
				err = volumemigration.ValidateVolumesUpdateMigration(vmi, nil, nil)
			} else {
				err = volumemigration.ValidateVolumesUpdateMigration(vmi, nil, vmi.Status.MigratedVolumes)
			}
			if exectedRes == nil {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(Equal(exectedRes))
			}
		},
			Entry("with nil VMI", nil, fmt.Errorf("VMI is empty")),
			Entry("with valid migrated volumes", libvmi.New(libvmistatus.WithStatus(
				v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName:         "disk0",
							SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: "src"},
							DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: "dst"},
						},
					},
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   v1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionFalse,
							Reason: v1.VirtualMachineInstanceReasonDisksNotMigratable,
						},
					},
				})), nil),
			Entry("with valid migrated volumes but unmigratable VMI", libvmi.New(libvmistatus.WithStatus(
				v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName:         "disk0",
							SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: "src"},
							DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: "dst"},
						},
					},
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:    v1.VirtualMachineInstanceIsStorageLiveMigratable,
							Status:  k8sv1.ConditionFalse,
							Reason:  v1.VirtualMachineInstanceReasonNotMigratable,
							Message: "non migratable test condition",
						},
					},
				})), fmt.Errorf("cannot migrate the volumes as the VMI isn't migratable: non migratable test condition")),
			Entry("with valid migrated volumes but with an additional RWO volume", libvmi.New(libvmistatus.WithStatus(
				v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName:         "disk0",
							SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: "src"},
							DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: "dst"},
						},
					},
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   v1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionFalse,
							Reason: v1.VirtualMachineInstanceReasonDisksNotMigratable,
						},
					},
					VolumeStatus: []v1.VolumeStatus{
						{
							Name: "disk0",
							PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName:   "src",
								AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
							},
						},
						{
							Name: "disk1",
							PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName:   "src",
								AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
							},
						},
					},
				})), fmt.Errorf("cannot migrate the VM. The volume disk1 is RWO and not included in the migration volumes")),
			Entry("with valid migrated volumes and persistent storage", libvmi.New(libvmi.WithName("test"), libvmi.WithTPM(true),
				libvmistatus.WithStatus(
					v1.VirtualMachineInstanceStatus{
						MigratedVolumes: []v1.StorageMigratedVolumeInfo{
							{
								VolumeName:         "disk0",
								SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: "src"},
								DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: "dst"},
							},
						},
						Conditions: []v1.VirtualMachineInstanceCondition{
							{
								Type:   v1.VirtualMachineInstanceIsMigratable,
								Status: k8sv1.ConditionFalse,
								Reason: v1.VirtualMachineInstanceReasonDisksNotMigratable,
							},
						},
						VolumeStatus: []v1.VolumeStatus{
							{
								Name: "disk0",
								PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
									ClaimName:   "src",
									AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
								},
							},
							{
								Name: "persistent-state-for-test",
								PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
									ClaimName:   "persistent-state-for-test",
									AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
								},
							},
						},
					})), nil),
		)
	})

	Context("PatchVMIVolumes", func() {
		var (
			ctrl          *gomock.Controller
			virtClient    *kubecli.MockKubevirtClient
			fakeClientset *fake.Clientset
		)
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			fakeClientset = fake.NewSimpleClientset()
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceNone).Return(fakeClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceNone)).AnyTimes()
		})

		It("should patch the VMI volumes", func() {
			volName := "disk0"
			vmi := libvmi.New(libvmi.WithPersistentVolumeClaim(volName, "vol0"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithMigratedVolume(v1.StorageMigratedVolumeInfo{VolumeName: volName}))))
			vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithPersistentVolumeClaim(volName, "vol1")))
			_, err := fakeClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceNone).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = fakeClientset.KubevirtV1().VirtualMachines(metav1.NamespaceNone).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = volumemigration.PatchVMIVolumes(virtClient, vmi, vm)
			Expect(err).ToNot(HaveOccurred())
			updatedVMI, err := fakeClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceNone).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVMI.Spec.Volumes).To(ContainElement(MatchFields(IgnoreExtras,
				Fields{
					"Name": Equal("disk0"),
					"VolumeSource": MatchFields(IgnoreExtras,
						Fields{
							"PersistentVolumeClaim": PointTo(MatchFields(IgnoreExtras,
								Fields{
									"PersistentVolumeClaimVolumeSource": MatchFields(IgnoreExtras,
										Fields{
											"ClaimName": Equal("vol1"),
										}),
								}),
							),
						}),
				}),
			))
		})

		DescribeTable("should not patch the VMI volumes", func(vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine) {
			vmiRes, err := volumemigration.PatchVMIVolumes(virtClient, vmi, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(vmiRes).To(Equal(vmi))
		},
			Entry("without the migrated volumes set", libvmi.New(libvmi.WithPersistentVolumeClaim("disk0", "vol0")),
				libvmi.NewVirtualMachine(libvmi.New(libvmi.WithPersistentVolumeClaim("disk0", "vol0")))),
			Entry("without any updates with a VM using a PVC", libvmi.New(libvmi.WithPersistentVolumeClaim("disk0", "vol0"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithMigratedVolume(v1.StorageMigratedVolumeInfo{VolumeName: "vol0"})))),
				libvmi.NewVirtualMachine(libvmi.New(libvmi.WithPersistentVolumeClaim("disk0", "vol0"))),
			),
			// The image pull policy for the container disks is set by the mutating webhook on the VMI spec but not on the VM.
			// This entry test simulates the scenario when the pull policy isn't set on the VM and the default is applied only
			// on the VMI spec.
			Entry("without any updates with a VM using a PVC and a containerdisk", libvmi.New(libvmi.WithPersistentVolumeClaim("disk0", "vol0"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithMigratedVolume(v1.StorageMigratedVolumeInfo{VolumeName: "vol0"}))),
				withContainerDisk("vol1", virtpointer.P(k8sv1.PullIfNotPresent))),
				libvmi.NewVirtualMachine(libvmi.New(libvmi.WithPersistentVolumeClaim("disk0", "vol0"),
					withContainerDisk("vol1", nil))),
			),
		)
	})
})

func addPVC(vmi *v1.VirtualMachineInstance, diskName, claim string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claim},
			},
		},
	})
}

func withShareableVolume(diskName, claim string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
			v1.Disk{
				Name: diskName,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio},
				},
				Shareable: virtpointer.P(true),
			})
		addPVC(vmi, diskName, claim)
	}
}

func withFilesystemVolume(diskName, claim string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems,
			v1.Filesystem{
				Name:     diskName,
				Virtiofs: &v1.FilesystemVirtiofs{},
			})
		addPVC(vmi, diskName, claim)
	}
}

func withHotpluggedVolume(diskName, claim string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
			v1.Disk{
				Name: diskName,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio},
				},
			})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: diskName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claim},
					Hotpluggable:                      true,
				},
			},
		})
	}
}

func withContainerDisk(volName string, pullPolicy *k8sv1.PullPolicy) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		var policy k8sv1.PullPolicy
		if pullPolicy == nil {
			policy = ""
		} else {
			policy = *pullPolicy
		}
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volName,
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{
					Image:           "image",
					ImagePullPolicy: policy,
				},
			},
		})
	}
}

func generateDiskNameFromIndex(i int) string {
	return fmt.Sprintf("disk%d", i)
}

func addVMIOptionsForVolumes(vols []string) []libvmi.Option {
	var ops []libvmi.Option
	for i, v := range vols {
		ops = append(ops, libvmi.WithPersistentVolumeClaim(generateDiskNameFromIndex(i), v))
	}
	return ops
}
