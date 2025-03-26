package apply_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Preference.Devices", func() {
	var (
		vmi                  *virtv1.VirtualMachineInstance
		instancetypeSpec     *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec       *v1beta1.VirtualMachinePreferenceSpec
		userDefinedBlockSize *virtv1.BlockSize

		field      = k8sfield.NewPath("spec", "template", "spec")
		vmiApplier = apply.NewVMIApplier()
	)

	BeforeEach(func() {
		vmi = libvmi.New()

		userDefinedBlockSize = &virtv1.BlockSize{
			Custom: &virtv1.CustomBlockSize{
				Logical:  512,
				Physical: 512,
			},
		}
		vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.P(false)
		vmi.Spec.Domain.Devices.AutoattachMemBalloon = pointer.P(false)
		vmi.Spec.Domain.Devices.Disks = []virtv1.Disk{
			{
				Cache:     virtv1.CacheWriteBack,
				IO:        virtv1.IONative,
				BlockSize: userDefinedBlockSize,
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{
						Bus: virtv1.DiskBusSCSI,
					},
				},
			},
			{
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{},
				},
			},
			{
				DiskDevice: virtv1.DiskDevice{
					CDRom: &virtv1.CDRomTarget{
						Bus: virtv1.DiskBusSATA,
					},
				},
			},
			{
				DiskDevice: virtv1.DiskDevice{
					CDRom: &virtv1.CDRomTarget{},
				},
			},
			{
				DiskDevice: virtv1.DiskDevice{
					LUN: &virtv1.LunTarget{
						Bus: virtv1.DiskBusSATA,
					},
				},
			},
			{
				DiskDevice: virtv1.DiskDevice{
					LUN: &virtv1.LunTarget{},
				},
			},
		}
		vmi.Spec.Domain.Devices.Inputs = []virtv1.Input{
			{
				Bus:  "usb",
				Type: "tablet",
			},
			{},
		}
		vmi.Spec.Domain.Devices.Interfaces = []virtv1.Interface{
			{
				Name:  "primary",
				Model: "e1000",
			},
			{
				Name: "secondary",
			},
		}
		vmi.Spec.Domain.Devices.Sound = &virtv1.SoundDevice{}

		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Devices: &v1beta1.DevicePreferences{
				PreferredAutoattachGraphicsDevice:   pointer.P(true),
				PreferredAutoattachMemBalloon:       pointer.P(true),
				PreferredAutoattachPodInterface:     pointer.P(true),
				PreferredAutoattachSerialConsole:    pointer.P(true),
				PreferredAutoattachInputDevice:      pointer.P(true),
				PreferredDiskDedicatedIoThread:      pointer.P(true),
				PreferredDisableHotplug:             pointer.P(true),
				PreferredUseVirtioTransitional:      pointer.P(true),
				PreferredNetworkInterfaceMultiQueue: pointer.P(true),
				PreferredBlockMultiQueue:            pointer.P(true),
				PreferredDiskBlockSize: &virtv1.BlockSize{
					Custom: &virtv1.CustomBlockSize{
						Logical:  4096,
						Physical: 4096,
					},
				},
				PreferredDiskCache:           virtv1.CacheWriteThrough,
				PreferredDiskIO:              virtv1.IONative,
				PreferredDiskBus:             virtv1.DiskBusVirtio,
				PreferredCdromBus:            virtv1.DiskBusSCSI,
				PreferredLunBus:              virtv1.DiskBusSATA,
				PreferredInputBus:            virtv1.InputBusVirtio,
				PreferredInputType:           virtv1.InputTypeTablet,
				PreferredInterfaceModel:      virtv1.VirtIO,
				PreferredSoundModel:          "ac97",
				PreferredRng:                 &virtv1.Rng{},
				PreferredInterfaceMasquerade: &virtv1.InterfaceMasquerade{},
			},
		}
	})

	// TODO - break this up into smaller more targeted tests
	Context("PreferredInterfaceMasquerade", func() {
		It("should be applied to interface on Pod network", func() {
			vmi.Spec.Networks = []virtv1.Network{{
				Name: vmi.Spec.Domain.Devices.Interfaces[0].Name,
				NetworkSource: virtv1.NetworkSource{
					Pod: &virtv1.PodNetwork{},
				},
			}}
			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Spec.Domain.Devices.Interfaces[0].Masquerade).ToNot(BeNil())
			Expect(vmi.Spec.Domain.Devices.Interfaces[1].Masquerade).To(BeNil())
		})
		It("should not be applied on interface that has another binding set", func() {
			vmi.Spec.Domain.Devices.Interfaces[0].SRIOV = &virtv1.InterfaceSRIOV{}
			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Spec.Domain.Devices.Interfaces[0].Masquerade).To(BeNil())
			Expect(vmi.Spec.Domain.Devices.Interfaces[0].SRIOV).ToNot(BeNil())
		})
		It("should not be applied on interface that is not on Pod network", func() {
			vmi.Spec.Networks = []virtv1.Network{{
				Name: vmi.Spec.Domain.Devices.Interfaces[0].Name,
			}}
			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Spec.Domain.Devices.Interfaces[0].Masquerade).To(BeNil())
		})
	})

	It("should apply to VMI", func() {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Devices.AutoattachGraphicsDevice).To(HaveValue(BeFalse()))
		Expect(vmi.Spec.Domain.Devices.AutoattachMemBalloon).To(HaveValue(BeFalse()))
		Expect(vmi.Spec.Domain.Devices.AutoattachInputDevice).To(HaveValue(BeTrue()))
		Expect(vmi.Spec.Domain.Devices.Disks[0].Cache).To(Equal(virtv1.CacheWriteBack))
		Expect(vmi.Spec.Domain.Devices.Disks[0].IO).To(Equal(virtv1.IONative))
		Expect(vmi.Spec.Domain.Devices.Disks[0].BlockSize).To(HaveValue(Equal(*userDefinedBlockSize)))
		Expect(vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal(virtv1.DiskBusSCSI))
		Expect(vmi.Spec.Domain.Devices.Disks[2].DiskDevice.CDRom.Bus).To(Equal(virtv1.DiskBusSATA))
		Expect(vmi.Spec.Domain.Devices.Disks[4].DiskDevice.LUN.Bus).To(Equal(virtv1.DiskBusSATA))
		Expect(vmi.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(virtv1.InputBusUSB))
		Expect(vmi.Spec.Domain.Devices.Inputs[0].Type).To(Equal(virtv1.InputTypeTablet))
		Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal("e1000"))

		// Assert that everything that isn't defined in the VM/VMI should use Preferences
		Expect(vmi.Spec.Domain.Devices.AutoattachPodInterface).To(HaveValue(Equal(*preferenceSpec.Devices.PreferredAutoattachPodInterface)))
		Expect(vmi.Spec.Domain.Devices.AutoattachSerialConsole).To(HaveValue(Equal(*preferenceSpec.Devices.PreferredAutoattachSerialConsole)))
		Expect(vmi.Spec.Domain.Devices.DisableHotplug).To(Equal(*preferenceSpec.Devices.PreferredDisableHotplug))
		Expect(vmi.Spec.Domain.Devices.UseVirtioTransitional).To(HaveValue(Equal(*preferenceSpec.Devices.PreferredUseVirtioTransitional)))
		Expect(vmi.Spec.Domain.Devices.Disks[1].Cache).To(Equal(preferenceSpec.Devices.PreferredDiskCache))
		Expect(vmi.Spec.Domain.Devices.Disks[1].IO).To(Equal(preferenceSpec.Devices.PreferredDiskIO))
		Expect(vmi.Spec.Domain.Devices.Disks[1].BlockSize).To(HaveValue(Equal(*preferenceSpec.Devices.PreferredDiskBlockSize)))
		Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))
		Expect(vmi.Spec.Domain.Devices.Disks[3].DiskDevice.CDRom.Bus).To(Equal(preferenceSpec.Devices.PreferredCdromBus))
		Expect(vmi.Spec.Domain.Devices.Disks[5].DiskDevice.LUN.Bus).To(Equal(preferenceSpec.Devices.PreferredLunBus))
		Expect(vmi.Spec.Domain.Devices.Inputs[1].Bus).To(Equal(preferenceSpec.Devices.PreferredInputBus))
		Expect(vmi.Spec.Domain.Devices.Inputs[1].Type).To(Equal(preferenceSpec.Devices.PreferredInputType))
		Expect(vmi.Spec.Domain.Devices.Interfaces[1].Model).To(Equal(preferenceSpec.Devices.PreferredInterfaceModel))
		Expect(vmi.Spec.Domain.Devices.Sound.Model).To(Equal(preferenceSpec.Devices.PreferredSoundModel))
		Expect(vmi.Spec.Domain.Devices.Rng).To(HaveValue(Equal(*preferenceSpec.Devices.PreferredRng)))
		Expect(vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue).
			To(HaveValue(Equal(*preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue)))
		Expect(vmi.Spec.Domain.Devices.BlockMultiQueue).To(HaveValue(Equal(*preferenceSpec.Devices.PreferredBlockMultiQueue)))
	})

	It("Should apply when a VMI disk doesn't have a DiskDevice target defined", func() {
		vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk = nil

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))
	})

	It("[test_id:CNV-9817] Should ignore preference when a VMI disk have a DiskDevice defined", func() {
		diskTypeForTest := virtv1.DiskBusSCSI

		vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus = diskTypeForTest
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(diskTypeForTest))
	})

	Context("PreferredDiskDedicatedIoThread", func() {
		DescribeTable("should be ignored when", func(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec) {
			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			for _, disk := range vmi.Spec.Domain.Devices.Disks {
				if disk.DiskDevice.Disk != nil {
					Expect(disk.DedicatedIOThread).To(BeNil())
				}
			}
		},
			Entry("unset", &v1beta1.VirtualMachinePreferenceSpec{
				Devices: &v1beta1.DevicePreferences{},
			}),
			Entry("false", &v1beta1.VirtualMachinePreferenceSpec{
				Devices: &v1beta1.DevicePreferences{
					PreferredDiskDedicatedIoThread: pointer.P(false),
				},
			}),
		)
		It("should only apply to virtio disk devices", func() {
			preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
				Devices: &v1beta1.DevicePreferences{
					PreferredDiskDedicatedIoThread: pointer.P(true),
				},
			}
			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			for _, disk := range vmi.Spec.Domain.Devices.Disks {
				if disk.DiskDevice.Disk != nil {
					if disk.DiskDevice.Disk.Bus == virtv1.DiskBusVirtio {
						Expect(disk.DedicatedIOThread).To(HaveValue(BeTrue()))
					} else {
						Expect(disk.DedicatedIOThread).To(BeNil())
					}
				}
			}
		})
	})

	Context("PreferredTPM", func() {
		DescribeTable("should",
			func(vmiTPM, preferenceTPM, expectedTPM *virtv1.TPMDevice) {
				vmi.Spec.Domain.Devices.TPM = vmiTPM
				preferenceSpec.Devices.PreferredTPM = preferenceTPM
				Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
				Expect(vmi.Spec.Domain.Devices.TPM).To(Equal(expectedTPM))
			},
			Entry("only apply when TPM device is nil within VMI spec",
				nil,
				&virtv1.TPMDevice{Persistent: pointer.P(true)},
				&virtv1.TPMDevice{Persistent: pointer.P(true)},
			),
			Entry("not apply when TPM device is provided within VMI spec",
				&virtv1.TPMDevice{Persistent: pointer.P(true)},
				&virtv1.TPMDevice{},
				&virtv1.TPMDevice{Persistent: pointer.P(true)},
			),
			Entry("not apply when TPM device is provided in the preference but disabled within VMI spec",
				&virtv1.TPMDevice{Enabled: pointer.P(false)},
				&virtv1.TPMDevice{},
				&virtv1.TPMDevice{Enabled: pointer.P(false)},
			),
			Entry("not apply when TPM device is explicitly Enabled in the preference but disabled within VMI spec",
				&virtv1.TPMDevice{Enabled: pointer.P(false)},
				&virtv1.TPMDevice{Enabled: pointer.P(true)},
				&virtv1.TPMDevice{Enabled: pointer.P(false)},
			),
		)
	})
})
