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

package libvmi

import (
	kvirtv1 "kubevirt.io/api/core/v1"
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

func WithPasstInterfaceWithPort() Option {
	return WithInterface(InterfaceWithPasstBindingPlugin([]kvirtv1.Port{{Port: 1234, Protocol: "TCP"}}...))
}

// InterfaceDeviceWithMasqueradeBinding returns an Interface named "default" with masquerade binding.
func InterfaceDeviceWithMasqueradeBinding(ports ...kvirtv1.Port) kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: kvirtv1.DefaultPodNetwork().Name,
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

// InterfaceDeviceWithSRIOVBinding returns an Interface with SRIOV binding.
func InterfaceDeviceWithSRIOVBinding(name string) kvirtv1.Interface {
	return kvirtv1.Interface{
		Name: name,
		InterfaceBindingMethod: kvirtv1.InterfaceBindingMethod{
			SRIOV: &kvirtv1.InterfaceSRIOV{},
		},
	}
}

// InterfaceWithPasstBinding returns an Interface named "default" with passt binding plugin.
func InterfaceWithPasstBindingPlugin(ports ...kvirtv1.Port) kvirtv1.Interface {
	const passtBindingName = "passt"
	return kvirtv1.Interface{
		Name:    kvirtv1.DefaultPodNetwork().Name,
		Binding: &kvirtv1.PluginBinding{Name: passtBindingName},
		Ports:   ports,
	}
}

// InterfaceWithMacvtapBindingPlugin returns an Interface named "default" with "macvtap" binding plugin.
func InterfaceWithMacvtapBindingPlugin(name string) *kvirtv1.Interface {
	const macvtapBindingName = "macvtap"
	return &kvirtv1.Interface{
		Name:    name,
		Binding: &kvirtv1.PluginBinding{Name: macvtapBindingName},
	}
}

func InterfaceWithBindingPlugin(name string, binding kvirtv1.PluginBinding, ports ...kvirtv1.Port) kvirtv1.Interface {
	return kvirtv1.Interface{
		Name:    name,
		Binding: &binding,
		Ports:   ports,
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

// WithHostname sets the hostname parameter.
func WithHostname(hostname string) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Hostname = hostname
	}
}

// WithSubdomain sets the subdomain parameter.
func WithSubdomain(subdomain string) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Subdomain = subdomain
	}
}

// WithAutoAttachPodInterface sets the autoattachPodInterface parameter.
func WithAutoAttachPodInterface(enabled bool) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.AutoattachPodInterface = &enabled
	}
}
