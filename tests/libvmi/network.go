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
	kvirtv1 "kubevirt.io/api/core/v1"
)

const DefaultInterfaceName = "default"

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
func InterfaceDeviceWithMasqueradeBinding(ports ...kvirtv1.Port) kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: "default",
		InterfaceBindingMethod: kvirtv1.InterfaceBindingMethod{
			Masquerade: &kvirtv1.InterfaceMasquerade{},
		},
		Ports: ports,
	}
}

// InterfaceDeviceWithBridgeBinding returns an Interface with bridge binding.
func InterfaceDeviceWithBridgeBinding(name string) kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: name,
		InterfaceBindingMethod: kvirtv1.InterfaceBindingMethod{
			Bridge: &kvirtv1.InterfaceBridge{},
		},
	}
}

// InterfaceDeviceWithSlirpBinding returns an Interface with SLIRP binding.
func InterfaceDeviceWithSlirpBinding(name string, ports ...kvirtv1.Port) kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: name,
		InterfaceBindingMethod: kvirtv1.InterfaceBindingMethod{
			Slirp: &kvirtv1.InterfaceSlirp{},
		},
		Ports: ports,
	}
}

// InterfaceDeviceWithSRIOVBinding returns an Interface with SRIOV binding.
func InterfaceDeviceWithSRIOVBinding(name string) kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: name,
		InterfaceBindingMethod: kvirtv1.InterfaceBindingMethod{
			SRIOV: &kvirtv1.InterfaceSRIOV{},
		},
	}
}

// InterfaceWithMac decorates an existing Interface with a MAC address.
func InterfaceWithMac(iface *kvirtv1.Interface, macAddress string) *kvirtv1.Interface {
	iface.MacAddress = macAddress
	return iface
}

// MultusNetwork returns a Network with the given name, associated to the given nad
func MultusNetwork(name, nadName string) *kvirtv1.Network {
	return &kvirtv1.Network{
		Name: name,
		NetworkSource: kvirtv1.NetworkSource{
			Multus: &kvirtv1.MultusNetwork{
				NetworkName: nadName,
			},
		},
	}
}
