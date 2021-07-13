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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package infraconfigurators

import (
	"fmt"
	"net"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

type mockNetfilterFrontend func(*netdriver.MockNetworkHandler, iptables.Protocol, string, string, string)

var _ = Describe("Masquerade infrastructure configurator", func() {
	var (
		ctrl    *gomock.Controller
		handler *netdriver.MockNetworkHandler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		handler = netdriver.NewMockNetworkHandler(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	const (
		bridgeIfaceName = "k6t-eth0"
	)

	newVMIMasqueradeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := v1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}

	Context("discover link information", func() {
		const (
			expectedVMInternalIPStr   = "10.0.2.2/24"
			expectedVMGatewayIPStr    = "10.0.2.1/24"
			expectedVMInternalIPv6Str = "fd10:0:2::2/120"
			expectedVMGatewayIPv6Str  = "fd10:0:2::1/120"
			ifaceName                 = "eth0"
			launcherPID               = 1000
		)

		var (
			masqueradeConfigurator *MasqueradePodNetworkConfigurator
			podLink                *netlink.GenericLink
			vmi                    *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			vmi = newVMIMasqueradeInterface("default", "vm1")
			masqueradeConfigurator = NewMasqueradePodNetworkConfigurator(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], bridgeIfaceName, &vmi.Spec.Networks[0], launcherPID, handler)
		})

		When("the pod link is defined", func() {
			BeforeEach(func() {
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 1000}}
				handler.EXPECT().LinkByName(ifaceName).Return(podLink, nil)
			})

			When("the pod interface has an IPv4 address", func() {
				When("and is missing an IPv6 address", func() {
					BeforeEach(func() {
						handler.EXPECT().IsIpv6Enabled(ifaceName).Return(false, nil)
					})

					It("should succeed discovering the pod link info", func() {
						Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
						Expect(masqueradeConfigurator.podNicLink).To(Equal(podLink))
						expectedGwIP, _ := netlink.ParseAddr(expectedVMGatewayIPStr)
						Expect(masqueradeConfigurator.vmGatewayAddr).To(Equal(expectedGwIP))
						expectedVMIP, _ := netlink.ParseAddr(expectedVMInternalIPStr)
						Expect(masqueradeConfigurator.vmIPv4Addr).To(Equal(*expectedVMIP))
					})
				})

				When("and we fail to understand if there's an IPv6 configuration", func() {
					BeforeEach(func() {
						handler.EXPECT().IsIpv6Enabled(ifaceName).Return(true, fmt.Errorf("failed to check pod's IPv6 configuration"))
					})

					It("should fail to discover the pod's link information", func() {
						Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
					})
				})
			})

			When("the pod interface has both IPv4 and IPv6 addresses", func() {
				BeforeEach(func() {
					handler.EXPECT().IsIpv6Enabled(ifaceName).Return(true, nil)
				})

				It("should succeed reading the pod link info", func() {
					Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
					Expect(masqueradeConfigurator.podNicLink).To(Equal(podLink))
					expectedGwIP, _ := netlink.ParseAddr(expectedVMGatewayIPStr)
					Expect(masqueradeConfigurator.vmGatewayAddr).To(Equal(expectedGwIP))
					expectedVMIP, _ := netlink.ParseAddr(expectedVMInternalIPStr)
					Expect(masqueradeConfigurator.vmIPv4Addr).To(Equal(*expectedVMIP))
					expectedGwIPv6, _ := netlink.ParseAddr(expectedVMGatewayIPv6Str)
					Expect(masqueradeConfigurator.vmGatewayIpv6Addr).To(Equal(expectedGwIPv6))
					expectedVMIPv6, _ := netlink.ParseAddr(expectedVMInternalIPv6Str)
					Expect(masqueradeConfigurator.vmIPv6Addr).To(Equal(*expectedVMIPv6))
				})
			})
		})

		When("the pod link information cannot be retrieved", func() {
			BeforeEach(func() {
				handler.EXPECT().LinkByName(ifaceName).Return(nil, fmt.Errorf("cannot get pod link"))
			})

			It("should fail to discover the pod's link information", func() {
				Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
			})
		})
	})

	Context("preparing network infrastructure", func() {
		const (
			ifaceName     = "eth0"
			ipv6GwStr     = "fd10:0:2::1/120"
			launcherPID   = 1000
			mtu           = 1000
			namespace     = "default"
			queueCount    = uint32(0)
			tapDeviceName = "tap0"
			vmIPv6Str     = "fd10:0:2::2/120"
			vmName        = "vm1"
		)

		var (
			inPodBridge     *netlink.Bridge
			podLink         *netlink.GenericLink
			gatewayAddr     *netlink.Addr
			podIP           netlink.Addr
			gatewayIPv6Addr *netlink.Addr
			podIPv6         *netlink.Addr
			vmi             *v1.VirtualMachineInstance
			dhcpConfig      *cache.DHCPConfig
		)

		BeforeEach(func() {
			vmi = newVMIMasqueradeInterface(namespace, vmName)
			podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
			gatewayAddr = &netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 0, 2, 1), Mask: net.CIDRMask(24, 32)}}
			podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 0, 2, 2), Mask: net.CIDRMask(24, 32)}}
			podIPv6, _ = netlink.ParseAddr(vmIPv6Str)
			gatewayIPv6Addr, _ = netlink.ParseAddr(ipv6GwStr)
			inPodBridge = podBridge(bridgeIfaceName, mtu)
			dhcpConfig = expectedDhcpConfig(ifaceName, podIP, *gatewayAddr, vmIPv6Str, ipv6GwStr, mtu)
		})

		When("the pod features a properly configured primary link", func() {
			table.DescribeTable("should work with", func(mockNetfilterFrontendFunc mockNetfilterFrontend, additionalIPProtocol ...iptables.Protocol) {
				masqueradeConfigurator := newMockedMasqueradeConfigurator(
					vmi,
					&vmi.Spec.Domain.Devices.Interfaces[0],
					bridgeIfaceName,
					&vmi.Spec.Networks[0],
					launcherPID,
					handler,
					podLink,
					podIP,
					*gatewayAddr,
					*podIPv6,
					*gatewayIPv6Addr)
				mockCreateMasqueradeInfraCreation(handler, inPodBridge, tapDeviceName, queueCount, launcherPID, mtu)
				mockVML3Config(*masqueradeConfigurator, ifaceName, inPodBridge, additionalIPProtocol...)
				mockNATNetfilterRules(handler, *dhcpConfig, mockNetfilterFrontendFunc, additionalIPProtocol...)

				Expect(masqueradeConfigurator.PreparePodNetworkInterface()).To(Succeed())
			},
				table.Entry("NFTables backend on an IPv4 cluster", mockNetfilterNFTables),
				table.Entry("IPTables backend on an IPv4 cluster", mockNetfilterIPTables),
				table.Entry("NFTables backend on a dual stack cluster", mockNetfilterNFTables, iptables.ProtocolIPv6),
				table.Entry("IPTables backend on a dual stack cluster", mockNetfilterIPTables, iptables.ProtocolIPv6),
			)
		})
	})
})

