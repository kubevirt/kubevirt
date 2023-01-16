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

package procsys

import (
	"fmt"
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
