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
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Validating Webhook", func() {
	Context("with VM disk", func() {
		It("should accept a valid disk", func() {
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

			errors := validateDisks(vm)
			// len  == 2 because missing volume and multiple targets set
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
		It("should accept valid volume", func() {
			vm := v1.NewMinimalVM("testvm")

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})

			errors := validateVolumes(vm)
			Expect(len(errors)).To(Equal(0))
		})
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
		table.DescribeTable("should verify cloud-init userdata length", func(userDataLen int, expectedErrors int) {
			vm := v1.NewMinimalVM("testvm")

			// generate fake userdata
			userdata := ""
			for i := 0; i < userDataLen; i++ {
				userdata = fmt.Sprintf("%sa", userdata)
			}

			vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserData: userdata,
					},
				},
			})

			errors := validateVolumes(vm)
			Expect(len(errors)).To(Equal(expectedErrors))
		},
			table.Entry("should accept userdata under max limit", 10, 0),
			table.Entry("should accept userdata equal max limit", cloudInitMaxLen, 0),
			table.Entry("should reject userdata greater than max limit", cloudInitMaxLen+1, 1),
		)
	})
})