func podBridge(ifaceName string, mtu int) *netlink.Bridge {
	inPodBridgeMAC, _ := net.ParseMAC("02:00:00:00:00:00")
	return &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu, HardwareAddr: inPodBridgeMAC}}
}

func expectedDhcpConfig(ifaceName string, podIP netlink.Addr, gatewayAddr netlink.Addr, podIPv6Addr string, gatewayIPv6Addr string, mtu int) *cache.DHCPConfig {
	ipv6GwAddr, _ := netlink.ParseAddr(gatewayIPv6Addr)
	ipv6VmAddr, _ := netlink.ParseAddr(podIPv6Addr)
	return &cache.DHCPConfig{
		Name:                ifaceName,
		IP:                  podIP,
		IPv6:                *ipv6VmAddr,
		Mtu:                 uint16(mtu),
		AdvertisingIPAddr:   gatewayAddr.IP.To4(),
		AdvertisingIPv6Addr: ipv6GwAddr.IP,
	}
}

func newMockedMasqueradeConfigurator(
	vmi *v1.VirtualMachineInstance,
	iface *v1.Interface,
	bridgeIfaceName string,
	network *v1.Network,
	launcherPID int,
	handler *netdriver.MockNetworkHandler,
	link netlink.Link,
	podIP netlink.Addr,
	gatewayIP netlink.Addr,
	ipv6PodIP netlink.Addr,
	ipv6GatewayAddr netlink.Addr) *MasqueradePodNetworkConfigurator {

	mc := NewMasqueradePodNetworkConfigurator(vmi, iface, bridgeIfaceName, network, launcherPID, handler)
	mc.podNicLink = link
	mc.vmGatewayAddr = &gatewayIP
	mc.vmIPv4Addr = podIP
	mc.vmGatewayIpv6Addr = &ipv6GatewayAddr
	mc.vmIPv6Addr = ipv6PodIP
	return mc
}

