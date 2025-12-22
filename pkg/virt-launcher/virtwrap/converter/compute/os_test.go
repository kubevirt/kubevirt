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

package compute_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("OS Domain Configurator", func() {
	DescribeTable("should configure SMBIOS sysinfo when required", func(isSMBiosNeeded bool, expectedSMBios *api.SMBios) {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(compute.NewOSDomainConfigurator(isSMBiosNeeded, nil).Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				OS: api.OS{
					SMBios: expectedSMBios,
				},
			},
		}

		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("SMBIOS is required", true, &api.SMBios{Mode: "sysinfo"}),
		Entry("SMBIOS is not required", false, nil),
	)

	It("should set machine type", func() {
		vmi := libvmi.New(withMachineType("test"))
		var domain api.Domain

		Expect(compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				OS: api.OS{
					Type: api.OSType{
						Machine: "test",
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	DescribeTable("configures boot menu based on VMI start strategy", func(vmi *v1.VirtualMachineInstance, expectedBootMenu *api.BootMenu) {
		var domain api.Domain

		Expect(compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				OS: api.OS{
					BootMenu: expectedBootMenu,
				},
			},
		}

		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("boot menu is enabled when VMI starts paused",
			libvmi.New(withStartStrategy(v1.StartStrategyPaused)),
			&api.BootMenu{
				Enable:  "yes",
				Timeout: pointer.P(uint(10000)),
			},
		),
		Entry("boot menu is disabled when VMI does not start paused", libvmi.New(), nil),
	)

	DescribeTable("should configures BIOS UseSerial", func(useSerial bool, expectedBIOS *api.BIOS) {
		var domain api.Domain
		vmi := libvmi.New(withBIOSUseSerial(useSerial))

		Expect(compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				OS: api.OS{
					BIOS: expectedBIOS,
				},
			},
		}

		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("when BIOS UseSerial is enabled", true, &api.BIOS{UseSerial: "yes"}),
		Entry("when BIOS UseSerial is disabled", false, nil),
	)

	DescribeTable("configures kernel args based on VMI firmware settings", func(vmi *v1.VirtualMachineInstance, expectedKernelArgs string) {
		var domain api.Domain

		Expect(compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				OS: api.OS{
					KernelArgs: expectedKernelArgs,
				},
			},
		}

		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("kernel args are set when specified", libvmi.New(withKernelArgs("test-args")), "test-args"),
		Entry("kernel args are not set when firmware is nil", libvmi.New(), ""),
	)

	Context("EFI configuration", func() {
		DescribeTable("should configure EFI bootloader with pflash type", func(secureBoot bool) {
			vmi := libvmi.New(withEFIBootloader(secureBoot))
			var domain api.Domain
			efiConfig := &compute.EFIConfiguration{
				EFICode:      "/usr/share/OVMF/OVMF_CODE.fd",
				EFIVars:      "/usr/share/OVMF/OVMF_VARS.fd",
				SecureLoader: secureBoot,
			}

			Expect(compute.NewOSDomainConfigurator(false, efiConfig).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					OS: api.OS{
						BootLoader: &api.Loader{
							Path:     "/usr/share/OVMF/OVMF_CODE.fd",
							ReadOnly: "yes",
							Secure:   boolToYesNo(secureBoot),
							Type:     "pflash",
						},
						NVRam: &api.NVRam{
							Template: "/usr/share/OVMF/OVMF_VARS.fd",
							NVRam:    "/var/lib/libvirt/qemu/nvram/" + vmi.Name + "_VARS.fd",
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("without secure boot", false),
			Entry("with secure boot", true),
		)
	})

	Context("ACPI configuration", func() {
		DescribeTable("should configure ACPI table from secret volume", func(tableType string, withACPIOption func(string) libvmi.Option) {
			volumeName := tableType + "-secret"
			vmi := libvmi.New(
				withACPIOption(volumeName),
				withSecretVolume(volumeName),
			)
			var domain api.Domain

			Expect(compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					OS: api.OS{
						ACPI: &api.OSACPI{
							Table: []api.ACPITable{
								{
									Type: tableType,
									Path: filepath.Join(config.SecretSourceDir, volumeName, tableType+".bin"),
								},
							},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
			Entry("SLIC table", "slic", withACPISlic),
			Entry("MSDM table", "msdm", withACPIMsdm),
		)

		It("should configure both SLIC and MSDM ACPI tables", func() {
			slicVolume := "slic-secret"
			msdmVolume := "msdm-secret"
			vmi := libvmi.New(
				withACPISlicAndMsdm(slicVolume, msdmVolume),
				withSecretVolume(slicVolume),
				withSecretVolume(msdmVolume),
			)
			var domain api.Domain

			Expect(compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					OS: api.OS{
						ACPI: &api.OSACPI{
							Table: []api.ACPITable{
								{
									Type: "slic",
									Path: filepath.Join(config.SecretSourceDir, slicVolume, "slic.bin"),
								},
								{
									Type: "msdm",
									Path: filepath.Join(config.SecretSourceDir, msdmVolume, "msdm.bin"),
								},
							},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		})

		It("should return error when ACPI volume is not found", func() {
			vmi := libvmi.New(withACPISlic("missing-volume"))
			var domain api.Domain

			err := compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Firmware's volume for slic was not found"))
		})

		It("should return error when ACPI is set but no tables are specified", func() {
			vmi := libvmi.New(withEmptyACPI())
			var domain api.Domain

			err := compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("No ACPI tables were set. Expecting at least one."))
		})

		It("should return error when ACPI volume is not a secret", func() {
			volumeName := "config-volume"
			vmi := libvmi.New(
				withACPISlic(volumeName),
				withConfigMapVolume(volumeName),
			)
			var domain api.Domain

			err := compute.NewOSDomainConfigurator(false, nil).Configure(vmi, &domain)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Firmware's volume type is unsupported for slic"))
		})
	})
})

func withMachineType(machineType string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Machine = &v1.Machine{
			Type: machineType,
		}
	}
}

func withStartStrategy(strategy v1.StartStrategy) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.StartStrategy = &strategy
	}
}

func withBIOSUseSerial(useSerial bool) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.BIOS = &v1.BIOS{
			UseSerial: pointer.P(useSerial),
		}
	}
}

func withKernelArgs(args string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.KernelBoot = &v1.KernelBoot{
			KernelArgs: args,
		}
	}
}

func withEFIBootloader(secureBoot bool) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{
			SecureBoot: pointer.P(secureBoot),
		}
	}
}

func withACPISlic(volumeName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.ACPI = &v1.ACPI{
			SlicNameRef: volumeName,
		}
	}
}

func withACPIMsdm(volumeName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.ACPI = &v1.ACPI{
			MsdmNameRef: volumeName,
		}
	}
}

func withACPISlicAndMsdm(slicVolume, msdmVolume string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.ACPI = &v1.ACPI{
			SlicNameRef: slicVolume,
			MsdmNameRef: msdmVolume,
		}
	}
}

func withEmptyACPI() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.ACPI = &v1.ACPI{}
	}
}

func withSecretVolume(volumeName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: volumeName,
				},
			},
		})
	}
}

func withConfigMapVolume(volumeName string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name:         volumeName,
			VolumeSource: v1.VolumeSource{},
		})
	}
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
