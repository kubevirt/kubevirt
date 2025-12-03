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

type usbConfigurator interface {
	IsUSBNeeded(vmi *v1.VirtualMachineInstance) bool
}

type ControllerDomainConfigurator struct {
	usbConfigurator usbConfigurator
}

type ControllerOption func(*ControllerDomainConfigurator)

func NewControllerDomainConfigurator(opts ...ControllerOption) ControllerDomainConfigurator {
	c := ControllerDomainConfigurator{}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

func WithUSBConfigurator(usbConfigurator usbConfigurator) ControllerOption {
	return func(c *ControllerDomainConfigurator) {
		c.usbConfigurator = usbConfigurator
	}
}

func (c ControllerDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	c.configureUSBController(vmi, domain)
	return nil
}

func (c ControllerDomainConfigurator) configureUSBController(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	// USB controller is disabled by default
	usbController := api.Controller{
		Type:  "usb",
		Index: "0",
		Model: "none",
	}
	if c.usbConfigurator.IsUSBNeeded(vmi) {
		usbController.Model = "qemu-xhci"
	}
	domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, usbController)
}
