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
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
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
		appendRegistryDisk(vm, "r0")

		// create a fake disk file
		volumeMountDir := generateVMBaseDir(vm)
		err = os.MkdirAll(volumeMountDir+"/disk_r0", 0750)
		Expect(err).ToNot(HaveOccurred())

		filePath := volumeMountDir + "/disk_r0/disk-image." + diskExtension
		_, err := os.Create(filePath)

		err = TakeOverRegistryDisks(vm)
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
			table.DescribeTable("by verifying mapping of ",
				func(diskType string) {
					VerifyDiskType(diskType)
				},
				table.Entry("qcow2 disk", "qcow2"),
				table.Entry("raw disk", "raw"),
			)
			It("by verifying error when no disk is present", func() {

				vm := v1.NewMinimalVM("fake-vm")
				appendRegistryDisk(vm, "r0")

				err := TakeOverRegistryDisks(vm)
				Expect(err).To(HaveOccurred())
			})
			It("by verifying container generation", func() {
				vm := v1.NewMinimalVM("fake-vm")
				appendRegistryDisk(vm, "r1")
				appendRegistryDisk(vm, "r0")
				containers, volumes, err := GenerateContainers(vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(containers)).To(Equal(2))
				Expect(len(volumes)).To(Equal(2))
			})

			It("by removing unseen registry disk data", func() {
				var domains []string

				domains = append(domains, "fakens1/fakedomain1")
				domains = append(domains, "fakens1/fakedomain2")
				domains = append(domains, "fakens2/fakedomain1")
				domains = append(domains, "fakens2/fakedomain2")
				domains = append(domains, "fakens3/fakedomain1")
				domains = append(domains, "fakens4/fakedomain1")

				for _, dom := range domains {
					err := os.MkdirAll(fmt.Sprintf("%s/%s/some-other-dir", tmpDir, dom), 0755)
					Expect(err).ToNot(HaveOccurred())
					msg := "fake content"
					bytes := []byte(msg)
					err = ioutil.WriteFile(fmt.Sprintf("%s/%s/some-file", tmpDir, dom), bytes, 0644)
					Expect(err).ToNot(HaveOccurred())
				}

				vmStore := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})

				err := vmStore.Add(v1.NewVMReferenceFromNameWithNS("fakens1", "fakedomain1"))
				Expect(err).ToNot(HaveOccurred())

				// verifies VM data in finalized state are removed
				vm := v1.NewVMReferenceFromNameWithNS("fakens1", "fakedomain2")
				vm.Status.Phase = v1.Succeeded
				err = vmStore.Add(vm)
				Expect(err).ToNot(HaveOccurred())

				err = CleanupOrphanedEphemeralDisks(vmStore)
				Expect(err).ToNot(HaveOccurred())

				// expect this domain to still exist
				_, err = os.Stat(fmt.Sprintf("%s/fakens1/fakedomain1", tmpDir))
				Expect(err).ToNot(HaveOccurred())

				// expect these domains to not exist
				for idx, dom := range domains {
					exists := true
					if idx == 0 {
						continue
					}
					_, err = os.Stat(fmt.Sprintf("%s/%s", tmpDir, dom))
					if os.IsNotExist(err) {
						exists = false
					}
					Expect(exists).To(Equal(false))
				}

				// verify cleaning up a non-existent directory does not fail
				err = SetLocalDirectory(tmpDir + "/made/this/path/up")
				Expect(err).ToNot(HaveOccurred())
				err = CleanupOrphanedEphemeralDisks(vmStore)
				Expect(err).ToNot(HaveOccurred())

			})

			It("by verifying data cleanup", func() {
				vm := v1.NewMinimalVM("fake-vm")
				appendRegistryDisk(vm, "r0")
				volumeMountDir := generateVMBaseDir(vm)
				err = os.MkdirAll(volumeMountDir, 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.MkdirAll(volumeMountDir+"/disk_r0", 0755)
				Expect(err).ToNot(HaveOccurred())
				err = os.MkdirAll(volumeMountDir+"/disk_r1", 0755)
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

func appendRegistryDisk(vm *v1.VirtualMachine, diskName string) {
	vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{},
		},
	})
	vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			RegistryDisk: &v1.RegistryDiskSource{
				Image: "someimage:v1.2.3.4",
			},
		},
	})
}
