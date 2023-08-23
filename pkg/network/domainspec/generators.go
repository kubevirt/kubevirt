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
	"fmt"
	"io"

	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

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

func (b *PasstLibvirtSpecGenerator) Generate() error {
	err := exec.Command("pgrep", "passt").Run()
	if err == nil {
		return fmt.Errorf("passt process is already running")
	}
	// remove passt interface from domain spec devices interfaces
	foundDomainInterface := false
	for i, iface := range b.domain.Spec.Devices.Interfaces {
		if iface.Alias.GetName() == b.vmiSpecIface.Name {
			b.domain.Spec.Devices.Interfaces = append(b.domain.Spec.Devices.Interfaces[:i], b.domain.Spec.Devices.Interfaces[i+1:]...)
			foundDomainInterface = true
			break
		}
	}
	if !foundDomainInterface {
		return fmt.Errorf("failed to find interface %s in vmi spec", b.vmiSpecIface.Name)
	}

	ports := b.generatePorts()
	args := append([]string{"--runas", "107", "-e"}, ports...)
	cmd := exec.Command("/usr/bin/passt", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps: []uintptr{unix.CAP_NET_BIND_SERVICE},
	}

	// connect passt's stderr to our own stdout in order to see the logs in the container logs
	var reader io.ReadCloser
	reader, err = cmd.StderrPipe()
	if err != nil {
		log.Log.Reason(err).Error("failed to get passt stderr")
		return err
	}
	go func() {
		const bufferSize = 1024
		const maxBufferSize = 512 * bufferSize
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, bufferSize), maxBufferSize)
		for scanner.Scan() {
			log.Log.Info(fmt.Sprintf("passt: %s", scanner.Text()))
		}
		if err = scanner.Err(); err != nil {
			log.Log.Reason(err).Error("failed to read passt logs")
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Log.Reason(err).Error("failed to start passt")
		return err
	}

	err = cmd.Wait()
	if err != nil {
		log.Log.Reason(err).Error("failed waiting for passt going to background")
		return err
	}

	return nil
}

func (b *PasstLibvirtSpecGenerator) generatePorts() []string {
	tcpPorts := []string{}
	udpPorts := []string{}

	if len(b.vmiSpecIface.Ports) == 0 {
		if istio.ProxyInjectionEnabled(b.vmi) {
			for _, port := range istio.ReservedPorts() {
				tcpPorts = append(tcpPorts, fmt.Sprintf("~%d", port))
			}
		} else {
			tcpPorts = append(tcpPorts, "all")
		}
		udpPorts = append(udpPorts, "all")
	}
	for _, port := range b.vmiSpecIface.Ports {
		if strings.EqualFold(port.Protocol, "TCP") || port.Protocol == "" {
			tcpPorts = append(tcpPorts, fmt.Sprintf("%d", port.Port))
		} else if strings.EqualFold(port.Protocol, "UDP") {
			udpPorts = append(udpPorts, fmt.Sprintf("%d", port.Port))
		} else {
			log.Log.Errorf("protocol %s is not supported by passt", port.Protocol)
		}
	}

	if len(tcpPorts) != 0 {
		tcpPorts = append([]string{"-t"}, strings.Join(tcpPorts, ","))
	}

	if len(udpPorts) != 0 {
		udpPorts = append([]string{"-u"}, strings.Join(udpPorts, ","))
	}
	return append(tcpPorts, udpPorts...)
}
