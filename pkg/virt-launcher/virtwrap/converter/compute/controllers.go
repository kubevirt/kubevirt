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
)

type ControllersDomainConfigurator struct {
	isUSBNeeded      bool
	scsiModel        string
	controllerDriver *api.ControllerDriver
}

type controllersOption func(*ControllersDomainConfigurator)

func NewControllersDomainConfigurator(options ...controllersOption) ControllersDomainConfigurator {
	var configurator ControllersDomainConfigurator

	for _, f := range options {
		f(&configurator)
	}

	return configurator
}

func (c ControllersDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, newUSBController(c.isUSBNeeded))

	if needsSCSIController(vmi) {
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, newSCSIController(c.scsiModel, c.controllerDriver))
	}

	return nil
}

func ControllersWithUSBNeeded(isUSBNeeded bool) controllersOption {
	return func(c *ControllersDomainConfigurator) {
		c.isUSBNeeded = isUSBNeeded
	}
}

func ControllersWithSCSIModel(scsiModel string) controllersOption {
	return func(c *ControllersDomainConfigurator) {
		c.scsiModel = scsiModel
	}
}

func ControllersWithControllerDriver(controllerDriver *api.ControllerDriver) controllersOption {
	return func(c *ControllersDomainConfigurator) {
		c.controllerDriver = controllerDriver
	}
}

func newUSBController(usbNeeded bool) api.Controller {
	usbControllerModel := "none"

	if usbNeeded {
		usbControllerModel = "qemu-xhci"
	}

	return api.Controller{
		Type:  "usb",
		Index: "0",
		Model: usbControllerModel,
	}
}

func newSCSIController(controllerModel string, controllerDriver *api.ControllerDriver) api.Controller {
	return api.Controller{
		Type:   "scsi",
		Index:  "0",
		Model:  controllerModel,
		Driver: controllerDriver,
	}
}

func needsSCSIController(vmi *v1.VirtualMachineInstance) bool {
	if !vmi.Spec.Domain.Devices.DisableHotplug {
		return true
	}

	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if getBusFromDisk(disk) == v1.DiskBusSCSI {
			return true
		}
	}

	return false
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
