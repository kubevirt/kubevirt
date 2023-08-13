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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package nmstate

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strconv"

	"golang.org/x/sys/unix"

	vishnetlink "github.com/vishvananda/netlink"
)

func (n NMState) Apply(spec *Spec) error {
	for _, iface := range spec.Interfaces {
		link, err := n.readLink(iface)
		if err != nil {
			return err
		}
		linkExists := link != nil

		if iface.State == IfaceStateAbsent {
			if linkExists {
				if derr := n.deleteInterface(iface); derr != nil {
					return fmt.Errorf("failed to delete link [%s]: %v", iface.Name, derr)
				}
			}
		} else {
			if iface.CopyMacFrom != "" {
				if iface.MacAddress != "" {
					return fmt.Errorf("specifying both MAC address and copy-mac-from fields is not supported [%s]", iface.Name)
				}
				macSourceLink, err := n.adapter.LinkByName(iface.CopyMacFrom)
				if err != nil {
					return fmt.Errorf("unable to find mac shource link [%s]: %v", iface.Name, err)
				}
				iface.MacAddress = macSourceLink.Attrs().HardwareAddr.String()
			}
			if !linkExists {
				if link, err = n.createInterface(iface); err != nil {
					return err
				}
			}

			if linkType := normalizeLinkTypeName(link); iface.TypeName != "" && linkType != iface.TypeName {
				return fmt.Errorf("type collision on link %s: actual %s, requested %s", iface.Name, linkType, iface.TypeName)
			}
			if serr := n.setupInterface(iface, link); serr != nil {
				return fmt.Errorf("failed to setup link [%s]: %v", iface.Name, serr)
			}
		}
	}

	err := n.setupLinuxStack(spec.LinuxStack)

	return err
}

func (n NMState) readLink(iface Interface) (vishnetlink.Link, error) {
	var link vishnetlink.Link
	var err error
	if iface.Index > 0 {
		link, err = n.adapter.LinkByIndex(iface.Index)
	} else {
		link, err = n.adapter.LinkByName(iface.Name)
	}
	if err != nil {
		var errNotFound vishnetlink.LinkNotFoundError
		if !errors.As(err, &errNotFound) {
			return nil, fmt.Errorf("failed reading link [%s]: %w", iface.Name, err)
		}
		return nil, nil
	}
	return link, nil
}

func (n NMState) deleteInterface(iface Interface) error {
	link := &vishnetlink.GenericLink{}
	link.Name = iface.Name
	return n.adapter.LinkDel(link)
}

var linkConfigByType = map[string]func(iface Interface) (vishnetlink.Link, error){
	TypeBridge: func(iface Interface) (vishnetlink.Link, error) {
		return initLink(iface.Name, &vishnetlink.Bridge{})
	},
	TypeDummy: func(iface Interface) (vishnetlink.Link, error) {
		return initLink(iface.Name, &vishnetlink.Dummy{})
	},
	TypeTap: func(iface Interface) (vishnetlink.Link, error) {
		ifaceTap := iface.Tap
		if ifaceTap == nil {
			ifaceTap = &TapDevice{}
		}
		return initLink(iface.Name, &vishnetlink.Tuntap{
			Mode:   unix.IFF_TAP,
			Queues: ifaceTap.Queues,
			Owner:  uint32(ifaceTap.UID),
			Group:  uint32(ifaceTap.GID),
		})
	},
}

func (n NMState) createInterface(iface Interface) (vishnetlink.Link, error) {
	var link vishnetlink.Link

	if f, exists := linkConfigByType[iface.TypeName]; exists {
		var err error
		if link, err = f(iface); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported interface type [%s]: %q", iface.Name, iface.TypeName)
	}

	// Tap devices require special (backend) creation (due to SELinux labeling)
	if iface.TypeName == TypeTap {
		return n.createTap(iface, link)
	}

	if err := n.adapter.LinkAdd(link); err != nil {
		return nil, err
	}

	return link, nil
}

func (n NMState) createTap(iface Interface, link vishnetlink.Link) (vishnetlink.Link, error) {
	l := link.(*vishnetlink.Tuntap)
	l.MTU = iface.MTU
	metadata := iface.Metadata
	if metadata == nil {
		metadata = &IfaceMetadata{Pid: os.Getpid()}
	}
	return link, n.adapter.AddTapDeviceWithSELinuxLabel(l.Name, l.MTU, l.Queues, int(l.Owner), metadata.Pid)
}

