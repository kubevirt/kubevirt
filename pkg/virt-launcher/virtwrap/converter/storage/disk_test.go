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
	"encoding/xml"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

var _ = Describe("Convert_v1_Disk_To_api_Disk", func() {
	DescribeTable("Should define disk capacity as the minimum of capacity and request", func(arch string, requests, capacity, expected int64) {
		context := &convertertypes.ConverterContext{Architecture: archconverter.NewConverter(arch)}
		v1Disk := v1.Disk{
			Name: "myvolume",
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{Bus: v1.VirtIO},
			},
		}
		apiDisk := api.Disk{}
		devicePerBus := map[string]storage.DeviceNamer{}
		numQueues := uint(2)
		volumeStatusMap := make(map[string]v1.VolumeStatus)
		volumeStatusMap["myvolume"] = v1.VolumeStatus{
			PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
				Capacity: k8sv1.ResourceList{
					k8sv1.ResourceStorage: *resource.NewQuantity(capacity, resource.DecimalSI),
				},
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: *resource.NewQuantity(requests, resource.DecimalSI),
				},
			},
		}
		storage.Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, &numQueues, volumeStatusMap)
		Expect(apiDisk.Capacity).ToNot(BeNil())
		Expect(*apiDisk.Capacity).To(Equal(expected))
	},
		Entry("Higher request than capacity on amd64", amd64, int64(9999), int64(1111), int64(1111)),
		Entry("Higher request than capacity on arm64", arm64, int64(9999), int64(1111), int64(1111)),
		Entry("Higher request than capacity on s390x", s390x, int64(9999), int64(1111), int64(1111)),
		Entry("Lower request than capacity on amd64", amd64, int64(1111), int64(9999), int64(1111)),
		Entry("Lower request than capacity on arm64", arm64, int64(1111), int64(9999), int64(1111)),
		Entry("Lower request than capacity on s390x", s390x, int64(1111), int64(9999), int64(1111)),
	)

	DescribeTable("Should assign scsi controller to", func(diskDevice v1.DiskDevice) {
		context := &convertertypes.ConverterContext{}
		v1Disk := v1.Disk{
			Name:       "myvolume",
			DiskDevice: diskDevice,
		}
		apiDisk := api.Disk{}
		devicePerBus := map[string]storage.DeviceNamer{}
		numQueues := uint(2)
		volumeStatusMap := make(map[string]v1.VolumeStatus)
		volumeStatusMap["myvolume"] = v1.VolumeStatus{}
		storage.Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, &numQueues, volumeStatusMap)
		Expect(apiDisk.Address).ToNot(BeNil())
		Expect(apiDisk.Address.Bus).To(Equal("0"))
		Expect(apiDisk.Address.Controller).To(Equal("0"))
		Expect(apiDisk.Address.Type).To(Equal("drive"))
		Expect(apiDisk.Address.Unit).To(Equal("0"))
	},
		Entry("LUN-type disk", v1.DiskDevice{
			LUN: &v1.LunTarget{Bus: "scsi"},
		}),
		Entry("Disk-type disk", v1.DiskDevice{
			Disk: &v1.DiskTarget{Bus: "scsi"},
		}),
	)

	DescribeTable("Should add boot order when provided", func(arch, expectedModel string) {
		order := uint(1)
		kubevirtDisk := &v1.Disk{
			Name:      "mydisk",
			BootOrder: &order,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.VirtIO,
				},
			},
		}
		convertedDisk := fmt.Sprintf(`<Disk device="disk" type="" model="%s">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver name="qemu" type="" discard="unmap"></driver>
  <alias name="ua-mydisk"></alias>
  <boot order="1"></boot>
</Disk>`, expectedModel)
		xml := diskToDiskXML(arch, kubevirtDisk)
		Expect(xml).To(Equal(convertedDisk))
	},
		Entry("on amd64", amd64, "virtio-non-transitional"),
		Entry("on arm64", arm64, "virtio-non-transitional"),
		Entry("on s390x", s390x, "virtio"),
	)

	DescribeTable("should set disk I/O mode if requested", func(arch string) {
		v1Disk := &v1.Disk{
			IO: "native",
		}
		xml := diskToDiskXML(arch, v1Disk)
		expectedXML := `<Disk device="" type="">
  <source></source>
  <target></target>
  <driver io="native" name="qemu" type=""></driver>
  <alias name="ua-"></alias>
</Disk>`
		Expect(xml).To(Equal(expectedXML))
	},
		Entry("on amd64", amd64),
		Entry("on arm64", arm64),
		Entry("on s390x", s390x),
	)

	DescribeTable("should not set disk I/O mode if not requested", func(arch string) {
		v1Disk := &v1.Disk{}
		xml := diskToDiskXML(arch, v1Disk)
		expectedXML := `<Disk device="" type="">
  <source></source>
  <target></target>
  <driver name="qemu" type=""></driver>
  <alias name="ua-"></alias>
</Disk>`
		Expect(xml).To(Equal(expectedXML))
	},
		Entry("on amd64", amd64),
		Entry("on arm64", arm64),
		Entry("on s390x", s390x),
	)

	DescribeTable("Should omit boot order when not provided", func(arch, expectedModel string) {
		kubevirtDisk := &v1.Disk{
			Name: "mydisk",
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.VirtIO,
				},
			},
		}
		var convertedDisk = fmt.Sprintf(`<Disk device="disk" type="" model="%s">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver name="qemu" type="" discard="unmap"></driver>
  <alias name="ua-mydisk"></alias>
</Disk>`, expectedModel)
		xml := diskToDiskXML(arch, kubevirtDisk)
		Expect(xml).To(Equal(convertedDisk))
	},
		Entry("on amd64", amd64, "virtio-non-transitional"),
		Entry("on arm64", arm64, "virtio-non-transitional"),
		Entry("on s390x", s390x, "virtio"),
	)

	DescribeTable("should set sharable and the cache if requested", func(arch, expectedModel string) {
		v1Disk := &v1.Disk{
			Name: "mydisk",
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.VirtIO,
				},
			},
			Shareable: pointer.P(true),
		}
		var expectedXML = fmt.Sprintf(`<Disk device="disk" type="" model="%s">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver cache="none" name="qemu" type="" discard="unmap"></driver>
  <alias name="ua-mydisk"></alias>
  <shareable></shareable>
</Disk>`, expectedModel)
		xml := diskToDiskXML(arch, v1Disk)
		Expect(xml).To(Equal(expectedXML))
	},
		Entry("on amd64", amd64, "virtio-non-transitional"),
		Entry("on arm64", arm64, "virtio-non-transitional"),
		Entry("on s390x", s390x, "virtio"),
	)

	It("should assign queues to a device if requested", func() {
		expectedQueues := uint(2)

		v1Disk := v1.Disk{
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{Bus: v1.VirtIO},
			},
		}
		apiDisk := api.Disk{}
		devicePerBus := map[string]storage.DeviceNamer{}
		numQueues := uint(2)
		storage.Convert_v1_Disk_To_api_Disk(&convertertypes.ConverterContext{Architecture: archconverter.NewConverter(amd64)}, &v1Disk, &apiDisk, devicePerBus, &numQueues, make(map[string]v1.VolumeStatus))
		Expect(apiDisk.Device).To(Equal("disk"), "expected disk device to be defined")
		Expect(*(apiDisk.Driver.Queues)).To(Equal(expectedQueues), "expected queues to be 2")
	})

	It("should not assign queues to a device if omitted", func() {
		v1Disk := v1.Disk{
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{},
			},
		}
		apiDisk := api.Disk{}
		devicePerBus := map[string]storage.DeviceNamer{}
		Expect(storage.Convert_v1_Disk_To_api_Disk(&convertertypes.ConverterContext{Architecture: archconverter.NewConverter(amd64)}, &v1Disk, &apiDisk, devicePerBus, nil, make(map[string]v1.VolumeStatus))).
			To(Succeed())
		Expect(apiDisk.Device).To(Equal("disk"), "expected disk device to be defined")
		Expect(apiDisk.Driver.Queues).To(BeNil(), "expected no queues to be requested")
	})
})

func diskToDiskXML(arch string, disk *v1.Disk) string {
	devicePerBus := make(map[string]storage.DeviceNamer)
	libvirtDisk := &api.Disk{}
	Expect(storage.Convert_v1_Disk_To_api_Disk(&convertertypes.ConverterContext{Architecture: archconverter.NewConverter(arch), UseVirtioTransitional: false}, disk, libvirtDisk, devicePerBus, nil, make(map[string]v1.VolumeStatus))).To(Succeed())
	data, err := xml.MarshalIndent(libvirtDisk, "", "  ")
	Expect(err).ToNot(HaveOccurred())
	return string(data)
}
