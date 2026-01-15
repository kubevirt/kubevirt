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
	"fmt"
	"strconv"

	"github.com/vishvananda/netlink"
	"golang.org/x/exp/slices"

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

type PasstLibvirtSpecGenerator struct {
	vmiSpecIface *v1.Interface
	domain       *api.Domain
	netLinkIface netlink.Link
	handler      netdriver.NetworkHandler
}

func NewPasstLibvirtSpecGenerator(
	iface *v1.Interface,
	domain *api.Domain,
	netLinkIface netlink.Link,
	handler netdriver.NetworkHandler,
) *PasstLibvirtSpecGenerator {
	return &PasstLibvirtSpecGenerator{
		vmiSpecIface: iface,
		domain:       domain,
		netLinkIface: netLinkIface,
		handler:      handler,
	}
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

func (g *PasstLibvirtSpecGenerator) Generate() error {
	idx := slices.IndexFunc(g.domain.Spec.Devices.Interfaces, func(iface api.Interface) bool {
		return iface.Alias.GetName() == g.vmiSpecIface.Name
	})

	const notFound = -1
	if notFound == idx {
		return fmt.Errorf("spec interface %s not found in domain", g.vmiSpecIface.Name)
	}

	userDefinedMAC, err := virtnetlink.RetrieveMacAddressFromVMISpecIface(g.vmiSpecIface)
	if err != nil {
		return err
	}
	var mac string
	if userDefinedMAC != nil {
		mac = userDefinedMAC.String()
	} else {
		mac = g.netLinkIface.Attrs().HardwareAddr.String()
	}
	g.domain.Spec.Devices.Interfaces[idx].MAC = &api.MAC{MAC: mac}

	return nil
}
