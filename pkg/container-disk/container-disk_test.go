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

package containerdisk

import (
	"io/ioutil"
	"os"
	"os/user"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("ContainerDisk", func() {
	tmpDir, _ := ioutil.TempDir("", "containerdisktest")
	owner, err := user.Current()
	if err != nil {
		panic(err)
	}

	VerifyDiskType := func(diskExtension string) {
		vmi := v1.NewMinimalVMI("fake-vmi")
		appendContainerDisk(vmi, "r0")

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

	Describe("container-disk", func() {
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
				appendContainerDisk(vmi, "r0")

				err := SetFilePermissions(vmi)
				Expect(err).To(HaveOccurred())
			})
			It("by verifying that resources are set if the VMI wants the guaranteed QOS class", func() {

				vmi := v1.NewMinimalVMI("fake-vmi")
				appendContainerDisk(vmi, "r0")
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceCPU:    resource.MustParse("1"),
						k8sv1.ResourceMemory: resource.MustParse("64M"),
					},
					Limits: k8sv1.ResourceList{
						k8sv1.ResourceCPU:    resource.MustParse("1"),
						k8sv1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				containers := GenerateContainers(vmi, "libvirt-runtime", "/var/run/libvirt")
				Expect(containers[0].Resources.Limits).To(HaveLen(2))
			})
			It("by verifying container generation", func() {
				vmi := v1.NewMinimalVMI("fake-vmi")
				appendContainerDisk(vmi, "r1")
				appendContainerDisk(vmi, "r0")
				containers := GenerateContainers(vmi, "libvirt-runtime", "/var/run/libvirt")
				Expect(err).ToNot(HaveOccurred())

				Expect(len(containers)).To(Equal(2))
				Expect(containers[0].ImagePullPolicy).To(Equal(k8sv1.PullAlways))
				Expect(containers[1].ImagePullPolicy).To(Equal(k8sv1.PullAlways))
			})
		})
	})
})

func appendContainerDisk(vmi *v1.VirtualMachineInstance, diskName string) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image:           "someimage:v1.2.3.4",
				ImagePullPolicy: k8sv1.PullAlways,
			},
		},
	})
}
