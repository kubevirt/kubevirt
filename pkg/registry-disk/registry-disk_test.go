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
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("RegistryDisk", func() {
	tmpDir, _ := ioutil.TempDir("", "registrydisktest")
	owner, err := user.Current()
	if err != nil {
		panic(err)
	}

	VerifyDiskType := func(diskExtension string) {
		vmi := v1.NewMinimalVMI("fake-vmi")
		appendRegistryDisk(vmi, "r0")

		// create a fake disk file
		volumeMountDir := generateVMIBaseDir(vmi)
		err = os.MkdirAll(volumeMountDir+"/disk_r0", 0750)
		Expect(err).ToNot(HaveOccurred())

		filePath := volumeMountDir + "/disk_r0/disk-image." + diskExtension
		_, err := os.Create(filePath)

		err = SetFilePermissions(vmi)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		os.MkdirAll(tmpDir, 0755)
		err := SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		SetLocalDataOwner(owner.Username)
	})

	AfterEach(func() {
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

				vmi := v1.NewMinimalVMI("fake-vmi")
				appendRegistryDisk(vmi, "r0")

				err := SetFilePermissions(vmi)
				Expect(err).To(HaveOccurred())
			})
			It("by verifying container generation", func() {
				vmi := v1.NewMinimalVMI("fake-vmi")
				appendRegistryDisk(vmi, "r1")
				appendRegistryDisk(vmi, "r0")
				containers := GenerateContainers(vmi, "libvirt-runtime", "/var/run/libvirt")
				Expect(err).ToNot(HaveOccurred())

				Expect(len(containers)).To(Equal(2))
			})
		})
	})
})

func appendRegistryDisk(vmi *v1.VirtualMachineInstance, diskName string) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			RegistryDisk: &v1.RegistryDiskSource{
				Image: "someimage:v1.2.3.4",
			},
		},
	})
}
