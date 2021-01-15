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
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

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
		vmi.UID = "1234"
		appendContainerDisk(vmi, "r0")

		expectedVolumeMountDir := fmt.Sprintf("%s/%s", tmpDir, string(vmi.UID))

		// create a fake disk file
		volumeMountDir := GetVolumeMountDirOnGuest(vmi)
		err = os.MkdirAll(volumeMountDir, 0750)
		Expect(err).ToNot(HaveOccurred())
		Expect(expectedVolumeMountDir).To(Equal(volumeMountDir))

		filePath := filepath.Join(volumeMountDir + "/disk_0.img")
		_, err := os.Create(filePath)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		os.MkdirAll(tmpDir, 0755)
		err := SetLocalDirectory(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		setLocalDataOwner(owner.Username)
		err = setPodsDirectory(tmpDir)
		Expect(err).ToNot(HaveOccurred())
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
			})

			It("by verifying host directory locations", func() {
				vmi := v1.NewMinimalVMI("fake-vmi")
				vmi.UID = "6789"
				vmi.Status.ActivePods = map[types.UID]string{
					"1234": "myhost",
				}

				// should not be found if dir doesn't exist
				path, found, err := GetVolumeMountDirOnHost(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
				Expect(path).To(Equal(""))

				// should be found if dir does exist
				expectedPath := fmt.Sprintf("%s/1234/volumes/kubernetes.io~empty-dir/container-disks", tmpDir)
				os.MkdirAll(expectedPath, 0755)
				path, found, err = GetVolumeMountDirOnHost(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(path).To(Equal(expectedPath))

				// should be able to generate legacy socket path dir
				legacySocket := GetLegacyVolumeMountDirOnHost(vmi)
				Expect(legacySocket).To(Equal(filepath.Join(tmpDir, "6789")))

				// should return error if disk target doesn't exist
				targetPath, err := GetDiskTargetPathFromHostView(vmi, 1)
				expectedPath = fmt.Sprintf("%s/1234/volumes/kubernetes.io~empty-dir/container-disks/disk_1.img", tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(targetPath).To(Equal(expectedPath))

			})

			It("by verifying error occurs if multiple host directory locations exist somehow", func() {
				vmi := v1.NewMinimalVMI("fake-vmi")
				vmi.UID = "6789"
				vmi.Status.ActivePods = map[types.UID]string{
					"1234": "myhost",
					"5678": "myhost",
				}

				// should not return error if only one dir exists
				expectedPath := fmt.Sprintf("%s/1234/volumes/kubernetes.io~empty-dir/container-disks", tmpDir)
				os.MkdirAll(expectedPath, 0755)
				path, found, err := GetVolumeMountDirOnHost(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(path).To(Equal(expectedPath))

				// return error if two dirs exist
				secondPath := fmt.Sprintf("%s/5678/volumes/kubernetes.io~empty-dir/container-disks", tmpDir)
				os.MkdirAll(secondPath, 0755)
				path, found, err = GetVolumeMountDirOnHost(vmi)
				Expect(err).To(HaveOccurred())
				Expect(found).To(BeFalse())
				Expect(path).To(Equal(""))
			})

			It("by verifying launcher directory locations", func() {
				vmi := v1.NewMinimalVMI("fake-vmi")
				vmi.UID = "6789"

				// This should fail if no file exists
				path, err := GetDiskTargetPartFromLauncherView(1)
				Expect(err).To(HaveOccurred())
				Expect(path).To(Equal(""))

				expectedPath := fmt.Sprintf("%s/disk_1.img", tmpDir)
				_, err = os.Create(expectedPath)
				Expect(err).ToNot(HaveOccurred())

				// this should pass once file exists
				path, err = GetDiskTargetPartFromLauncherView(1)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expectedPath))
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
				containers := GenerateContainers(vmi, "libvirt-runtime", "bin-volume")
				Expect(err).ToNot(HaveOccurred())

				Expect(len(containers)).To(Equal(2))
				Expect(containers[0].ImagePullPolicy).To(Equal(k8sv1.PullAlways))
				Expect(containers[1].ImagePullPolicy).To(Equal(k8sv1.PullAlways))
			})

			Context("which checks socket paths", func() {

				var vmi *v1.VirtualMachineInstance
				var tmpDir string
				BeforeEach(func() {

					tmpDir, err = ioutil.TempDir("", "something")
					Expect(err).ToNot(HaveOccurred())
					err := os.MkdirAll(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks", tmpDir, "poduid"), 0777)
					Expect(err).ToNot(HaveOccurred())
					f, err := os.Create(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_0.sock", tmpDir, "poduid"))
					Expect(err).ToNot(HaveOccurred())
					f.Close()
					f, err = os.Create(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_1.sock", tmpDir, "poduid"))
					Expect(err).ToNot(HaveOccurred())
					f.Close()
					vmi = v1.NewMinimalVMI("fake-vmi")
					vmi.Status.ActivePods = map[types.UID]string{"poduid": ""}
					appendContainerDisk(vmi, "r0")
					appendContainerDisk(vmi, "r1")
					appendContainerDisk(vmi, "r2")
				})

				AfterEach(func() {
					os.RemoveAll(tmpDir)
				})

				It("should fail if the base directory only exists", func() {
					_, err = NewSocketPathGetter(tmpDir)(vmi, 2)
					Expect(err).To(HaveOccurred())
				})

				It("shoud succeed if the the socket is there", func() {
					path1, err := NewSocketPathGetter(tmpDir)(vmi, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(path1).To(Equal(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_0.sock", tmpDir, "poduid")))
					path2, err := NewSocketPathGetter(tmpDir)(vmi, 1)
					Expect(err).ToNot(HaveOccurred())
					Expect(path2).To(Equal(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_1.sock", tmpDir, "poduid")))
				})
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
