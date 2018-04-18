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

	Context("with admission review", func() {
		It("reject invalid disk", func() {
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
		})
		It("reject invalid volume", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk:          &v1.RegistryDiskSource{},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
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
			Expect(resp.Allowed).To(Equal(false))
		})
		table.DescribeTable("should accept valid volumes",
			func(volumeSource v1.VolumeSource) {
				vm := v1.NewMinimalVM("testvm")

				vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
					Name:         "testvolume",
					VolumeSource: volumeSource,
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
			},
			table.Entry("with pvc volume source", v1.VolumeSource{PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{}}),
			table.Entry("with cloud-init volume source", v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "fake"}}),
			table.Entry("with registryDisk volume source", v1.VolumeSource{RegistryDisk: &v1.RegistryDiskSource{}}),
			table.Entry("with ephemeral volume source", v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}),
			table.Entry("with emptyDisk volume source", v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}),
		)
		table.DescribeTable("should accept valid disks",
			func(disk v1.Disk, volume v1.Volume) {
				vm := v1.NewMinimalVM("testvm")

				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, disk)
				vm.Spec.Volumes = append(vm.Spec.Volumes, volume)

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
			},
			table.Entry("with Disk target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}},
				v1.Volume{Name: "testvolume", VolumeSource: v1.VolumeSource{RegistryDisk: &v1.RegistryDiskSource{Image: "fake"}}},
			),
			table.Entry("with LUN target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{}}},
				v1.Volume{Name: "testvolume", VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{}}},
			),
			table.Entry("with Floppy target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Floppy: &v1.FloppyTarget{}}},
				v1.Volume{Name: "testvolume", VolumeSource: v1.VolumeSource{RegistryDisk: &v1.RegistryDiskSource{Image: "fake"}}},
			),
			table.Entry("with CDRom target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{}}},
				v1.Volume{Name: "testvolume", VolumeSource: v1.VolumeSource{RegistryDisk: &v1.RegistryDiskSource{Image: "fake"}}},
			),
		)
	})
	Context("with VM disk", func() {
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

			errors := validateDisks(vm)
			Expect(len(errors)).To(Equal(0))
		})
		It("should reject disk with missing volume", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})

			errors := validateDisks(vm)
			Expect(len(errors)).To(Equal(1))
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

			errors := validateDisks(vm)
			Expect(len(errors)).To(Equal(1))
		})
		It("should generate multiple errors", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})

			errors := validateDisks(vm)
			// missing volume and multiple targets set. should result in 2 errors
			Expect(len(errors)).To(Equal(2))
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

				errors := validateDisks(vm)
				Expect(len(errors)).To(Equal(expectedErrors))
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
	Context("with VM volume", func() {
		It("should reject volume no volume source set", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
			})

			errors := validateVolumes(vm)
			Expect(len(errors)).To(Equal(1))
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

			errors := validateVolumes(vm)
			Expect(len(errors)).To(Equal(1))
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

			errors := validateVolumes(vm)
			Expect(len(errors)).To(Equal(expectedErrors))
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

			errors := validateVolumes(vm)
			Expect(len(errors)).To(Equal(1))
		})
	})
})
