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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package masquerade

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/driver/nft"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/netmachinery"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
)

type nftable interface {
	AddTable(family nft.IPFamily, name string) error
	AddChain(family nft.IPFamily, table, name string, chainspec ...string) error
	AddRule(family nft.IPFamily, table, chain string, rulespec ...string) error
}

type MasqPod struct {
	nftable        nftable
	istioEnabled   bool
	migrationPorts []int
}

const (
	natTable = "nat"

	preroutingChain          = "prerouting"
	postroutingChain         = "postrouting"
	inputChain               = "input"
	outputChain              = "output"
	kubevirtPreInboundChain  = "KUBEVIRT_PREINBOUND"
	kubevirtPostInboundChain = "KUBEVIRT_POSTINBOUND"
)

type option func(*MasqPod)

func New(opts ...option) MasqPod {
	m := MasqPod{nftable: nft.NFTBin{}}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func WithIstio(enabled bool) option {
	return func(m *MasqPod) {
		m.istioEnabled = enabled
	}
}

func WithNftableAdapter(h nftable) option {
	return func(m *MasqPod) {
		m.nftable = h
	}
}

// WithLegacyMigrationPorts is used for legacy setups where migration ports are in use
// When set, the configuration should skip forwarding for the reserved migration ports.
func WithLegacyMigrationPorts() option {
	const LibvirtDirectMigrationPort = 49152
	const LibvirtBlockMigrationPort = 49153
	return func(m *MasqPod) {
		m.migrationPorts = []int{LibvirtDirectMigrationPort, LibvirtBlockMigrationPort}
	}
}

func (m MasqPod) Setup(bridgeIfaceSpec, podIfaceSpec *nmstate.Interface, vmiIface v1.Interface) error {
	if bridgeIfaceSpec.IPv4.Enabled != nil && *bridgeIfaceSpec.IPv4.Enabled {
		if err := m.setupNATByFamily(nft.IPv4, podIfaceSpec, bridgeIfaceSpec, vmiIface); err != nil {
			return err
		}
	}
	if bridgeIfaceSpec.IPv6.Enabled != nil && *bridgeIfaceSpec.IPv6.Enabled {
		if err := m.setupNATByFamily(nft.IPv6, podIfaceSpec, bridgeIfaceSpec, vmiIface); err != nil {
			return err
		}
	}
	return nil
}

func (m MasqPod) setupNATByFamily(family nft.IPFamily, podIfaceSpec, bridgeIfaceSpec *nmstate.Interface, vmiIface v1.Interface) error {

	if err := m.nftable.AddTable(family, natTable); err != nil {
		return err
	}
	if err := m.nftable.AddChain(family, natTable, preroutingChain, "{ type nat hook prerouting priority -100; }"); err != nil {
		return err
	}
	if err := m.nftable.AddChain(family, natTable, inputChain, "{ type nat hook input priority 100; }"); err != nil {
		return err
	}
	if err := m.nftable.AddChain(family, natTable, outputChain, "{ type nat hook output priority -100; }"); err != nil {
		return err
	}
	if err := m.nftable.AddChain(family, natTable, postroutingChain, "{ type nat hook postrouting priority 100; }"); err != nil {
		return err
	}
	if err := m.nftable.AddChain(family, natTable, kubevirtPreInboundChain); err != nil {
		return err
	}
	if err := m.nftable.AddChain(family, natTable, kubevirtPostInboundChain); err != nil {
		return err
	}

	guestIP := guestIPByGatewayInterface(family, *bridgeIfaceSpec)
	if err := m.nftable.AddRule(family, natTable, postroutingChain, string(family), "saddr", guestIP, "counter", "masquerade"); err != nil {
		return err
	}
	if err := m.nftable.AddRule(family, natTable, preroutingChain, "iifname", podIfaceSpec.Name, "counter", "jump", kubevirtPreInboundChain); err != nil {
		return err
	}
	if err := m.nftable.AddRule(family, natTable, postroutingChain, "oifname", bridgeIfaceSpec.Name, "counter", "jump", kubevirtPostInboundChain); err != nil {
		return err
	}

	if len(m.migrationPorts) > 0 {
		if err := m.skipForwardPorts(family, m.migrationPorts...); err != nil {
			return err
		}
	}

	addressesToDnat := []string{ipLoopback(family)}
	if m.istioEnabled && family == nft.IPv4 {
		addressesToDnat = append(addressesToDnat, podIfaceSpec.IPv4.Address[0].IP)
	}
	addressesToDnatSpec := fmt.Sprintf("{ %s }", strings.Join(addressesToDnat, ", "))

	for _, port := range vmiIface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}
		protocol := strings.ToLower(port.Protocol)
		addressesToSnat := []string{ipLoopback(family)}

		if m.istioEnabled {
			var portsToForward []int
			for _, nonProxiedPort := range istio.NonProxiedPorts() {
				if int(port.Port) == nonProxiedPort {
					portsToForward = append(portsToForward, nonProxiedPort)
				}
			}
			if err := m.forwardPorts(family, guestIP, "tcp", portsToForward...); err != nil {
				return err
			}

			if family == nft.IPv4 {
				addressesToSnat = append(addressesToSnat, istio.GetLoopbackAddress())
			}
		} else {
			if err := m.forwardPorts(family, guestIP, protocol, int(port.Port)); err != nil {
				return err
			}
		}

		addressesToSnatSpec := fmt.Sprintf("{ %s }", strings.Join(addressesToSnat, ", "))
		gw := guestIPGateway(family, *bridgeIfaceSpec).String()
		if err := m.nftable.AddRule(family, natTable, kubevirtPostInboundChain, protocol, "dport", strconv.Itoa(int(port.Port)), string(family), "saddr", addressesToSnatSpec, "counter", "snat", "to", gw); err != nil {
			return err
		}

		if err := m.nftable.AddRule(family, natTable, outputChain, string(family), "daddr", addressesToDnatSpec, protocol, "dport", strconv.Itoa(int(port.Port)), "counter", "dnat", "to", guestIP); err != nil {
			return err
		}
	}

	if len(vmiIface.Ports) == 0 {
		addressesToSnat := []string{ipLoopback(family)}
		if m.istioEnabled {
			// Skip forwarding for the reserved istio ports
			if err := m.skipForwardPorts(family, istio.ReservedPorts()...); err != nil {
				return err
			}
			if err := m.forwardPorts(family, guestIP, "tcp", istio.NonProxiedPorts()...); err != nil {
				return err
			}
			if family == nft.IPv4 {
				addressesToSnat = append(addressesToSnat, istio.GetLoopbackAddress())
			}
		} else {
			if err := m.nftable.AddRule(family, natTable, kubevirtPreInboundChain, "counter", "dnat", "to", guestIP); err != nil {
				return err
			}
		}
		addressesToSnatSpec := fmt.Sprintf("{ %s }", strings.Join(addressesToSnat, ", "))
		gw := guestIPGateway(family, *bridgeIfaceSpec).String()
		if err := m.nftable.AddRule(family, natTable, kubevirtPostInboundChain, string(family), "saddr", addressesToSnatSpec, "counter", "snat", "to", gw); err != nil {
			return err
		}
		if err := m.nftable.AddRule(family, natTable, outputChain, string(family), "daddr", addressesToDnatSpec, "counter", "dnat", "to", guestIP); err != nil {
			return err
		}
	}

	return nil
}

