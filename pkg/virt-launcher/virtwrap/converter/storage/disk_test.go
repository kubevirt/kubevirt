package storage_test

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Disk Configurator", func() {
	const (
		blockPVCName = "pvc_block_test"
		blockDVName  = "dv_block_test"
		fileDVName   = "test-file-dv"
		filePVCName  = "test-file-pvc"
	)

	isBlockPVC := map[string]bool{
		blockPVCName: true,
		filePVCName:  false,
	}
	isBlockDV := map[string]bool{
		blockDVName: true,
		fileDVName:  false,
	}
	ephemeralDiskImageCreator := &fake.MockEphemeralDiskImageCreator{BaseDir: "/var/run/libvirt/kubevirt-ephemeral-disk/"}

	DescribeTable("Should define disk capacity as the minimum of capacity and request", func(arch string, requests, capacity, expected int64) {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture("amd64"),
		)
		vmi := libvmi.New(
			libvmi.WithDataVolume(blockDVName, blockPVCName),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{
							Name: blockDVName,
							PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
								Capacity: k8sv1.ResourceList{
									k8sv1.ResourceStorage: *resource.NewQuantity(capacity, resource.DecimalSI),
								},
								Requests: k8sv1.ResourceList{
									k8sv1.ResourceStorage: *resource.NewQuantity(requests, resource.DecimalSI),
								},
							},
						},
					),
				),
			),
		)
		var domain api.Domain

		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain.Spec.Devices.Disks[0].Capacity).ToNot(BeNil())
		Expect(*domain.Spec.Devices.Disks[0].Capacity).To(Equal(expected))
	},
		multiArchEntry("Higher request than capacity", int64(9999), int64(1111), int64(1111)),
		multiArchEntry("Lower request than capacity", int64(1111), int64(9999), int64(1111)),
	)

	DescribeTable("Should assign scsi controller to", func(diskDevice v1.DiskDevice) {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture("amd64"),
		)

		vmi := libvmi.New(
			libvmi.WithDataVolume(blockDVName, blockPVCName),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{
							Name: blockDVName,
						},
					),
				),
			),
		)
		vmi.Spec.Domain.Devices.Disks[0].DiskDevice = diskDevice

		domain := &api.Domain{}
		Expect(configurator.Configure(vmi, domain)).To(Succeed())

		Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
		disk := domain.Spec.Devices.Disks[0]

		Expect(disk.Address).ToNot(BeNil())
		Expect(disk.Address.Bus).To(Equal("0"))
		Expect(disk.Address.Controller).To(Equal("0"))
		Expect(disk.Address.Type).To(Equal("drive"))
		Expect(disk.Address.Unit).To(Equal("0"))
	},
		Entry("LUN-type disk", v1.DiskDevice{
			LUN: &v1.LunTarget{Bus: "scsi"},
		}),
		Entry("Disk-type disk", v1.DiskDevice{
			Disk: &v1.DiskTarget{Bus: "scsi"},
		}),
	)

	DescribeTable("Should add boot order when provided", func(arch, expectedModel string) {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture(arch),
		)

		vmi := libvmi.New(
			libvmi.WithDataVolume(blockDVName, blockPVCName, withBootOrder(1)),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{Name: blockDVName},
					),
				),
			),
		)

		domain := &api.Domain{}
		Expect(configurator.Configure(vmi, domain)).To(Succeed())

		Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
		disk := domain.Spec.Devices.Disks[0]

		Expect(disk.BootOrder).ToNot(BeNil())
		Expect(disk.BootOrder.Order).To(Equal(uint(1)))

		Expect(disk.Model).To(Equal(expectedModel))
	},
		Entry("on amd64", "amd64", "virtio-non-transitional"),
		Entry("on arm64", "arm64", "virtio-non-transitional"),
		Entry("on s390x", "s390x", "virtio"),
	)

	DescribeTable("should set disk I/O mode if requested", func(arch string) {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture(arch),
		)

		vmi := libvmi.New(
			libvmi.WithDataVolume(blockDVName, blockPVCName, withIOMode("native")),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{Name: blockDVName},
					),
				),
			),
		)

		domain := &api.Domain{}
		Expect(configurator.Configure(vmi, domain)).To(Succeed())

		Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
		disk := domain.Spec.Devices.Disks[0]

		Expect(disk.Driver).ToNot(BeNil())
		Expect(disk.Driver.IO).To(Equal(v1.DriverIO("native")))
	},
		multiArchEntry(""),
	)

	DescribeTable("should not set disk I/O mode if not requested", func(arch string) {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture(arch),
		)

		vmi := libvmi.New(
			libvmi.WithDataVolume(blockDVName, blockPVCName),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{Name: blockDVName},
					),
				),
			),
		)

		domain := &api.Domain{}
		Expect(configurator.Configure(vmi, domain)).To(Succeed())

		Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
		disk := domain.Spec.Devices.Disks[0]

		Expect(disk.Driver).ToNot(BeNil())
		Expect(disk.Driver.IO).To(BeEmpty())
	},
		multiArchEntry(""),
	)

	DescribeTable("Should omit boot order when not provided", func(arch, expectedModel string) {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture(arch),
		)

		vmi := libvmi.New(
			libvmi.WithDataVolume(blockDVName, blockPVCName),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{Name: blockDVName},
					),
				),
			),
		)

		domain := &api.Domain{}
		Expect(configurator.Configure(vmi, domain)).To(Succeed())

		Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
		disk := domain.Spec.Devices.Disks[0]

		Expect(disk.BootOrder).To(BeNil())

		Expect(disk.Model).To(Equal(expectedModel))
	},
		Entry("on amd64", "amd64", "virtio-non-transitional"),
		Entry("on arm64", "arm64", "virtio-non-transitional"),
		Entry("on s390x", "s390x", "virtio"),
	)

	DescribeTable("should set sharable and the cache if requested", func(arch, expectedModel string) {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture(arch),
		)

		vmi := libvmi.New(
			libvmi.WithDataVolume(blockDVName, blockPVCName, withShareable()),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{Name: blockDVName},
					),
				),
			),
		)

		domain := &api.Domain{}
		Expect(configurator.Configure(vmi, domain)).To(Succeed())

		Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
		disk := domain.Spec.Devices.Disks[0]

		Expect(disk.Shareable).ToNot(BeNil())
		Expect(disk.Driver.Cache).To(Equal("none"))
		Expect(disk.Model).To(Equal(expectedModel))
	},
		Entry("on amd64", "amd64", "virtio-non-transitional"),
		Entry("on arm64", "arm64", "virtio-non-transitional"),
		Entry("on s390x", "s390x", "virtio"),
	)

	DescribeTable("should configure hotplug disks",
		func(diskOption libvmi.Option, volumeName string, ignoreDiscard bool, isBlockMode bool) {
			var discardIgnore []string

			if ignoreDiscard {
				discardIgnore = append(discardIgnore, volumeName)
			}

			hotpluggedVolumeStatus := v1.VolumeStatus{
				Name:  volumeName,
				Phase: v1.HotplugVolumeMounted,
			}
			hotplugVolumes := map[string]v1.VolumeStatus{volumeName: hotpluggedVolumeStatus}

			configurator := storage.NewDiskConfigurator(
				storage.WithArchitecture("amd64"),
				storage.WithHotplugVolumes(hotplugVolumes),
				storage.WithIsBlockPVC(isBlockPVC),
				storage.WithIsBlockDV(isBlockDV),
				storage.WithVolumesDiscardIgnore(discardIgnore),
			)

			vmi := libvmi.New(
				diskOption,
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithPhase(v1.Running),
						libvmistatus.WithVolumeStatus(hotpluggedVolumeStatus),
					),
				),
			)

			domain := &api.Domain{}
			Expect(configurator.Configure(vmi, domain)).To(Succeed())

			Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
			disk := domain.Spec.Devices.Disks[0]

			Expect(disk.Driver.Name).To(Equal("qemu"))
			Expect(disk.Driver.Type).To(Equal("raw"))
			Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))

			if ignoreDiscard {
				Expect(disk.Driver.Discard).To(BeEmpty())
			} else {
				Expect(disk.Driver.Discard).To(Equal("unmap"))
			}

			basePath := filepath.Join(v1.HotplugDiskDir, volumeName)

			if isBlockMode {
				Expect(disk.Type).To(Equal("block"))
				Expect(disk.Source.Dev).To(Equal(basePath))
			} else {
				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Source.File).To(Equal(fmt.Sprintf("%s.img", basePath)))
			}
		},
		Entry("filesystem PVC", libvmi.WithHotplugPersistentVolumeClaim(filePVCName, filePVCName), filePVCName, false, false),
		Entry("block mode PVC", libvmi.WithHotplugPersistentVolumeClaim(blockPVCName, blockPVCName), blockPVCName, false, true),
		Entry("'discard ignore' PVC", libvmi.WithHotplugPersistentVolumeClaim(blockPVCName, blockPVCName), blockPVCName, true, true),
		Entry("filesystem DV", libvmi.WithHotplugDataVolume(fileDVName, filePVCName), fileDVName, false, false),
		Entry("block mode DV", libvmi.WithHotplugDataVolume(blockDVName, blockPVCName), blockDVName, false, true),
		Entry("'discard ignore' DV", libvmi.WithHotplugDataVolume(blockDVName, blockPVCName), blockDVName, true, true),
	)

	It("should generate the block backingstore disk within the domain", func() {
		configurator := storage.NewDiskConfigurator(
			storage.WithArchitecture("amd64"),
			storage.WithEphemeralDiskCreator(ephemeralDiskImageCreator),
			storage.WithIsBlockDV(isBlockDV),
			storage.WithIsBlockPVC(isBlockPVC),
		)

		vmi := libvmi.New(
			libvmi.WithEphemeralPersistentVolumeClaim(blockPVCName, "test-ephemeral"),
			libvmistatus.WithStatus(
				libvmistatus.New(
					libvmistatus.WithPhase(v1.Running),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{Name: "test-ephemeral"},
					),
				),
			),
		)

		domain := &api.Domain{}
		Expect(configurator.Configure(vmi, domain)).To(Succeed())

		Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
		disk := domain.Spec.Devices.Disks[0]

		By("Checking if the disk backing store type is block")
		Expect(disk.BackingStore).ToNot(BeNil())
		Expect(disk.BackingStore.Type).To(Equal("block"))
		By("Checking if the disk backing store device path is appropriately configured")
		Expect(disk.BackingStore.Source.Dev).To(Equal(storage.GetBlockDeviceVolumePath(blockPVCName)))
	})

	Context("virtio block multi-queue", func() {
		DescribeTable("should assign queues to a device if requested", func(expectedQueues uint, useBlkMq bool) {
			configurator := storage.NewDiskConfigurator(
				storage.WithArchitecture("amd64"),
				storage.WithVcpus(expectedQueues),
				storage.WithUseBlkMQ(useBlkMq),
			)

			vmi := libvmi.New(
				libvmi.WithDataVolume(blockDVName, blockPVCName),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithPhase(v1.Running),
						libvmistatus.WithVolumeStatus(
							v1.VolumeStatus{Name: blockDVName},
						),
					),
				),
			)
			domain := &api.Domain{}
			Expect(configurator.Configure(vmi, domain)).To(Succeed())

			Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
			disk := domain.Spec.Devices.Disks[0]
			Expect(disk.Driver).ToNot(BeNil(), "expected disk device to be defined")
			if useBlkMq {
				Expect(disk.Driver.Queues).ToNot(BeNil())
				Expect(*disk.Driver.Queues).To(Equal(expectedQueues), "expected queues to be 2")
			} else {
				Expect(disk.Driver.Queues).To(BeNil())
			}
		},
			Entry("with 2 queues", uint(2), true),
			Entry("without block multi-queue", uint(0), false),
		)
	})

	Context("With BlockIO", func() {
		const ()
		It("Should add blockio fields when custom sizes are provided", func() {
			configurator := storage.NewDiskConfigurator(
				storage.WithArchitecture("amd64"),
			)

			vmi := libvmi.New(
				libvmi.WithDataVolume(blockDVName, blockPVCName, withCustomBlockSize(1234, 1234, 1234)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithPhase(v1.Running),
						libvmistatus.WithVolumeStatus(
							v1.VolumeStatus{Name: blockDVName},
						),
					),
				),
			)

			domain := &api.Domain{}
			Expect(configurator.Configure(vmi, domain)).To(Succeed())

			Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
			disk := domain.Spec.Devices.Disks[0]

			Expect(disk.BlockIO).ToNot(BeNil())

			Expect(disk.BlockIO.LogicalBlockSize).To(Equal(uint(1234)))
			Expect(disk.BlockIO.PhysicalBlockSize).To(Equal(uint(1234)))
			Expect(*disk.BlockIO.DiscardGranularity).To(Equal(uint(1234)))
		})

		DescribeTable("Should detect disk BlockIO settings", func(dvName, pvcName string) {
			configurator := storage.NewDiskConfigurator(
				storage.WithArchitecture("amd64"),
				storage.WithIsBlockPVC(isBlockPVC),
				storage.WithIsBlockDV(isBlockDV),
				storage.WithBlockIoInspector(NewMockBlockIOInspector(512, 512, 512)),
			)

			vmi := libvmi.New(
				libvmi.WithDataVolume(fileDVName, filePVCName, withBlockSizeMatchVolume()),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithPhase(v1.Running),
						libvmistatus.WithVolumeStatus(
							v1.VolumeStatus{Name: blockDVName},
						),
					),
				),
			)

			domain := &api.Domain{}
			Expect(configurator.Configure(vmi, domain)).To(Succeed())

			Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
			disk := domain.Spec.Devices.Disks[0]

			Expect(disk.BlockIO).ToNot(BeNil())

			blockIO := disk.BlockIO
			Expect(blockIO.LogicalBlockSize).To(Equal(blockIO.PhysicalBlockSize))
			// The default for most filesystems nowadays is 4096 but it can be changed.
			// As such, relying on a specific value is flakey unless
			// we create a disk image and filesystem just for this test.
			// For now, as long as we have a value, the exact value doesn't matter.
			Expect(blockIO.LogicalBlockSize).ToNot(BeZero())
			Expect(blockIO.DiscardGranularity).ToNot(BeNil())
			Expect(*blockIO.DiscardGranularity).To(Equal(blockIO.LogicalBlockSize))
		},
			Entry("with block disk", blockDVName, blockPVCName),
			Entry("with file disk", fileDVName, filePVCName),
		)
	})

	Context("CBT", func() {
		DescribeTable("should create domain disk with datastore for block volumes with CBT enabled", func(diskOption libvmi.Option, volumeName string) {
			vmi := libvmi.New(
				diskOption,
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithPhase(v1.Running),
						libvmistatus.WithVolumeStatus(
							v1.VolumeStatus{Name: volumeName},
						),
					),
				),
			)

			cbtPath := "/var/lib/libvirt/qemu/cbt/" + volumeName + ".qcow2"

			configurator := storage.NewDiskConfigurator(
				storage.WithArchitecture("amd64"),
				storage.WithIsBlockPVC(isBlockPVC),
				storage.WithIsBlockDV(isBlockDV),
				storage.WithApplyCBT(map[string]string{volumeName: cbtPath}),
			)

			domain := &api.Domain{}
			Expect(configurator.Configure(vmi, domain)).To(Succeed())

			Expect(domain.Spec.Devices.Disks).To(HaveLen(1))
			disk := domain.Spec.Devices.Disks[0]

			// Verify CBT configuration
			Expect(disk.Type).To(Equal("file"))
			Expect(disk.Source.File).To(Equal(cbtPath))
			Expect(disk.Source.Name).To(Equal(volumeName))
			Expect(disk.Driver.Type).To(Equal("qcow2"))
			Expect(disk.Driver.ErrorPolicy).To(Equal(v1.DiskErrorPolicyStop))
			Expect(disk.Driver.Discard).To(Equal("unmap"))

			// Verify datastore configuration for block volumes
			Expect(disk.Source.DataStore).ToNot(BeNil())
			Expect(disk.Source.DataStore.Type).To(Equal("block"))
			Expect(disk.Source.DataStore.Format).ToNot(BeNil())
			Expect(disk.Source.DataStore.Format.Type).To(Equal("raw"))
			Expect(disk.Source.DataStore.Source).ToNot(BeNil())
			Expect(disk.Source.DataStore.Source.Dev).To(Equal(storage.GetBlockDeviceVolumePath(volumeName)))
		},
			Entry("PVC", libvmi.WithPersistentVolumeClaim(blockPVCName, blockPVCName), blockPVCName),
			Entry("DataVolume", libvmi.WithDataVolume(blockDVName, blockPVCName), blockDVName),
		)
	})

	Context("Device Naming", func() {
		DescribeTable("Should assign device names correctly based on bus and existing usage",
			func(diskOptions []libvmi.Option, volumeStatusOptions []libvmistatus.Option, expectedTargets map[string]string) {
				configurator := storage.NewDiskConfigurator(
					storage.WithArchitecture("amd64"),
				)
				statusOptions := []libvmistatus.Option{
					libvmistatus.WithPhase(v1.Running),
				}
				statusOptions = append(statusOptions, volumeStatusOptions...)
				options := []libvmi.Option{
					libvmistatus.WithStatus(
						libvmistatus.New(statusOptions...),
					),
				}
				options = append(options, diskOptions...)
				vmi := libvmi.New(options...)

				domain := &api.Domain{}
				Expect(configurator.Configure(vmi, domain)).To(Succeed())

				Expect(domain.Spec.Devices.Disks).To(HaveLen(len(diskOptions)))

				for _, outputDisk := range domain.Spec.Devices.Disks {
					diskName := outputDisk.Alias.GetName()
					expectedTarget, exists := expectedTargets[diskName]

					Expect(exists).To(BeTrue(), fmt.Sprintf("Unexpected disk found in output: %s", diskName))
					Expect(outputDisk.Target.Device).To(Equal(expectedTarget), fmt.Sprintf("Incorrect target for disk %s", diskName))
				}
			},

			Entry("Should assign sequential names for SATA disks",
				[]libvmi.Option{
					libvmi.WithDataVolume("disk1", "pvc1", withBus(v1.DiskBusSATA)),
					libvmi.WithDataVolume("disk2", "pvc2", withBus(v1.DiskBusSATA)),
				},
				nil,
				map[string]string{
					"disk1": "sda",
					"disk2": "sdb",
				},
			),
			Entry("Should use 'vda' for VirtIO and 'sda' for SCSI",
				[]libvmi.Option{
					libvmi.WithDataVolume("virtio-disk", "pvc1", withBus(v1.DiskBusVirtio)),
					libvmi.WithDataVolume("scsi-disk", "pvc2", withBus(v1.DiskBusSCSI)),
				},
				nil,
				map[string]string{
					"virtio-disk": "vda",
					"scsi-disk":   "sda",
				},
			),
			Entry("Should skip names that are already claimed in VolumeStatus",
				[]libvmi.Option{
					libvmi.WithDataVolume("old-disk", "pvc1", withBus(v1.DiskBusVirtio)),
					libvmi.WithDataVolume("new-disk", "pvc2", withBus(v1.DiskBusVirtio)),
				},
				[]libvmistatus.Option{
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{
							Name:   "old-disk",
							Target: "vda",
						},
					),
				},
				map[string]string{
					"old-disk": "vda",
					"new-disk": "vdb",
				},
			),
			Entry("Should preserve existing target if disk name matches VolumeStatus",
				[]libvmi.Option{
					libvmi.WithDataVolume("my-disk", "pvc1", withBus(v1.DiskBusVirtio)),
				},
				[]libvmistatus.Option{
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{
							Name:   "my-disk",
							Target: "vdz",
						},
					),
				},
				map[string]string{
					"my-disk": "vdz",
				},
			),
			Entry("Should fill a gap",
				[]libvmi.Option{
					libvmi.WithDataVolume("disk1", "pvc1", withBus(v1.DiskBusVirtio)),
					libvmi.WithDataVolume("disk2", "pvc2", withBus(v1.DiskBusVirtio)),
					libvmi.WithDataVolume("disk3", "pvc3", withBus(v1.DiskBusVirtio)),
				},
				[]libvmistatus.Option{
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{
							Name:   "disk1",
							Target: "vda",
						},
					),
					libvmistatus.WithVolumeStatus(
						v1.VolumeStatus{
							Name:   "disk3",
							Target: "vdc",
						},
					),
				},
				map[string]string{
					"disk1": "vda",
					"disk2": "vdb",
					"disk3": "vdc",
				},
			),
		)

		DescribeTable("FormatDeviceName should generate correct device strings",
			func(prefix string, index int, expected string) {
				res := storage.FormatDeviceName(prefix, index)
				Expect(res).To(Equal(expected))
			},
			Entry("Index 0", "sd", 0, "sda"),
			Entry("Index 1", "sd", 1, "sdb"),
			Entry("Index 25 (Rollover boundary)", "sd", 25, "sdz"),
			Entry("Index 26 (First double char)", "sd", 26, "sdaa"),
			Entry("Index 701 (Three chars)", "sd", 702, "sdaaa"),
		)
	})

})

