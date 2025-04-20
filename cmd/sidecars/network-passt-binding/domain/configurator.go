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

package domain

import (
	"fmt"
	"strings"

	vishnetlink "github.com/vishvananda/netlink"
	vmschema "kubevirt.io/api/core/v1"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"

	"kubevirt.io/kubevirt/pkg/network/driver/netlink"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type linkFinder interface {
	LinkByName(name string) (vishnetlink.Link, error)
}

type NetworkConfiguratorOptions struct {
	IstioProxyInjectionEnabled bool
	UseVirtioTransitional      bool
}

type PasstNetworkConfigurator struct {
	vmiSpecIface *vmschema.Interface
	options      NetworkConfiguratorOptions
	linkFinder   linkFinder
}

const (
	// PasstPluginName passt binding plugin name should be registered to Kubevirt through Kubevirt CR
	PasstPluginName = "passt"
	// PasstLogFilePath passt log file path Kubevirt consume and record
	PasstLogFilePath = "/var/run/kubevirt/passt.log"
)

func NewPasstNetworkConfigurator(ifaces []vmschema.Interface, networks []vmschema.Network, opts NetworkConfiguratorOptions, linkFinder linkFinder) (*PasstNetworkConfigurator, error) {
	network := vmispec.LookupPodNetwork(networks)
	if network == nil {
		return nil, fmt.Errorf("pod network not found")
	}
	iface := vmispec.LookupInterfaceByName(ifaces, network.Name)
	if iface == nil {
		return nil, fmt.Errorf("no interface found")
	}
	if iface.Binding == nil || iface.Binding != nil && iface.Binding.Name != PasstPluginName {
		return nil, fmt.Errorf("interface %q is not set with Passt network binding plugin", network.Name)
	}
	if linkFinder == nil {
		linkFinder = &netlink.NetLink{}
	}

	return &PasstNetworkConfigurator{
		vmiSpecIface: iface,
		options:      opts,
		linkFinder:   linkFinder,
	}, nil
}

func (p PasstNetworkConfigurator) Mutate(domainSpec *domainschema.DomainSpec) (*domainschema.DomainSpec, error) {
	generatedIface, err := p.generateInterface()
	if err != nil {
		return nil, fmt.Errorf("failed to generate domain interface spec: %v", err)
	}

	domainSpecCopy := domainSpec.DeepCopy()
	if iface := lookupIfaceByAliasName(domainSpecCopy.Devices.Interfaces, p.vmiSpecIface.Name); iface != nil {
		*iface = *generatedIface
	} else {
		domainSpecCopy.Devices.Interfaces = append(domainSpecCopy.Devices.Interfaces, *generatedIface)
	}

	log.Log.Infof("passt interface is added to domain spec successfully: %+v", generatedIface)

	return domainSpecCopy, nil
}

func lookupIfaceByAliasName(ifaces []domainschema.Interface, name string) *domainschema.Interface {
	for i, iface := range ifaces {
		if iface.Alias != nil && iface.Alias.GetName() == name {
			return &ifaces[i]
		}
	}

	return nil
}

func (p PasstNetworkConfigurator) generateInterface() (*domainschema.Interface, error) {
	sourceLinkName, err := p.discoverSourceLinkName()
	if err != nil {
		return nil, err
	}

	var pciAddress *domainschema.Address
	if p.vmiSpecIface.PciAddress != "" {
		var err error
		pciAddress, err = device.NewPciAddressField(p.vmiSpecIface.PciAddress)
		if err != nil {
			return nil, err
		}
	}

	var ifaceModel string
	if p.vmiSpecIface.Model == "" {
		ifaceModel = vmschema.VirtIO
	} else {
		ifaceModel = p.vmiSpecIface.Model
	}

	var ifaceModelType string
	if ifaceModel == vmschema.VirtIO {
		if p.options.UseVirtioTransitional {
			ifaceModelType = "virtio-transitional"
		} else {
			ifaceModelType = "virtio-non-transitional"
		}
	} else {
		ifaceModelType = p.vmiSpecIface.Model
	}
	model := &domainschema.Model{Type: ifaceModelType}

	var mac *domainschema.MAC
	if p.vmiSpecIface.MacAddress != "" {
		mac = &domainschema.MAC{MAC: p.vmiSpecIface.MacAddress}
	}

	var acpi *domainschema.ACPI
	if p.vmiSpecIface.ACPIIndex > 0 {
		acpi = &domainschema.ACPI{Index: uint(p.vmiSpecIface.ACPIIndex)}
	}

	const (
		ifaceTypeUser     = "user"
		ifaceBackendPasst = "passt"
	)
	return &domainschema.Interface{
		Alias:       domainschema.NewUserDefinedAlias(p.vmiSpecIface.Name),
		Model:       model,
		Address:     pciAddress,
		MAC:         mac,
		ACPI:        acpi,
		Type:        ifaceTypeUser,
		Source:      domainschema.InterfaceSource{Device: sourceLinkName},
		Backend:     &domainschema.InterfaceBackend{Type: ifaceBackendPasst, LogFile: PasstLogFilePath},
		PortForward: p.generatePortForward(),
	}, nil
}

func (p PasstNetworkConfigurator) generatePortForward() []domainschema.InterfacePortForward {
	var tcpPortsRange, udpPortsRange []domainschema.InterfacePortForwardRange

	if p.options.IstioProxyInjectionEnabled {
		for _, port := range istio.ReservedPorts() {
			tcpPortsRange = append(tcpPortsRange, domainschema.InterfacePortForwardRange{Start: uint(port), Exclude: "yes"})
		}
	}

	const (
		protoTCP = "tcp"
		protoUDP = "udp"
	)

	for _, port := range p.vmiSpecIface.Ports {
		if strings.EqualFold(port.Protocol, protoTCP) || port.Protocol == "" {
			tcpPortsRange = append(tcpPortsRange, domainschema.InterfacePortForwardRange{Start: uint(port.Port)})
		} else if strings.EqualFold(port.Protocol, protoUDP) {
			udpPortsRange = append(udpPortsRange, domainschema.InterfacePortForwardRange{Start: uint(port.Port)})
		} else {
			log.Log.Errorf("protocol %s is not supported by passt", port.Protocol)
		}
	}

	var portsFwd []domainschema.InterfacePortForward
	if len(udpPortsRange) == 0 && len(tcpPortsRange) == 0 {
		portsFwd = append(portsFwd, domainschema.InterfacePortForward{Proto: protoTCP})
		portsFwd = append(portsFwd, domainschema.InterfacePortForward{Proto: protoUDP})
	}
	if len(tcpPortsRange) > 0 {
		portsFwd = append(portsFwd, domainschema.InterfacePortForward{Proto: protoTCP, Ranges: tcpPortsRange})
	}
	if len(udpPortsRange) > 0 {
		portsFwd = append(portsFwd, domainschema.InterfacePortForward{Proto: protoUDP, Ranges: udpPortsRange})
	}

	return portsFwd
}

func (p PasstNetworkConfigurator) discoverSourceLinkName() (string, error) {
	// optionalLinkName link name to look for before falling back to eth0 if the
	// link do not exist
	// FIXME: This will be configurable in the future.
	const (
		// optionalLinkName comes from https://github.com/ovn-org/ovn-kubernetes/blob/master/go-controller/pkg/cni/udn/primary_network.go#L30
		optionalLinkName = "ovn-udn1"
	)

	if _, err := p.linkFinder.LinkByName(optionalLinkName); err != nil {
		if _, notFound := err.(vishnetlink.LinkNotFoundError); notFound {
			return namescheme.PrimaryPodInterfaceName, nil
		}
		return "", err
	}
	return optionalLinkName, nil
}
