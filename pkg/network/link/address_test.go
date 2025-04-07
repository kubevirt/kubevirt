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

package link

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

var _ = Describe("Common Methods", func() {
	createNetwork := func(cidr string, ipv6Cidr string) *v1.Network {
		return &v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{
					VMNetworkCIDR:     cidr,
					VMIPv6NetworkCIDR: ipv6Cidr,
				},
			},
		}
	}
	Context("GenerateMasqueradeGatewayAndVmIPAddrs function", func() {
		It("Should return 2 addresses", func() {
			gw, vm, err := GenerateMasqueradeGatewayAndVmIPAddrs(createNetwork("10.0.0.0/30", ""), netdriver.IPv4)
			Expect(err).ToNot(HaveOccurred())
			Expect(gw.IPNet.String()).To(Equal("10.0.0.1/30"))
			Expect(vm.IPNet.String()).To(Equal("10.0.0.2/30"))
		})
		It("Should return 2 IPV6 addresses", func() {
			gw, vm, err := GenerateMasqueradeGatewayAndVmIPAddrs(createNetwork("", "fd10:0:2::/120"), netdriver.IPv6)
			Expect(err).ToNot(HaveOccurred())
			Expect(gw.IPNet.String()).To(Equal("fd10:0:2::1/120"))
			Expect(vm.IPNet.String()).To(Equal("fd10:0:2::2/120"))
		})
		It("Should return 2 default addresses", func() {
			gw, vm, err := GenerateMasqueradeGatewayAndVmIPAddrs(createNetwork("", ""), netdriver.IPv4)
			Expect(err).ToNot(HaveOccurred())
			Expect(gw.IPNet.String()).To(Equal("10.0.2.1/24"))
			Expect(vm.IPNet.String()).To(Equal("10.0.2.2/24"))
		})
		It("Should return 2 default IPV6 addresses", func() {
			gw, vm, err := GenerateMasqueradeGatewayAndVmIPAddrs(createNetwork("", ""), netdriver.IPv6)
			Expect(err).ToNot(HaveOccurred())
			Expect(gw.IPNet.String()).To(Equal("fd10:0:2::1/120"))
			Expect(vm.IPNet.String()).To(Equal("fd10:0:2::2/120"))
		})
		It("Should fail when the subnet is too small", func() {
			_, _, err := GenerateMasqueradeGatewayAndVmIPAddrs(createNetwork("10.0.0.0/31", ""), netdriver.IPv4)
			Expect(err).To(HaveOccurred())
		})
		It("Should fail when the IPV6 subnet is too small", func() {
			_, _, err := GenerateMasqueradeGatewayAndVmIPAddrs(createNetwork("", "fd10:0:2::/127"), netdriver.IPv6)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("RetrieveMacAddressFromVMISpecIface function", func() {
		It("Should return nil when the spec doesn't contain a MAC address", func() {
			iface := &v1.Interface{}
			mac, err := RetrieveMacAddressFromVMISpecIface(iface)
			Expect(err).ToNot(HaveOccurred())
			Expect(mac).To(BeNil())
		})
		It("Should return an error if the spec contain MAC address with wrong format", func() {
			iface := &v1.Interface{
				MacAddress: "abcd",
			}
			mac, err := RetrieveMacAddressFromVMISpecIface(iface)
			Expect(err).To(HaveOccurred())
			Expect(mac).To(BeNil())
		})

		DescribeTable("Should return the spec parsed MAC address", func(rawMACAddress string) {
			iface := &v1.Interface{
				MacAddress: rawMACAddress,
			}
			mac, err := RetrieveMacAddressFromVMISpecIface(iface)
			Expect(err).ToNot(HaveOccurred())
			expectedMac, _ := net.ParseMAC(rawMACAddress)
			Expect(mac).To(Equal(&expectedMac))
		},
			Entry("lowercase and colon separated", "de:ad:00:00:be:af"),
			Entry("uppercase and colon separated", "DE:AD:00:00:BE:AF"),
			Entry("lowercase and dash separated", "de-ad-00-00-be-af"),
			Entry("uppercase and dash separated", "DE-AD-00-00-BE-AF"),
		)
	})
	Context("GetFakeBridgeIP function", func() {
		It("Should return empty string when interface name is not in the interface list", func() {
			ip := GetFakeBridgeIP([]v1.Interface{v1.Interface{Name: "aaaa"}}, &v1.Interface{Name: "abcd"})
			Expect(ip).To(Equal(""))
		})
		It("Should return the correct ip when the interface is the first in the list", func() {
			ip := GetFakeBridgeIP([]v1.Interface{v1.Interface{Name: "abcd"}}, &v1.Interface{Name: "abcd"})
			Expect(ip).To(Equal(fmt.Sprintf(bridgeFakeIP, 0)))
		})
		It("Should return the correct ip when the interface is not the first in the list", func() {
			ip := GetFakeBridgeIP([]v1.Interface{v1.Interface{Name: "aaaa"}, v1.Interface{Name: "abcd"}}, &v1.Interface{Name: "abcd"})
			Expect(ip).To(Equal(fmt.Sprintf(bridgeFakeIP, 1)))
		})
	})

	Context("FilterPodNetworkRoutes function", func() {
		const (
			mac = "12:34:56:78:9A:BC"
		)

		defRoute := netlink.Route{
			Gw: net.IPv4(10, 35, 0, 1),
		}
		staticRoute := netlink.Route{
			Dst: &net.IPNet{IP: net.IPv4(10, 45, 0, 10), Mask: net.CIDRMask(32, 32)},
			Gw:  net.IPv4(10, 25, 0, 1),
		}
		gwRoute := netlink.Route{
			Dst: &net.IPNet{IP: net.IPv4(10, 35, 0, 1), Mask: net.CIDRMask(32, 32)},
		}
		nicRoute := netlink.Route{Src: net.IPv4(10, 35, 0, 6)}
		emptyRoute := netlink.Route{}
		staticRouteList := []netlink.Route{defRoute, gwRoute, nicRoute, emptyRoute, staticRoute}

		address := &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}
		fakeMac, _ := net.ParseMAC(mac)
		testDhcpConfig := &cache.DHCPConfig{
			Name:              namescheme.PrimaryPodInterfaceName,
			IP:                netlink.Addr{IPNet: address},
			MAC:               fakeMac,
			Mtu:               uint16(1410),
			AdvertisingIPAddr: net.IPv4(10, 35, 0, 1),
		}

		It("should remove empty routes, and routes matching nic, leaving others intact", func() {
			expectedRouteList := []netlink.Route{defRoute, gwRoute, staticRoute}
			Expect(FilterPodNetworkRoutes(staticRouteList, testDhcpConfig)).To(Equal(expectedRouteList))
		})
	})
})