// setupInterface updates an existing link with the desired interface configuration.
// When applicable, a change is applied only if required.
// However, there are cases where the link does not contain the information needed to determine if a change
// is required or not. For such cases, a set is performed anyway (to save kernel access).
func (n NMState) setupInterface(iface Interface, link vishnetlink.Link) error {
	if op := link.Attrs().OperState; op == vishnetlink.OperUp || op == vishnetlink.OperLowerLayerDown {
		if err := n.adapter.LinkSetDown(link); err != nil {
			return err
		}
		defer func() { _ = n.adapter.LinkSetUp(link) }()
	}

	if iface.Name != link.Attrs().Name {
		if err := n.adapter.LinkSetName(link, iface.Name); err != nil {
			return err
		}
		// When trying to update the IP addresses the link name sometimes invalidates the operation
		// as it is inconsistent with the address label. Update the link with the new name to overcome this issue.
		// Ref: https://github.com/vishvananda/netlink/blob/a8a91c050431c7e23691dec368a86bd39b39da29/addr_linux.go#L79
		link.Attrs().Name = iface.Name
	}

	if iface.MacAddress != "" && iface.MacAddress != link.Attrs().HardwareAddr.String() {
		mac, err := net.ParseMAC(iface.MacAddress)
		if err != nil {
			return err
		}
		if err = n.adapter.LinkSetHardwareAddr(link, mac); err != nil {
			return err
		}
	}

	if iface.MTU > 0 && iface.MTU != link.Attrs().MTU {
		if err := n.adapter.LinkSetMTU(link, iface.MTU); err != nil {
			return err
		}
	}

	if iface.Ethtool.Feature.TxChecksum != nil && !(*iface.Ethtool.Feature.TxChecksum) {
		if err := n.adapter.TXChecksumOff(iface.Name); err != nil {
			return err
		}
	}

	if iface.Controller != "" {
		bridgeLink := vishnetlink.Bridge{}
		bridgeLink.Name = iface.Controller
		if err := n.adapter.LinkSetMaster(link, &bridgeLink); err != nil {
			return err
		}
	}

	if err := n.setupInterfaceIP(iface, link); err != nil {
		return err
	}

	if iface.State == IfaceStateUp {
		if err := n.adapter.LinkSetUp(link); err != nil {
			return err
		}
	}

	if val := iface.LinuxStack.IP4RouteLocalNet; val != nil && *val {
		if err := n.adapter.IPv4EnableRouteLocalNet(iface.Name); err != nil {
			return err
		}
	}

	if val := iface.LinuxStack.PortLearning; val != nil && !*val {
		if err := n.adapter.LinkSetLearningOff(link); err != nil {
			return err
		}
	}
	return nil
}

func (n NMState) setupInterfaceIP(iface Interface, link vishnetlink.Link) error {
	if err := n.deleteIPAddresses(link); err != nil {
		return err
	}
	if iface.IPv4.Enabled != nil && *iface.IPv4.Enabled {
		if err := n.addIPAddresses(iface.IPv4.Address, link); err != nil {
			return err
		}
	}
	if iface.IPv6.Enabled != nil && *iface.IPv6.Enabled {
		if err := n.addIPAddresses(iface.IPv6.Address, link); err != nil {
			return err
		}
	}
	return nil
}

func (n NMState) addIPAddresses(ips []IPAddress, link vishnetlink.Link) error {
	for _, ip := range ips {
		addr, err := n.adapter.ParseAddr(ip.IP + "/" + strconv.Itoa(ip.PrefixLen))
		if err != nil {
			return err
		}
		if err := n.adapter.AddrAdd(link, addr); err != nil {
			// If the address already exists, ignore the error as there is nothing to do.
			if !errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("failed to add IP address [%v] on link %s: %v", addr, link.Attrs().Name, err)
			}
		}
	}
	return nil
}

func (n NMState) deleteIPAddresses(link vishnetlink.Link) error {
	addresses, err := n.adapter.AddrList(link, vishnetlink.FAMILY_ALL)
	if err != nil {
		return err
	}
	for _, addr := range addresses {
		if err := n.adapter.AddrDel(link, &addr); err != nil {
			return fmt.Errorf("failed to delete IP address %v from link %s: %v", addr, link.Attrs().Name, err)
		}
	}
	return nil
}

// initLink sets the link name and initializes the generic attributes.
// It expects as input a concrete link (with optional specific concrete attributes set already).
func initLink(ifaceName string, link vishnetlink.Link) (vishnetlink.Link, error) {
	linkAttrs := vishnetlink.NewLinkAttrs()
	linkAttrs.Name = ifaceName
	*link.Attrs() = linkAttrs
	return link, nil
}

func (n NMState) setupLinuxStack(linuxStack LinuxStack) error {
	if val := linuxStack.IPv4.Forwarding; val != nil && *val {
		if err := n.adapter.IPv4EnableForwarding(); err != nil {
			return err
		}
	}
	if val := linuxStack.IPv6.Forwarding; val != nil && *val {
		if err := n.adapter.IPv6EnableForwarding(); err != nil {
			return err
		}
	}
	if val := linuxStack.IPv4.ArpIgnore; val != nil {
		if err := n.adapter.IPv4SetArpIgnore("all", *val); err != nil {
			return err
		}
	}
	if fromto := linuxStack.IPv4.PingGroupRange; fromto != nil && len(fromto) == 2 {
		if err := n.adapter.IPv4SetPingGroupRange(fromto[0], fromto[1]); err != nil {
			return err
		}
	}
	if startPort := linuxStack.IPv4.UnprivilegedPortStart; startPort != nil {
		if err := n.adapter.IPv4SetUnprivilegedPortStart(*startPort); err != nil {
			return err
		}
	}
	return nil
}
