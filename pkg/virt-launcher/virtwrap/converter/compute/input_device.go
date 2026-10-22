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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute/input"
)

type InputDeviceDomainConfigurator struct {
	additionalInputDevices input.DeviceConfigurator
}

func NewInputDeviceDomainConfigurator(additionalInputDevices input.DeviceConfigurator) InputDeviceDomainConfigurator {
	return InputDeviceDomainConfigurator{
		additionalInputDevices: additionalInputDevices,
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
		if i.additionalInputDevices != nil {
			i.additionalInputDevices.ConfigureInputDevices(vmi, domain)
		}
	}

	return nil
}

func apiInputDeviceFromV1InputDevice(inputDevice v1.Input) (api.Input, error) {
	var bus v1.InputBus

	switch inputDevice.Bus {
	case v1.InputBusVirtio, v1.InputBusUSB:
		bus = inputDevice.Bus
	case "":
		bus = v1.InputBusUSB
	default:
		return api.Input{}, fmt.Errorf("input contains unsupported bus %s", inputDevice.Bus)
	}

	if inputDevice.Type != v1.InputTypeTablet {
		return api.Input{}, fmt.Errorf("input contains unsupported type %s", inputDevice.Type)
	}

	domainInputDevice := api.Input{
		Bus:   bus,
		Type:  inputDevice.Type,
		Alias: api.NewUserDefinedAlias(inputDevice.Name),
	}

	if bus == v1.InputBusVirtio {
		domainInputDevice.Model = v1.VirtIO
	}

	return domainInputDevice, nil
}
