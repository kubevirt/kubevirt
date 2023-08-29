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
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/istio"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
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

func NewSlirpLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *SlirpLibvirtSpecGenerator {
	return &SlirpLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
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

func NewPasstLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain, vmi *v1.VirtualMachineInstance) *PasstLibvirtSpecGenerator {
	return &PasstLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
		vmi:          vmi,
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

type SlirpLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
}

func (b *SlirpLibvirtSpecGenerator) Generate() error {
	// remove slirp interface from domain spec devices interfaces
	var foundIfaceModelType string
	for i, iface := range b.domain.Spec.Devices.Interfaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			b.domain.Spec.Devices.Interfaces = append(
				b.domain.Spec.Devices.Interfaces[:i],
				b.domain.Spec.Devices.Interfaces[i+1:]...,
			)
			foundIfaceModelType = iface.Model.Type
			break
		}
	}

	if foundIfaceModelType == "" {
		return fmt.Errorf("failed to find interface %s in vmi spec", b.vmiSpecIface.Name)
	}

	qemuArg := fmt.Sprintf(`{"driver":%q,"netdev":%q,"id":%q`, foundIfaceModelType, b.vmiSpecIface.Name, b.vmiSpecIface.Name)
	if b.vmiSpecIface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		qemuArg += fmt.Sprintf(`,"mac":%q`, b.vmiSpecIface.MacAddress)
	}
	qemuArg += "}"
	// Add interface configuration to qemuArgs
	b.domain.Spec.QEMUCmd.QEMUArg = append(
		b.domain.Spec.QEMUCmd.QEMUArg,
		api.Arg{Value: "-device"},
		api.Arg{Value: qemuArg},
	)
	return nil
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
	vmiSpecIface *v1.Interface
	domain       *api.Domain
	vmi          *v1.VirtualMachineInstance
}

const (
	passtLogFile      = "/var/run/kubevirt/passt.log"
	ifaceTypeUser     = "user"
	ifaceBackendPasst = "passt"
)

func (b *PasstLibvirtSpecGenerator) Generate() error {
	var passtIfaceFound bool
	var passtIfaceIdx int
	for i, iface := range b.domain.Spec.Devices.Interfaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			passtIfaceFound = true
			passtIfaceIdx = i
			break
		}
	}
	if !passtIfaceFound {
		return fmt.Errorf("failed to find interface %s in vmi spec", b.vmiSpecIface.Name)
	}

	iface := b.generateInterface(&b.domain.Spec.Devices.Interfaces[passtIfaceIdx])
	b.domain.Spec.Devices.Interfaces[passtIfaceIdx] = *iface

	go func() {
		for {
			if _, err := os.Stat(passtLogFile); !errors.Is(err, os.ErrNotExist) {
				break
			}
			time.Sleep(time.Second)
		}

		logFile, err := os.Open(passtLogFile)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to open file: %s", passtLogFile)
			return
		}
		defer func() {
			if closeErr := logFile.Close(); closeErr != nil {
				log.Log.Reason(err).Errorf("failed to close file: %s", passtLogFile)
			}
		}()

		const bufferSize = 1024
		const maxBufferSize = 512 * bufferSize
		scanner := bufio.NewScanner(logFile)
		scanner.Buffer(make([]byte, bufferSize), maxBufferSize)
		for scanner.Scan() {
			log.Log.Info(fmt.Sprintf("passt: %s", scanner.Text()))
		}
		if err = scanner.Err(); err != nil {
			log.Log.Reason(err).Error("failed to read passt logs")
		}
	}()

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
	ifaceCopy.Source = api.InterfaceSource{Device: namescheme.PrimaryPodInterfaceName}
	ifaceCopy.Backend = &api.InterfaceBackend{Type: ifaceBackendPasst, LogFile: passtLogFile}
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
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoTCP})
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoUDP})
	}
	if len(tcpPortsRange) > 0 {
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoTCP, Ranges: tcpPortsRange})
	}
	if len(udpPortsRange) > 0 {
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoUDP, Ranges: udpPortsRange})
	}

	return portsFwd
}
