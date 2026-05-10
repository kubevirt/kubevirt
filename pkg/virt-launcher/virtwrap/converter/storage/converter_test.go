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

package storage_test

import (
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

var _ = Describe("ConvertDisks", func() {
	Context("with CBT volumes", func() {
		var (
			vmi *v1.VirtualMachineInstance
			c   *convertertypes.ConverterContext
		)

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}

			c = &convertertypes.ConverterContext{
				Architecture:   archconverter.NewConverter(runtime.GOARCH),
				VirtualMachine: vmi,
				AllowEmulation: true,
			}
		})

		DescribeTable("should create domain disk with datastore for filesystem volumes with CBT enabled",
			func(volumeName string, createVolumeSource func(string) v1.VolumeSource) {
				cbtPath := "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"

				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					Name: volumeName,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				}}
				vmi.Spec.Volumes = []v1.Volume{{
					Name:         volumeName,
					VolumeSource: createVolumeSource(volumeName),
				}}

				c.ApplyCBT = map[string]string{volumeName: cbtPath}

				dom := &api.Domain{}
				Expect(storage.ConvertDisks(vmi, dom, c)).To(Succeed())

				Expect(dom.Spec.Devices.Disks).To(HaveLen(1))
				disk := dom.Spec.Devices.Disks[0]

				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Source.File).To(Equal(cbtPath))
				Expect(disk.Driver.Type).To(Equal("qcow2"))
				Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))
				Expect(disk.Driver.Discard).To(Equal("unmap"))

				Expect(disk.Source.DataStore).ToNot(BeNil())
				Expect(disk.Source.DataStore.Type).To(Equal("file"))
				Expect(disk.Source.DataStore.Format).ToNot(BeNil())
				Expect(disk.Source.DataStore.Format.Type).To(Equal("raw"))
				Expect(disk.Source.DataStore.Source).ToNot(BeNil())
				Expect(disk.Source.DataStore.Source.File).ToNot(BeEmpty())
			},
			Entry("PVC", "test-pvc",
				func(name string) v1.VolumeSource {
					return v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: name,
							},
						},
					}
				},
			),
			Entry("DataVolume", "test-dv",
				func(name string) v1.VolumeSource {
					return v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: name,
						},
					}
				},
			),
			Entry("HostDisk", "test-hostdisk",
				func(name string) v1.VolumeSource {
					return v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path: "/var/run/kubevirt-private/vmi-disks/" + name + "/disk.img",
							Type: v1.HostDiskExistsOrCreate,
						},
					}
				},
			),
		)

		DescribeTable("should create domain disk with datastore for block volumes with CBT enabled",
			func(volumeName string, createVolumeSource func(string) v1.VolumeSource, setupContext func(*convertertypes.ConverterContext, string)) {
				cbtPath := "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"

				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					Name: volumeName,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				}}
				vmi.Spec.Volumes = []v1.Volume{{
					Name:         volumeName,
					VolumeSource: createVolumeSource(volumeName),
				}}

				c.ApplyCBT = map[string]string{volumeName: cbtPath}
				setupContext(c, volumeName)

				dom := &api.Domain{}
				Expect(storage.ConvertDisks(vmi, dom, c)).To(Succeed())

				Expect(dom.Spec.Devices.Disks).To(HaveLen(1))
				disk := dom.Spec.Devices.Disks[0]

				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Source.File).To(Equal(cbtPath))
				Expect(disk.Source.Name).To(Equal(volumeName))
				Expect(disk.Driver.Type).To(Equal("qcow2"))
				Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))
				Expect(disk.Driver.Discard).To(Equal("unmap"))

				Expect(disk.Source.DataStore).ToNot(BeNil())
				Expect(disk.Source.DataStore.Type).To(Equal("block"))
				Expect(disk.Source.DataStore.Format).ToNot(BeNil())
				Expect(disk.Source.DataStore.Format.Type).To(Equal("raw"))
				Expect(disk.Source.DataStore.Source).ToNot(BeNil())
				Expect(disk.Source.DataStore.Source.Dev).To(Equal(storage.GetBlockDeviceVolumePath(volumeName)))
			},
			Entry("PVC", "test-block-pvc",
				func(name string) v1.VolumeSource {
					return v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: name,
							},
						},
					}
				},
				func(c *convertertypes.ConverterContext, name string) {
					c.IsBlockPVC = map[string]bool{name: true}
				},
			),
			Entry("DataVolume", "test-block-dv",
				func(name string) v1.VolumeSource {
					return v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: name,
						},
					}
				},
				func(c *convertertypes.ConverterContext, name string) {
					c.IsBlockDV = map[string]bool{name: true}
				},
			),
		)
	})

	Context("with hotplug CBT volumes", func() {
		var (
			vmi *v1.VirtualMachineInstance
			c   *convertertypes.ConverterContext
		)

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}

			c = &convertertypes.ConverterContext{
				Architecture:   archconverter.NewConverter(runtime.GOARCH),
				VirtualMachine: vmi,
				AllowEmulation: true,
				IsBlockPVC: map[string]bool{
					"test-block-pvc": true,
				},
				IsBlockDV: map[string]bool{
					"test-block-dv": true,
				},
				VolumesDiscardIgnore: []string{
					"test-discard-ignore",
				},
			}
		})

		DescribeTable("should create domain disk with datastore for hotplug volumes with CBT enabled",
			func(volumeName string, volSource v1.VolumeSource, isBlock bool) {
				cbtPath := "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"

				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{{
					Name: volumeName,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				}}
				vmi.Spec.Volumes = []v1.Volume{{
					Name:         volumeName,
					VolumeSource: volSource,
				}}

				c.ApplyCBT = map[string]string{volumeName: cbtPath}
				c.HotplugVolumes = map[string]v1.VolumeStatus{
					volumeName: {Name: volumeName, Phase: v1.HotplugVolumeMounted, HotplugVolume: &v1.HotplugVolumeStatus{}},
				}
				if isBlock {
					if volSource.PersistentVolumeClaim != nil {
						c.IsBlockPVC[volumeName] = true
					} else if volSource.DataVolume != nil {
						c.IsBlockDV[volumeName] = true
					}
				}

				dom := &api.Domain{}
				Expect(storage.ConvertDisks(vmi, dom, c)).To(Succeed())

				Expect(dom.Spec.Devices.Disks).To(HaveLen(1))
				disk := dom.Spec.Devices.Disks[0]

				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Source.File).To(Equal(cbtPath))
				Expect(disk.Driver.Type).To(Equal("qcow2"))
				Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))
				Expect(disk.Driver.Discard).To(Equal("unmap"))

				Expect(disk.Source.DataStore).ToNot(BeNil())
				Expect(disk.Source.DataStore.Format).ToNot(BeNil())
				Expect(disk.Source.DataStore.Format.Type).To(Equal("raw"))
				Expect(disk.Source.DataStore.Source).ToNot(BeNil())
				if isBlock {
					Expect(disk.Source.DataStore.Type).To(Equal("block"))
					Expect(disk.Source.DataStore.Source.Dev).To(Equal(storage.GetHotplugBlockDeviceVolumePath(volumeName)))
				} else {
					Expect(disk.Source.DataStore.Type).To(Equal("file"))
					Expect(disk.Source.DataStore.Source.File).To(Equal(storage.GetHotplugFilesystemVolumePath(volumeName)))
				}
			},
			Entry("filesystem PVC", "test-hotplug-pvc",
				v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "test-hotplug-pvc"},
					Hotpluggable:                      true,
				}}, false),
			Entry("filesystem DataVolume", "test-hotplug-dv",
				v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "test-hotplug-dv", Hotpluggable: true}}, false),
			Entry("block PVC", "test-hotplug-block-pvc",
				v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "test-hotplug-block-pvc"},
					Hotpluggable:                      true,
				}}, true),
			Entry("block DataVolume", "test-hotplug-block-dv",
				v1.VolumeSource{DataVolume: &v1.DataVolumeSource{Name: "test-hotplug-block-dv", Hotpluggable: true}}, true),
		)
	})
})
