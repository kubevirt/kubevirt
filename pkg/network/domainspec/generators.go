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

package domainspec

import (
	"errors"
	"strconv"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const linkIfaceFailFmt = "failed to get a link for interface: %s"

type LibvirtSpecGenerator interface {
	Generate() error
}

func NewTapLibvirtSpecGenerator(
	iface *v1.Interface,
	network v1.Network,
	domain *api.Domain,
	podInterfaceName string,
	handler netdriver.NetworkHandler,
) *TapLibvirtSpecGenerator {
	return &TapLibvirtSpecGenerator{
		vmiSpecIface:     iface,
		vmiSpecNetwork:   network,
		domain:           domain,
		podInterfaceName: podInterfaceName,
		handler:          handler,
	}
}

type TapLibvirtSpecGenerator struct {
	vmiSpecIface     *v1.Interface
	vmiSpecNetwork   v1.Network
	domain           *api.Domain
	podInterfaceName string
	handler          netdriver.NetworkHandler
}

func (b *TapLibvirtSpecGenerator) Generate() error {
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

func (b *TapLibvirtSpecGenerator) discoverDomainIfaceSpec() (*api.Interface, error) {
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

	targetName, err := b.getTargetName()
	if err != nil {
		return nil, err
	}
	return &api.Interface{
		MAC: &api.MAC{MAC: mac.String()},
		MTU: &api.MTU{Size: strconv.Itoa(podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  targetName,
			Managed: "no",
		},
	}, nil
}

// The method tries to find a tap device based on the hashed network name
// in case such device doesn't exist, the pod interface is used as the target
func (b *TapLibvirtSpecGenerator) getTargetName() (string, error) {
	tapName := virtnetlink.GenerateTapDeviceName(b.podInterfaceName, b.vmiSpecNetwork)
	if _, err := b.handler.LinkByName(tapName); err != nil {
		var linkNotFoundErr netlink.LinkNotFoundError
		if errors.As(err, &linkNotFoundErr) {
			return b.podInterfaceName, nil
		}
		return "", err
	}
	return tapName, nil
}

func NewPasstLibvirtSpecGenerator(
	iface *v1.Interface,
	network v1.Network,
	domain *api.Domain,
	podInterfaceName string,
	handler netdriver.NetworkHandler,
) *PasstLibvirtSpecGenerator {
	return &PasstLibvirtSpecGenerator{
		vmiSpecIface:     iface,
		vmiSpecNetwork:   network,
		domain:           domain,
		podInterfaceName: podInterfaceName,
		handler:          handler,
	}
}

type PasstLibvirtSpecGenerator struct {
	vmiSpecIface     *v1.Interface
	vmiSpecNetwork   v1.Network
	domain           *api.Domain
	podInterfaceName string
	handler          netdriver.NetworkHandler
}

func (p *PasstLibvirtSpecGenerator) Generate() error {
	ips, err := p.discoverPodIPs()
	if err != nil {
		return err
	}

	domainIface := LookupIfaceByAliasName(p.domain.Spec.Devices.Interfaces, p.vmiSpecIface.Name)
	if domainIface != nil {
		domainIface.IPs = ips
	}

	return nil
}

func (p *PasstLibvirtSpecGenerator) discoverPodIPs() ([]api.InterfaceIP, error) {
	podNicLink, err := p.handler.LinkByName(p.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf(linkIfaceFailFmt, p.podInterfaceName)
		return nil, err
	}

	var ips []api.InterfaceIP

	ipv4, err := p.firstGlobalUnicastAddr(podNicLink, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}
	if ipv4 != nil {
		prefixLen, _ := ipv4.Mask.Size()
		ips = append(ips, api.InterfaceIP{
			Family:  "ipv4",
			Address: ipv4.IP.String(),
			Prefix:  strconv.Itoa(prefixLen),
		})
	}

	ipv6, err := p.firstGlobalUnicastAddr(podNicLink, netlink.FAMILY_V6)
	if err != nil {
		return nil, err
	}
	if ipv6 != nil {
		ips = append(ips, api.InterfaceIP{
			Family:  "ipv6",
			Address: ipv6.IP.String(),
		})
	}

	return ips, nil
}

func (p *PasstLibvirtSpecGenerator) firstGlobalUnicastAddr(link netlink.Link, family int) (*netlink.Addr, error) {
	addrs, err := p.handler.AddrList(link, family)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get addresses for interface: %s", p.podInterfaceName)
		return nil, err
	}
	for i, addr := range addrs {
		if addr.IP.IsGlobalUnicast() {
			return &addrs[i], nil
		}
	}
	return nil, nil
}
