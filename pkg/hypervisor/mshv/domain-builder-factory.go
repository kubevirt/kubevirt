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

package mshv

import (
	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

func MakeDomainBuilder(vmi *v1.VirtualMachineInstance, c *types.ConverterContext) *types.DomainBuilder {
	architecture := c.Architecture.GetArchitecture()
	virtioModel := virtio.InterpretTransitionalModelType(
		vmi.Spec.Domain.Devices.UseVirtioTransitional,
		architecture,
	)
	scsiControllerModel := c.Architecture.SCSIControllerModel(virtioModel)

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

	builder := types.NewDomainBuilder(
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
		NewMshvDomainConfigurator(c.AllowEmulation, c.HypervisorDeviceAvailable),
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
		),
		compute.NewQemuCmdDomainConfigurator(c.Architecture.ShouldVerboseLogsBeEnabled()),
		compute.NewCPUDomainConfigurator(c.Architecture.SupportCPUHotplug(), c.Architecture.RequiresMPXCPUValidation()),
		compute.NewIOThreadsDomainConfigurator(uint(ioThreadCount)),
		compute.MemoryConfigurator{},
	)

	return &builder
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

func convertEFIConfiguration(input *types.EFIConfiguration) *compute.EFIConfiguration {
	if input == nil {
		return nil
	}

	return &compute.EFIConfiguration{
		EFICode:      input.EFICode,
		EFIVars:      input.EFIVars,
		SecureLoader: input.SecureLoader,
	}
}
