/* This file is part of the KubeVirt project
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

package converter

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/kvm"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/mshv"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

func Convert_v1_VirtualMachineInstance_To_api_Domain(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *convertertypes.ConverterContext) (err error) {

	precond.MustNotBeNil(vmi)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	var controllerDriver *api.ControllerDriver
	if c.UseLaunchSecuritySEV || c.UseLaunchSecurityPV {
		controllerDriver = &api.ControllerDriver{
			IOMMU: "on",
		}
	}

	hasIOThreads := iothreads.HasIOThreads(vmi)
	var ioThreadCount, autoThreads int
	if hasIOThreads {
		ioThreadCount, autoThreads = iothreads.GetIOThreadsCountType(vmi)
	}

	architecture := c.Architecture.GetArchitecture()
	virtioModel := virtio.InterpretTransitionalModelType(
		vmi.Spec.Domain.Devices.UseVirtioTransitional,
		architecture,
	)
	scsiControllerModel := c.Architecture.SCSIControllerModel(virtioModel)

	configurators := []convertertypes.Configurator{
		metadata.DomainConfigurator{},
		network.NewDomainConfigurator(
			network.WithDomainAttachmentByInterfaceName(c.DomainAttachmentByInterfaceName),
			network.WithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			network.WithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			network.WithROMTuningSupport(c.Architecture.IsROMTuningSupported()),
			network.WithVirtioModel(virtioModel),
		),
		compute.TPMDomainConfigurator{},
		compute.VSOCKDomainConfigurator{},
		compute.NewLaunchSecurityDomainConfigurator(architecture),
		compute.ChannelsDomainConfigurator{},
		compute.ClockDomainConfigurator{},
		compute.NewRNGDomainConfigurator(
			compute.RNGWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.RNGWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			compute.RNGWithVirtioModel(virtioModel),
		),
		compute.NewInputDeviceDomainConfigurator(architecture),
		compute.NewBalloonDomainConfigurator(
			compute.BalloonWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.BalloonWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			compute.BalloonWithFreePageReporting(c.FreePageReporting),
			compute.BalloonWithMemBalloonStatsPeriod(c.MemBalloonStatsPeriod),
			compute.BalloonWithVirtioModel(virtioModel),
		),
		compute.NewGraphicsDomainConfigurator(architecture, c.BochsForEFIGuests),
		compute.SoundDomainConfigurator{},
		compute.NewHostDeviceDomainConfigurator(
			c.GenericHostDevices,
			c.GPUHostDevices,
			c.SRIOVDevices,
		),
		compute.NewWatchdogDomainConfigurator(architecture),
		compute.NewConsoleDomainConfigurator(c.SerialConsoleLog),
		compute.PanicDevicesDomainConfigurator{},
		compute.NewHypervisorFeaturesDomainConfigurator(c.Architecture.HasVMPort(), c.UseLaunchSecurityTDX),
		compute.NewSysInfoDomainConfigurator(convertCmdv1SMBIOSToComputeSMBIOS(c.SMBios)),
		compute.NewOSDomainConfigurator(c.Architecture.IsSMBiosNeeded(), convertEFIConfiguration(c.EFIConfiguration)),
		storage.NewVirtiofsConfigurator(),
		compute.UsbRedirectDeviceDomainConfigurator{},
		compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(c.Architecture.IsUSBNeeded(vmi)),
			compute.ControllersWithSCSIModel(scsiControllerModel),
			compute.ControllersWithSCSIIOThreads(uint(autoThreads)),
			compute.ControllersWithControllerDriver(controllerDriver),
			compute.ControllersWithSupportPCIHole64Disabling(c.Architecture.SupportPCIHole64Disabling()),
			compute.ControllersWithVirtioSerialModel(virtioModel),
		),
		compute.NewQemuCmdDomainConfigurator(c.Architecture.ShouldVerboseLogsBeEnabled()),
		compute.NewCPUDomainConfigurator(c.Architecture.SupportCPUHotplug(), c.Architecture.RequiresMPXCPUValidation()),
		compute.NewIOThreadsDomainConfigurator(uint(ioThreadCount)),
		compute.MemoryConfigurator{},
		compute.RebootPolicyDomainConfigurator{},
	}

	switch c.HypervisorName {
	case v1.HyperVDirectHypervisorName:
		configurators = append(configurators, mshv.NewMshvDomainConfigurator(c.AllowEmulation, c.HypervisorDeviceAvailable))
	default:
		configurators = append(configurators, kvm.NewKvmDomainConfigurator(c.AllowEmulation, c.HypervisorDeviceAvailable))
	}

	builder := convertertypes.NewDomainBuilder(configurators...)
	if err := builder.Build(vmi, domain); err != nil {
		return err
	}

	var isMemfdRequired = false
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		domain.Spec.MemoryBacking = &api.MemoryBacking{
			HugePages: &api.HugePages{},
		}
		if val := vmi.Annotations[v1.MemfdMemoryBackend]; val != "false" {
			isMemfdRequired = true
		}
	}
	// virtiofs require shared access
	if util.IsVMIVirtiofsEnabled(vmi) || netvmispec.HasPasstBinding(vmi) {
		if domain.Spec.MemoryBacking == nil {
			domain.Spec.MemoryBacking = &api.MemoryBacking{}
		}
		domain.Spec.MemoryBacking.Access = &api.MemoryBackingAccess{
			Mode: "shared",
		}
		isMemfdRequired = true
	}

	if isMemfdRequired {
		// Set memfd as memory backend to solve SELinux restrictions
		// See the issue: https://github.com/kubevirt/kubevirt/issues/3781
		domain.Spec.MemoryBacking.Source = &api.MemoryBackingSource{Type: "memfd"}

		// NUMA is required in order to use memfd
		if domain.Spec.CPU.NUMA == nil {
			domain.Spec.CPU.NUMA = &api.NUMA{
				Cells: []api.NUMACell{
					{
						ID:     "0",
						CPUs:   fmt.Sprintf("0-%d", domain.Spec.VCPU.CPUs-1),
						Memory: uint64(vcpu.GetVirtualMemory(vmi).Value() / int64(1024)),
						Unit:   "KiB",
					},
				},
			}
		}
	}

	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	var numBlkQueues *uint
	virtioBlkMQRequested := (vmi.Spec.Domain.Devices.BlockMultiQueue != nil) && (*vmi.Spec.Domain.Devices.BlockMultiQueue)
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	vcpus := uint(cpuCount)
	if vcpus == 0 {
		vcpus = uint(1)
	}

	if virtioBlkMQRequested {
		numBlkQueues = &vcpus
	}

	volumeStatusMap := make(map[string]v1.VolumeStatus)
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		volumeStatusMap[volumeStatus.Name] = volumeStatus
	}

	prefixMap := storage.NewDeviceNamer(vmi.Status.VolumeStatus, vmi.Spec.Domain.Devices.Disks)
	currentAutoThread := uint(1)
	currentDedicatedThread := uint(autoThreads + 1)
	supplementalIOThreads := iothreads.SupplementalPoolThreadCount(vmi)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := api.Disk{}
		emptyCDRom := false

		err := storage.ConvertV1DiskToAPIDisk(c, &disk, &newDisk, prefixMap, numBlkQueues, volumeStatusMap)
		if err != nil {
			return err
		}
		volume := volumes[disk.Name]
		if volume == nil {
			if disk.CDRom == nil {
				return fmt.Errorf("no matching volume with name %s found", disk.Name)
			}
			emptyCDRom = true
		}

		hpStatus, hpOk := c.HotplugVolumes[disk.Name]
		switch {
		case emptyCDRom:
			err = storage.Convert_v1_Missing_Volume_To_api_Disk(&newDisk)
		case hpOk:
			err = storage.Convert_v1_Hotplug_Volume_To_api_Disk(volume, &newDisk, c)
		default:
			err = storage.Convert_v1_Volume_To_api_Disk(volume, &newDisk, c, volumeIndices[disk.Name])
		}

		if err != nil {
			return err
		}

		if err := storage.ConvertV1BlockSizeToAPIBlockIO(&disk, &newDisk, c.Architecture.GetArchitecture()); err != nil {
			return err
		}

		_, isPermVolume := c.PermanentVolumes[disk.Name]
		// if len(c.PermanentVolumes) == 0, it means the vmi is not ready yet, add all disks
		permReady := isPermVolume || len(c.PermanentVolumes) == 0
		hotplugReady := hpOk && (hpStatus.Phase == v1.HotplugVolumeMounted || hpStatus.Phase == v1.VolumeReady)

		if permReady || hotplugReady || emptyCDRom {
			domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
		}
		if err := storage.SetErrorPolicy(&disk, &newDisk); err != nil {
			return err
		}
		if hasIOThreads {
			currentDedicatedThread, currentAutoThread = storage.AssignDiskIOThread(&disk, &newDisk, supplementalIOThreads, autoThreads, currentDedicatedThread, currentAutoThread)
		}
	}

	if vmi.Spec.Domain.CPU != nil {
		// Adjust guest vcpu config. Currently will handle vCPUs to pCPUs pinning
		if vmi.IsCPUDedicated() {
			err = vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, c.Topology, c.CPUSet, hasIOThreads)
			if err != nil {
				return err
			}

			if c.PCINUMAAwareTopologyEnabled {
				if c.Architecture.SupportPCIePlacement() {
					if err := PlacePCIDevicesWithNUMAAlignment(&domain.Spec); err != nil {
						log.Log.Reason(err).Warningf("Failed to process PCIe NUMA-aware topology, falling back to default placement")
					}
				} else {
					log.Log.Infof("Skipping PCIe NUMA alignment: architecture %s does not support PCIe placement", c.Architecture.GetArchitecture())
				}
			}
		}
	}

	if val := vmi.Annotations[v1.PlacePCIDevicesOnRootComplex]; val == "true" {
		if c.Architecture.SupportPCIePlacement() {
			if err := PlacePCIDevicesOnRootComplex(&domain.Spec); err != nil {
				return err
			}
		} else {
			log.Log.Infof("Skipping PCIe root complex placement: architecture %s does not support PCIe placement", c.Architecture.GetArchitecture())
		}
	}

	return nil
}

func GracePeriodSeconds(vmi *v1.VirtualMachineInstance) int64 {
	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}
	return gracePeriodSeconds
}

func convertCmdv1SMBIOSToComputeSMBIOS(input *cmdv1.SMBios) *compute.SMBIOS {
	if input == nil {
		return nil
	}

	return &compute.SMBIOS{
		Manufacturer: input.Manufacturer,
		Product:      input.Product,
		Version:      input.Version,
		SKU:          input.Sku,
		Family:       input.Family,
	}
}

func convertEFIConfiguration(input *convertertypes.EFIConfiguration) *compute.EFIConfiguration {
	if input == nil {
		return nil
	}

	return &compute.EFIConfiguration{
		EFICode:      input.EFICode,
		EFIVars:      input.EFIVars,
		SecureLoader: input.SecureLoader,
	}
}
