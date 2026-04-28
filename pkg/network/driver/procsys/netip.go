/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package procsys

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"kubevirt.io/kubevirt/pkg/util/sysctl"
)

type ProcSys struct{}

var sysCtl = sysctl.New()

var (
	enable = "1"
)

func (p ProcSys) IPv4SetArpIgnore(iface string, replyMode ArpReplyMode) error {
	return sysCtl.SetSysctl(fmt.Sprintf(sysctl.Ipv4ArpIgnore, iface), strconv.Itoa(int(replyMode)))
}

func (p ProcSys) IPv4GetArpIgnore(iface string) (ArpReplyMode, error) {
	val, err := sysCtl.GetSysctl(fmt.Sprintf(sysctl.Ipv4ArpIgnore, iface))
	if err != nil {
		return 0, err
	}
	ival, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return ArpReplyMode(ival), err
}

func (p ProcSys) IPv4EnableForwarding() error {
	return sysCtl.SetSysctl(sysctl.NetIPv4Forwarding, enable)
}

func (p ProcSys) IPv4GetForwarding() (bool, error) {
	val, err := sysCtl.GetSysctl(sysctl.NetIPv4Forwarding)
	return val == enable, err
}

func (p ProcSys) IPv6EnableForwarding() error {
	return sysCtl.SetSysctl(sysctl.NetIPv6Forwarding, enable)
}

func (p ProcSys) IPv6GetForwarding() (bool, error) {
	val, err := sysCtl.GetSysctl(sysctl.NetIPv6Forwarding)

	//when ipv6 has disabled, val will not exist, so need default return false
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return val == enable, err
}

func (p ProcSys) IPv4SetPingGroupRange(from, to int) error {
	return sysCtl.SetSysctl(sysctl.PingGroupRange, fmt.Sprintf("%d %d", from, to))
}

func (p ProcSys) IPv4GetPingGroupRange() (int, int, error) {
	val, err := sysCtl.GetSysctl(sysctl.PingGroupRange)
	if err != nil {
		return 0, 0, err
	}
	var from, to int
	_, err = fmt.Sscanf(val, "%d %d", &from, &to)
	return from, to, err
}

func (p ProcSys) IPv4EnableRouteLocalNet(linkName string) error {
	routeLocalNetForLink := fmt.Sprintf(sysctl.IPv4RouteLocalNet, linkName)
	return sysCtl.SetSysctl(routeLocalNetForLink, enable)
}

func (p ProcSys) IPv4GetRouteLocalNet(linkName string) (bool, error) {
	routeLocalNetForLink := fmt.Sprintf(sysctl.IPv4RouteLocalNet, linkName)
	val, err := sysCtl.GetSysctl(routeLocalNetForLink)
	return val == enable, err
}

func (p ProcSys) IPv4SetUnprivilegedPortStart(port int) error {
	return sysCtl.SetSysctl(sysctl.UnprivilegedPortStart, strconv.Itoa(port))
}

func (p ProcSys) IPv4GetUnprivilegedPortStart() (int, error) {
	val, err := sysCtl.GetSysctl(sysctl.UnprivilegedPortStart)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}
