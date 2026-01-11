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
	isUSBNeeded bool
}

func NewControllersDomainConfigurator(isUSBNeeded bool) ControllersDomainConfigurator {
	return ControllersDomainConfigurator{
		isUSBNeeded: isUSBNeeded,
	}
}

func (c ControllersDomainConfigurator) Configure(_ *v1.VirtualMachineInstance, domain *api.Domain) error {
	domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, newUSBController(c.isUSBNeeded))
	return nil
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