func mockCreateMasqueradeInfraCreation(handler *netdriver.MockNetworkHandler, bridge *netlink.Bridge, tapName string, queueCout uint32, launcherPID int, mtu int) {
	handler.EXPECT().LinkAdd(bridge).Return(nil)
	handler.EXPECT().LinkSetUp(bridge).Return(nil)
	handler.EXPECT().DisableTXOffloadChecksum(bridge.Name).Return(nil)
	handler.EXPECT().CreateTapDevice(tapName, queueCout, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
	handler.EXPECT().BindTapDeviceToBridge(tapName, bridge.Name).Return(nil)
}

func mockVML3Config(configurator MasqueradePodNetworkConfigurator, podIface string, inPodBridge *netlink.Bridge, optionalIPProtocol ...iptables.Protocol) {
	protos := protocols(optionalIPProtocol...)
	hasIPv6Config := len(protos) > 1
	mockedHandler := configurator.handler.(*netdriver.MockNetworkHandler)
	mockedHandler.EXPECT().IsIpv6Enabled(podIface).Return(hasIPv6Config, nil).Times(2) // once on create bridge, another on prepare pod network

	for _, l3Protocol := range protos {
		gatewayAddr := configurator.vmGatewayAddr
		if l3Protocol == iptables.ProtocolIPv6 {
			gatewayAddr = configurator.vmGatewayIpv6Addr
		}
		mockedHandler.EXPECT().AddrAdd(inPodBridge, gatewayAddr).Return(nil)
	}
}

func mockNATNetfilterRules(handler *netdriver.MockNetworkHandler, dhcpConfig cache.DHCPConfig, mockFrontendFunc mockNetfilterFrontend, optionalIPProtocol ...iptables.Protocol) {
	getNFTIPString := func(proto iptables.Protocol) string {
		ipString := "ip"
		if proto == iptables.ProtocolIPv6 {
			ipString = "ip6"
		}
		return ipString
	}

	for _, proto := range protocols(optionalIPProtocol...) {
		vmIP := dhcpConfig.IP.IP.String()
		gwIP := dhcpConfig.AdvertisingIPAddr.String()
		if proto == iptables.ProtocolIPv6 {
			vmIP = dhcpConfig.IPv6.IP.String()
			gwIP = dhcpConfig.AdvertisingIPv6Addr.String()
		}
		handler.EXPECT().ConfigureIpForwarding(proto).Return(nil)
		mockFrontendFunc(handler, proto, getNFTIPString(proto), vmIP, gwIP)
	}
}

func mockNetfilterIPTables(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIPString string, vmIP string, gwIP string) {
	handler.EXPECT().NftablesLoad(proto).Return(fmt.Errorf("nft not found"))
	handler.EXPECT().HasNatIptables(proto).Return(true)
	mockIPTablesBackend(handler, proto, nftIPString, vmIP, gwIP)
}

func mockNetfilterNFTables(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIPString string, vmIP string, gwIP string) {
	handler.EXPECT().NftablesLoad(proto).Return(nil)
	handler.EXPECT().HasNatIptables(proto).Return(true).Times(0)
	handler.EXPECT().HasNatIptables(proto).Return(false).Times(0)
	mockNFTablesFrontend(handler, proto, nftIPString, vmIP, gwIP)
}

func mockNFTablesFrontend(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIPString string, vmIP string, gwIP string) {
	handler.EXPECT().GetNFTIPString(proto).Return(nftIPString).AnyTimes()
	handler.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
	handler.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)

	handler.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", nftIPString, "saddr", vmIP, "counter", "masquerade").Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "prerouting", "iifname", "eth0", "counter", "jump", "KUBEVIRT_PREINBOUND").Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", "oifname", "k6t-eth0", "counter", "jump", "KUBEVIRT_POSTINBOUND").Return(nil)

	for _, chain := range []string{"output", "KUBEVIRT_POSTINBOUND"} {
		handler.EXPECT().NftablesAppendRule(proto, "nat", chain, "tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(PortsUsedByLiveMigration(), ", ")), nftIPString, "saddr", GetLoopbackAdrress(proto), "counter", "return").Return(nil)
	}

	handler.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND", "counter", "dnat", "to", vmIP).Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND", nftIPString, "saddr", fmt.Sprintf("{ %s }", GetLoopbackAdrress(proto)), "counter", "snat", "to", gwIP).Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "output", nftIPString, "daddr", fmt.Sprintf("{ %s }", GetLoopbackAdrress(proto)), "counter", "dnat", "to", vmIP).Return(nil)
}

