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
 * Copyright The KubeVirt Authors.
 *
 */

package virtiofs

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("ContainerPath helpers", func() {

	Context("FindVolumeMountForPath", func() {
		var container *k8sv1.Container

		BeforeEach(func() {
			container = &k8sv1.Container{
				VolumeMounts: []k8sv1.VolumeMount{
					{
						Name:      "root-mount",
						MountPath: "/data",
					},
					{
						Name:      "nested-mount",
						MountPath: "/data/nested",
					},
					{
						Name:      "other-mount",
						MountPath: "/other",
					},
				},
			}
		})

		DescribeTable("should find correct mount and subpath",
			func(path, expectedMount, expectedSubPath string) {
				mount, subPath := FindVolumeMountForPath(container, path)
				if expectedMount == "" {
					Expect(mount).To(BeNil())
				} else {
					Expect(mount).ToNot(BeNil())
					Expect(mount.Name).To(Equal(expectedMount))
					Expect(subPath).To(Equal(expectedSubPath))
				}
			},
			Entry("exact match", "/data", "root-mount", ""),
			Entry("exact nested match", "/data/nested", "nested-mount", ""),
			Entry("subpath under root mount", "/data/subdir/file", "root-mount", "subdir/file"),
			Entry("subpath under nested mount (prefers more specific)", "/data/nested/file", "nested-mount", "file"),
			Entry("no matching mount", "/nonexistent/path", "", ""),
		)
	})

	Context("GetContainerPathVolumesWithFilesystems", func() {
		It("should return nil for nil VMI", func() {
			result := GetContainerPathVolumesWithFilesystems(nil)
			Expect(result).To(BeNil())
		})

		It("should return nil for VMI without filesystems", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{},
					},
				},
			}
			result := GetContainerPathVolumesWithFilesystems(vmi)
			Expect(result).To(BeNil())
		})

		It("should return containerPath volume with matching filesystem", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Filesystems: []v1.Filesystem{
								{
									Name:     "fs1",
									Virtiofs: &v1.FilesystemVirtiofs{},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "fs1",
							VolumeSource: v1.VolumeSource{
								ContainerPath: &v1.ContainerPathVolumeSource{
									Path: "/data",
								},
							},
						},
					},
				},
			}
			result := GetContainerPathVolumesWithFilesystems(vmi)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("fs1"))
		})

		It("should not return containerPath volume without matching filesystem", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Filesystems: []v1.Filesystem{
								{
									Name:     "other-fs",
									Virtiofs: &v1.FilesystemVirtiofs{},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "fs1",
							VolumeSource: v1.VolumeSource{
								ContainerPath: &v1.ContainerPathVolumeSource{
									Path: "/data",
								},
							},
						},
					},
				},
			}
			result := GetContainerPathVolumesWithFilesystems(vmi)
			Expect(result).To(BeEmpty())
		})
	})

	Context("MissingContainerPathContainers", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Filesystems: []v1.Filesystem{
								{
									Name:     "fs1",
									Virtiofs: &v1.FilesystemVirtiofs{},
								},
								{
									Name:     "fs2",
									Virtiofs: &v1.FilesystemVirtiofs{},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "fs1",
							VolumeSource: v1.VolumeSource{
								ContainerPath: &v1.ContainerPathVolumeSource{
									Path: "/data1",
								},
							},
						},
						{
							Name: "fs2",
							VolumeSource: v1.VolumeSource{
								ContainerPath: &v1.ContainerPathVolumeSource{
									Path: "/data2",
								},
							},
						},
					},
				},
			}
		})

		It("should return all containers when none exist", func() {
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{Name: "compute"},
					},
				},
			}
			result := MissingContainerPathContainers(vmi, pod)
			Expect(result).To(HaveLen(2))
			Expect(result).To(ContainElements("virtiofs-fs1", "virtiofs-fs2"))
		})

		It("should return only missing containers", func() {
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{Name: "compute"},
						{Name: "virtiofs-fs1"},
					},
				},
			}
			result := MissingContainerPathContainers(vmi, pod)
			Expect(result).To(HaveLen(1))
			Expect(result).To(ContainElement("virtiofs-fs2"))
		})

		It("should return nil when all containers exist", func() {
			pod := &k8sv1.Pod{
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{Name: "compute"},
						{Name: "virtiofs-fs1"},
						{Name: "virtiofs-fs2"},
					},
				},
			}
			result := MissingContainerPathContainers(vmi, pod)
			Expect(result).To(BeNil())
		})
	})
})
