/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package netlink

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func (n NetLink) LinkList() ([]netlink.Link, error) {
	return netlink.LinkList()
}

func (n NetLink) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

func (n NetLink) LinkByIndex(index int) (netlink.Link, error) {
	return netlink.LinkByIndex(index)
}

func (n NetLink) LinkSetHardwareAddr(link netlink.Link, hwaddr net.HardwareAddr) error {
	return withErrDescr(netlink.LinkSetHardwareAddr(link, hwaddr), "LinkSetHardwareAddr")
}

func (n NetLink) LinkSetMTU(link netlink.Link, mtu int) error {
	return withErrDescr(netlink.LinkSetMTU(link, mtu), "LinkSetMTU")
}

func (n NetLink) LinkSetDown(link netlink.Link) error {
	return withErrDescr(netlink.LinkSetDown(link), "LinkSetDown")
}

func (n NetLink) LinkSetUp(link netlink.Link) error {
	return withErrDescr(netlink.LinkSetUp(link), "LinkSetUp")
}

func (n NetLink) LinkSetName(link netlink.Link, name string) error {
	return withErrDescr(netlink.LinkSetName(link, name), "LinkSetName")
}

func (n NetLink) LinkAdd(link netlink.Link) error {
	return withErrDescr(netlink.LinkAdd(link), "LinkAdd")
}

func (n NetLink) LinkDel(link netlink.Link) error {
	return withErrDescr(netlink.LinkDel(link), "LinkDel")
}

func (n NetLink) LinkSetLearningOff(link netlink.Link) error {
	return withErrDescr(netlink.LinkSetLearning(link, false), "LinkSetLearningOff")
}

func (n NetLink) LinkGetProtinfo(link netlink.Link) (netlink.Protinfo, error) {
	return netlink.LinkGetProtinfo(link)
}

func (n NetLink) LinkSetMaster(link netlink.Link, master *netlink.Bridge) error {
	return withErrDescr(netlink.LinkSetMaster(link, master), "LinkSetMaster")
}

func withErrDescr(err error, description string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", description, err)
	}
	return nil
}
