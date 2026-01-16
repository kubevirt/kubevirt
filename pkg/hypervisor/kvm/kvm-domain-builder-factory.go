package kvm

import (
	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/builder"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

type KvmDomainBuilderFactory struct{}

func (f KvmDomainBuilderFactory) MakeDomainBuilder(vmi *v1.VirtualMachineInstance, c *builder.ConverterContext) *builder.DomainBuilder {
	architecture := c.Architecture.GetArchitecture()
	virtioModel := virtio.InterpretTransitionalModelType(
		vmi.Spec.Domain.Devices.UseVirtioTransitional,
		architecture,
	)

	builder := builder.NewDomainBuilder(
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
		compute.NewHypervisorDomainConfigurator(c.AllowEmulation, c.HypervisorAvailable),
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
		compute.NewControllersDomainConfigurator(c.Architecture.IsUSBNeeded(vmi)),
		compute.NewQemuCmdDomainConfigurator(c.Architecture.ShouldVerboseLogsBeEnabled()),
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

func convertEFIConfiguration(input *builder.EFIConfiguration) *compute.EFIConfiguration {
	if input == nil {
		return nil
	}

	return &compute.EFIConfiguration{
		EFICode:      input.EFICode,
		EFIVars:      input.EFIVars,
		SecureLoader: input.SecureLoader,
	}
}
