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
 * Copyright 2022 Intel Corporation.
 *
 */

package converter

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	openapiClient "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/openapi/cloud-hypervisor/client"
)

func ConvertVirtualMachineInstanceToVmConfig(vmi *v1.VirtualMachineInstance, vmConfig *openapiClient.VmConfig) (err error) {
	if err := convertVirtualMachineInstanceSpecToVmConfig(vmi, vmConfig); err != nil {
		return err
	}

	if err := convertVirtualMachineInstanceStatusToVmConfig(vmi, vmConfig); err != nil {
		return err
	}

	return nil
}

func convertVirtualMachineInstanceSpecToVmConfig(vmi *v1.VirtualMachineInstance, vmConfig *openapiClient.VmConfig) (err error) {
	if err := convertDomainSpecToVmConfig(vmi, vmConfig); err != nil {
		return err
	}

	if err := convertVolumesToVmConfig(vmi, vmConfig); err != nil {
		return err
	}

	if err := convertNetworksToVmConfig(vmi, vmConfig); err != nil {
		return err
	}

	if err := convertAccessCredentialsToVmConfig(vmi, vmConfig); err != nil {
		return err
	}

	return nil
}

func convertVirtualMachineInstanceStatusToVmConfig(vmi *v1.VirtualMachineInstance, vmConfig *openapiClient.VmConfig) (err error) {
	return nil
}

func convertDomainSpecToVmConfig(vmi *v1.VirtualMachineInstance, vmConfig *openapiClient.VmConfig) (err error) {
	domainSpec := vmi.Spec.Domain
	resources := domainSpec.Resources
	cpu := domainSpec.CPU
	memory := domainSpec.Memory
	firmware := domainSpec.Firmware
	features := domainSpec.Features
	devices := domainSpec.Devices

	// CPUs
	vmConfig.SetCpus(openapiClient.CpusConfig{})
	if requestCpus, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
		vmConfig.Cpus.SetBootVcpus(int32(requestCpus.Value()))
		vmConfig.Cpus.SetMaxVcpus(vmConfig.Cpus.BootVcpus)
	}
	if limitCpus, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
		vmConfig.Cpus.SetMaxVcpus(int32(limitCpus.Value()))
	}
	if cpu != nil {
		// Topology
		topology := openapiClient.NewCpuTopology()
		topology.SetCoresPerDie(int32(cpu.Cores))
		topology.SetPackages(int32(cpu.Sockets))
		topology.SetThreadsPerCore(int32(cpu.Threads))
		topology.SetDiesPerPackage(1)
		vmConfig.Cpus.SetTopology(*topology)

		bootVcpus := int32(cpu.Cores * cpu.Sockets * cpu.Threads)
		vmConfig.Cpus.SetBootVcpus(bootVcpus)
		if bootVcpus > vmConfig.Cpus.MaxVcpus {
			vmConfig.Cpus.SetMaxVcpus(bootVcpus)
		}

		// Features
		features := openapiClient.NewCpuFeatures()
		for _, feature := range cpu.Features {
			if feature.Name == "amx" {
				features.SetAmx(true)
				break
			}
		}
		vmConfig.Cpus.Features = features
	}

	// Memory
	vmConfig.SetMemory(openapiClient.MemoryConfig{})
	if requestRam, ok := resources.Requests[k8sv1.ResourceMemory]; ok {
		vmConfig.Memory.Size = requestRam.Value()
	}
	if limitRam, ok := resources.Limits[k8sv1.ResourceMemory]; ok {
		maxRam := limitRam.Value()
		if maxRam > vmConfig.Memory.Size {
			hotplugSize := maxRam - vmConfig.Memory.Size
			vmConfig.Memory.SetHotplugMethod("virtio-mem")
			vmConfig.Memory.SetHotplugSize(hotplugSize)
		}
	}
	if memory != nil && memory.Hugepages != nil {
		vmConfig.Memory.SetHugepages(true)
		vmConfig.Memory.SetHugepageSize(2 << 20)
	}
	if devices.Filesystems != nil {
		for _, fs := range devices.Filesystems {
			if fs.Virtiofs != nil {
				vmConfig.Memory.SetShared(true)
				break
			}
		}
	}

	// Firmware
	if firmware != nil {
		// SMBIOS serial number
		if firmware.Serial != "" {
			vmConfig.SetPlatform(openapiClient.PlatformConfig{})
			vmConfig.Platform.SetSerialNumber(firmware.Serial)
		}

		// By default, the provided VmConfig has been set to load the
		// EFI firmware since the value isn't provided by the VMI
		// specification.
		//
		// In case the VMI specification doesn't expect the VM to boot
		// from an EFI image, we fallback to the direct kernel boot or
		// initramfs method.
		if firmware.KernelBoot != nil && firmware.KernelBoot.Container != nil {
			log.Log.Object(vmi).Infof("Kernel boot defined for VMI")

			kernelBoot := firmware.KernelBoot

			// Kernel boot parameters
			vmConfig.SetCmdline(openapiClient.CmdLineConfig{
				Args: kernelBoot.KernelArgs,
			})

			container := kernelBoot.Container

			// Kernel
			if container.KernelPath != "" {
				kernelPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(container.KernelPath)
				vmConfig.Kernel.SetPath(kernelPath)
			}
			// Initramfs
			if container.InitrdPath != "" {
				initrdPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(container.InitrdPath)
				vmConfig.SetInitramfs(openapiClient.InitramfsConfig{
					Path: initrdPath,
				})
			}
		}
	}

	// Features
	if features != nil &&
		features.Hyperv != nil &&
		features.Hyperv.SyNIC != nil &&
		features.Hyperv.SyNIC.Enabled != nil {
		//vmConfig.Cpus.SetKvmHyperv(*features.Hyperv.SyNIC.Enabled)
	}

	/// Devices
	if devices.UseVirtioTransitional != nil && *devices.UseVirtioTransitional {
		return fmt.Errorf("Transitional virtio is not supported")
	}

	// Disks
	blockMultiQueue := (devices.BlockMultiQueue != nil) && (*devices.BlockMultiQueue)
	var diskConfigs []openapiClient.DiskConfig
	for _, disk := range devices.Disks {
		if disk.LUN != nil || disk.CDRom != nil {
			continue
		}

		diskConfig := openapiClient.DiskConfig{}
		diskConfig.SetId(disk.Name)

		if blockMultiQueue {
			diskConfig.SetNumQueues(vmConfig.Cpus.BootVcpus)
		}
		diskConfigs = append(diskConfigs, diskConfig)
	}
	vmConfig.SetDisks(diskConfigs)

	// Watchdog
	if devices.Watchdog != nil {
		vmConfig.SetWatchdog(true)
	}

	// Network interfaces
	netMultiQueue := (devices.NetworkInterfaceMultiQueue != nil) && (*devices.NetworkInterfaceMultiQueue)
	var netConfigs []openapiClient.NetConfig
	for _, iface := range devices.Interfaces {
		if iface.Bridge == nil && iface.Masquerade == nil {
			continue
		}

		netConfig := openapiClient.NetConfig{}
		netConfig.SetId(iface.Name)

		if netMultiQueue {
			netConfig.SetNumQueues(vmConfig.Cpus.BootVcpus * 2)
		}
		netConfigs = append(netConfigs, netConfig)
	}
	vmConfig.SetNet(netConfigs)

	// Console
	vmConfig.SetConsole(openapiClient.ConsoleConfig{
		Mode: "Off",
	})
	if devices.AutoattachSerialConsole == nil || *devices.AutoattachSerialConsole {
		vmConfig.SetSerial(openapiClient.ConsoleConfig{
			Mode: "Pty",
		})
	}

	// Filesystems

	// Host devices

	/// End of Devices

	return nil
}

