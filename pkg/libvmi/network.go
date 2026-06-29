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

// InterfaceOption represents an action that configures an Interface.
type InterfaceOption func(iface *kvirtv1.Interface)

// NewInterface instantiates a new Interface, building its properties based on the specified options.
func NewInterface(name string, opts ...InterfaceOption) kvirtv1.Interface {
	iface := kvirtv1.Interface{Name: name}
	for _, opt := range opts {
		opt(&iface)
	}
	return iface
}

// WithMasqueradeBinding sets the masquerade binding method.
func WithMasqueradeBinding() InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.InterfaceBindingMethod = kvirtv1.InterfaceBindingMethod{
			Masquerade: &kvirtv1.InterfaceMasquerade{},
		}
	}
}

// WithBridgeBinding sets the bridge binding method.
func WithBridgeBinding() InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.InterfaceBindingMethod = kvirtv1.InterfaceBindingMethod{
			Bridge: &kvirtv1.InterfaceBridge{},
		}
	}
}

// WithSRIOVBinding sets the SRIOV binding method.
func WithSRIOVBinding() InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.InterfaceBindingMethod = kvirtv1.InterfaceBindingMethod{
			SRIOV: &kvirtv1.InterfaceSRIOV{},
		}
	}
}

// WithPasstBinding sets the passt binding method.
func WithPasstBinding() InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.InterfaceBindingMethod = kvirtv1.InterfaceBindingMethod{
			PasstBinding: &kvirtv1.InterfacePasstBinding{},
		}
	}
}

// WithBindingPlugin sets a plugin binding.
func WithBindingPlugin(binding kvirtv1.PluginBinding) InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.Binding = &binding
	}
}

// WithMac sets the MAC address.
func WithMac(macAddress string) InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.MacAddress = macAddress
	}
}

// WithPorts sets the ports.
func WithPorts(ports ...kvirtv1.Port) InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.Ports = ports
	}
}

// WithPciAddress sets the guest PCI address.
func WithPciAddress(pciAddress string) InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.PciAddress = pciAddress
	}
}

// WithTag sets a tag.
func WithTag(tag string) InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.Tag = tag
	}
}

// WithModel sets the interface model.
func WithModel(model string) InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.Model = model
	}
}

// WithState sets the interface state.
func WithState(state kvirtv1.InterfaceState) InterfaceOption {
	return func(iface *kvirtv1.Interface) {
		iface.State = state
	}
}

// InterfaceDeviceWithMasqueradeBinding returns an Interface named "default" with masquerade binding.
func InterfaceDeviceWithMasqueradeBinding(ports ...kvirtv1.Port) kvirtv1.Interface {
	return NewInterface(kvirtv1.DefaultPodNetwork().Name, WithMasqueradeBinding(), WithPorts(ports...))
}

// InterfaceDeviceWithBridgeBinding returns an Interface with bridge binding.
func InterfaceDeviceWithBridgeBinding(name string) kvirtv1.Interface {
	return NewInterface(name, WithBridgeBinding())
}

// InterfaceDeviceWithSRIOVBinding returns an Interface with SRIOV binding.
func InterfaceDeviceWithSRIOVBinding(name string) kvirtv1.Interface {
	return NewInterface(name, WithSRIOVBinding())
}

// InterfaceDeviceWithPasstBinding returns an Interface with passtBinding.
func InterfaceDeviceWithPasstBinding(name string) kvirtv1.Interface {
	return NewInterface(name, WithPasstBinding())
}

// InterfaceWithPasstBinding returns an Interface named "default" with passt binding plugin.
func InterfaceWithPasstBindingPlugin(ports ...kvirtv1.Port) kvirtv1.Interface {
	const passtBindingName = "passt"
	return NewInterface(kvirtv1.DefaultPodNetwork().Name,
		WithBindingPlugin(kvirtv1.PluginBinding{Name: passtBindingName}), WithPorts(ports...))
}

// InterfaceWithMacvtapBindingPlugin returns an Interface named "default" with "macvtap" binding plugin.
func InterfaceWithMacvtapBindingPlugin(name string) kvirtv1.Interface {
	const macvtapBindingName = "macvtap"
	return NewInterface(name, WithBindingPlugin(kvirtv1.PluginBinding{Name: macvtapBindingName}))
}

func InterfaceWithBindingPlugin(name string, binding kvirtv1.PluginBinding, ports ...kvirtv1.Port) kvirtv1.Interface {
	return NewInterface(name, WithBindingPlugin(binding), WithPorts(ports...))
}

// InterfaceWithMac decorates an existing Interface with a MAC address.
func InterfaceWithMac(iface kvirtv1.Interface, macAddress string) kvirtv1.Interface {
	WithMac(macAddress)(&iface)
	return iface
}

// InterfaceWithPciAddress decorates an existing Interface with a guest PCI address.
func InterfaceWithPciAddress(iface kvirtv1.Interface, pciAddress string) kvirtv1.Interface {
	WithPciAddress(pciAddress)(&iface)
	return iface
}

// InterfaceWithTag decorates an existing Interface with a tag.
func InterfaceWithTag(iface kvirtv1.Interface, tag string) kvirtv1.Interface {
	WithTag(tag)(&iface)
	return iface
}

// InterfaceWithModel decorates an existing Interface with a model.
func InterfaceWithModel(iface kvirtv1.Interface, model string) kvirtv1.Interface {
	WithModel(model)(&iface)
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

// DRANetwork returns a DRA-backed Network with resourceClaim source.
func DRANetwork(name, claimName, requestName string) *kvirtv1.Network {
	return &kvirtv1.Network{
		Name: name,
		NetworkSource: kvirtv1.NetworkSource{
			ResourceClaim: &kvirtv1.ClaimRequest{
				ClaimName:   claimName,
				RequestName: requestName,
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

// WithNetworkInterfaceMultiQueue sets the networkInterfaceMultiQueue field.
func WithNetworkInterfaceMultiQueue(enabled bool) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = &enabled
	}
}
