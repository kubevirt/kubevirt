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

package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("GetVirtualMachineInstanceVolumes", func() {

	const backendVolume = "persistent-state-for-"

	Context("when no flags are provided (FLAG_INCLUDE_ALL)", func() {
		It("should return all volumes", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{
						{Name: "rootdisk"},
					},
					Domain: v1.DomainSpec{
						Firmware: &v1.Firmware{
							Bootloader: &v1.Bootloader{
								EFI: &v1.EFI{
									Persistent: func(b bool) *bool { return &b }(true),
								},
							},
						},
						Devices: v1.Devices{
							TPM: &v1.TPMDevice{
								Persistent: func(b bool) *bool { return &b }(true),
							},
						},
					},
				},
			}

			volumes := GetVirtualMachineInstanceVolumes(vmi, FLAG_INCLUDE_ALL)

			Expect(volumes).To(HaveLen(2))
			Expect(volumes[0].Name).To(Equal("rootdisk"))
			Expect(volumes[1].Name).To(Equal(backendVolume))
		})
	})

	Context("when FLAG_EXCLUDE_EFI is provided", func() {
		It("should exclude EFI volumes", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{
						{Name: "rootdisk"},
					},
					Domain: v1.DomainSpec{
						Firmware: &v1.Firmware{
							Bootloader: &v1.Bootloader{
								EFI: &v1.EFI{
									Persistent: func(b bool) *bool { return &b }(true),
								},
							},
						},
					},
				},
			}

			volumes := GetVirtualMachineInstanceVolumes(vmi, FLAG_EXCLUDE_EFI)

			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].Name).To(Equal("rootdisk"))
		})
	})

	Context("when FLAG_EXCLUDE_TPM is provided", func() {
		It("should exclude TPM volumes", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{
						{Name: "rootdisk"},
					},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							TPM: &v1.TPMDevice{
								Persistent: func(b bool) *bool { return &b }(true),
							},
						},
					},
				},
			}

			volumes := GetVirtualMachineInstanceVolumes(vmi, FLAG_EXCLUDE_TPM)

			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].Name).To(Equal("rootdisk"))
		})
	})

	Context("when multiple flags are provided", func() {
		It("should exclude both EFI and TPM volumes", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{
						{Name: "rootdisk"},
					},
					Domain: v1.DomainSpec{
						Firmware: &v1.Firmware{
							Bootloader: &v1.Bootloader{
								EFI: &v1.EFI{
									Persistent: func(b bool) *bool { return &b }(true),
								},
							},
						},
						Devices: v1.Devices{
							TPM: &v1.TPMDevice{
								Persistent: func(b bool) *bool { return &b }(true),
							},
						},
					},
				},
			}

			volumes := GetVirtualMachineInstanceVolumes(vmi, FLAG_EXCLUDE_EFI|FLAG_EXCLUDE_TPM)

			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].Name).To(Equal("rootdisk"))
		})
	})
})
