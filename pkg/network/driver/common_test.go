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

package driver

import (
	"fmt"
	"net"

	"github.com/coreos/go-iptables/iptables"
	"github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

var _ = Describe("Common Methods", func() {
	Context("GetAvailableAddrsFromCIDR function", func() {
		It("Should return 2 addresses", func() {
			networkHandler := NetworkUtilsHandler{}
			gw, vm, err := networkHandler.GetHostAndGwAddressesFromCIDR("10.0.0.0/30")
			Expect(err).ToNot(HaveOccurred())
			Expect(gw).To(Equal("10.0.0.1/30"))
			Expect(vm).To(Equal("10.0.0.2/30"))
		})
		It("Should return 2 IPV6 addresses", func() {
			networkHandler := NetworkUtilsHandler{}
			gw, vm, err := networkHandler.GetHostAndGwAddressesFromCIDR("fd10:0:2::/120")
			Expect(err).ToNot(HaveOccurred())
			Expect(gw).To(Equal("fd10:0:2::1/120"))
			Expect(vm).To(Equal("fd10:0:2::2/120"))
		})
		It("Should fail when the subnet is too small", func() {
			networkHandler := NetworkUtilsHandler{}
			_, _, err := networkHandler.GetHostAndGwAddressesFromCIDR("10.0.0.0/31")
			Expect(err).To(HaveOccurred())
		})
		It("Should fail when the IPV6 subnet is too small", func() {
			networkHandler := NetworkUtilsHandler{}
			_, _, err := networkHandler.GetHostAndGwAddressesFromCIDR("fd10:0:2::/127")
			Expect(err).To(HaveOccurred())
		})
	})
	Context("composeNftablesLoad function", func() {
		table.DescribeTable("should compose the correct command",
			func(protocol iptables.Protocol, protocolVersionNum string) {
				cmd := composeNftablesLoad(protocol)
				Expect(cmd.Path).To(HaveSuffix("nft"))
				Expect(cmd.Args).To(Equal([]string{
					"nft",
					"-f",
					fmt.Sprintf("/etc/nftables/ipv%s-nat.nft", protocolVersionNum)}))
			},
			table.Entry("ipv4", iptables.ProtocolIPv4, "4"),
			table.Entry("ipv6", iptables.ProtocolIPv6, "6"),
		)
	})
})

var _ = Describe("DhcpConfig", func() {
	const ipv4Cidr = "10.0.0.200/24"
	const ipv4Address = "10.0.0.200"
	const ipv4Mask = "ffffff00"
	const ipv6Cidr = "fd10:0:2::2/120"
	const mac = "de:ad:00:00:be:ef"
	const ipv4Gateway = "10.0.0.1"
	const mtu = 1450
	const vifName = "test-vif"

	Context("String", func() {
		It("returns correct string representation", func() {
			vif := createDummyVIF(vifName, ipv4Cidr, ipv4Gateway, "", mac, mtu)
			Expect(vif.String()).To(Equal(fmt.Sprintf("DhcpConfig: { Name: %s, IP: %s, Mask: %s, IPv6: <nil>, MAC: %s, Gateway: %s, MTU: %d, IPAMDisabled: false}", vifName, ipv4Address, ipv4Mask, mac, ipv4Gateway, mtu)))
		})
		It("returns correct string representation with ipv6", func() {
			vif := createDummyVIF(vifName, ipv4Cidr, ipv4Gateway, ipv6Cidr, mac, mtu)
			Expect(vif.String()).To(Equal(fmt.Sprintf("DhcpConfig: { Name: %s, IP: %s, Mask: %s, IPv6: %s, MAC: %s, Gateway: %s, MTU: %d, IPAMDisabled: false}", vifName, ipv4Address, ipv4Mask, ipv6Cidr, mac, ipv4Gateway, mtu)))
		})
	})
})

func createDummyVIF(vifName, ipv4cidr, ipv4gateway, ipv6cidr, macStr string, mtu uint16) *cache.DhcpConfig {
	addr, _ := netlink.ParseAddr(ipv4cidr)
	mac, _ := net.ParseMAC(macStr)
	gw := net.ParseIP(ipv4gateway)
	vif := &cache.DhcpConfig{
		Name:    vifName,
		IP:      *addr,
		MAC:     mac,
		Gateway: gw,
		Mtu:     mtu,
	}
	if ipv6cidr != "" {
		ipv6Addr, _ := netlink.ParseAddr(ipv6cidr)
		vif.IPv6 = *ipv6Addr
	}

	return vif
}

var _ = Describe("infocache", func() {

})
