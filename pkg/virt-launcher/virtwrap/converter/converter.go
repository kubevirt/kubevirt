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

package converter

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
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
		compute.VSOCKDomainConfigurator{ProcPath: c.VSOCKProcPath},
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
		compute.NewGraphicsDomainConfigurator(architecture, c.BochsForEFIGuests, c.AllowCrossArchEmulation),
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
		storage.NewDiskConfigurator(
			storage.DiskWithArchitecture(architecture),
			storage.DiskWithVirtioModel(virtioModel),
			storage.DiskWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			storage.DiskWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			storage.DiskWithVolumesDiscardIgnore(c.VolumesDiscardIgnore),
			storage.DiskWithHotplugVolumes(c.HotplugVolumes),
			storage.DiskWithPermanentVolumes(c.PermanentVolumes),
			storage.DiskWithIsBlockPVC(c.IsBlockPVC),
			storage.DiskWithIsBlockDV(c.IsBlockDV),
			storage.DiskWithApplyCBT(c.ApplyCBT),
			storage.DiskWithDisksInfo(c.DisksInfo),
			storage.DiskWithEphemeralDiskCreator(c.EphemeraldiskCreator),
		),
		compute.UsbRedirectDeviceDomainConfigurator{},
		compute.NewControllersDomainConfigurator(
			compute.ControllersWithUSBNeeded(c.Architecture.IsUSBNeeded(vmi)),
			compute.ControllersWithSCSIModel(scsiControllerModel),
			compute.ControllersWithSCSIIOThreads(uint(autoThreads)),
			compute.ControllersWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.ControllersWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			compute.ControllersWithSupportPCIHole64Disabling(c.Architecture.SupportPCIHole64Disabling()),
			compute.ControllersWithVirtioSerialModel(virtioModel),
		),
		compute.NewQemuCmdDomainConfigurator(c.Architecture.ShouldVerboseLogsBeEnabled()),
		compute.NewCPUDomainConfigurator(
			compute.CPUWithHotplugSupported(c.Architecture.SupportCPUHotplug()),
			compute.CPUWithMPXCPUValidation(c.Architecture.RequiresMPXCPUValidation()),
			compute.CPUWithCrossArchEmulation(c.AllowCrossArchEmulation),
			compute.CPUWithMemfdSupported(c.Architecture.IsMemfdSupported()),
		),
		compute.NewIOThreadsDomainConfigurator(uint(ioThreadCount)),
		compute.MemoryConfigurator{},
		compute.NewMemoryBackingConfigurator(c.Architecture.IsMemfdSupported()),
		compute.RebootPolicyDomainConfigurator{},
		compute.NewIOMMUFDConfigurator(c.IOMMUFDEnabled),
	}

	switch c.HypervisorName {
	case v1.HyperVDirectHypervisorName:
		configurators = append(configurators, mshv.NewMshvDomainConfigurator(c.AllowEmulation, c.HypervisorDeviceAvailable))
	default:
		if c.AllowCrossArchEmulation {
			configurators = append(configurators, kvm.NewKvmDomainConfiguratorWithCrossArch(c.AllowEmulation, c.HypervisorDeviceAvailable, c.AllowCrossArchEmulation, c.HostArchitecture))
		} else {
			configurators = append(configurators, kvm.NewKvmDomainConfigurator(c.AllowEmulation, c.HypervisorDeviceAvailable))
		}
	}

	builder := convertertypes.NewDomainBuilder(configurators...)
	if err := builder.Build(vmi, domain); err != nil {
		return err
	}

	if vmi.Spec.Domain.CPU != nil && vmi.IsCPUDedicated() {
		// Adjust guest vcpu config. Currently will handle vCPUs to pCPUs pinning
		if err := vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, c.Topology, c.CPUSet); err != nil {
			return err
		}

		if graceIOVirtualizationRequested(c) {
			return configureGraceIOVirtualization(&domain.Spec, c.GraceHostDeviceAliases, c.IOMMUFDEnabled)
		}

		if c.PCINUMAAwareTopologyEnabled {
			if c.Architecture.SupportPCIePlacement() {
				// Strict PCI placement is tied to the VMI API request. At this
				// point the request has already been converted into domain NUMA
				// state, but deriving strictness from the XML shape would make
				// other NUMATune users strict by accident.
				strictPCIPlacement := vmi.Spec.Domain.CPU.NUMA != nil &&
					vmi.Spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil
				var opts []PCIPlacementOption
				if strictPCIPlacement {
					opts = append(opts, WithStrictPCINUMAPlacement())
				}
				if err := PlacePCIDevicesWithNUMAAlignment(&domain.Spec, opts...); err != nil {
					if strictPCIPlacement {
						return fmt.Errorf("failed to process strict PCIe NUMA-aware topology: %w", err)
					}
					log.Log.Reason(err).Warningf("Failed to process PCIe NUMA-aware topology, falling back to default placement")
				}
			} else {
				log.Log.Infof("Skipping PCIe NUMA alignment: architecture %s does not support PCIe placement", c.Architecture.GetArchitecture())
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

func GetVolumeNameByDisk(disk api.Disk) string {
	return disk.Alias.GetName()
}

// GetVolumeNameByTarget returns the volume name associated to the device target in the domain (e.g vda)
func GetVolumeNameByTarget(domain *api.Domain, target string) string {
	for _, d := range domain.Spec.Devices.Disks {
		if d.Target.Device == target {
			return GetVolumeNameByDisk(d)
		}
	}
	return ""
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
		EFICode:                   input.EFICode,
		EFIVars:                   input.EFIVars,
		SecureLoader:              input.SecureLoader,
		UsesFirmwareAutoSelection: input.UsesFirmwareAutoSelection,
	}
}
