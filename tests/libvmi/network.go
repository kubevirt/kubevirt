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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libvmi

import (
	kvirtv1 "kubevirt.io/client-go/api/v1"
)

// WithInterface adds a Domain Device Interface.
func WithInterface(iface kvirtv1.Interface) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Interfaces = append(
			vmi.Spec.Domain.Devices.Interfaces, iface,
		)
	}
}

// WithNetwork adds a network object.
func WithNetwork(network *kvirtv1.Network) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Networks = append(vmi.Spec.Networks, *network)
	}
}

// InterfaceDeviceWithMasqueradeBinding returns an Interface named "default" with masquerade binding.
func InterfaceDeviceWithMasqueradeBinding() kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: "default",
		InterfaceBindingMethod: kvirtv1.InterfaceBindingMethod{
			Masquerade: &kvirtv1.InterfaceMasquerade{},
		},
	}
}

// InterfaceDeviceWithBridgeBinding returns an Interface named "default" with bridge binding.
func InterfaceDeviceWithBridgeBinding() kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: "default",
		InterfaceBindingMethod: kvirtv1.InterfaceBindingMethod{
			Bridge: &kvirtv1.InterfaceBridge{},
		},
	}
}
