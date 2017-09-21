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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package registrydisk

import (
	"io/ioutil"
	"os"
	"os/user"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

var _ = Describe("RegistryDisk", func() {
	tmpDir, _ := ioutil.TempDir("", "registrydisktest")
	owner, err := user.Current()
	if err != nil {
		panic(err)
	}

	VerifyDiskType := func(diskExtension string) {
		vm := v1.NewMinimalVM("fake-vm")
		vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
			Type:   "ContainerRegistryDisk:v1alpha",
			Device: "disk",
			Source: v1.DiskSource{
				Name: "someimage:v1.2.3.4",
			},
			Target: v1.DiskTarget{
				Device: "vda",
			},
		})

		// create a fake disk file
		volumeMountDir := generateVMBaseDir(vm)
		err = os.MkdirAll(volumeMountDir+"/disk0", 0750)
		Expect(err).ToNot(HaveOccurred())

		filePath := volumeMountDir + "/disk0/disk-image." + diskExtension
		_, err := os.Create(filePath)

		vm, err = MapRegistryDisks(vm)
		Expect(err).ToNot(HaveOccurred())

		// verify file gets renamed by virt-handler to prevent container from
		// removing it before VM is exited
		exists, err := diskutils.FileExists(filePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(false))

		// verify file rename takes place
		exists, err = diskutils.FileExists(filePath + ".virt")
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(Equal(true))

		Expect(vm.Spec.Domain.Devices.Disks[0].Type).To(Equal("file"))
		Expect(vm.Spec.Domain.Devices.Disks[0].Target.Device).To(Equal("vda"))
		Expect(vm.Spec.Domain.Devices.Disks[0].Driver).ToNot(Equal(nil))
		Expect(vm.Spec.Domain.Devices.Disks[0].Driver.Type).To(Equal(diskExtension))
		Expect(vm.Spec.Domain.Devices.Disks[0].Source).ToNot(Equal(nil))
		Expect(vm.Spec.Domain.Devices.Disks[0].Source.File).To(Equal(filePath + ".virt"))

		err = CleanupEphemeralDisks(vm)
		exists, err = diskutils.FileExists(volumeMountDir)
		Expect(err).ToNot(HaveOccurred())

		Expect(exists).To(Equal(false))
	}

	BeforeSuite(func() {
		err := SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		SetLocalDataOwner(owner.Username)
	})

	AfterSuite(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("registry-disk", func() {
		Context("verify helper functions", func() {
			It("by verifying error when no disk is present", func() {

				vm := v1.NewMinimalVM("fake-vm")
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Type:   "ContainerRegistryDisk:v1alpha",
					Device: "disk",
					Source: v1.DiskSource{
						Name: "someimage:v1.2.3.4",
					},
					Target: v1.DiskTarget{
						Device: "vda",
					},
				})

				vm, err := MapRegistryDisks(vm)
				Expect(err).To(HaveOccurred())
			})

			It("by verifying mapping of qcow2 disk", func() {
				VerifyDiskType("qcow2")
			})

			It("by verifying mapping of raw disk", func() {
				VerifyDiskType("raw")
			})

			It("by verifying container generation", func() {
				vm := v1.NewMinimalVM("fake-vm")
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Type:   "ContainerRegistryDisk:v1alpha",
					Device: "disk",
					Source: v1.DiskSource{
						Name: "someimage:v1.2.3.4",
					},
					Target: v1.DiskTarget{
						Device: "vda",
					},
				})
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Type:   "ContainerRegistryDisk:v1alpha",
					Device: "disk",
					Source: v1.DiskSource{
						Name: "someimage:v1.2.3.4",
					},
					Target: v1.DiskTarget{
						Device: "vdb",
					},
				})

				containers, volumes, err := GenerateContainers(vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(containers)).To(Equal(2))
				Expect(len(volumes)).To(Equal(2))
			})

			It("by verifying data cleanup", func() {

				vm := v1.NewMinimalVM("fake-vm")
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Type:   "ContainerRegistryDisk:v1alpha",
					Device: "disk",
					Source: v1.DiskSource{
						Name: "someimage:v1.2.3.4",
					},
					Target: v1.DiskTarget{
						Device: "vda",
					},
				})

				volumeMountDir := generateVMBaseDir(vm)
				err = os.MkdirAll(volumeMountDir, 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.MkdirAll(volumeMountDir+"/disk0", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.MkdirAll(volumeMountDir+"/disk1", 0755)
				Expect(err).ToNot(HaveOccurred())

				exists, err := diskutils.FileExists(volumeMountDir)
				Expect(err).ToNot(HaveOccurred())

				Expect(exists).To(Equal(true))

				err = CleanupEphemeralDisks(vm)
				exists, err = diskutils.FileExists(volumeMountDir)
				Expect(err).ToNot(HaveOccurred())

				Expect(exists).To(Equal(false))

			})
		})
	})
})
