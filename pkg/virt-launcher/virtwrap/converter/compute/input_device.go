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
		for i := range vmi.Spec.Domain.Devices.Inputs {
			inputDevice := api.Input{}
			err := convert_v1_Input_To_api_InputDevice(&vmi.Spec.Domain.Devices.Inputs[i], &inputDevice)
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

func convert_v1_Input_To_api_InputDevice(input *v1.Input, inputDevice *api.Input) error {
	if input.Bus != v1.InputBusVirtio && input.Bus != v1.InputBusUSB && input.Bus != "" {
		return fmt.Errorf("input contains unsupported bus %s", input.Bus)
	}

	if input.Bus != v1.InputBusVirtio && input.Bus != v1.InputBusUSB {
		input.Bus = v1.InputBusUSB
	}

	if input.Type != v1.InputTypeTablet {
		return fmt.Errorf("input contains unsupported type %s", input.Type)
	}

	inputDevice.Bus = input.Bus
	inputDevice.Type = input.Type
	inputDevice.Alias = api.NewUserDefinedAlias(input.Name)

	if input.Bus == v1.InputBusVirtio {
		inputDevice.Model = v1.VirtIO
	}
	return nil
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
