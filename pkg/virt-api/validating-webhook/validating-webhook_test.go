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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package validating_webhook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1beta1 "k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Validating Webhook", func() {
	Context("with VM admission review", func() {
		It("reject invalid VM spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vmBytes, _ := json.Marshal(&vm)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
					Object: runtime.RawExtension{
						Raw: vmBytes,
					},
				},
			}

			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0].volumeName"))
		})
		It("should accept valid vm spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})
			vmBytes, _ := json.Marshal(&vm)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"},
					Object: runtime.RawExtension{
						Raw: vmBytes,
					},
				},
			}
			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})
	Context("with VMRS admission review", func() {
		It("reject invalid VM spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vmrs := &v1.VirtualMachineReplicaSet{
				Spec: v1.VMReplicaSetSpec{
					Template: &v1.VMTemplateSpec{
						Spec: vm.Spec,
					},
				},
			}
			vmrsBytes, _ := json.Marshal(&vmrs)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VMReplicaSetGroupVersionKind.Group,
						Version:  v1.VMReplicaSetGroupVersionKind.Version,
						Resource: "virtualmachinereplicasets",
					},
					Object: runtime.RawExtension{
						Raw: vmrsBytes,
					},
				},
			}

			resp := admitVMRS(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[0].volumeName"))
		})
		It("should accept valid vm spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})

			vmrs := &v1.VirtualMachineReplicaSet{
				Spec: v1.VMReplicaSetSpec{
					Template: &v1.VMTemplateSpec{
						Spec: vm.Spec,
					},
				},
			}
			vmrsBytes, _ := json.Marshal(&vmrs)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VMReplicaSetGroupVersionKind.Group,
						Version:  v1.VMReplicaSetGroupVersionKind.Version,
						Resource: "virtualmachinereplicasets",
					},
					Object: runtime.RawExtension{
						Raw: vmrsBytes,
					},
				},
			}

			resp := admitVMRS(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})

	Context("with OVM admission review", func() {
		It("reject invalid VM spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			ovm := &v1.OfflineVirtualMachine{
				Spec: v1.OfflineVirtualMachineSpec{
					Running: false,
					Template: &v1.VMTemplateSpec{
						Spec: vm.Spec,
					},
				},
			}
			ovmBytes, _ := json.Marshal(&ovm)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.OfflineVirtualMachineGroupVersionKind.Group,
						Version:  v1.OfflineVirtualMachineGroupVersionKind.Version,
						Resource: "offlinevirtualmachines",
					},
					Object: runtime.RawExtension{
						Raw: ovmBytes,
					},
				},
			}

			resp := admitOVMs(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[0].volumeName"))
		})
		It("should accept valid vm spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})

			ovm := &v1.OfflineVirtualMachine{
				Spec: v1.OfflineVirtualMachineSpec{
					Running: false,
					Template: &v1.VMTemplateSpec{
						Spec: vm.Spec,
					},
				},
			}
			ovmBytes, _ := json.Marshal(&ovm)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.OfflineVirtualMachineGroupVersionKind.Group,
						Version:  v1.OfflineVirtualMachineGroupVersionKind.Version,
						Resource: "offlinevirtualmachines",
					},
					Object: runtime.RawExtension{
						Raw: ovmBytes,
					},
				},
			}

			resp := admitOVMs(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})
	Context("with VMPreset admission review", func() {
		It("reject invalid VM spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})
			vmPreset := &v1.VirtualMachinePreset{
				Spec: v1.VirtualMachinePresetSpec{
					Domain: &vm.Spec.Domain,
				},
			}
			vmPresetBytes, _ := json.Marshal(&vmPreset)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachinePresetGroupVersionKind.Group,
						Version:  v1.VirtualMachinePresetGroupVersionKind.Version,
						Resource: "virtualmachinepresets",
					},
					Object: runtime.RawExtension{
						Raw: vmPresetBytes,
					},
				},
			}

			resp := admitVMPreset(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0]"))
		})
		It("should accept valid vm spec", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})

			vmPreset := &v1.VirtualMachinePreset{
				Spec: v1.VirtualMachinePresetSpec{
					Domain: &v1.DomainSpec{},
				},
			}
			vmPresetBytes, _ := json.Marshal(&vmPreset)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachinePresetGroupVersionKind.Group,
						Version:  v1.VirtualMachinePresetGroupVersionKind.Version,
						Resource: "virtualmachinepresets",
					},
					Object: runtime.RawExtension{
						Raw: vmPresetBytes,
					},
				},
			}

			resp := admitVMPreset(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})

	Context("with VM spec", func() {
		It("should accept disk and volume lists equal to max element length", func() {
			vm := v1.NewMinimalVM("testvm")

			for i := 0; i < arrayLenMax; i++ {
				diskName := fmt.Sprintf("testDisk%d", i)
				volumeName := fmt.Sprintf("testVolume%d", i)
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       diskName,
					VolumeName: volumeName,
				})
				vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{},
					},
				})
			}

			causes := validateVirtualMachineSpec("fake.", &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject disk lists greater than max element length", func() {
			vm := v1.NewMinimalVM("testvm")

			for i := 0; i <= arrayLenMax; i++ {
				diskName := "testDisk"
				volumeName := "testVolume"
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       diskName,
					VolumeName: volumeName,
				})
			}

			causes := validateVirtualMachineSpec("fake.", &vm.Spec)
			// if this is processed correctly, it should result in a single error
			// If multiple causes occurred, then the spec was processed too far.
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks"))
		})
		It("should reject volume lists greater than max element length", func() {
			vm := v1.NewMinimalVM("testvm")

			for i := 0; i <= arrayLenMax; i++ {
				volumeName := "testVolume"
				vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{},
					},
				})
			}

			causes := validateVirtualMachineSpec("fake.", &vm.Spec)
			// if this is processed correctly, it should result in a single error
			// If multiple causes occurred, then the spec was processed too far.
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.volumes"))
		})

		It("should reject disk with missing volume", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})

			causes := validateVirtualMachineSpec("fake.", &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].volumeName"))
		})
		It("should reject multiple disks referencing same volume", func() {
			vm := v1.NewMinimalVM("testvm")

			// verify two disks referencing the same volume are rejected
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume",
			})

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})
			causes := validateVirtualMachineSpec("fake.", &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].volumeName"))
		})
		It("should generate multiple causes", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})

			causes := validateVirtualMachineSpec("fake.", &vm.Spec)
			// missing volume and multiple targets set. should result in 2 causes
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].volumeName"))
			Expect(causes[1].Field).To(Equal("fake.domain.devices.disks[0]"))
		})
		table.DescribeTable("should verify LUN is mapped to PVC volume",
			func(volume *v1.Volume, expectedErrors int) {
				vm := v1.NewMinimalVM("testvm")
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       "testdisk",
					VolumeName: "testvolume",
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{},
					},
				})
				vm.Spec.Volumes = append(vm.Spec.Volumes, *volume)

				causes := validateVirtualMachineSpec("fake.", &vm.Spec)
				Expect(len(causes)).To(Equal(expectedErrors))
			},
			table.Entry("and reject non PVC sources",
				&v1.Volume{
					Name: "testvolume",
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{},
					},
				}, 1),
			table.Entry("and accept PVC sources",
				&v1.Volume{
					Name: "testvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
					},
				}, 0),
		)

	})
	Context("with Volume", func() {
		table.DescribeTable("should accept valid volumes",
			func(volumeSource v1.VolumeSource) {
				vm := v1.NewMinimalVM("testvm")
				vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
					Name:         "testvolume",
					VolumeSource: volumeSource,
				})

				causes := validateVolumes("fake.", vm.Spec.Volumes)
				Expect(len(causes)).To(Equal(0))
			},
			table.Entry("with pvc volume source", v1.VolumeSource{PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{}}),
			table.Entry("with cloud-init volume source", v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "fake"}}),
			table.Entry("with registryDisk volume source", v1.VolumeSource{RegistryDisk: &v1.RegistryDiskSource{}}),
			table.Entry("with ephemeral volume source", v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}),
			table.Entry("with emptyDisk volume source", v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}),
		)
		It("should reject volume with no volume source set", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
			})

			causes := validateVolumes("fake", vm.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volume with multiple volume sources set", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk:          &v1.RegistryDiskSource{},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
				},
			})

			causes := validateVolumes("fake", vm.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volumes with duplicate names", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})
			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})

			causes := validateVolumes("fake", vm.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})
		table.DescribeTable("should verify cloud-init userdata length", func(userDataLen int, expectedErrors int, base64Encode bool) {
			vm := v1.NewMinimalVM("testvm")

			// generate fake userdata
			userdata := ""
			for i := 0; i < userDataLen; i++ {
				userdata = fmt.Sprintf("%sa", userdata)
			}

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}})

			if base64Encode {
				vm.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(userdata))
			} else {
				vm.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserData = userdata
			}

			causes := validateVolumes("fake", vm.Spec.Volumes)
			Expect(len(causes)).To(Equal(expectedErrors))
			for _, cause := range causes {
				Expect(cause.Field).To(ContainSubstring("fake[0].cloudInitNoCloud"))
			}
		},
			table.Entry("should accept userdata under max limit", 10, 0, false),
			table.Entry("should accept userdata equal max limit", cloudInitMaxLen, 0, false),
			table.Entry("should reject userdata greater than max limit", cloudInitMaxLen+1, 1, false),
			table.Entry("should accept userdata base64 under max limit", 10, 0, true),
			table.Entry("should accept userdata base64 equal max limit", cloudInitMaxLen, 0, true),
			table.Entry("should reject userdata base64 greater than max limit", cloudInitMaxLen+1, 1, true),
		)

		It("should reject cloud-init with invalid base64 data", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserDataBase64: "#######garbage******",
					},
				},
			})

			causes := validateVolumes("fake", vm.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud.userDataBase64"))
		})
	})
	Context("with Disk", func() {
		table.DescribeTable("should accept valid disks",
			func(disk v1.Disk) {
				vm := v1.NewMinimalVM("testvm")

				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, disk)

				causes := validateDisks("fake.", vm.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(0))

			},
			table.Entry("with Disk target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}},
			),
			table.Entry("with LUN target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{}}},
			),
			table.Entry("with Floppy target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Floppy: &v1.FloppyTarget{}}},
			),
			table.Entry("with CDRom target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{}}},
			),
		)
		It("should allow disk without a target", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				// disk without a target defaults to DiskTarget
			})
			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{Image: "fake"},
				},
			})

			causes := validateDisks("fake.", vm.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject disks with duplicate names ", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume2",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			causes := validateDisks("fake", vm.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})

		It("should reject disk with multiple targets ", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})
			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{Image: "fake"},
				},
			})

			causes := validateDisks("fake", vm.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
	})
})
