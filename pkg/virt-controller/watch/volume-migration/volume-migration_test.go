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
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	volumemigration "kubevirt.io/kubevirt/pkg/virt-controller/watch/volume-migration"
)

var _ = Describe("Volume Migration", func() {
	Context("ValidateVolumes", func() {
		DescribeTable("should validate the migrated volumes", func(vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine, expectError error) {
			err := volumemigration.ValidateVolumes(vmi, vm)
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
			Entry("with an invalid hotplugged volume", libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol0"), withFilesystemVolume("disk1", "vol1"),
			), libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk0", "vol2"), withHotpluggedVolume("disk1", "vol4"),
			)), fmt.Errorf("invalid volumes to update with migration: hotplugged: [disk1]")),
		)
	})

	Context("VolumeMigrationCancel", func() {
		var (
			ctrl         *gomock.Controller
			virtClient   *kubecli.MockKubevirtClient
			vmiInterface *kubecli.MockVirtualMachineInstanceInterface
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
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			virtClient.EXPECT().VirtualMachineInstance(ns).Return(vmiInterface).AnyTimes()
		})

		shouldPatchVMI := func(vmi *v1.VirtualMachineInstance) {
			// The first patch operation is for the volumes in the VMI spec
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), metav1.PatchOptions{}).
				DoAndReturn(func(ctx context.Context, name, patchType, patch, opts interface{}, subs ...interface{}) (*v1.VirtualMachineInstance, error) {
					originalVMIBytes, err := json.Marshal(vmi)
					Expect(err).ToNot(HaveOccurred())
					patchBytes := patch.([]byte)

					patchJSON, err := jsonpatch.DecodePatch(patchBytes)
					Expect(err).ToNot(HaveOccurred())
					newVMIBytes, err := patchJSON.Apply(originalVMIBytes)
					Expect(err).ToNot(HaveOccurred())

					var newVMI *v1.VirtualMachineInstance
					err = json.Unmarshal(newVMIBytes, &newVMI)
					Expect(err).ToNot(HaveOccurred())
					return newVMI, nil
				})
			// The second patch operation is for the conditions and the migrated volumes
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), metav1.PatchOptions{}).
				Do(func(ctx context.Context, name, patchType, patch, opts interface{}, subs ...interface{}) {
					originalVMIBytes, err := json.Marshal(vmi)
					Expect(err).ToNot(HaveOccurred())
					patchBytes := patch.([]byte)

					patchJSON, err := jsonpatch.DecodePatch(patchBytes)
					Expect(err).ToNot(HaveOccurred())
					newVMIBytes, err := patchJSON.Apply(originalVMIBytes)
					Expect(err).ToNot(HaveOccurred())

					var newVMI *v1.VirtualMachineInstance
					err = json.Unmarshal(newVMIBytes, &newVMI)
					Expect(err).ToNot(HaveOccurred())

					condManager := controller.NewVirtualMachineInstanceConditionManager()
					c := condManager.GetCondition(newVMI, v1.VirtualMachineInstanceVolumesChange)
					Expect(c).ToNot(BeNil())
					Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(newVMI.Status.MigratedVolumes).To(BeEmpty())
				})
		}
		DescribeTable("should evaluate the volume migration cancellation", func(vmiVols, vmVols []string, migVols []migVolumes, expectRes bool, expectErr error, expectCancellation bool) {
			vmi := libvmi.New(append(addVMIOptionsForVolumes(vmiVols), libvmi.WithNamespace(ns))...)
			vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
				Type: v1.VirtualMachineInstanceVolumesChange, Status: k8sv1.ConditionTrue})
			for _, v := range migVols {
				vmi.Status.MigratedVolumes = append(vmi.Status.MigratedVolumes, v1.StorageMigratedVolumeInfo{
					VolumeName:         v.volName,
					SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: v.src},
					DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: v.dst},
				})
			}
			vm := libvmi.NewVirtualMachine(libvmi.New(append(addVMIOptionsForVolumes(vmVols), libvmi.WithNamespace(ns))...))

			if expectCancellation {
				shouldPatchVMI(vmi)
			}
			res, err := volumemigration.VolumeMigrationCancel(virtClient, vmi, vm)
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
				{generateDiskNameFromIndex(0), "src0", "dst0"}, {generateDiskNameFromIndex(1), "src1", "dst1"}, {generateDiskNameFromIndex(2), "src2", "dst2"}},
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
				Type: v1.VirtualMachineInstanceVolumesChange, Status: k8sv1.ConditionTrue}, true),
			Entry("without false condition", &v1.VirtualMachineInstanceCondition{
				Type: v1.VirtualMachineInstanceVolumesChange, Status: k8sv1.ConditionFalse}, false),
		)
	})

	Context("PatchVMIStatusWithMigratedVolumes", func() {
		var (
			ctrl         *gomock.Controller
			virtClient   *kubecli.MockKubevirtClient
			vmiInterface *kubecli.MockVirtualMachineInstanceInterface
			pvcStore     cache.Store
		)
		const ns = k8sv1.NamespaceDefault
		type migVolumes struct {
			src string
			dst string
		}

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			virtClient.EXPECT().VirtualMachineInstance(ns).Return(vmiInterface).AnyTimes()
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
		shouldPatchVMI := func(vmi *v1.VirtualMachineInstance, expectedMigVols map[string]migVolumes) {
			vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), metav1.PatchOptions{}).
				Do(func(ctx context.Context, name, patchType, patch, opts interface{}, subs ...interface{}) {
					originalVMIBytes, err := json.Marshal(vmi)
					Expect(err).ToNot(HaveOccurred())
					patchBytes := patch.([]byte)

					patchJSON, err := jsonpatch.DecodePatch(patchBytes)
					Expect(err).ToNot(HaveOccurred())
					newVMIBytes, err := patchJSON.Apply(originalVMIBytes)
					Expect(err).ToNot(HaveOccurred())

					var newVMI *v1.VirtualMachineInstance
					err = json.Unmarshal(newVMIBytes, &newVMI)
					Expect(err).ToNot(HaveOccurred())
					for _, migVol := range vmi.Status.MigratedVolumes {
						v, ok := expectedMigVols[migVol.VolumeName]
						Expect(ok).To(BeTrue())
						Expect(migVol.SourcePVCInfo).ToNot(BeNil())
						Expect(migVol.DestinationPVCInfo).ToNot(BeNil())
						Expect(migVol.SourcePVCInfo.ClaimName).To(Equal(v.src))
						Expect(migVol.DestinationPVCInfo.ClaimName).To(Equal(v.dst))
						Expect(migVol.SourcePVCInfo.VolumeMode).To(HaveValue(Equal(k8sv1.PersistentVolumeFilesystem)))
						Expect(migVol.DestinationPVCInfo.VolumeMode).To(HaveValue(Equal(k8sv1.PersistentVolumeFilesystem)))
					}
				})
		}
		DescribeTable("should update the migrated volumes in the vmi", func(vmiVols, vmVols []string, expectedMigVols map[string]migVolumes) {
			shouldAddPVCsIntoTheStore(vmiVols, vmVols)
			vmi := libvmi.New(append(addVMIOptionsForVolumes(vmiVols), libvmi.WithNamespace(ns))...)
			vm := libvmi.NewVirtualMachine(libvmi.New(append(addVMIOptionsForVolumes(vmVols), libvmi.WithNamespace(ns))...))
			if len(expectedMigVols) > 0 {
				shouldPatchVMI(vmi, expectedMigVols)
			}

			Expect(volumemigration.PatchVMIStatusWithMigratedVolumes(virtClient, pvcStore, vmi, vm)).ToNot(HaveOccurred())
		},
			Entry("with an update of a volume", []string{"src0"}, []string{"dst0"},
				map[string]migVolumes{generateDiskNameFromIndex(0): migVolumes{src: "src0", dst: "dst0"}}),
			Entry("with an update of multiple volumes", []string{"src0", "src1"}, []string{"dst0", "dst1"},
				map[string]migVolumes{generateDiskNameFromIndex(0): migVolumes{src: "src0", dst: "dst0"},
					generateDiskNameFromIndex(1): migVolumes{src: "src1", dst: "dst1"}}),
			Entry("without any update", []string{"vol0"}, []string{"vol0"}, map[string]migVolumes{}),
		)

	})
})

func addPVC(vmi *v1.VirtualMachineInstance, diskName, claim string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claim}},
		},
	})
}

func withShareableVolume(diskName, claim string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
			v1.Disk{Name: diskName,
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
			v1.Disk{Name: diskName,
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
