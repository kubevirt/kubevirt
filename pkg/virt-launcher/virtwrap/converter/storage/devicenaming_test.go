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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
)

var _ = Describe("device naming", func() {
	It("format device name should return correct value", func() {
		res := storage.FormatDeviceName("sd", 0)
		Expect(res).To(Equal("sda"))
		res = storage.FormatDeviceName("sd", 1)
		Expect(res).To(Equal("sdb"))
		// 25 is z 26 starting at 0
		res = storage.FormatDeviceName("sd", 25)
		Expect(res).To(Equal("sdz"))
		res = storage.FormatDeviceName("sd", 26*2-1)
		Expect(res).To(Equal("sdaz"))
		res = storage.FormatDeviceName("sd", 26*26-1)
		Expect(res).To(Equal("sdyz"))
	})

	It("makeDeviceName should generate proper name", func() {
		prefixMap := make(map[string]storage.DeviceNamer)
		res, index := storage.MakeDeviceName("test1", v1.VirtIO, prefixMap)
		Expect(res).To(Equal("vda"))
		Expect(index).To(Equal(0))
		for i := 2; i < 10; i++ {
			storage.MakeDeviceName(fmt.Sprintf("test%d", i), v1.VirtIO, prefixMap)
		}
		delete(prefixMap["vd"].UsedDeviceMap, "vdd")
		By("Verifying next value is vdd")
		res, index = storage.MakeDeviceName("something", v1.VirtIO, prefixMap)
		Expect(index).To(Equal(3))
		Expect(res).To(Equal("vdd"))
		res, index = storage.MakeDeviceName("something_else", v1.VirtIO, prefixMap)
		Expect(res).To(Equal("vdj"))
		Expect(index).To(Equal(9))
		By("verifying existing returns correct value")
		res, index = storage.MakeDeviceName("something", v1.VirtIO, prefixMap)
		Expect(res).To(Equal("vdd"))
		Expect(index).To(Equal(3))
		By("Verifying a new bus returns from start")
		res, index = storage.MakeDeviceName("something", "scsi", prefixMap)
		Expect(res).To(Equal("sda"))
		Expect(index).To(Equal(0))
	})

	It("makeDeviceName should fill target gap when mixed bus types share same prefix", func() {
		By("Setting up prefixMap simulating: SCSI boot=sda, SATA cdrom=sdc, SATA cloudinit=sdd, sdb available")
		prefixMap := map[string]storage.DeviceNamer{
			"sd": {
				ExistingNameMap: map[string]string{
					"boot":      "sda",
					"cdrom":     "sdc",
					"cloudinit": "sdd",
				},
				UsedDeviceMap: map[string]string{
					"sda": "boot",
					"sdc": "cdrom",
					"sdd": "cloudinit",
				},
			},
		}

		By("Adding new SCSI disk - should get sdb (the available gap)")
		res, index := storage.MakeDeviceName("newhotplug", v1.DiskBusSCSI, prefixMap)
		Expect(res).To(Equal("sdb"), "new disk should fill the gap at sdb")
		Expect(index).To(Equal(1))

		By("Adding another SCSI disk - should get sde (next available after sdd)")
		res, index = storage.MakeDeviceName("anotherhotplug", v1.DiskBusSCSI, prefixMap)
		Expect(res).To(Equal("sde"))
		Expect(index).To(Equal(4))
	})

	Describe("NewDeviceNamer", func() {
		DescribeTable("should correctly build prefix map for different disk types",
			func(volumeStatuses []v1.VolumeStatus, disks []v1.Disk, expectedPrefixes []string, expectedMappings map[string]map[string]string) {
				prefixMap := storage.NewDeviceNamer(volumeStatuses, disks)

				Expect(prefixMap).To(HaveLen(len(expectedPrefixes)))
				for _, prefix := range expectedPrefixes {
					Expect(prefixMap).To(HaveKey(prefix))
				}

				for prefix, mappings := range expectedMappings {
					namer := prefixMap[prefix]
					for diskName, target := range mappings {
						Expect(namer.ExistingNameMap).To(HaveKeyWithValue(diskName, target))
						Expect(namer.UsedDeviceMap).To(HaveKeyWithValue(target, diskName))
					}
				}
			},
			Entry("Disk with virtio bus",
				[]v1.VolumeStatus{{Name: "mydisk", Target: "vda"}},
				[]v1.Disk{{Name: "mydisk", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}}},
				[]string{"vd"},
				map[string]map[string]string{"vd": {"mydisk": "vda"}},
			),
			Entry("Disk with SATA bus",
				[]v1.VolumeStatus{{Name: "mydisk", Target: "sda"}},
				[]v1.Disk{{Name: "mydisk", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusSATA}}}},
				[]string{"sd"},
				map[string]map[string]string{"sd": {"mydisk": "sda"}},
			),
			Entry("LUN with SCSI bus",
				[]v1.VolumeStatus{{Name: "mylun", Target: "sda"}},
				[]v1.Disk{{Name: "mylun", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI}}}},
				[]string{"sd"},
				map[string]map[string]string{"sd": {"mylun": "sda"}},
			),
			Entry("CDRom with SATA bus",
				[]v1.VolumeStatus{{Name: "mycdrom", Target: "sda"}},
				[]v1.Disk{{Name: "mycdrom", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{Bus: v1.DiskBusSATA}}}},
				[]string{"sd"},
				map[string]map[string]string{"sd": {"mycdrom": "sda"}},
			),
			Entry("Mixed devices with different buses",
				[]v1.VolumeStatus{
					{Name: "disk1", Target: "vda"},
					{Name: "lun1", Target: "sda"},
					{Name: "cdrom1", Target: "sdb"},
				},
				[]v1.Disk{
					{Name: "disk1", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}},
					{Name: "lun1", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI}}},
					{Name: "cdrom1", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{Bus: v1.DiskBusSATA}}},
				},
				[]string{"vd", "sd"},
				map[string]map[string]string{
					"vd": {"disk1": "vda"},
					"sd": {"lun1": "sda", "cdrom1": "sdb"},
				},
			),
			Entry("Multiple disks sharing same prefix",
				[]v1.VolumeStatus{
					{Name: "disk1", Target: "vda"},
					{Name: "disk2", Target: "vdb"},
				},
				[]v1.Disk{
					{Name: "disk1", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}},
					{Name: "disk2", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}},
				},
				[]string{"vd"},
				map[string]map[string]string{"vd": {"disk1": "vda", "disk2": "vdb"}},
			),
		)

		It("should return empty map for empty inputs", func() {
			prefixMap := storage.NewDeviceNamer([]v1.VolumeStatus{}, []v1.Disk{})
			Expect(prefixMap).To(BeEmpty())
		})

		It("should not populate maps when volume status has no target", func() {
			volumeStatuses := []v1.VolumeStatus{{Name: "mydisk", Target: ""}}
			disks := []v1.Disk{{Name: "mydisk", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}}}

			prefixMap := storage.NewDeviceNamer(volumeStatuses, disks)

			Expect(prefixMap).To(HaveKey("vd"))
			namer := prefixMap["vd"]
			Expect(namer.ExistingNameMap).To(BeEmpty())
			Expect(namer.UsedDeviceMap).To(BeEmpty())
		})

		It("should skip disks with no Disk, LUN, or CDRom defined", func() {
			volumeStatuses := []v1.VolumeStatus{{Name: "unknown", Target: "vda"}}
			disks := []v1.Disk{{Name: "unknown", DiskDevice: v1.DiskDevice{}}}

			prefixMap := storage.NewDeviceNamer(volumeStatuses, disks)

			Expect(prefixMap).To(BeEmpty())
		})

		It("should handle disk without matching volume status", func() {
			volumeStatuses := []v1.VolumeStatus{}
			disks := []v1.Disk{{Name: "mydisk", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}}}}

			prefixMap := storage.NewDeviceNamer(volumeStatuses, disks)

			Expect(prefixMap).To(HaveKey("vd"))
			namer := prefixMap["vd"]
			Expect(namer.ExistingNameMap).To(BeEmpty())
			Expect(namer.UsedDeviceMap).To(BeEmpty())
		})

		It("should allow new disk to fill gap after hotplug detach with mixed disk types", func() {
			By("Simulating post-detach state: boot=sda, detached=sdb(missing), cdrom=sdc, cloudinit=sdd")
			volumeStatuses := []v1.VolumeStatus{
				{Name: "boot", Target: "sda"},
				{Name: "cdrom", Target: "sdc"},
				{Name: "cloudinit", Target: "sdd"},
			}
			disks := []v1.Disk{
				{Name: "boot", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusSCSI}}},
				{Name: "cdrom", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{Bus: v1.DiskBusSATA}}},
				{Name: "cloudinit", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{Bus: v1.DiskBusSATA}}},
			}

			prefixMap := storage.NewDeviceNamer(volumeStatuses, disks)

			By("Verifying the prefix map correctly tracks all used targets")
			Expect(prefixMap).To(HaveKey("sd"))
			namer := prefixMap["sd"]
			Expect(namer.UsedDeviceMap).To(HaveKeyWithValue("sda", "boot"))
			Expect(namer.UsedDeviceMap).To(HaveKeyWithValue("sdc", "cdrom"))
			Expect(namer.UsedDeviceMap).To(HaveKeyWithValue("sdd", "cloudinit"))
			Expect(namer.UsedDeviceMap).NotTo(HaveKey("sdb"))

			By("Simulating domain conversion: calling makeDeviceName for each disk in order")
			bootTarget, _ := storage.MakeDeviceName("boot", v1.DiskBusSCSI, prefixMap)
			Expect(bootTarget).To(Equal("sda"), "boot disk should keep its existing target")

			cdromTarget, _ := storage.MakeDeviceName("cdrom", v1.DiskBusSATA, prefixMap)
			Expect(cdromTarget).To(Equal("sdc"), "cdrom should keep its existing target, not get reassigned")

			cloudinitTarget, _ := storage.MakeDeviceName("cloudinit", v1.DiskBusSATA, prefixMap)
			Expect(cloudinitTarget).To(Equal("sdd"), "cloudinit should keep its existing target, not get reassigned")

			By("Adding a new hotplug SCSI disk - it should get sdb (the gap)")
			newDiskName, index := storage.MakeDeviceName("newhotplug", v1.DiskBusSCSI, prefixMap)
			Expect(newDiskName).To(Equal("sdb"), "new disk should fill the gap at sdb, not collide with sdc/sdd")
			Expect(index).To(Equal(1))
		})
	})
})
