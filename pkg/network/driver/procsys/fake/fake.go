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

package fake

import "kubevirt.io/kubevirt/pkg/network/driver/procsys"

type ProcSys struct {
	linuxStackData netStackData
	ifacesData     map[string]netIfData
}

type netStackData struct {
	arpIgnoreMode   procsys.ArpReplyMode
	ipv4Forwarding  bool
	ipv6Forwarding  bool
	pingGroupRange  [2]int
	unprivPortStart int
}

type netIfData struct {
	routeLocalNet bool
}

func New() *ProcSys {
	return &ProcSys{ifacesData: map[string]netIfData{}}
}

func (p *ProcSys) IPv4GetForwarding() (bool, error) {
	return p.linuxStackData.ipv4Forwarding, nil
}

func (p *ProcSys) IPv4EnableForwarding() error {
	p.linuxStackData.ipv4Forwarding = true
	return nil
}

func (p *ProcSys) IPv6GetForwarding() (bool, error) {
	return p.linuxStackData.ipv6Forwarding, nil
}

func (p *ProcSys) IPv6EnableForwarding() error {
	p.linuxStackData.ipv6Forwarding = true
	return nil
}

func (p *ProcSys) IPv4GetPingGroupRange() (int, int, error) {
	r := p.linuxStackData.pingGroupRange
	return r[0], r[1], nil
}

func (p *ProcSys) IPv4SetPingGroupRange(from int, to int) error {
	p.linuxStackData.pingGroupRange = [2]int{from, to}
	return nil
}

func (p *ProcSys) IPv4GetUnprivilegedPortStart() (int, error) {
	return p.linuxStackData.unprivPortStart, nil
}

func (p *ProcSys) IPv4SetUnprivilegedPortStart(port int) error {
	p.linuxStackData.unprivPortStart = port
	return nil
}

func (p *ProcSys) IPv4GetArpIgnore(ifaceName string) (procsys.ArpReplyMode, error) {
	if ifaceName == "all" {
		return p.linuxStackData.arpIgnoreMode, nil
	}
	return 0, nil
}

func (p *ProcSys) IPv4SetArpIgnore(ifaceName string, mode procsys.ArpReplyMode) error {
	if ifaceName == "all" {
		p.linuxStackData.arpIgnoreMode = mode
	}
	return nil
}

func (p *ProcSys) IPv4GetRouteLocalNet(ifaceName string) (bool, error) {
	if d, exists := p.ifacesData[ifaceName]; exists {
		return d.routeLocalNet, nil
	}
	return false, nil
}

func (p *ProcSys) IPv4EnableRouteLocalNet(ifaceName string) error {
	if _, exists := p.ifacesData[ifaceName]; !exists {
		p.ifacesData[ifaceName] = netIfData{}
	}
	d := p.ifacesData[ifaceName]
	d.routeLocalNet = true
	p.ifacesData[ifaceName] = d
	return nil
}