func convertVolumesToVmConfig(vmi *v1.VirtualMachineInstance, vmConfig *openapiClient.VmConfig) (err error) {
	volumes := vmi.Spec.Volumes

	if vmConfig.Disks == nil {
		return nil
	}

	disksConfig := map[string]*openapiClient.DiskConfig{}
	for i, disk := range *vmConfig.Disks {
		disksConfig[disk.GetId()] = &(*vmConfig.Disks)[i]
	}

	for i, volume := range volumes {
		diskConfig, ok := disksConfig[volume.Name]
		if !ok {
			return fmt.Errorf("Could not find disk associated with volume '%s'", volume.Name)
		}

		if volume.ContainerDisk != nil {
			diskConfig.Path = containerdisk.GetRawDiskTargetPathFromLauncherView(i)
		} else if volume.EmptyDisk != nil {
			diskConfig.Path = emptydisk.NewEmptyRawDiskCreator().FilePathForVolumeName(volume.Name)
		} else if volume.CloudInitNoCloud != nil {
			diskConfig.Path = cloudinit.GetIsoFilePath(cloudinit.DataSourceNoCloud, vmi.Name, vmi.Namespace)
		} else if volume.CloudInitConfigDrive != nil {
			diskConfig.Path = cloudinit.GetIsoFilePath(cloudinit.DataSourceConfigDrive, vmi.Name, vmi.Namespace)
		}
	}

	return nil
}

func convertNetworksToVmConfig(vmi *v1.VirtualMachineInstance, vmConfig *openapiClient.VmConfig) (err error) {
	return nil
}

func convertAccessCredentialsToVmConfig(vmi *v1.VirtualMachineInstance, vmConfig *openapiClient.VmConfig) (err error) {
	return nil
}
