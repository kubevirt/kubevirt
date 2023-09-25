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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package domainspec

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/istio"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const linkIfaceFailFmt = "failed to get a link for interface: %s"

type LibvirtSpecGenerator interface {
	Generate() error
}

func NewMacvtapLibvirtSpecGenerator(
	iface *v1.Interface,
	domain *api.Domain,
	podInterfaceName string,
	handler netdriver.NetworkHandler,
) *MacvtapLibvirtSpecGenerator {
	return &MacvtapLibvirtSpecGenerator{
		vmiSpecIface:     iface,
		domain:           domain,
		podInterfaceName: podInterfaceName,
		handler:          handler,
	}
}

func NewMasqueradeLibvirtSpecGenerator(
	iface *v1.Interface,
	vmiSpecNetwork *v1.Network,
	domain *api.Domain,
	podInterfaceName string,
	handler netdriver.NetworkHandler,
) *MasqueradeLibvirtSpecGenerator {
	return &MasqueradeLibvirtSpecGenerator{
		vmiSpecIface:     iface,
		vmiSpecNetwork:   vmiSpecNetwork,
		domain:           domain,
		podInterfaceName: podInterfaceName,
		handler:          handler,
	}
}

func NewBridgeLibvirtSpecGenerator(
	iface *v1.Interface,
	domain *api.Domain,
	cachedDomainInterface api.Interface,
	podInterfaceName string,
	handler netdriver.NetworkHandler,
) *BridgeLibvirtSpecGenerator {
	return &BridgeLibvirtSpecGenerator{
		vmiSpecIface:          iface,
		domain:                domain,
		cachedDomainInterface: cachedDomainInterface,
		podInterfaceName:      podInterfaceName,
		handler:               handler,
	}
}

func NewPasstLibvirtSpecGenerator(
	iface *v1.Interface,
	domain *api.Domain,
	podIfaceName string,
	vmi *v1.VirtualMachineInstance,
) *PasstLibvirtSpecGenerator {
	return &PasstLibvirtSpecGenerator{
		vmiSpecIface:     iface,
		domain:           domain,
		podInterfaceName: podIfaceName,
		vmi:              vmi,
	}
}

type BridgeLibvirtSpecGenerator struct {
	vmiSpecIface          *v1.Interface
	domain                *api.Domain
	cachedDomainInterface api.Interface
	podInterfaceName      string
	handler               netdriver.NetworkHandler
}

func (b *BridgeLibvirtSpecGenerator) Generate() error {
	domainIface, err := b.discoverDomainIfaceSpec()
	if err != nil {
		return err
	}
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func (b *BridgeLibvirtSpecGenerator) discoverDomainIfaceSpec() (*api.Interface, error) {
	podNicLink, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf(linkIfaceFailFmt, b.podInterfaceName)
		return nil, err
	}
	_, dummy := podNicLink.(*netlink.Dummy)
	if dummy {
		newPodNicName := virtnetlink.GenerateNewBridgedVmiInterfaceName(b.podInterfaceName)
		podNicLink, err = b.handler.LinkByName(newPodNicName)
		if err != nil {
			log.Log.Reason(err).Errorf(linkIfaceFailFmt, newPodNicName)
			return nil, err
		}
	}

	b.cachedDomainInterface.MTU = &api.MTU{Size: strconv.Itoa(podNicLink.Attrs().MTU)}

	b.cachedDomainInterface.Target = &api.InterfaceTarget{
		Device:  virtnetlink.GenerateTapDeviceName(b.podInterfaceName),
		Managed: "no"}
	return &b.cachedDomainInterface, nil
}

type MasqueradeLibvirtSpecGenerator struct {
	vmiSpecIface     *v1.Interface
	vmiSpecNetwork   *v1.Network
	domain           *api.Domain
	handler          netdriver.NetworkHandler
	podInterfaceName string
}

func (b *MasqueradeLibvirtSpecGenerator) Generate() error {
	domainIface, err := b.discoverDomainIfaceSpec()
	if err != nil {
		return err
	}
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func (b *MasqueradeLibvirtSpecGenerator) discoverDomainIfaceSpec() (*api.Interface, error) {
	var domainIface api.Interface
	podNicLink, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf(linkIfaceFailFmt, b.podInterfaceName)
		return nil, err
	}

	mac, err := virtnetlink.RetrieveMacAddressFromVMISpecIface(b.vmiSpecIface)
	if err != nil {
		return nil, err
	}

	domainIface.MTU = &api.MTU{Size: strconv.Itoa(podNicLink.Attrs().MTU)}
	domainIface.Target = &api.InterfaceTarget{
		Device:  virtnetlink.GenerateTapDeviceName(podNicLink.Attrs().Name),
		Managed: "no",
	}

	if mac != nil {
		domainIface.MAC = &api.MAC{MAC: mac.String()}
	}
	return &domainIface, nil
}

