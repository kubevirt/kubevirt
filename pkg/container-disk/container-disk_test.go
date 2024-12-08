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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
)

var _ = Describe("ContainerDisk", func() {
	var tmpDir string
	const someImage = "someimage:v1.2.3.4"

	VerifyDiskType := func(diskExtension string) {
		vmi := libvmi.New(libvmi.WithContainerDisk("r0", someImage))
		vmi.UID = "1234"

		expectedVolumeMountDir := fmt.Sprintf("%s/%s", tmpDir, string(vmi.UID))

		// create a fake disk file
		volumeMountDir := GetVolumeMountDirOnGuest(vmi)
		err := os.MkdirAll(volumeMountDir, 0750)
		Expect(err).ToNot(HaveOccurred())
		Expect(expectedVolumeMountDir).To(Equal(volumeMountDir))

		filePath := filepath.Join(volumeMountDir, "disk_0.img")
		_, err = os.Create(filePath)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "containerdisktest")
		Expect(err).ToNot(HaveOccurred())
		Expect(os.MkdirAll(tmpDir, 0755)).To(Succeed())
		err = SetLocalDirectory(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		err = setPodsDirectory(tmpDir)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Describe("container-disk", func() {
		Context("verify helper functions", func() {
			DescribeTable("by verifying mapping of ",
				func(diskType string) {
					VerifyDiskType(diskType)
				},
				Entry("qcow2 disk", "qcow2"),
				Entry("raw disk", "raw"),
			)
			It("by verifying error when no disk is present", func() {
				vmi := libvmi.New()

				By("Trying to get the volume mount directory on guest and expecting an error")
				volumeMountDir := GetVolumeMountDirOnGuest(vmi)
				_, err := os.Stat(filepath.Join(volumeMountDir, "disk_0.img"))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(os.IsNotExist, "IsNotExist"))
			})

			It("by verifying host directory locations", func() {
				vmi := libvmi.New(
					libvmistatus.WithStatus(
						libvmistatus.New(libvmistatus.WithActivePod("1234", "myhost")),
					))

				vmi.UID = "6789"

				// should not be found if dir doesn't exist
				By("Checking if the directory does not exist")
				path, err := GetVolumeMountDirOnHost(vmi)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())

				// should be found if dir does exist
				expectedPath := fmt.Sprintf("%s/1234/volumes/kubernetes.io~empty-dir/container-disks", tmpDir)
				Expect(os.MkdirAll(expectedPath, 0755)).To(Succeed())
				path, err = GetVolumeMountDirOnHost(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(expectedPath))

				// should be able to generate legacy socket path dir
				legacySocket := GetLegacyVolumeMountDirOnHost(vmi)
				Expect(legacySocket).To(Equal(filepath.Join(tmpDir, "6789")))

				// should return error if disk target dir doesn't exist
				targetPath, err := GetDiskTargetDirFromHostView(vmi)
				expectedPath = fmt.Sprintf("%s/1234/volumes/kubernetes.io~empty-dir/container-disks", tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(targetPath.Raw())).To(Equal(expectedPath))
			})

			It("by verifying error occurs if multiple host directory locations exist somehow", func() {
				vmi := libvmi.New(libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithActivePod("1234", "myhost"),
						libvmistatus.WithActivePod("5678", "myhost"),
					)))

				vmi.UID = "6789"

				// should not return error if only one dir exists
				expectedPath := fmt.Sprintf("%s/1234/volumes/kubernetes.io~empty-dir/container-disks", tmpDir)
				Expect(os.MkdirAll(expectedPath, 0755)).To(Succeed())
				path, err := GetVolumeMountDirOnHost(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(expectedPath))

				// return error if two dirs exist
				secondPath := fmt.Sprintf("%s/5678/volumes/kubernetes.io~empty-dir/container-disks", tmpDir)
				Expect(os.MkdirAll(secondPath, 0755)).To(Succeed())
				path, err = GetVolumeMountDirOnHost(vmi)
				Expect(err).To(HaveOccurred())
			})

			It("by verifying launcher directory locations", func() {
				vmi := libvmi.New()
				vmi.UID = "6789"

				path, err := GetDiskTargetPartFromLauncherView(1)
				Expect(err).To(HaveOccurred())
				Expect(path).To(Equal(""))

				expectedPath := fmt.Sprintf("%s/disk_1.img", tmpDir)
				_, err = os.Create(expectedPath)
				Expect(err).ToNot(HaveOccurred())

				path, err = GetDiskTargetPartFromLauncherView(1)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(expectedPath))
			})

			DescribeTable("by verifying that resources are set if the VMI wants the guaranteed QOS class", func(req, lim, expectedReq, expectedLimit k8sv1.ResourceList) {
				clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					SupportContainerResources: []v1.SupportContainerResources{
						{
							Type: v1.ContainerDisk,
							Resources: v1.ResourceRequirementsWithoutClaims{
								Requests: req,
								Limits:   lim,
							},
						},
					},
				})

				vmi := libvmi.New(
					libvmi.WithContainerDisk("r0", someImage),
					libvmi.WithResourceCPU("1"),
					libvmi.WithResourceMemory("64M"),
					libvmi.WithLimitCPU("1"),
					libvmi.WithLimitMemory("64M"),
				)

				containers := GenerateContainers(vmi, clusterConfig, nil, "libvirt-runtime", "/var/run/libvirt")

				Expect(containers[0].Resources.Requests).To(ContainElements(*expectedReq.Cpu(), *expectedReq.Memory(), *expectedReq.StorageEphemeral()))
				Expect(containers[0].Resources.Limits).To(BeEquivalentTo(expectedLimit))
			},
				Entry("defaults not overridden", k8sv1.ResourceList{}, k8sv1.ResourceList{}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:              resource.MustParse("10m"),
					k8sv1.ResourceMemory:           resource.MustParse("40M"),
					k8sv1.ResourceEphemeralStorage: resource.MustParse("50M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("10m"),
					k8sv1.ResourceMemory: resource.MustParse("40M"),
				}),
				Entry("defaults overridden", k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("1m"),
					k8sv1.ResourceMemory: resource.MustParse("25M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("100m"),
					k8sv1.ResourceMemory: resource.MustParse("400M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:              resource.MustParse("100m"),
					k8sv1.ResourceMemory:           resource.MustParse("400M"),
					k8sv1.ResourceEphemeralStorage: resource.MustParse("50M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("100m"),
					k8sv1.ResourceMemory: resource.MustParse("400M"),
				}),
			)

			DescribeTable("by verifying that resources are set from config", func(req, lim, expectedReq, expectedLimit k8sv1.ResourceList) {
				clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					SupportContainerResources: []v1.SupportContainerResources{
						{
							Type: v1.ContainerDisk,
							Resources: v1.ResourceRequirementsWithoutClaims{
								Requests: req,
								Limits:   lim,
							},
						},
					},
				})

				vmi := libvmi.New(
					libvmi.WithContainerDisk("r0", someImage),
				)

				containers := GenerateContainers(vmi, clusterConfig, nil, "libvirt-runtime", "/var/run/libvirt")

				Expect(containers[0].Resources.Requests).To(ContainElements(*expectedReq.Cpu(), *expectedReq.Memory(), *expectedReq.StorageEphemeral()))
				Expect(containers[0].Resources.Limits).To(BeEquivalentTo(expectedLimit))
			},
				Entry("defaults not overridden", k8sv1.ResourceList{}, k8sv1.ResourceList{}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:              resource.MustParse("1m"),
					k8sv1.ResourceMemory:           resource.MustParse("1M"),
					k8sv1.ResourceEphemeralStorage: resource.MustParse("50M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("10m"),
					k8sv1.ResourceMemory: resource.MustParse("40M"),
				}),
				Entry("defaults overridden", k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("2m"),
					k8sv1.ResourceMemory: resource.MustParse("25M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("110m"),
					k8sv1.ResourceMemory: resource.MustParse("400M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:              resource.MustParse("2m"),
					k8sv1.ResourceMemory:           resource.MustParse("25M"),
					k8sv1.ResourceEphemeralStorage: resource.MustParse("50M"),
				}, k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("110m"),
					k8sv1.ResourceMemory: resource.MustParse("400M"),
				}),
			)

			It("by verifying that ephemeral storage request is set to every container", func() {
				clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					SupportContainerResources: []v1.SupportContainerResources{},
				})

				vmi := libvmi.New(
					libvmi.WithContainerDisk("r0", someImage),
				)

				containers := GenerateContainers(vmi, clusterConfig, nil, "libvirt-runtime", "/var/run/libvirt")

				expectedEphemeralStorageRequest := resource.MustParse(ephemeralStorageOverheadSize)

				containerResourceSpecs := make([]k8sv1.ResourceList, 0)
				for _, container := range containers {
					containerResourceSpecs = append(containerResourceSpecs, container.Resources.Requests)
				}

				for _, containerResourceSpec := range containerResourceSpecs {
					Expect(containerResourceSpec).To(HaveKeyWithValue(k8sv1.ResourceEphemeralStorage, expectedEphemeralStorageRequest))
				}
			})

			It("by verifying container generation", func() {
				clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					SupportContainerResources: []v1.SupportContainerResources{},
				})

				By("Creating a new VMI with multiple container disks")
				vmi := libvmi.New(
					libvmi.WithContainerDiskAndPullPolicy("r1", someImage, "Always"),
					libvmi.WithContainerDiskAndPullPolicy("r0", someImage, "Always"),
				)

				By("Generating containers with the given VMI and cluster configuration")
				containers := GenerateContainers(vmi, clusterConfig, nil, "libvirt-runtime", "bin-volume")
				Expect(containers).To(HaveLen(2))
				Expect(containers[0].ImagePullPolicy).To(Equal(k8sv1.PullAlways))
				Expect(containers[1].ImagePullPolicy).To(Equal(k8sv1.PullAlways))
			})
			Context("which checks socket paths", func() {

				var vmi *v1.VirtualMachineInstance
				var tmpDir string

				BeforeEach(func() {
					var err error
					tmpDir, err = os.MkdirTemp("", "something")
					Expect(err).ToNot(HaveOccurred())
					err = os.MkdirAll(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks", tmpDir, "poduid"), 0777)
					Expect(err).ToNot(HaveOccurred())
					f, err := os.Create(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_0.sock", tmpDir, "poduid"))
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())
					f, err = os.Create(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_1.sock", tmpDir, "poduid"))
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())

					By("Creating a new VMI with multiple container disks")
					vmi = libvmi.New(
						libvmi.WithContainerDisk("r0", someImage),
						libvmi.WithContainerDisk("r1", someImage),
						libvmi.WithContainerDisk("r2", someImage),
						libvmistatus.WithStatus(
							libvmistatus.New(
								libvmistatus.WithActivePod("poduid", ""),
							)),
					)
				})

				AfterEach(func() {
					Expect(os.RemoveAll(tmpDir)).To(Succeed())
				})

				It("should fail if the base directory only exists", func() {
					_, err := NewSocketPathGetter(tmpDir)(vmi, 2)
					Expect(err).To(HaveOccurred())
				})

				It("should succeed if the socket is there", func() {
					path1, err := NewSocketPathGetter(tmpDir)(vmi, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(path1).To(Equal(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_0.sock", tmpDir, "poduid")))
					path2, err := NewSocketPathGetter(tmpDir)(vmi, 1)
					Expect(err).ToNot(HaveOccurred())
					Expect(path2).To(Equal(fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks/disk_1.sock", tmpDir, "poduid")))
				})
			})
		})

		Context("should use the right containerID", func() {
			It("for a new migration pod with two containerDisks", func() {
				clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					SupportContainerResources: []v1.SupportContainerResources{},
				})

				By("Creating a new VMI with container and non-container disks")
				vmi := libvmi.New(
					libvmi.WithContainerDisk("disk1", someImage),
					libvmi.WithDataVolume("disk3", "some-data-volume"),
					libvmi.WithContainerDisk("disk2", someImage),
				)

				pod := createMigrationSourcePod(vmi)

				imageIDs := ExtractImageIDsFromSourcePod(vmi, pod)
				Expect(imageIDs).To(HaveKeyWithValue("disk1", "someimage@sha256:0"))
				Expect(imageIDs).To(HaveKeyWithValue("disk2", "someimage@sha256:1"))
				Expect(imageIDs).To(HaveLen(2))

				newContainers := GenerateContainers(vmi, clusterConfig, imageIDs, "a-name", "something")
				Expect(newContainers[0].Image).To(Equal("someimage@sha256:0"))
				Expect(newContainers[1].Image).To(Equal("someimage@sha256:1"))
			})
			It("for a new migration pod with a containerDisk and a kernel image", func() {
				clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					SupportContainerResources: []v1.SupportContainerResources{},
				})

				By("Creating a new VMI with a container disk and a kernel boot image")
				vmi := libvmi.New(
					libvmi.WithKernelBootContainer(someImage),
					libvmi.WithContainerDisk("disk1", someImage),
					libvmi.WithDataVolume("disk3", "some-data-volume"),
				)

				pod := createMigrationSourcePod(vmi)

				By("Extracting image IDs from the source pod")
				imageIDs := ExtractImageIDsFromSourcePod(vmi, pod)
				Expect(imageIDs).To(HaveKeyWithValue("disk1", "someimage@sha256:0"))
				Expect(imageIDs).To(HaveKeyWithValue("kernel-boot-volume", "someimage@sha256:bootcontainer"))
				Expect(imageIDs).To(HaveLen(2))

				newContainers := GenerateContainers(vmi, clusterConfig, imageIDs, "a-name", "something")
				newBootContainer := GenerateKernelBootContainer(vmi, clusterConfig, imageIDs, "a-name", "something")
				newContainers = append(newContainers, *newBootContainer)
				Expect(newContainers[0].Image).To(Equal("someimage@sha256:0"))
				Expect(newContainers[1].Image).To(Equal("someimage@sha256:bootcontainer"))
			})

			It("should return the source image tag if it can't detect a reproducible imageID", func() {
				By("Creating a new VMI with a container disk")
				vmi := libvmi.New(
					libvmi.WithContainerDisk("disk1", someImage),
				)

				By("Creating a migration source pod")
				pod := createMigrationSourcePod(vmi)
				pod.Status.ContainerStatuses[0].ImageID = "rubbish"

				imageIDs := ExtractImageIDsFromSourcePod(vmi, pod)

				Expect(imageIDs["disk1"]).To(Equal(vmi.Spec.Volumes[0].ContainerDisk.Image))
			})

			DescribeTable("It should detect the image ID from", func(imageID string) {
				expected := "myregistry.io/myimage@sha256:4gjffGJlg4"
				res := toPullableImageReference("myregistry.io/myimage", imageID)
				Expect(res).To(Equal(expected))
				res = toPullableImageReference("myregistry.io/myimage:1234", imageID)
				Expect(res).To(Equal(expected))
				res = toPullableImageReference("myregistry.io/myimage:latest", imageID)
				Expect(res).To(Equal(expected))
			},
				Entry("docker", "docker://sha256:4gjffGJlg4"),
				Entry("dontainerd", "sha256:4gjffGJlg4"),
				Entry("cri-o", "myregistry/myimage@sha256:4gjffGJlg4"),
			)

			DescribeTable("It should detect the base image from", func(given, expected string) {
				res := toPullableImageReference(given, "docker://sha256:4gjffGJlg4")
				Expect(strings.Split(res, "@sha256:")[0]).To(Equal(expected))
			},
				Entry("image with registry and no tags or shasum", "myregistry.io/myimage", "myregistry.io/myimage"),
				Entry("image with registry and tag", "myregistry.io/myimage:latest", "myregistry.io/myimage"),
				Entry("image with registry and shasum", "myregistry.io/myimage@sha256:123534", "myregistry.io/myimage"),
				Entry("image with registry and no tags or shasum and custom port", "myregistry.io:5000/myimage", "myregistry.io:5000/myimage"),
				Entry("image with registry and tag and custom port", "myregistry.io:5000/myimage:latest", "myregistry.io:5000/myimage"),
				Entry("image with registry and shasum and custom port", "myregistry.io:5000/myimage@sha256:123534", "myregistry.io:5000/myimage"),
				Entry("image with registry and shasum and custom port and group", "myregistry.io:5000/mygroup/myimage@sha256:123534", "myregistry.io:5000/mygroup/myimage"),
			)
		})

		Context("when generating the container", func() {
			DescribeTable("when generating the container", func(testFunc func(*k8sv1.Container)) {
				clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					SupportContainerResources: []v1.SupportContainerResources{},
				})
				By("Creating a new VMI with a container disk")
				vmi := libvmi.New(
					libvmi.WithContainerDisk("disk1", someImage),
				)

				pod := createMigrationSourcePod(vmi)
				imageIDs := ExtractImageIDsFromSourcePod(vmi, pod)

				newContainers := GenerateContainers(vmi, clusterConfig, imageIDs, "a-name", "something")

				testFunc(&newContainers[0])
			},
				Entry("AllowPrivilegeEscalation should be false", func(c *k8sv1.Container) {
					Expect(*c.SecurityContext.AllowPrivilegeEscalation).To(BeFalse())
				}),
				Entry("all capabilities should be dropped", func(c *k8sv1.Container) {
					Expect(c.SecurityContext.Capabilities.Drop).To(Equal([]k8sv1.Capability{"ALL"}))
				}),
			)
		})
	})
})

func createMigrationSourcePod(vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
		SupportContainerResources: []v1.SupportContainerResources{},
	})
	pod := &k8sv1.Pod{Status: k8sv1.PodStatus{}}
	containers := GenerateContainers(vmi, clusterConfig, nil, "a-name", "something")

	for idx, container := range containers {
		status := k8sv1.ContainerStatus{
			Name:    container.Name,
			Image:   container.Image,
			ImageID: fmt.Sprintf("finalimg@sha256:%v", idx),
		}
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, status)
	}
	bootContainer := GenerateKernelBootContainer(vmi, clusterConfig, nil, "a-name", "something")
	if bootContainer != nil {
		status := k8sv1.ContainerStatus{
			Name:    bootContainer.Name,
			Image:   bootContainer.Image,
			ImageID: fmt.Sprintf("finalimg@sha256:%v", "bootcontainer"),
		}
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, status)
	}

	return pod
}
