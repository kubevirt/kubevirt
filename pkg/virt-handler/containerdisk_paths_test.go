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

package virthandler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("ContainerDisk path annotation", func() {

	newVMI := func(volumes []v1.Volume) *v1.VirtualMachineInstance {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testvmi",
				Namespace: "default",
			},
		}
		vmi.Spec.Volumes = volumes
		return vmi
	}

	newDomain := func(naming string, disks []api.Disk) *api.Domain {
		domain := &api.Domain{}
		domain.Spec.Metadata.KubeVirt.ContainerDiskNaming = naming
		domain.Spec.Devices.Disks = disks
		return domain
	}

	newDisk := func(aliasName, filePath string) api.Disk {
		return api.Disk{
			Alias:  api.NewUserDefinedAlias(aliasName),
			Source: api.DiskSource{File: filePath},
		}
	}

	Context("isLegacyContainerDiskNaming", func() {
		DescribeTable("correctly identifies naming style",
			func(naming string, expected bool) {
				domain := newDomain(naming, nil)
				Expect(isLegacyContainerDiskNaming(domain)).To(Equal(expected))
			},
			Entry("returns true when ContainerDiskNaming is empty", "", true),
			Entry("returns true when ContainerDiskNaming is not v2", "v1", true),
			Entry("returns false when ContainerDiskNaming is v2", "v2", false),
		)
	})

	Context("buildContainerDiskPathMap", func() {
		type testCase struct {
			volumes   []v1.Volume
			disks     []api.Disk
			naming    string
			expectMap map[string]string
		}

		DescribeTable("builds path map correctly",
			func(tc testCase) {
				vmi := newVMI(tc.volumes)
				domain := newDomain(tc.naming, tc.disks)
				result := buildContainerDiskPathMap(vmi, domain)
				Expect(result).To(Equal(tc.expectMap))
			},
			Entry("empty map when no container disk volumes",
				testCase{
					volumes: []v1.Volume{
						{Name: "pvcvol", VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
						}},
					},
					disks:     []api.Disk{newDisk("ua-pvcvol", "/var/run/kubevirt-private/vmi-disks/pvcvol/disk.img")},
					naming:    "",
					expectMap: map[string]string{},
				}),
			Entry("maps volume name to legacy index-based path",
				testCase{
					volumes: []v1.Volume{
						{Name: "mydisk", VolumeSource: v1.VolumeSource{
							ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"},
						}},
					},
					disks:     []api.Disk{newDisk("ua-mydisk", "/var/run/kubevirt/container-disks/disk_2.img")},
					naming:    "",
					expectMap: map[string]string{"mydisk": "/var/run/kubevirt/container-disks/disk_2.img"},
				}),
			Entry("maps multiple container disk volumes correctly",
				testCase{
					volumes: []v1.Volume{
						{Name: "disk0", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "img0"}}},
						{Name: "disk1", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "img1"}}},
					},
					disks: []api.Disk{
						newDisk("ua-disk0", "/var/run/kubevirt/container-disks/disk_0.img"),
						newDisk("ua-disk1", "/var/run/kubevirt/container-disks/disk_1.img"),
					},
					naming: "",
					expectMap: map[string]string{
						"disk0": "/var/run/kubevirt/container-disks/disk_0.img",
						"disk1": "/var/run/kubevirt/container-disks/disk_1.img",
					},
				}),
			Entry("skips disks with no alias",
				testCase{
					volumes: []v1.Volume{
						{Name: "mydisk", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"}}},
					},
					disks:     []api.Disk{{Source: api.DiskSource{File: "/var/run/kubevirt/container-disks/disk_0.img"}}},
					naming:    "",
					expectMap: map[string]string{},
				}),
			Entry("skips disks with empty file path",
				testCase{
					volumes: []v1.Volume{
						{Name: "mydisk", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"}}},
					},
					disks:     []api.Disk{newDisk("ua-mydisk", "")},
					naming:    "",
					expectMap: map[string]string{},
				}),
			Entry("skips non-index-based filenames",
				testCase{
					volumes: []v1.Volume{
						{Name: "mydisk", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"}}},
					},
					disks:     []api.Disk{newDisk("ua-mydisk", "/var/run/kubevirt/container-disks/somethingelse.img")},
					naming:    "",
					expectMap: map[string]string{},
				}),
		)
	})

	Context("syncContainerDiskPathAnnotation", func() {
		It("returns nil when domain is nil", func() {
			c := &VirtualMachineController{}
			vmi := newVMI(nil)
			Expect(c.syncContainerDiskPathAnnotation(vmi, nil)).To(Succeed())
		})

		It("returns nil when domain is v2 style", func() {
			c := &VirtualMachineController{}
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				}},
			})
			domain := newDomain("v2", nil)
			Expect(c.syncContainerDiskPathAnnotation(vmi, domain)).To(Succeed())
		})

		It("returns nil when annotation already exists", func() {
			c := &VirtualMachineController{}
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				}},
			})
			vmi.Annotations = map[string]string{
				v1.ContainerDiskPathsAnnotation: `{"mydisk":"/path/disk_0.img"}`,
			}
			domain := newDomain("", nil)
			Expect(c.syncContainerDiskPathAnnotation(vmi, domain)).To(Succeed())
		})

		It("returns nil when VMI has no container disks", func() {
			c := &VirtualMachineController{}
			vmi := newVMI([]v1.Volume{
				{Name: "pvcvol", VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
				}},
			})
			domain := newDomain("", nil)
			Expect(c.syncContainerDiskPathAnnotation(vmi, domain)).To(Succeed())
		})

		It("returns nil when path map is empty", func() {
			c := &VirtualMachineController{}
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{},
				}},
			})
			// domain has no disks so pathMap will be empty
			domain := newDomain("", nil)
			Expect(c.syncContainerDiskPathAnnotation(vmi, domain)).To(Succeed())
		})
	})

	Context("isLegacyDiskFilename", func() {
		DescribeTable("correctly identifies legacy vs v2 filenames",
			func(filename string, expected bool) {
				Expect(isLegacyDiskFilename(filename)).To(Equal(expected))
			},
			Entry("legacy index 0", "disk_0.img", true),
			Entry("legacy index 2", "disk_2.img", true),
			Entry("legacy index 10", "disk_10.img", true),
			Entry("v2 volume name", "disk_myvolume.img", false),
			Entry("v2 volume name with numbers", "disk_vol1.img", false),
			Entry("unrelated file", "somethingelse.img", false),
			Entry("empty middle", "disk_.img", false),
		)
	})
})