func (m MasqPod) skipForwardPorts(family nft.IPFamily, ports ...int) error {
	loopback := ipLoopback(family)
	fmtPorts := formatPorts(ports)
	portsSpec := fmt.Sprintf("{ %s }", strings.Join(fmtPorts, ", "))
	if err := m.nftable.AddRule(family, natTable, outputChain, "tcp", "dport", portsSpec, string(family), "saddr", loopback, "counter", "return"); err != nil {
		return fmt.Errorf("failed to define skip forwarding for: %s/%s, err: %v", family, fmtPorts, err)
	}
	if err := m.nftable.AddRule(family, natTable, kubevirtPostInboundChain, "tcp", "dport", portsSpec, string(family), "saddr", loopback, "counter", "return"); err != nil {
		return fmt.Errorf("failed to define skip forwarding for: %s/%s, err: %v", family, fmtPorts, err)
	}
	return nil
}

func formatPorts(ports []int) []string {
	var formattedPorts []string
	for _, p := range ports {
		formattedPorts = append(formattedPorts, strconv.Itoa(p))
	}
	return formattedPorts
}

func (m MasqPod) forwardPorts(family nft.IPFamily, toIP string, protocol string, ports ...int) error {
	if len(ports) == 0 {
		return nil
	}
	p := strings.Trim(strings.Replace(fmt.Sprint(ports), " ", ", ", -1), "[]")
	portsSpec := fmt.Sprintf("{ %s }", p)
	return m.nftable.AddRule(family, natTable, kubevirtPreInboundChain, protocol, "dport", portsSpec, "counter", "dnat", "to", toIP)
}

func ipLoopback(family nft.IPFamily) string {
	if family == nft.IPv4 {
		return ip.IPv4Loopback
	}
	return net.IPv6loopback.String()
}

// guestIPByGatewayInterface calculates and returns the expected guest IP.
// The bridge IP is the guest default gateway and the next address is the one expected on the guest interface.
func guestIPByGatewayInterface(family nft.IPFamily, bridgeIface nmstate.Interface) string {
	ipAddr := guestIPGateway(family, bridgeIface)
	netmachinery.NextIP(ipAddr)
	return ipAddr.String()
}

func guestIPGateway(family nft.IPFamily, bridgeIface nmstate.Interface) net.IP {
	var ipAddr net.IP
	switch family {
	case nft.IPv4:
		ipAddr = net.ParseIP(bridgeIface.IPv4.Address[0].IP)
	case nft.IPv6:
		ipAddr = net.ParseIP(bridgeIface.IPv6.Address[0].IP)
	}
	return ipAddr
}
