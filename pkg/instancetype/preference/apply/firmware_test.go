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

//nolint:dupl
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

var _ = Describe("Preference.Firmware", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		field      = k8sfield.NewPath("spec", "template", "spec")
		vmiApplier = apply.NewVMIApplier()
	)

	BeforeEach(func() {
		vmi = libvmi.New()
	})

	It("should apply BIOS preferences full to VMI", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredUseBios:                 pointer.P(true),
				PreferredUseBiosSerial:           pointer.P(true),
				DeprecatedPreferredUseEfi:        pointer.P(false),
				DeprecatedPreferredUseSecureBoot: pointer.P(false),
			},
		}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(HaveValue(Equal(*preferenceSpec.Firmware.PreferredUseBiosSerial)))
	})

	It("should apply SecureBoot preferences full to VMI", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredUseBios:                 pointer.P(false),
				PreferredUseBiosSerial:           pointer.P(false),
				DeprecatedPreferredUseEfi:        pointer.P(true),
				DeprecatedPreferredUseSecureBoot: pointer.P(true),
			},
		}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(HaveValue(Equal(*preferenceSpec.Firmware.DeprecatedPreferredUseSecureBoot)))
	})

	It("should not overwrite user defined Bootloader.BIOS with DeprecatedPreferredUseEfi - bug #10313", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				DeprecatedPreferredUseEfi:        pointer.P(true),
				DeprecatedPreferredUseSecureBoot: pointer.P(true),
			},
		}
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{
			Bootloader: &virtv1.Bootloader{
				BIOS: &virtv1.BIOS{
					UseSerial: pointer.P(false),
				},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).To(BeNil())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(HaveValue(BeFalse()))
	})

	It("should not overwrite user defined value with PreferredUseBiosSerial - bug #10313", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredUseBios:       pointer.P(true),
				PreferredUseBiosSerial: pointer.P(true),
			},
		}
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{
			Bootloader: &virtv1.Bootloader{
				BIOS: &virtv1.BIOS{
					UseSerial: pointer.P(false),
				},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(HaveValue(BeFalse()))
	})

	It("should not overwrite user defined Bootloader.EFI with PreferredUseBios - bug #10313", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredUseBios:       pointer.P(true),
				PreferredUseBiosSerial: pointer.P(true),
			},
		}
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{
			Bootloader: &virtv1.Bootloader{
				EFI: &virtv1.EFI{
					SecureBoot: pointer.P(false),
				},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS).To(BeNil())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(HaveValue(BeFalse()))
	})

	It("should not overwrite user defined value with DeprecatedPreferredUseSecureBoot - bug #10313", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				DeprecatedPreferredUseEfi:        pointer.P(true),
				DeprecatedPreferredUseSecureBoot: pointer.P(true),
			},
		}
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{
			Bootloader: &virtv1.Bootloader{
				EFI: &virtv1.EFI{
					SecureBoot: pointer.P(false),
				},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(HaveValue(BeFalse()))
	})

	It("should apply PreferredEfi", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredEfi: &virtv1.EFI{
					Persistent: pointer.P(true),
					SecureBoot: pointer.P(true),
				},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(HaveValue(BeNil()))
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.Persistent).To(HaveValue(BeTrue()))
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(HaveValue(BeTrue()))
	})

	It("should ignore DeprecatedPreferredUseEfi and DeprecatedPreferredUseSecureBoot when using PreferredEfi", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredEfi: &virtv1.EFI{
					Persistent: pointer.P(true),
				},
				DeprecatedPreferredUseEfi:        pointer.P(false),
				DeprecatedPreferredUseSecureBoot: pointer.P(false),
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(HaveValue(BeNil()))
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.Persistent).To(HaveValue(BeTrue()))
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(BeNil())
	})

	It("should not overwrite EFI when using PreferredEfi - bug #12985", func() {
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{
			Bootloader: &virtv1.Bootloader{
				EFI: &virtv1.EFI{
					SecureBoot: pointer.P(false),
				},
			},
		}
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredEfi: &virtv1.EFI{
					SecureBoot: pointer.P(true),
					Persistent: pointer.P(true),
				},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).ToNot(BeNil())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).ToNot(BeNil())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(HaveValue(BeFalse()))
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.Persistent).To(BeNil())
	})

	It("should not apply PreferredEfi when VM already using BIOS - bug #12985", func() {
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{
			Bootloader: &virtv1.Bootloader{
				BIOS: &virtv1.BIOS{},
			},
		}
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Firmware: &v1beta1.FirmwarePreferences{
				PreferredEfi: &virtv1.EFI{},
			},
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS).ToNot(BeNil())
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).To(BeNil())
	})
})
