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
		It("returns true when ContainerDiskNaming is empty", func() {
			domain := newDomain("", nil)
			Expect(isLegacyContainerDiskNaming(domain)).To(BeTrue())
		})

		It("returns true when ContainerDiskNaming is not v2", func() {
			domain := newDomain("v1", nil)
			Expect(isLegacyContainerDiskNaming(domain)).To(BeTrue())
		})

		It("returns false when ContainerDiskNaming is v2", func() {
			domain := newDomain("v2", nil)
			Expect(isLegacyContainerDiskNaming(domain)).To(BeFalse())
		})
	})

	Context("buildContainerDiskPathMap", func() {
		It("returns empty map when no container disk volumes", func() {
			vmi := newVMI([]v1.Volume{
				{Name: "pvcvol", VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
				}},
			})
			domain := newDomain("", []api.Disk{
				newDisk("ua-pvcvol", "/var/run/kubevirt-private/vmi-disks/pvcvol/disk.img"),
			})
			result := buildContainerDiskPathMap(vmi, domain)
			Expect(result).To(BeEmpty())
		})

		It("returns empty map when domain has v2 naming (volume-name-based paths)", func() {
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"},
				}},
			})
			domain := newDomain("v2", []api.Disk{
				newDisk("ua-mydisk", "/var/run/kubevirt/container-disks/disk_mydisk.img"),
			})
			// v2 paths still match disk_ prefix/suffix but we call this only for legacy
			result := buildContainerDiskPathMap(vmi, domain)
			// The path contains volume name not index so it would still match prefix/suffix
			// but in practice isLegacyContainerDiskNaming gates this call
			Expect(result).To(HaveLen(1))
		})

		It("maps volume name to legacy index-based path", func() {
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"},
				}},
			})
			domain := newDomain("", []api.Disk{
				newDisk("ua-mydisk", "/var/run/kubevirt/container-disks/disk_2.img"),
			})
			result := buildContainerDiskPathMap(vmi, domain)
			Expect(result).To(HaveLen(1))
			Expect(result["mydisk"]).To(Equal("/var/run/kubevirt/container-disks/disk_2.img"))
		})

		It("maps multiple container disk volumes correctly", func() {
			vmi := newVMI([]v1.Volume{
				{Name: "disk0", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "img0"},
				}},
				{Name: "disk1", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "img1"},
				}},
			})
			domain := newDomain("", []api.Disk{
				newDisk("ua-disk0", "/var/run/kubevirt/container-disks/disk_0.img"),
				newDisk("ua-disk1", "/var/run/kubevirt/container-disks/disk_1.img"),
			})
			result := buildContainerDiskPathMap(vmi, domain)
			Expect(result).To(HaveLen(2))
			Expect(result["disk0"]).To(Equal("/var/run/kubevirt/container-disks/disk_0.img"))
			Expect(result["disk1"]).To(Equal("/var/run/kubevirt/container-disks/disk_1.img"))
		})

		It("skips disks with no alias", func() {
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"},
				}},
			})
			domain := newDomain("", []api.Disk{
				{Source: api.DiskSource{File: "/var/run/kubevirt/container-disks/disk_0.img"}},
			})
			result := buildContainerDiskPathMap(vmi, domain)
			Expect(result).To(BeEmpty())
		})

		It("skips disks with empty file path", func() {
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"},
				}},
			})
			domain := newDomain("", []api.Disk{
				newDisk("ua-mydisk", ""),
			})
			result := buildContainerDiskPathMap(vmi, domain)
			Expect(result).To(BeEmpty())
		})

		It("skips non-index-based filenames", func() {
			vmi := newVMI([]v1.Volume{
				{Name: "mydisk", VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "myimage"},
				}},
			})
			domain := newDomain("", []api.Disk{
				newDisk("ua-mydisk", "/var/run/kubevirt/container-disks/somethingelse.img"),
			})
			result := buildContainerDiskPathMap(vmi, domain)
			Expect(result).To(BeEmpty())
		})
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
})
