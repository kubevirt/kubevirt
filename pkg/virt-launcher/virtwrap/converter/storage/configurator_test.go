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

	k8sv1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/storage/volumepath"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

const (
	amd64 = "amd64"
	arm64 = "arm64"
	s390x = "s390x"
)

var _ = Describe("DiskConfigurator", func() {
	It("should produce no disks when the VMI has no volumes", func() {
		vmi := libvmi.New()
		c := newConverterContext(amd64)
		var domain api.Domain

		Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(api.Domain{}))
	})

	Context("volume source conversion", func() {
		It("should convert a PVC volume in filesystem mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a PVC volume in block mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"), ccWithBlockPVC("mypvc"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("block"),
				diskWithSource(api.DiskSource{Name: "mypvc", Dev: volumepath.BlockDevice("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a hotplug PVC volume in filesystem mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("hotplug-pvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithHotplugVolume("hotplug-pvc", v1.HotplugVolumeMounted))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-pvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.HotplugFilesystem("hotplug-pvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a hotplug PVC volume in block mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("hotplug-pvc", "my-claim"),
			)
			c := newConverterContext(amd64,
				ccWithHotplugVolume("hotplug-pvc", v1.HotplugVolumeMounted),
				ccWithBlockPVC("hotplug-pvc"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-pvc",
				diskWithDevice("disk"),
				diskWithType("block"),
				diskWithSource(api.DiskSource{Dev: volumepath.HotplugBlockDevice("hotplug-pvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a hotplug DataVolume in filesystem mode", func() {
			vmi := libvmi.New(
				libvmi.WithDataVolume("hotplug-dv", "my-dv"),
			)
			c := newConverterContext(amd64, ccWithHotplugVolume("hotplug-dv", v1.HotplugVolumeMounted))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-dv",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.HotplugFilesystem("hotplug-dv")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a hotplug DataVolume in block mode", func() {
			vmi := libvmi.New(
				libvmi.WithDataVolume("hotplug-dv", "my-dv"),
			)
			c := newConverterContext(amd64,
				ccWithHotplugVolume("hotplug-dv", v1.HotplugVolumeMounted),
				ccWithBlockDV("hotplug-dv"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-dv",
				diskWithDevice("disk"),
				diskWithType("block"),
				diskWithSource(api.DiskSource{Dev: volumepath.HotplugBlockDevice("hotplug-dv")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should exclude hotplug volumes that are not ready", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("perm-disk", "perm-claim"),
				libvmi.WithPersistentVolumeClaim("hotplug-pvc", "my-claim"),
			)
			c := newConverterContext(amd64,
				ccWithPermanentVolume("perm-disk"),
				ccWithHotplugVolume("hotplug-pvc", v1.VolumeBound),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("perm-disk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("perm-disk")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should include hotplug volumes with VolumeReady phase", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("hotplug-pvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithHotplugVolume("hotplug-pvc", v1.VolumeReady))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-pvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.HotplugFilesystem("hotplug-pvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should omit discard for volumes in the discard-ignore list", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"), ccWithDiscardIgnore("mypvc"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should omit discard for hotplug volumes in the discard-ignore list", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("hotplug-pvc", "my-claim"),
			)
			c := newConverterContext(amd64,
				ccWithHotplugVolume("hotplug-pvc", v1.HotplugVolumeMounted),
				ccWithDiscardIgnore("hotplug-pvc"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-pvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.HotplugFilesystem("hotplug-pvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should omit discard for hotplug DataVolumes in the discard-ignore list", func() {
			vmi := libvmi.New(
				libvmi.WithDataVolume("hotplug-dv", "my-dv"),
			)
			c := newConverterContext(amd64,
				ccWithHotplugVolume("hotplug-dv", v1.HotplugVolumeMounted),
				ccWithDiscardIgnore("hotplug-dv"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-dv",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.HotplugFilesystem("hotplug-dv")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a non-hotplug DataVolume in filesystem mode", func() {
			vmi := libvmi.New(
				libvmi.WithDataVolume("mydv", "my-dv"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mydv"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mydv",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mydv")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a non-hotplug DataVolume in block mode", func() {
			vmi := libvmi.New(
				libvmi.WithDataVolume("mydv", "my-dv"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mydv"), ccWithBlockDV("mydv"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mydv",
				diskWithDevice("disk"),
				diskWithType("block"),
				diskWithSource(api.DiskSource{Name: "mydv", Dev: volumepath.BlockDevice("mydv")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a PVC volume with CBT in filesystem mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"), ccWithCBT("mypvc", "/cbt/mypvc.qcow2"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{
					File: "/cbt/mypvc.qcow2",
					DataStore: &api.DataStore{
						Type: "file",
						Format: &api.DataStoreFormat{
							Type: "raw",
						},
						Source: &api.DiskSource{
							File: volumepath.Filesystem("mypvc"),
						},
					},
				}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a PVC volume with CBT in block mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"), ccWithBlockPVC("mypvc"), ccWithCBT("mypvc", "/cbt/mypvc.qcow2"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{
					Name: "mypvc",
					File: "/cbt/mypvc.qcow2",
					DataStore: &api.DataStore{
						Type: "block",
						Format: &api.DataStoreFormat{
							Type: "raw",
						},
						Source: &api.DiskSource{
							Dev: volumepath.BlockDevice("mypvc"),
						},
					},
				}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a hotplug PVC with CBT in filesystem mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("hotplug-pvc", "my-claim"),
			)
			c := newConverterContext(amd64,
				ccWithHotplugVolume("hotplug-pvc", v1.HotplugVolumeMounted),
				ccWithCBT("hotplug-pvc", "/cbt/hotplug-pvc.qcow2"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-pvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{
					File: "/cbt/hotplug-pvc.qcow2",
					DataStore: &api.DataStore{
						Type: "file",
						Format: &api.DataStoreFormat{
							Type: "raw",
						},
						Source: &api.DiskSource{
							File: volumepath.HotplugFilesystem("hotplug-pvc"),
						},
					},
				}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a hotplug PVC with CBT in block mode", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("hotplug-pvc", "my-claim"),
			)
			c := newConverterContext(amd64,
				ccWithHotplugVolume("hotplug-pvc", v1.HotplugVolumeMounted),
				ccWithBlockPVC("hotplug-pvc"),
				ccWithCBT("hotplug-pvc", "/cbt/hotplug-pvc.qcow2"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-pvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{
					File: "/cbt/hotplug-pvc.qcow2",
					DataStore: &api.DataStore{
						Type: "block",
						Format: &api.DataStoreFormat{
							Type: "raw",
						},
						Source: &api.DiskSource{
							Dev: volumepath.HotplugBlockDevice("hotplug-pvc"),
						},
					},
				}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a HostDisk volume", func() {
			vmi := libvmi.New(
				libvmi.WithHostDisk("myhostdisk", "/data/disk.img", v1.HostDiskExistsOrCreate),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("myhostdisk"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("myhostdisk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: hostdisk.GetMountedHostDiskPath("myhostdisk", "/data/disk.img")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a HostDisk volume with CBT", func() {
			vmi := libvmi.New(
				libvmi.WithHostDisk("myhostdisk", "/data/disk.img", v1.HostDiskExistsOrCreate),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("myhostdisk"), ccWithCBT("myhostdisk", "/cbt/myhostdisk.qcow2"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("myhostdisk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{
					File: "/cbt/myhostdisk.qcow2",
					DataStore: &api.DataStore{
						Type: "file",
						Format: &api.DataStoreFormat{
							Type: "raw",
						},
						Source: &api.DiskSource{
							File: hostdisk.GetMountedHostDiskPath("myhostdisk", "/data/disk.img"),
						},
					},
				}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a Sysprep volume", func() {
			vmi := libvmi.New(
				libvmi.WithSysprepConfigMap("mysysprep", "my-config"),
				libvmi.WithDisk("mysysprep", v1.DiskBusSATA),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mysysprep"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mysysprep",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: config.GetSysprepDiskPath("mysysprep")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSATA, Device: "sda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a CloudInitNoCloud volume", func() {
			vmi := libvmi.New(
				libvmi.WithCloudInitNoCloud(),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("cloudinitdisk"), ccWithVirtualMachine(vmi))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("cloudinitdisk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: cloudinit.GetIsoFilePath(cloudinit.DataSourceNoCloud, vmi.Name, vmi.Namespace)}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a CloudInitConfigDrive volume", func() {
			vmi := libvmi.New(
				libvmi.WithCloudInitConfigDrive(),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("cloudinitdisk"), ccWithVirtualMachine(vmi))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("cloudinitdisk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: cloudinit.GetIsoFilePath(cloudinit.DataSourceConfigDrive, vmi.Name, vmi.Namespace)}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert an EmptyDisk volume", func() {
			vmi := libvmi.New(
				libvmi.WithEmptyDisk("myemptydisk", v1.DiskBusVirtio, resource.MustParse("1Gi")),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("myemptydisk"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("myemptydisk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: emptydisk.NewEmptyDiskCreator().FilePathForVolumeName("myemptydisk")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a ContainerDisk volume", func() {
			vmi := libvmi.New(
				libvmi.WithContainerDisk("mycontainerdisk", "my-image:latest"),
			)
			ephemeralCreator := &fake.MockEphemeralDiskImageCreator{BaseDir: "/var/run/libvirt/kubevirt-ephemeral-disk/"}
			c := newConverterContext(amd64,
				ccWithPermanentVolume("mycontainerdisk"),
				ccWithEphemeraldiskCreator(ephemeralCreator),
				ccWithDiskInfo("mycontainerdisk", &disk.DiskInfo{Format: "qcow2"}),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mycontainerdisk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: ephemeralCreator.GetFilePath("mycontainerdisk")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
				diskWithBackingStore(api.BackingStore{
					Type: "file",
					Format: &api.BackingStoreFormat{
						Type: "qcow2",
					},
					Source: &api.DiskSource{
						File: containerdisk.GetDiskTargetPathFromLauncherView("mycontainerdisk"),
					},
				}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert an EphemeralVolume in filesystem mode", func() {
			vmi := libvmi.New(
				libvmi.WithEphemeralPersistentVolumeClaim("myephemeral", "my-claim"),
			)
			ephemeralCreator := &fake.MockEphemeralDiskImageCreator{BaseDir: "/var/run/libvirt/kubevirt-ephemeral-disk/"}
			c := newConverterContext(amd64,
				ccWithPermanentVolume("myephemeral"),
				ccWithEphemeraldiskCreator(ephemeralCreator),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("myephemeral",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: ephemeralCreator.GetFilePath("myephemeral")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSATA, Device: "sda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithBackingStore(api.BackingStore{
					Type: "file",
					Format: &api.BackingStoreFormat{
						Type: "raw",
					},
					Source: &api.DiskSource{
						File: volumepath.Filesystem("myephemeral"),
					},
				}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert an EphemeralVolume in block mode", func() {
			vmi := libvmi.New(
				libvmi.WithEphemeralPersistentVolumeClaim("myephemeral", "my-claim"),
			)
			ephemeralCreator := &fake.MockEphemeralDiskImageCreator{BaseDir: "/var/run/libvirt/kubevirt-ephemeral-disk/"}
			c := newConverterContext(amd64,
				ccWithPermanentVolume("myephemeral"),
				ccWithEphemeraldiskCreator(ephemeralCreator),
				ccWithBlockPVC("myephemeral"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("myephemeral",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: ephemeralCreator.GetFilePath("myephemeral")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSATA, Device: "sda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "qcow2", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithBackingStore(api.BackingStore{
					Type: "block",
					Format: &api.BackingStoreFormat{
						Type: "raw",
					},
					Source: &api.DiskSource{
						Name: "myephemeral",
						Dev:  volumepath.BlockDevice("myephemeral"),
					},
				}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a ConfigMap volume", func() {
			vmi := libvmi.New(
				libvmi.WithConfigMapDisk("my-config", "myconfigmap"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("myconfigmap"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("myconfigmap",
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: config.GetConfigMapDiskPath("myconfigmap")}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a Secret volume", func() {
			vmi := libvmi.New(
				libvmi.WithSecretDisk("my-secret", "mysecret"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mysecret"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mysecret",
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: config.GetSecretDiskPath("mysecret")}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a DownwardAPI volume", func() {
			vmi := libvmi.New(
				libvmi.WithDownwardAPIDisk("mydownwardapi"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mydownwardapi"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mydownwardapi",
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: config.GetDownwardAPIDiskPath("mydownwardapi")}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a ServiceAccount volume", func() {
			vmi := libvmi.New(
				libvmi.WithServiceAccountDisk("mysa"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mysa-disk"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mysa-disk",
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: config.GetServiceAccountDiskPath()}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a DownwardMetrics volume", func() {
			vmi := libvmi.New(
				libvmi.WithDownwardMetricsVolume("mymetrics"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mymetrics"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mymetrics",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: config.DownwardMetricDisk}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop}),
				diskWithModel("virtio-non-transitional"),
				diskWithReadOnly(),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert a hotplug HostDisk via the hotplug PVC path", func() {
			vmi := libvmi.New(
				libvmi.WithHostDisk("hotplug-hd", "/data/disk.img", v1.HostDiskExistsOrCreate),
			)
			c := newConverterContext(amd64, ccWithHotplugVolume("hotplug-hd", v1.HotplugVolumeMounted))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("hotplug-hd",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.HotplugFilesystem("hotplug-hd")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should convert an empty CDRom with no matching volume", func() {
			vmi := libvmi.New(
				libvmi.WithEmptyCDRom(v1.DiskBusSATA, "mycdrom"),
			)
			c := newConverterContext(amd64)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mycdrom",
				diskWithDevice("cdrom"),
				diskWithType("block"),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSATA, Device: "sda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithReadOnly(),
			))
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("disk-level conversion", func() {
		DescribeTable("should set disk capacity as the minimum of request and capacity",
			func(requests, capacity, expected int64) {
				vmi := libvmi.New(
					libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
					libvmistatus.WithStatus(libvmistatus.New(
						libvmistatus.WithVolumeStatus(v1.VolumeStatus{
							Name: "mypvc",
							PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
								Capacity: k8sv1.ResourceList{
									k8sv1.ResourceStorage: *resource.NewQuantity(capacity, resource.DecimalSI),
								},
								Requests: k8sv1.ResourceList{
									k8sv1.ResourceStorage: *resource.NewQuantity(requests, resource.DecimalSI),
								},
							},
						}),
					)),
				)
				c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
				var domain api.Domain

				Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := newDomainWithDisks(newDisk("mypvc",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
					diskWithModel("virtio-non-transitional"),
					diskWithCapacity(expected),
				))
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("higher request than capacity", int64(9999), int64(1111), int64(1111)),
			Entry("lower request than capacity", int64(1111), int64(9999), int64(1111)),
		)

		DescribeTable("should assign SCSI controller address", func(vmiOpt libvmi.Option, expectedDevice string) {
			vmi := libvmi.New(vmiOpt)
			c := newConverterContext(amd64, ccWithPermanentVolume("myvol"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("myvol",
				diskWithDevice(expectedDevice),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("myvol")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSCSI, Device: "sda"}),
				diskWithAddress(api.Address{Type: "drive", Controller: "0", Bus: "0", Unit: "0"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
			))
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("LUN-type disk", libvmi.WithPersistentVolumeClaimLun("myvol", "my-claim", false), "lun"),
			Entry("Disk-type disk", libvmi.WithPersistentVolumeClaim("myvol", "my-claim", libvmi.WithDiskBus(v1.DiskBusSCSI)), "disk"),
		)

		DescribeTable("should set the virtio model per architecture",
			func(architecture, expectedModel string) {
				vmi := libvmi.New(
					libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
				)
				c := newConverterContext(architecture, ccWithPermanentVolume("mypvc"))
				var domain api.Domain

				Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := newDomainWithDisks(newDisk("mypvc",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
					diskWithModel(expectedModel),
				))
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("amd64", amd64, "virtio-non-transitional"),
			Entry("arm64", arm64, "virtio-non-transitional"),
			Entry("s390x", s390x, "virtio"),
		)

		It("should assign queues when block multi-queue is enabled", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
				libvmi.WithBlockMultiQueue(),
				libvmi.WithCPUCount(2, 1, 1),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap", Queues: new(uint(2))}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should set disk IO mode when requested", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim", libvmi.WithDiskIO("native")),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", IO: "native", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		DescribeTable("should set custom block size",
			func(architecture, expectedModel string, logical, physical uint) {
				vmi := libvmi.New(
					libvmi.WithPersistentVolumeClaim("mypvc", "my-claim", libvmi.WithDiskCustomBlockSize(logical, physical)),
				)
				c := newConverterContext(architecture, ccWithPermanentVolume("mypvc"))
				var domain api.Domain

				Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := newDomainWithDisks(newDisk("mypvc",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
					diskWithModel(expectedModel),
					diskWithBlockIO(api.BlockIO{
						LogicalBlockSize:   logical,
						PhysicalBlockSize:  physical,
						DiscardGranularity: new(physical),
					}),
				))
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("on amd64", amd64, "virtio-non-transitional", uint(1234), uint(1234)),
			Entry("1024 on s390x", s390x, "virtio", uint(1024), uint(1024)),
			Entry("2048 on s390x", s390x, "virtio", uint(2048), uint(2048)),
			Entry("4096 on s390x", s390x, "virtio", uint(4096), uint(4096)),
		)

		DescribeTable("should reject custom block size exceeding s390x maximum",
			func(logical, physical uint) {
				vmi := libvmi.New(
					libvmi.WithPersistentVolumeClaim("mypvc", "my-claim", libvmi.WithDiskCustomBlockSize(logical, physical)),
				)
				c := newConverterContext(s390x, ccWithPermanentVolume("mypvc"))
				var domain api.Domain

				Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(
					MatchError(ContainSubstring("exceeds the maximum supported size")))
			},
			Entry("8192", uint(8192), uint(8192)),
			Entry("65536", uint(65536), uint(65536)),
			Entry("1 MiB", uint(1048576), uint(1048576)),
		)

		It("should detect block size via the injected detector for MatchVolume", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim", libvmi.WithDiskMatchVolumeBlockSize()),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
			stubDetector := func(_ *api.Disk) (*api.BlockIO, error) {
				return &api.BlockIO{
					LogicalBlockSize:  512,
					PhysicalBlockSize: 4096,
				}, nil
			}
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c, storage.DiskWithOptimalBlockIODetector(stubDetector)).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
				diskWithBlockIO(api.BlockIO{LogicalBlockSize: 512, PhysicalBlockSize: 4096}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should set shareable and cache=none", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim", libvmi.WithDiskShareable()),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", Cache: "none", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
				diskWithShareable(),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should set IOMMU driver when SEV is active on virtio disk", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim"),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"), ccWithSEV())
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap", IOMMU: "on"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should set boot order when provided", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mypvc", "my-claim", libvmi.WithDiskBootOrder(1)),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mypvc",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
				diskWithBootOrder(1),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should set LUN reservation source", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaimLun("mylun", "my-claim", true),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mylun"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mylun",
				diskWithDevice("lun"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{
					File: volumepath.Filesystem("mylun"),
					Reservations: &api.Reservations{
						Managed:   "no",
						Migration: "yes",
						SourceReservations: &api.SourceReservations{
							Type: "unix",
							Path: reservation.GetPrHelperSocketPath(),
							Mode: "client",
						},
					},
				}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSCSI, Device: "sda"}),
				diskWithAddress(api.Address{Type: "drive", Controller: "0", Bus: "0", Unit: "0"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("error policy", func() {
		DescribeTable("should set non-default error policy",
			func(policy v1.DiskErrorPolicy) {
				vmi := libvmi.New(
					libvmi.WithPersistentVolumeClaim("mypvc", "my-claim", libvmi.WithDiskErrorPolicy(policy)),
				)
				c := newConverterContext(amd64, ccWithPermanentVolume("mypvc"))
				var domain api.Domain

				Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

				expectedDomain := newDomainWithDisks(newDisk("mypvc",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("mypvc")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: policy, Discard: "unmap"}),
					diskWithModel("virtio-non-transitional"),
				))
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("report", v1.DiskErrorPolicyReport),
			Entry("ignore", v1.DiskErrorPolicyIgnore),
			Entry("enospace", v1.DiskErrorPolicyEnospace),
		)
	})

	Context("IOThreads", func() {
		It("should assign auto IOThreads to virtio disks", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk1", "claim1"),
				libvmi.WithPersistentVolumeClaim("disk2", "claim2"),
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicyAuto),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("disk1"), ccWithPermanentVolume("disk2"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(
				newDisk("disk1",
					diskWithDevice("disk"), diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("disk1")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap", IOThread: new(uint(1))}),
					diskWithModel("virtio-non-transitional"),
				),
				newDisk("disk2",
					diskWithDevice("disk"), diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("disk2")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vdb"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap", IOThread: new(uint(2))}),
					diskWithModel("virtio-non-transitional"),
				),
			)
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should assign dedicated IOThreads separately from auto threads", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("auto-disk", "claim1"),
				libvmi.WithPersistentVolumeClaim("dedicated-disk", "claim2", libvmi.WithDedicatedIOThreads(true)),
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicyAuto),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("auto-disk"), ccWithPermanentVolume("dedicated-disk"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(
				newDisk("auto-disk",
					diskWithDevice("disk"), diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("auto-disk")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap", IOThread: new(uint(1))}),
					diskWithModel("virtio-non-transitional"),
				),
				newDisk("dedicated-disk",
					diskWithDevice("disk"), diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("dedicated-disk")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vdb"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap", IOThread: new(uint(2))}),
					diskWithModel("virtio-non-transitional"),
				),
			)
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should not assign IOThreads to non-virtio disks", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaimLun("scsi-disk", "claim1", false),
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicyAuto),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("scsi-disk"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("scsi-disk",
				diskWithDevice("lun"), diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("scsi-disk")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSCSI, Device: "sda"}),
				diskWithAddress(api.Address{Type: "drive", Controller: "0", Bus: "0", Unit: "0"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should assign supplemental pool IOThreads when policy is set", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mydisk", "claim1"),
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicySupplementalPool),
				libvmi.WithIOThreads(v1.DiskIOThreads{
					SupplementalPoolThreadCount: new(uint32(2)),
				}),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mydisk"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mydisk",
				diskWithDevice("disk"), diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mydisk")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{
					Name:        "qemu",
					Type:        "raw",
					ErrorPolicy: v1.DiskErrorPolicyStop,
					Discard:     "unmap",
					IOThreads: &api.DiskIOThreads{
						IOThread: []api.DiskIOThread{
							{Id: 1},
							{Id: 2},
						},
					},
				}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})
	})

	Context("device naming", func() {
		It("should assign correct prefixes for different bus types", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("virtio-disk", "claim1"),
				libvmi.WithPersistentVolumeClaimLun("scsi-lun", "claim2", false),
			)
			c := newConverterContext(amd64,
				ccWithPermanentVolume("virtio-disk"),
				ccWithPermanentVolume("scsi-lun"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(
				newDisk("virtio-disk",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("virtio-disk")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
					diskWithModel("virtio-non-transitional"),
				),
				newDisk("scsi-lun",
					diskWithDevice("lun"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("scsi-lun")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSCSI, Device: "sda"}),
					diskWithAddress(api.Address{Type: "drive", Controller: "0", Bus: "0", Unit: "0"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				),
			)
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should preserve existing device names from volume status", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("disk1", "claim1"),
				libvmi.WithPersistentVolumeClaim("disk2", "claim2"),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithVolumeStatus(v1.VolumeStatus{Name: "disk1", Target: "vdb"}),
					libvmistatus.WithVolumeStatus(v1.VolumeStatus{Name: "disk2", Target: "vda"}),
				)),
			)
			c := newConverterContext(amd64,
				ccWithPermanentVolume("disk1"),
				ccWithPermanentVolume("disk2"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(
				newDisk("disk1",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("disk1")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vdb"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
					diskWithModel("virtio-non-transitional"),
				),
				newDisk("disk2",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("disk2")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
					diskWithModel("virtio-non-transitional"),
				),
			)
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should auto-assign device name when volume status has no target", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("mydisk", "claim1"),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithVolumeStatus(v1.VolumeStatus{Name: "mydisk", Target: ""}),
				)),
			)
			c := newConverterContext(amd64, ccWithPermanentVolume("mydisk"))
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(newDisk("mydisk",
				diskWithDevice("disk"),
				diskWithType("file"),
				diskWithSource(api.DiskSource{File: volumepath.Filesystem("mydisk")}),
				diskWithTarget(api.DiskTarget{Bus: v1.DiskBusVirtio, Device: "vda"}),
				diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				diskWithModel("virtio-non-transitional"),
			))
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should fill target gap after hotplug detach", func() {
			vmi := libvmi.New(
				libvmi.WithPersistentVolumeClaim("boot", "boot-claim", libvmi.WithDiskBus(v1.DiskBusSCSI)),
				libvmi.WithCDRom("cdrom", v1.DiskBusSATA, "cdrom-claim"),
				libvmi.WithPersistentVolumeClaim("newhotplug", "hotplug-claim", libvmi.WithDiskBus(v1.DiskBusSCSI)),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithVolumeStatus(v1.VolumeStatus{Name: "boot", Target: "sda"}),
					libvmistatus.WithVolumeStatus(v1.VolumeStatus{Name: "cdrom", Target: "sdc"}),
				)),
			)
			c := newConverterContext(amd64,
				ccWithPermanentVolume("boot"),
				ccWithPermanentVolume("cdrom"),
				ccWithPermanentVolume("newhotplug"),
			)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithDisks(
				newDisk("boot",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("boot")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSCSI, Device: "sda"}),
					diskWithAddress(api.Address{Type: "drive", Controller: "0", Bus: "0", Unit: "0"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				),
				newDisk("cdrom",
					diskWithDevice("cdrom"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("cdrom")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSATA, Device: "sdc"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
					diskWithReadOnly(),
				),
				newDisk("newhotplug",
					diskWithDevice("disk"),
					diskWithType("file"),
					diskWithSource(api.DiskSource{File: volumepath.Filesystem("newhotplug")}),
					diskWithTarget(api.DiskTarget{Bus: v1.DiskBusSCSI, Device: "sdb"}),
					diskWithAddress(api.Address{Type: "drive", Controller: "0", Bus: "0", Unit: "1"}),
					diskWithDriver(api.DiskDriver{Name: "qemu", Type: "raw", ErrorPolicy: v1.DiskErrorPolicyStop, Discard: "unmap"}),
				),
			)
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should generate correct device names at multi-letter boundaries", func() {
			// 26*26 disks to reach the last two-letter device suffix (vdyz)
			const diskCount = 26 * 26

			vmiOpts := make([]libvmi.Option, 0, diskCount)
			ccOpts := make([]converterContextOption, 0, diskCount)
			for i := range diskCount {
				name := fmt.Sprintf("disk%d", i)
				vmiOpts = append(vmiOpts, libvmi.WithPersistentVolumeClaim(name, name+"-claim"))
				ccOpts = append(ccOpts, ccWithPermanentVolume(name))
			}

			vmi := libvmi.New(vmiOpts...)
			c := newConverterContext(amd64, ccOpts...)
			var domain api.Domain

			Expect(storage.NewDiskConfigurator(c).Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.Devices.Disks[0].Target.Device).To(Equal("vda"))
			Expect(domain.Spec.Devices.Disks[1].Target.Device).To(Equal("vdb"))
			Expect(domain.Spec.Devices.Disks[25].Target.Device).To(Equal("vdz"))
			Expect(domain.Spec.Devices.Disks[26].Target.Device).To(Equal("vdaa"))
			Expect(domain.Spec.Devices.Disks[51].Target.Device).To(Equal("vdaz"))
			Expect(domain.Spec.Devices.Disks[675].Target.Device).To(Equal("vdyz"))
		})
	})
})

func newDomainWithDisks(disks ...api.Disk) api.Domain {
	return api.Domain{
		Spec: api.DomainSpec{
			Devices: api.Devices{
				Disks: disks,
			},
		},
	}
}

type diskOption func(*api.Disk)

func newDisk(name string, opts ...diskOption) api.Disk {
	d := api.Disk{
		Alias: api.NewUserDefinedAlias(name),
	}
	for _, opt := range opts {
		opt(&d)
	}
	return d
}

func diskWithType(t string) diskOption {
	return func(d *api.Disk) { d.Type = t }
}

func diskWithDevice(device string) diskOption {
	return func(d *api.Disk) { d.Device = device }
}

func diskWithSource(s api.DiskSource) diskOption {
	return func(d *api.Disk) { d.Source = s }
}

func diskWithTarget(t api.DiskTarget) diskOption {
	return func(d *api.Disk) { d.Target = t }
}

func diskWithDriver(drv api.DiskDriver) diskOption {
	return func(d *api.Disk) { d.Driver = &drv }
}

func diskWithModel(m string) diskOption {
	return func(d *api.Disk) { d.Model = m }
}

func diskWithAddress(a api.Address) diskOption {
	return func(d *api.Disk) { d.Address = &a }
}

func diskWithBlockIO(b api.BlockIO) diskOption {
	return func(d *api.Disk) { d.BlockIO = &b }
}

func diskWithBackingStore(b api.BackingStore) diskOption {
	return func(d *api.Disk) { d.BackingStore = &b }
}

func diskWithBootOrder(order uint) diskOption {
	return func(d *api.Disk) { d.BootOrder = &api.BootOrder{Order: order} }
}

func diskWithCapacity(capacity int64) diskOption {
	return func(d *api.Disk) { d.Capacity = &capacity }
}

func diskWithReadOnly() diskOption {
	return func(d *api.Disk) { d.ReadOnly = &api.ReadOnly{} }
}

func diskWithShareable() diskOption {
	return func(d *api.Disk) { d.Shareable = &api.Shareable{} }
}

type converterContextOption func(*convertertypes.ConverterContext)

func newConverterContext(architecture string, opts ...converterContextOption) *convertertypes.ConverterContext {
	c := &convertertypes.ConverterContext{
		Architecture:     archconverter.NewConverter(architecture),
		PermanentVolumes: map[string]v1.VolumeStatus{},
		HotplugVolumes:   map[string]v1.VolumeStatus{},
		IsBlockPVC:       map[string]bool{},
		IsBlockDV:        map[string]bool{},
		ApplyCBT:         map[string]string{},
	}
	for _, f := range opts {
		f(c)
	}
	return c
}

func ccWithPermanentVolume(name string) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.PermanentVolumes[name] = v1.VolumeStatus{}
	}
}

func ccWithHotplugVolume(name string, phase v1.VolumePhase) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.HotplugVolumes[name] = v1.VolumeStatus{Phase: phase}
	}
}

func ccWithBlockPVC(name string) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.IsBlockPVC[name] = true
	}
}

func ccWithBlockDV(name string) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.IsBlockDV[name] = true
	}
}

func ccWithDiscardIgnore(names ...string) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.VolumesDiscardIgnore = append(c.VolumesDiscardIgnore, names...)
	}
}

func ccWithSEV() converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.UseLaunchSecuritySEV = true
	}
}

func ccWithCBT(name, path string) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.ApplyCBT[name] = path
	}
}

func ccWithVirtualMachine(vmi *v1.VirtualMachineInstance) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.VirtualMachine = vmi
	}
}

func ccWithEphemeraldiskCreator(creator *fake.MockEphemeralDiskImageCreator) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		c.EphemeraldiskCreator = creator
	}
}

func ccWithDiskInfo(name string, info *disk.DiskInfo) converterContextOption {
	return func(c *convertertypes.ConverterContext) {
		if c.DisksInfo == nil {
			c.DisksInfo = map[string]*disk.DiskInfo{}
		}
		c.DisksInfo[name] = info
	}
}