func mockIPTablesBackend(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIPString string, vmIP string, gwIP string) {
	handler.EXPECT().GetNFTIPString(proto).Return(nftIPString).AnyTimes()
	handler.EXPECT().IptablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
	handler.EXPECT().IptablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)

	handler.EXPECT().IptablesAppendRule(proto, "nat",
		"POSTROUTING",
		"-s",
		vmIP,
		"-j",
		"MASQUERADE").Return(nil)
	handler.EXPECT().IptablesAppendRule(proto, "nat",
		"PREROUTING",
		"-i",
		"eth0",
		"-j",
		"KUBEVIRT_PREINBOUND").Return(nil)
	handler.EXPECT().IptablesAppendRule(proto, "nat",
		"POSTROUTING",
		"-o",
		"k6t-eth0",
		"-j",
		"KUBEVIRT_POSTINBOUND").Return(nil)
	handler.EXPECT().IptablesAppendRule(proto, "nat",
		"KUBEVIRT_PREINBOUND",
		"-j",
		"DNAT",
		"--to-destination",
		vmIP).Return(nil)
	handler.EXPECT().IptablesAppendRule(proto, "nat",
		"KUBEVIRT_POSTINBOUND",
		"--source",
		GetLoopbackAdrress(proto),
		"-j",
		"SNAT",
		"--to-source",
		gwIP).Return(nil)
	handler.EXPECT().IptablesAppendRule(proto, "nat",
		"OUTPUT",
		"--destination",
		GetLoopbackAdrress(proto),
		"-j",
		"DNAT",
		"--to-destination",
		vmIP).Return(nil)

	for _, chain := range []string{"OUTPUT", "KUBEVIRT_POSTINBOUND"} {
		handler.EXPECT().IptablesAppendRule(proto, "nat", chain,
			"-p", "tcp", "--match", "multiport",
			"--dports", fmt.Sprintf("%s", strings.Join(PortsUsedByLiveMigration(), ",")),
			"--source", GetLoopbackAdrress(proto), "-j", "RETURN").Return(nil)
	}
}

func protocols(optionalIPProtocol ...iptables.Protocol) []iptables.Protocol {
	return append(
		[]iptables.Protocol{iptables.ProtocolIPv4},
		optionalIPProtocol...)
}