type MacvtapLibvirtSpecGenerator struct {
	vmiSpecIface     *v1.Interface
	domain           *api.Domain
	podInterfaceName string
	handler          netdriver.NetworkHandler
}

func (b *MacvtapLibvirtSpecGenerator) Generate() error {
	domainIface, err := b.discoverDomainIfaceSpec()
	if err != nil {
		return err
	}
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			ifaces[i].MTU = domainIface.MTU
			ifaces[i].MAC = domainIface.MAC
			ifaces[i].Target = domainIface.Target
			break
		}
	}
	return nil
}

func (b *MacvtapLibvirtSpecGenerator) discoverDomainIfaceSpec() (*api.Interface, error) {
	podNicLink, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf(linkIfaceFailFmt, b.podInterfaceName)
		return nil, err
	}
	mac, err := virtnetlink.RetrieveMacAddressFromVMISpecIface(b.vmiSpecIface)
	if err != nil {
		return nil, err
	}
	if mac == nil {
		mac = &podNicLink.Attrs().HardwareAddr
	}

	return &api.Interface{
		MAC: &api.MAC{MAC: mac.String()},
		MTU: &api.MTU{Size: strconv.Itoa(podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  b.podInterfaceName,
			Managed: "no",
		},
	}, nil
}

type PasstLibvirtSpecGenerator struct {
	vmiSpecIface     *v1.Interface
	domain           *api.Domain
	podInterfaceName string
	vmi              *v1.VirtualMachineInstance
}

const (
	PasstLogFile      = "/var/run/kubevirt/passt.log" // #nosec G101
	ifaceTypeUser     = "user"
	ifaceBackendPasst = "passt"
)

func (b *PasstLibvirtSpecGenerator) Generate() error {
	domainIface := LookupIfaceByAliasName(b.domain.Spec.Devices.Interfaces, b.vmiSpecIface.Name)
	if domainIface == nil {
		return fmt.Errorf("failed to find interface %s in domain spec", b.vmiSpecIface.Name)
	}

	generatedIface := b.generateInterface(domainIface)
	*domainIface = *generatedIface

	return nil
}

const (
	protoTCP = "tcp"
	protoUDP = "udp"
)

func (b *PasstLibvirtSpecGenerator) generateInterface(iface *api.Interface) *api.Interface {
	ifaceCopy := iface.DeepCopy()

	var mac *api.MAC
	if b.vmiSpecIface.MacAddress != "" {
		mac = &api.MAC{MAC: b.vmiSpecIface.MacAddress}
	}

	ifaceCopy.Type = ifaceTypeUser
	ifaceCopy.Source = api.InterfaceSource{Device: b.podInterfaceName}
	ifaceCopy.Backend = &api.InterfaceBackend{Type: ifaceBackendPasst, LogFile: PasstLogFile}
	ifaceCopy.PortForward = b.generatePortForward()
	ifaceCopy.MAC = mac

	return ifaceCopy
}

func (b *PasstLibvirtSpecGenerator) generatePortForward() []api.InterfacePortForward {
	var tcpPortsRange, udpPortsRange []api.InterfacePortForwardRange

	if istio.ProxyInjectionEnabled(b.vmi) {
		for _, port := range istio.ReservedPorts() {
			tcpPortsRange = append(tcpPortsRange, api.InterfacePortForwardRange{Start: uint(port), Exclude: "yes"})
		}
	}

	for _, port := range b.vmiSpecIface.Ports {
		if strings.EqualFold(port.Protocol, protoTCP) || port.Protocol == "" {
			tcpPortsRange = append(tcpPortsRange, api.InterfacePortForwardRange{Start: uint(port.Port)})
		} else if strings.EqualFold(port.Protocol, protoUDP) {
			udpPortsRange = append(udpPortsRange, api.InterfacePortForwardRange{Start: uint(port.Port)})
		} else {
			log.Log.Errorf("protocol %s is not supported by passt", port.Protocol)
		}
	}

	var portsFwd []api.InterfacePortForward
	if len(udpPortsRange) == 0 && len(tcpPortsRange) == 0 {
		portsFwd = append(portsFwd,
			api.InterfacePortForward{Proto: protoTCP},
			api.InterfacePortForward{Proto: protoUDP},
		)
	}
	if len(tcpPortsRange) > 0 {
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoTCP, Ranges: tcpPortsRange})
	}
	if len(udpPortsRange) > 0 {
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoUDP, Ranges: udpPortsRange})
	}

	return portsFwd
}
