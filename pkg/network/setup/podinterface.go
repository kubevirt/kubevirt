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

package network

import (
	"fmt"
	"strconv"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type LibvirtSpecGenerator interface {
	generate() error
}

func newMacvtapLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain, podInterfaceName string, handler netdriver.NetworkHandler) *MacvtapLibvirtSpecGenerator {
	return &MacvtapLibvirtSpecGenerator{
		vmiSpecIface:     iface,
		domain:           domain,
		podInterfaceName: podInterfaceName,
		handler:          handler,
	}
}

func newMasqueradeLibvirtSpecGenerator(iface *v1.Interface, vmiSpecNetwork *v1.Network, domain *api.Domain, podInterfaceName string, handler netdriver.NetworkHandler) *MasqueradeLibvirtSpecGenerator {
	return &MasqueradeLibvirtSpecGenerator{
		vmiSpecIface:     iface,
		vmiSpecNetwork:   vmiSpecNetwork,
		domain:           domain,
		podInterfaceName: podInterfaceName,
		handler:          handler,
	}
}

func newSlirpLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain) *SlirpLibvirtSpecGenerator {
	return &SlirpLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
	}
}

func newBridgeLibvirtSpecGenerator(iface *v1.Interface, domain *api.Domain, cachedDomainInterface api.Interface, podInterfaceName string, handler netdriver.NetworkHandler) *BridgeLibvirtSpecGenerator {
	return &BridgeLibvirtSpecGenerator{
		vmiSpecIface:          iface,
		domain:                domain,
		cachedDomainInterface: cachedDomainInterface,
		podInterfaceName:      podInterfaceName,
		handler:               handler,
	}
}

type BridgeLibvirtSpecGenerator struct {
	vmiSpecIface          *v1.Interface
	domain                *api.Domain
	cachedDomainInterface api.Interface
	podInterfaceName      string
	handler               netdriver.NetworkHandler
}

func (b *BridgeLibvirtSpecGenerator) generate() error {
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
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return nil, err
	}
	_, dummy := podNicLink.(*netlink.Dummy)
	if dummy {
		newPodNicName := virtnetlink.GenerateNewBridgedVmiInterfaceName(b.podInterfaceName)
		podNicLink, err = b.handler.LinkByName(newPodNicName)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get a link for interface: %s", newPodNicName)
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

func (b *MasqueradeLibvirtSpecGenerator) generate() error {
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
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
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

func (b *SlirpLibvirtSpecGenerator) generate() error {
	err := downloadPasstBinaries()
	if err != nil {
		return err
	}

	output, err := exec.Command("passt").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	return nil
}

func downloadPasstBinaries() error {
	//curl -k https://passt.top/builds/static/passt --output /usr/bin/passt
	output, err := exec.Command("curl", "-k", "https://passt.top/builds/static/passt", "--output", "/usr/bin/passt").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	//curl -k https://passt.top/builds/static/qrap --output /usr/bin/qrap
	output, err = exec.Command("curl", "-k", "https://passt.top/builds/static/qrap", "--output", "/usr/bin/qrap").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	//curl -k https://gist.githubusercontent.com/AlonaKaplan/072680bd9d7e438bb90ee0e5a2423825/raw/54a0a59b6bf55465e06324b3cc259366c9f5b39e/gistfile1.txt --output /usr/bin/qrap.sh
	output, err = exec.Command("curl", "-k", "https://gist.githubusercontent.com/AlonaKaplan/072680bd9d7e438bb90ee0e5a2423825/raw/54a0a59b6bf55465e06324b3cc259366c9f5b39e/gistfile1.txt", "--output", "/usr/bin/qrap.sh").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	//chmod +x /usr/bin/passt
	output, err = exec.Command("chmod", "+x", "/usr/bin/passt").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	//chmod +x /usr/bin/qrap
	output, err = exec.Command("chmod", "+x", "/usr/bin/qrap").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	//chmod +x /usr/bin/qrap
	output, err = exec.Command("chmod", "+x", "/usr/bin/qrap.sh").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	return nil
}

type MacvtapLibvirtSpecGenerator struct {
	vmiSpecIface     *v1.Interface
	domain           *api.Domain
	podInterfaceName string
	handler          netdriver.NetworkHandler
}

func (b *MacvtapLibvirtSpecGenerator) generate() error {
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
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
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