const (
	amd64 = "amd64"
	arm64 = "arm64"
	s390x = "s390x"
)

// multiArchEntry returns a slice of Ginkgo TableEntry starting from one.
// It repeats the same TableEntry for every architecture.
// This is pretty useful when the same behavior is expected for every arch.
// **IMPORTANT**
// This requires the DescribeTable body func to have `arch string` as first
// parameter.

func multiArchEntry(text string, args ...any) []TableEntry {
	return []TableEntry{
		Entry(fmt.Sprintf("%s on %s", text, amd64), append([]any{amd64}, args...)...),
		Entry(fmt.Sprintf("%s on %s", text, arm64), append([]any{arm64}, args...)...),
		Entry(fmt.Sprintf("%s on %s", text, s390x), append([]any{s390x}, args...)...),
	}
}

func withBootOrder(bootOrder uint) libvmi.DiskOption {
	return func(disk *v1.Disk) {
		disk.BootOrder = &bootOrder
	}
}

func withIOMode(ioMode v1.DriverIO) libvmi.DiskOption {
	return func(disk *v1.Disk) {
		disk.IO = ioMode
	}
}

func withShareable() libvmi.DiskOption {
	return func(disk *v1.Disk) {
		disk.Shareable = pointer.P(true)
	}
}

func withBus(busType v1.DiskBus) libvmi.DiskOption {
	return func(disk *v1.Disk) {
		if disk.LUN != nil {
			disk.LUN.Bus = busType
		} else if disk.CDRom != nil {
			disk.CDRom.Bus = busType
		} else if disk.Disk != nil {
			disk.Disk.Bus = busType
		}
	}
}

