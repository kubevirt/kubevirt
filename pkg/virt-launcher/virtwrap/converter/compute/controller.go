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
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type scsiConfigurator interface {
	ScsiController(model string, driver *api.ControllerDriver) api.Controller
}

type ControllerDomainConfigurator struct {
	architecture          string
	scsiConfigurator      scsiConfigurator
	useVirtioTransitional bool
	useLaunchSecuritySEV  bool
	useLaunchSecurityPV   bool
}

type ControllerOption func(*ControllerDomainConfigurator)

func NewControllerDomainConfigurator(opts ...ControllerOption) ControllerDomainConfigurator {
	c := ControllerDomainConfigurator{}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

func WithArchitecture(architecture string) ControllerOption {
	return func(c *ControllerDomainConfigurator) {
		c.architecture = architecture
	}
}

func WithSCSIConfigurator(scsiConfigurator scsiConfigurator) ControllerOption {
	return func(c *ControllerDomainConfigurator) {
		c.scsiConfigurator = scsiConfigurator
	}
}

func WithUseVirtioTransitional(useVirtioTransitional bool) ControllerOption {
	return func(c *ControllerDomainConfigurator) {
		c.useVirtioTransitional = useVirtioTransitional
	}
}

func WithUseLaunchSecuritySEV(useLaunchSecuritySEV bool) ControllerOption {
	return func(c *ControllerDomainConfigurator) {
		c.useLaunchSecuritySEV = useLaunchSecuritySEV
	}
}

func WithUseLaunchSecurityPV(useLaunchSecurityPV bool) ControllerOption {
	return func(c *ControllerDomainConfigurator) {
		c.useLaunchSecurityPV = useLaunchSecurityPV
	}
}

func (c ControllerDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	c.configureUSBController(vmi, domain)
	c.configureSCSIController(vmi, domain)
	return nil
}

func (c ControllerDomainConfigurator) configureUSBController(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	// USB controller is disabled by default
	usbController := api.Controller{
		Type:  "usb",
		Index: "0",
		Model: "none",
	}

	switch c.architecture {
	case "amd64":
		if isUSBNeeded(vmi) {
			usbController.Model = "qemu-xhci"
		}
	case "arm64":
		usbController.Model = "qemu-xhci"
	}

	domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, usbController)
}

func (c ControllerDomainConfigurator) configureSCSIController(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if needsSCSIController(vmi) {
		var controllerDriver *api.ControllerDriver
		if c.useLaunchSecuritySEV || c.useLaunchSecurityPV {
			controllerDriver = &api.ControllerDriver{
				IOMMU: "on",
			}
		}

		scsiController := c.scsiConfigurator.ScsiController(virtio.InterpretTransitionalModelType(&c.useVirtioTransitional, c.architecture), controllerDriver)
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, scsiController)
	}
}

func needsSCSIController(vmi *v1.VirtualMachineInstance) bool {
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if getBusFromDisk(disk) == v1.DiskBusSCSI {
			return true
		}
	}
	return !vmi.Spec.Domain.Devices.DisableHotplug
}

func getBusFromDisk(disk v1.Disk) v1.DiskBus {
	if disk.LUN != nil {
		return disk.LUN.Bus
	}
	if disk.Disk != nil {
		return disk.Disk.Bus
	}
	if disk.CDRom != nil {
		return disk.CDRom.Bus
	}
	return ""
}

func isUSBNeeded(vmi *v1.VirtualMachineInstance) bool {
	for _, input := range vmi.Spec.Domain.Devices.Inputs {
		if input.Bus == "usb" {
			return true
		}
	}

	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if disk.Disk != nil && disk.Disk.Bus == v1.DiskBusUSB {
			return true
		}
	}

	if vmi.Spec.Domain.Devices.ClientPassthrough != nil {
		return true
	}

	return device.USBDevicesFound(vmi.Spec.Domain.Devices.HostDevices)
}
