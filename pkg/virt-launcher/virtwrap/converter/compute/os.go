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

package compute

import (
	"fmt"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type EFIConfiguration struct {
	EFICode      string
	EFIVars      string
	SecureLoader bool
}

type OSDomainConfigurator struct {
	isSMBiosNeeded   bool
	efiConfiguration *EFIConfiguration
}

func NewOSDomainConfigurator(isSMBiosNeeded bool, efiConfiguration *EFIConfiguration) OSDomainConfigurator {
	return OSDomainConfigurator{
		isSMBiosNeeded:   isSMBiosNeeded,
		efiConfiguration: efiConfiguration,
	}
}

func (o OSDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if err := o.convert_v1_Firmware_To_related_apis(vmi, domain); err != nil {
		return err
	}

	o.configureSMBios(domain)
	configureMachineType(vmi, domain)
	configureBootMenu(vmi, domain)

	return nil
}

func (o OSDomainConfigurator) configureSMBios(domain *api.Domain) {
	if o.isSMBiosNeeded {
		domain.Spec.OS.SMBios = &api.SMBios{
			Mode: "sysinfo",
		}
	}
}

func configureMachineType(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if machine := vmi.Spec.Domain.Machine; machine != nil {
		domain.Spec.OS.Type.Machine = machine.Type
	}
}

func configureBootMenu(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	// set bootmenu to give time to access bios
	if vmi.ShouldStartPaused() {
		const bootMenuTimeoutMS = uint(10000)
		domain.Spec.OS.BootMenu = &api.BootMenu{
			Enable:  "yes",
			Timeout: pointer.P(bootMenuTimeoutMS),
		}
	}
}

func (o OSDomainConfigurator) convert_v1_Firmware_To_related_apis(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	firmware := vmi.Spec.Domain.Firmware
	if firmware == nil {
		return nil
	}

	if vmi.IsBootloaderEFI() {
		domain.Spec.OS.BootLoader = &api.Loader{
			Path:     o.efiConfiguration.EFICode,
			ReadOnly: "yes",
			Secure:   boolToYesNo(&o.efiConfiguration.SecureLoader, false),
		}

		if util.IsSEVSNPVMI(vmi) || util.IsTDXVMI(vmi) {
			// Use stateless firmware for the TDX/SNP VMs
			domain.Spec.OS.BootLoader.Type = "rom"
			domain.Spec.OS.NVRam = nil
		} else {
			domain.Spec.OS.BootLoader.Type = "pflash"
			domain.Spec.OS.NVRam = &api.NVRam{
				Template: o.efiConfiguration.EFIVars,
				NVRam:    filepath.Join(services.PathForNVram(vmi), vmi.Name+"_VARS.fd"),
			}
		}
	}

	if firmware.Bootloader != nil && firmware.Bootloader.BIOS != nil {
		if firmware.Bootloader.BIOS.UseSerial != nil && *firmware.Bootloader.BIOS.UseSerial {
			domain.Spec.OS.BIOS = &api.BIOS{
				UseSerial: "yes",
			}
		}
	}

	if util.HasKernelBootContainerImage(vmi) {
		kb := firmware.KernelBoot

		log.Log.Object(vmi).Infof("kernel boot defined for VMI. Converting to domain XML")
		if kb.Container.KernelPath != "" {
			kernelPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.KernelPath)
			log.Log.Object(vmi).Infof("setting kernel path for kernel boot: %s", kernelPath)
			domain.Spec.OS.Kernel = kernelPath
		}

		if kb.Container.InitrdPath != "" {
			initrdPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.InitrdPath)
			log.Log.Object(vmi).Infof("setting initrd path for kernel boot: %s", initrdPath)
			domain.Spec.OS.Initrd = initrdPath
		}

	}

	// Define custom command-line arguments even if kernel-boot container is not defined
	if firmware.KernelBoot != nil {
		log.Log.Object(vmi).Infof("setting custom kernel arguments: %s", firmware.KernelBoot.KernelArgs)
		domain.Spec.OS.KernelArgs = firmware.KernelBoot.KernelArgs
	}

	if err := convert_v1_Firmware_ACPI_To_related_apis(firmware, domain, vmi.Spec.Volumes); err != nil {
		return err
	}

	return nil
}

func convert_v1_Firmware_ACPI_To_related_apis(firmware *v1.Firmware, domain *api.Domain, volumes []v1.Volume) error {
	if firmware.ACPI == nil {
		return nil
	}

	if firmware.ACPI.SlicNameRef == "" && firmware.ACPI.MsdmNameRef == "" {
		return fmt.Errorf("No ACPI tables were set. Expecting at least one.")
	}

	if domain.Spec.OS.ACPI == nil {
		domain.Spec.OS.ACPI = &api.OSACPI{}
	}

	if val, err := createACPITable("slic", firmware.ACPI.SlicNameRef, volumes); err != nil {
		return err
	} else if val != nil {
		domain.Spec.OS.ACPI.Table = append(domain.Spec.OS.ACPI.Table, *val)
	}

	if val, err := createACPITable("msdm", firmware.ACPI.MsdmNameRef, volumes); err != nil {
		return err
	} else if val != nil {
		domain.Spec.OS.ACPI.Table = append(domain.Spec.OS.ACPI.Table, *val)
	}

	// if field was set but volume was not found, helper function will return error
	return nil
}

func createACPITable(source, volumeName string, volumes []v1.Volume) (*api.ACPITable, error) {
	if volumeName == "" {
		return nil, nil
	}

	for _, volume := range volumes {
		if volume.Name != volumeName {
			continue
		}

		if volume.Secret == nil {
			// Unsupported. This should have been blocked by webhook, so warn user.
			return nil, fmt.Errorf("Firmware's volume type is unsupported for %s", source)
		}

		// Return path to table's binary data
		sourcePath := config.GetSecretSourcePath(volumeName)
		sourcePath = filepath.Join(sourcePath, fmt.Sprintf("%s.bin", source))
		return &api.ACPITable{
			Type: source,
			Path: sourcePath,
		}, nil
	}

	return nil, fmt.Errorf("Firmware's volume for %s was not found", source)
}