func withCustomBlockSize(logical, physical uint, discardGranularity uint) libvmi.DiskOption {
	return func(disk *v1.Disk) {
		disk.BlockSize = &v1.BlockSize{
			Custom: &v1.CustomBlockSize{
				Logical:            logical,
				Physical:           physical,
				DiscardGranularity: pointer.P(discardGranularity),
			},
		}
	}
}

func withBlockSizeMatchVolume() libvmi.DiskOption {
	return func(disk *v1.Disk) {
		disk.BlockSize = &v1.BlockSize{
			MatchVolume: &v1.FeatureState{
				Enabled: pointer.P(true),
			},
		}
	}
}

type MockBlockIOInspector struct {
	logical            uint
	physical           uint
	discardGranularity uint
}

func NewMockBlockIOInspector(logical, physical, discardGranularity uint) MockBlockIOInspector {
	return MockBlockIOInspector{
		logical:            logical,
		physical:           physical,
		discardGranularity: discardGranularity,
	}
}

func (m MockBlockIOInspector) GetDevBlockIO(path string) (*api.BlockIO, error) {
	return &api.BlockIO{
		LogicalBlockSize:   m.logical,
		PhysicalBlockSize:  m.physical,
		DiscardGranularity: pointer.P(m.discardGranularity),
	}, nil
}

func (m MockBlockIOInspector) GetFileBlockIO(path string) (*api.BlockIO, error) {
	return &api.BlockIO{
		LogicalBlockSize:   m.logical,
		PhysicalBlockSize:  m.physical,
		DiscardGranularity: pointer.P(m.discardGranularity),
	}, nil
}
