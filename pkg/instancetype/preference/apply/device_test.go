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
 */

package apply_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

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
		vmi.Spec.Domain.Devices.PanicDevices = []virtv1.PanicDevice{{}}

		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Devices: &v1beta1.DevicePreferences{
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
				PreferredPanicDeviceModel:    pointer.P(virtv1.Hyperv),
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
		Expect(vmi.Spec.Domain.Devices.PanicDevices[0].Model).To(Equal(preferenceSpec.Devices.PreferredPanicDeviceModel))
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

	Context("PreferredPanicDeviceModel", func() {
		DescribeTable("should",
			func(preferredPanicDeviceModel *virtv1.PanicDeviceModel, vmiPanicDevices, expectedPanicDevices []virtv1.PanicDevice) {
				vmi.Spec.Domain.Devices.PanicDevices = vmiPanicDevices
				preferenceSpec.Devices.PreferredPanicDeviceModel = preferredPanicDeviceModel
				Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
				Expect(vmi.Spec.Domain.Devices.PanicDevices).To(Equal(expectedPanicDevices))
			},
			Entry("not apply when preferredPanicDeviceModel is nil",
				nil,
				[]virtv1.PanicDevice{{Model: pointer.P(virtv1.Hyperv)}},
				[]virtv1.PanicDevice{{Model: pointer.P(virtv1.Hyperv)}},
			),
			Entry("not apply when panic devices is not provided in the VMI spec",
				pointer.P(virtv1.Hyperv),
				[]virtv1.PanicDevice{},
				[]virtv1.PanicDevice{},
			),
			Entry("only apply when  panic device model is nil for a panic device provided in the VMI spec",
				pointer.P(virtv1.Isa),
				[]virtv1.PanicDevice{{Model: pointer.P(virtv1.Hyperv)}, {}, {Model: pointer.P(virtv1.PanicDeviceModel(""))}},
				[]virtv1.PanicDevice{
					{Model: pointer.P(virtv1.Hyperv)},
					{Model: pointer.P(virtv1.Isa)},
					{Model: pointer.P(virtv1.PanicDeviceModel(""))},
				},
			),
		)
	})

	DescribeTable("PreferredAutoAttach should", func(preferenceValue, vmiValue *bool, match types.GomegaMatcher) {
		type autoAttachField struct {
			preference **bool
			vmi        **bool
		}
		autoAttachFields := map[string]autoAttachField{
			"PreferredAutoattachGraphicsDevice": {
				&preferenceSpec.Devices.PreferredAutoattachGraphicsDevice,
				&vmi.Spec.Domain.Devices.AutoattachGraphicsDevice,
			},
			"PreferredAutoattachMemBalloon": {
				&preferenceSpec.Devices.PreferredAutoattachMemBalloon,
				&vmi.Spec.Domain.Devices.AutoattachMemBalloon,
			},
			"PreferredAutoattachPodInterface": {
				&preferenceSpec.Devices.PreferredAutoattachPodInterface,
				&vmi.Spec.Domain.Devices.AutoattachPodInterface,
			},
			"PreferredAutoattachSerialConsole": {
				&preferenceSpec.Devices.PreferredAutoattachSerialConsole,
				&vmi.Spec.Domain.Devices.AutoattachSerialConsole,
			},
			"PreferredAutoattachInputDevice": {
				&preferenceSpec.Devices.PreferredAutoattachInputDevice,
				&vmi.Spec.Domain.Devices.AutoattachInputDevice,
			},
		}
		for name, f := range autoAttachFields {
			*f.preference = preferenceValue
			*f.vmi = vmiValue
			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(*f.vmi).To(match, fmt.Sprintf("%s not applied correctly", name))
		}
	},
		Entry("apply true when VMI value is nil", pointer.P(true), nil, HaveValue(BeTrue())),
		Entry("apply false when VMI value is nil", pointer.P(false), nil, HaveValue(BeFalse())),
		Entry("not apply nil when VMI value is nil", nil, nil, BeNil()),
		Entry("not apply nil when VMI value is true", nil, pointer.P(true), HaveValue(BeTrue())),
		Entry("not apply nil when VMI value is false", nil, pointer.P(false), HaveValue(BeFalse())),
		Entry("not apply true when VMI value is false", pointer.P(true), pointer.P(false), HaveValue(BeFalse())),
		Entry("not apply false when VMI value is true", pointer.P(false), pointer.P(true), HaveValue(BeTrue())),
	)
})
