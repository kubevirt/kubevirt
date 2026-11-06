/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You obtain a copy of the License at
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

package input

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type DeviceConfigurator interface {
	ConfigureInputDevices(vmi *v1.VirtualMachineInstance, domain *api.Domain)
}

func NewDeviceConfigurator(architecture string) DeviceConfigurator {
	switch architecture {
	case "arm64":
		return arm64InputDevicesConfigurator{}
	case "s390x":
		return s390xInputDevicesConfigurator{}
	default:
		return nil
	}
}

type arm64InputDevicesConfigurator struct{}

func (arm64InputDevicesConfigurator) ConfigureInputDevices(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if !hasTabletDevice(vmi) {
		domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
			api.Input{
				Bus:  "usb",
				Type: "tablet",
			},
		)
	}
	domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
		api.Input{
			Bus:  "usb",
			Type: "keyboard",
		},
	)
}

type s390xInputDevicesConfigurator struct{}

func (s390xInputDevicesConfigurator) ConfigureInputDevices(_ *v1.VirtualMachineInstance, domain *api.Domain) {
	domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
		api.Input{
			Bus:  "virtio",
			Type: "keyboard",
		},
	)
}

func hasTabletDevice(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.Inputs != nil {
		for _, device := range vmi.Spec.Domain.Devices.Inputs {
			if device.Type == "tablet" {
				return true
			}
		}
	}
	return false
}
