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

package compute

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type InputDeviceDomainConfigurator struct {
	architecture string
}

func NewInputDeviceDomainConfigurator(architecture string) InputDeviceDomainConfigurator {
	return InputDeviceDomainConfigurator{
		architecture: architecture,
	}
}

func (i InputDeviceDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Devices.Inputs != nil {
		inputDevices := make([]api.Input, 0)
		for _, specInput := range vmi.Spec.Domain.Devices.Inputs {
			inputDevice, err := apiInputDeviceFromV1InputDevice(specInput)
			if err != nil {
				return err
			}
			inputDevices = append(inputDevices, inputDevice)
		}
		domain.Spec.Devices.Inputs = inputDevices
	}

	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == nil || *vmi.Spec.Domain.Devices.AutoattachGraphicsDevice {
		if err := i.addArchitectureSpecificInputDevices(vmi, domain); err != nil {
			return err
		}
	}

	return nil
}

func apiInputDeviceFromV1InputDevice(input v1.Input) (api.Input, error) {
	var bus v1.InputBus

	switch input.Bus {
	case v1.InputBusVirtio, v1.InputBusUSB:
		bus = input.Bus
	case "":
		bus = v1.InputBusUSB
	default:
		return api.Input{}, fmt.Errorf("input contains unsupported bus %s", input.Bus)
	}

	if !v1.IsValidInputType(input.Type) {
		return api.Input{}, fmt.Errorf("input contains unsupported type %s", input.Type)
	}

	inputDevice := api.Input{
		Bus:   bus,
		Type:  input.Type,
		Alias: api.NewUserDefinedAlias(input.Name),
	}

	if bus == v1.InputBusVirtio {
		inputDevice.Model = v1.VirtIO
	}

	return inputDevice, nil
}

func (i InputDeviceDomainConfigurator) addArchitectureSpecificInputDevices(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	switch i.architecture {
	case "amd64":
		// No architecture-specific input devices required
	case "arm64":
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
	case "s390x":
		domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
			api.Input{
				Bus:  "virtio",
				Type: "keyboard",
			},
		)
	}
	return nil
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
