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

package network

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/coreos/go-iptables/iptables"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const bridgeFakeIP = "169.254.75.1%d/32"

var _ = Describe("Pod Network", func() {
	var mockNetwork *netdriver.MockNetworkHandler
	var ctrl *gomock.Controller
	var dummySwap *netlink.Dummy
	var primaryPodInterface *netlink.GenericLink
	var primaryPodInterfaceAfterNameChange *netlink.GenericLink
	var addrList []netlink.Addr
	var newPodInterfaceName string
	var routeList []netlink.Route
	var routeAddr netlink.Route
	var fakeMac net.HardwareAddr
	var fakeAddr netlink.Addr
	var updateFakeMac net.HardwareAddr
	var bridgeTest *netlink.Bridge
	var masqueradeBridgeTest *netlink.Bridge
	var bridgeAddr *netlink.Addr
	var testNic *cache.DHCPConfig
	var tmpDir string
	var masqueradeTestNic *cache.DHCPConfig
	var masqueradeDummyName string
	var masqueradeDummy *netlink.Dummy
	var masqueradeCidr string
	var masqueradeGwStr string
	var masqueradeGwAddr *netlink.Addr
	var masqueradeGwIp string
	var masqueradeVmStr string
	var masqueradeVmAddr *netlink.Addr
	var masqueradeVmIp string
	var masqueradeIpv6Cidr string
	var masqueradeIpv6GwStr string
	var masqueradeIpv6GwAddr *netlink.Addr
	var masqueradeGwIpv6 string
	var masqueradeIpv6VmStr string
	var masqueradeIpv6VmAddr *netlink.Addr
	var masqueradeVmIpv6 string
	var pid int
	var tapDeviceName string
	var queueNumber uint32
	var mtu int
	var cacheFactory cache.InterfaceCacheFactory
	var libvirtUser string

	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()
		var err error
		tmpDir, err = ioutil.TempDir("/tmp", "interface-cache")
		Expect(err).ToNot(HaveOccurred())
		cacheFactory = cache.NewInterfaceCacheFactoryWithBasePath(tmpDir)

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		testMac := "12:34:56:78:9A:BC"
		updateTestMac := "AF:B3:1F:78:2A:CA"
		mtu = 1410
		newPodInterfaceName = fmt.Sprintf("%s-nic", primaryPodInterfaceName)
		dummySwap = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: primaryPodInterfaceName}}
		primaryPodInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: primaryPodInterfaceName, MTU: mtu}}
		primaryPodInterfaceAfterNameChange = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: newPodInterfaceName, MTU: mtu}}
		address := &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}
		gw := net.IPv4(10, 35, 0, 1)
		fakeMac, _ = net.ParseMAC(testMac)
		updateFakeMac, _ = net.ParseMAC(updateTestMac)
		fakeAddr = netlink.Addr{IPNet: address}
		addrList = []netlink.Addr{fakeAddr}
		routeAddr = netlink.Route{Gw: gw}
		routeList = []netlink.Route{routeAddr}
		pid = os.Getpid()
		tapDeviceName = "tap0"
		libvirtUser = netdriver.LibvirtUserAndGroupId

		// Create a bridge
		bridgeTest = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: api.DefaultBridgeName,
			},
		}

		masqueradeBridgeMAC, _ := net.ParseMAC(link.StaticMasqueradeBridgeMAC)
		masqueradeBridgeTest = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name:         api.DefaultBridgeName,
				MTU:          mtu,
				HardwareAddr: masqueradeBridgeMAC,
			},
		}

		bridgeAddr, _ = netlink.ParseAddr(fmt.Sprintf(bridgeFakeIP, 0))
		tapDeviceName = "tap0"
		testNic = &cache.DHCPConfig{Name: primaryPodInterfaceName,
			IP:                fakeAddr,
			MAC:               fakeMac,
			Mtu:               uint16(mtu),
			AdvertisingIPAddr: gw,
		}

		masqueradeCidr = "10.0.2.0/30"
		masqueradeGwStr = "10.0.2.1/30"
		masqueradeGwAddr, _ = netlink.ParseAddr(masqueradeGwStr)
		masqueradeGwIp = masqueradeGwAddr.IP.String()
		masqueradeVmStr = "10.0.2.2/30"
		masqueradeVmAddr, _ = netlink.ParseAddr(masqueradeVmStr)
		masqueradeVmIp = masqueradeVmAddr.IP.String()
		masqueradeIpv6Cidr = "fd10:0:2::0/120"
		masqueradeIpv6GwStr = "fd10:0:2::1/120"
		masqueradeIpv6GwAddr, _ = netlink.ParseAddr(masqueradeIpv6GwStr)
		masqueradeGwIpv6 = masqueradeIpv6GwAddr.IP.String()
		masqueradeIpv6VmStr = "fd10:0:2::2/120"
		masqueradeIpv6VmAddr, _ = netlink.ParseAddr(masqueradeIpv6VmStr)
		masqueradeVmIpv6 = masqueradeIpv6VmAddr.IP.String()
		masqueradeDummyName = fmt.Sprintf("%s-nic", api.DefaultBridgeName)
		masqueradeDummy = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: masqueradeDummyName, MTU: mtu}}
		masqueradeTestNic = &cache.DHCPConfig{Name: primaryPodInterfaceName,
			IP:                  *masqueradeVmAddr,
			IPv6:                *masqueradeIpv6VmAddr,
			MAC:                 fakeMac,
			Mtu:                 uint16(mtu),
			AdvertisingIPAddr:   masqueradeGwAddr.IP.To4(),
			AdvertisingIPv6Addr: masqueradeIpv6GwAddr.IP.To16()}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	GetMasqueradeVmIp := func(protocol iptables.Protocol) string {
		if protocol == iptables.ProtocolIPv4 {
			return masqueradeVmIp
		}
		return masqueradeVmIpv6
	}

	GetMasqueradeGwIp := func(protocol iptables.Protocol) string {
		if protocol == iptables.ProtocolIPv4 {
			return masqueradeGwIp
		}
		return masqueradeGwIpv6
	}

	GetNFTIPString := func(proto iptables.Protocol) string {
		ipString := "ip"
		if proto == iptables.ProtocolIPv6 {
			ipString = "ip6"
		}
		return ipString
	}

	ipProtocols := func() [2]iptables.Protocol {
		return [2]iptables.Protocol{iptables.ProtocolIPv4, iptables.ProtocolIPv6}
	}

	queueNumber = uint32(0)

	TestPodInterfaceIPBinding := func(vm *v1.VirtualMachineInstance, domain *api.Domain) {
		//For Bridge tests
		mockNetwork.EXPECT().LinkSetName(primaryPodInterface, newPodInterfaceName).Return(nil)
		mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
		mockNetwork.EXPECT().LinkByName(newPodInterfaceName).Return(primaryPodInterfaceAfterNameChange, nil)
		mockNetwork.EXPECT().LinkAdd(dummySwap).Return(nil)
		mockNetwork.EXPECT().AddrReplace(dummySwap, &fakeAddr).Return(nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().RouteList(primaryPodInterface, netlink.FAMILY_V4).Return(routeList, nil)
		mockNetwork.EXPECT().AddrDel(primaryPodInterface, &fakeAddr).Return(nil)
		mockNetwork.EXPECT().LinkSetDown(primaryPodInterface).Return(nil)
		mockNetwork.EXPECT().SetRandomMac(newPodInterfaceName).Return(updateFakeMac, nil)
		mockNetwork.EXPECT().LinkSetUp(primaryPodInterfaceAfterNameChange).Return(nil)
		mockNetwork.EXPECT().LinkSetLearningOff(primaryPodInterfaceAfterNameChange).Return(nil)
		mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
		mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
		mockNetwork.EXPECT().ParseAddr(fmt.Sprintf(bridgeFakeIP, 0)).Return(bridgeAddr, nil)
		mockNetwork.EXPECT().LinkSetMaster(primaryPodInterfaceAfterNameChange, bridgeTest).Return(nil)
		mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(testNic, api.DefaultBridgeName, nil)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
		mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
		mockNetwork.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)

		// For masquerade tests
		mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterface.Name).Return(GetMasqueradeVmIp(iptables.ProtocolIPv4), GetMasqueradeVmIp(iptables.ProtocolIPv6), nil)
		mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeGwStr).Return(masqueradeGwAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeIpv6GwStr).Return(masqueradeIpv6GwAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeVmStr).Return(masqueradeVmAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeIpv6VmStr).Return(masqueradeIpv6VmAddr, nil)
		mockNetwork.EXPECT().LinkAdd(masqueradeDummy).Return(nil)
		mockNetwork.EXPECT().LinkByName(masqueradeDummyName).Return(masqueradeDummy, nil)
		mockNetwork.EXPECT().LinkSetUp(masqueradeDummy).Return(nil)
		mockNetwork.EXPECT().LinkAdd(masqueradeBridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkSetUp(masqueradeBridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkSetMaster(masqueradeDummy, masqueradeBridgeTest).Return(nil)
		mockNetwork.EXPECT().AddrAdd(masqueradeBridgeTest, masqueradeGwAddr).Return(nil)
		mockNetwork.EXPECT().AddrAdd(masqueradeBridgeTest, masqueradeIpv6GwAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(masqueradeTestNic, api.DefaultBridgeName, nil)
		mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
		// Global nat rules using iptables
		for _, proto := range ipProtocols() {
			for _, chain := range []string{"OUTPUT", "KUBEVIRT_POSTINBOUND"} {
				mockNetwork.EXPECT().IptablesAppendRule(proto, "nat", chain,
					"-p", "tcp", "--match", "multiport",
					"--dports", fmt.Sprintf("%s", strings.Join(infraconfigurators.PortsUsedByLiveMigration(), ",")),
					"--source", infraconfigurators.GetLoopbackAdrress(proto), "-j", "RETURN").Return(nil)
			}
		}
		mockNetwork.EXPECT().ConfigureIpForwarding(iptables.ProtocolIPv4).Return(nil)
		mockNetwork.EXPECT().ConfigureIpForwarding(iptables.ProtocolIPv6).Return(nil)
		mockNetwork.EXPECT().GetNFTIPString(iptables.ProtocolIPv4).Return("ip").AnyTimes()
		mockNetwork.EXPECT().GetNFTIPString(iptables.ProtocolIPv6).Return("ip6").AnyTimes()
		for _, proto := range ipProtocols() {
			mockNetwork.EXPECT().IptablesNewChain(proto, "nat", gomock.Any()).Return(nil).Times(2)
			mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
				"POSTROUTING",
				"-s",
				GetMasqueradeVmIp(proto),
				"-j",
				"MASQUERADE").Return(nil)
			mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
				"PREROUTING",
				"-i",
				"eth0",
				"-j",
				"KUBEVIRT_PREINBOUND").Return(nil)
			mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
				"POSTROUTING",
				"-o",
				"k6t-eth0",
				"-j",
				"KUBEVIRT_POSTINBOUND").Return(nil)
			mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
				"KUBEVIRT_PREINBOUND",
				"-j",
				"DNAT",
				"--to-destination",
				GetMasqueradeVmIp(proto)).Return(nil)
			mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
				"KUBEVIRT_POSTINBOUND",
				"--source",
				infraconfigurators.GetLoopbackAdrress(proto),
				"-j",
				"SNAT",
				"--to-source",
				GetMasqueradeGwIp(proto)).Return(nil)
			mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
				"OUTPUT",
				"--destination",
				infraconfigurators.GetLoopbackAdrress(proto),
				"-j",
				"DNAT",
				"--to-destination",
				GetMasqueradeVmIp(proto)).Return(nil)

			//Global net rules using nftable
			mockNetwork.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", GetNFTIPString(proto), "saddr", GetMasqueradeVmIp(proto), "counter", "masquerade").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "prerouting", "iifname", "eth0", "counter", "jump", "KUBEVIRT_PREINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", "oifname", "k6t-eth0", "counter", "jump", "KUBEVIRT_POSTINBOUND").Return(nil)
			for _, chain := range []string{"output", "KUBEVIRT_POSTINBOUND"} {
				mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", chain, "tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(infraconfigurators.PortsUsedByLiveMigration(), ", ")), GetNFTIPString(proto), "saddr", infraconfigurators.GetLoopbackAdrress(proto), "counter", "return").Return(nil)
			}
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND", "counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND", GetNFTIPString(proto), "saddr", fmt.Sprintf("{ %s }", infraconfigurators.GetLoopbackAdrress(proto)), "counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "output", GetNFTIPString(proto), "daddr", fmt.Sprintf("{ %s }", infraconfigurators.GetLoopbackAdrress(proto)), "counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)
		}
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
		podnic, err := newPhase1PodNIC(vm, &vm.Spec.Networks[0], mockNetwork, cacheFactory, &pid)
		Expect(err).ToNot(HaveOccurred())
		err = podnic.PlugPhase1()
		Expect(err).To(BeNil())

		// Calling SetupPhase1 a second time should result in
		// no mockNetwork function calls, as confirmed by mock object
		// limited number of calls expected for each mocked entry point.
		podnic, err = newPhase1PodNIC(vm, &vm.Spec.Networks[0], mockNetwork, cacheFactory, &pid)
		Expect(err).ToNot(HaveOccurred())
		err = podnic.PlugPhase1()
		Expect(err).To(BeNil())
	}

	Context("on successful setup", func() {
		It("should define a new DHCPConfig bind to a bridge", func() {
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			domain := NewDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			TestPodInterfaceIPBinding(vm, domain)
		})

		Context("func filterPodNetworkRoutes()", func() {
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

			It("should remove empty routes, and routes matching nic, leaving others intact", func() {
				expectedRouteList := []netlink.Route{defRoute, gwRoute, staticRoute}
				Expect(netdriver.FilterPodNetworkRoutes(staticRouteList, testNic)).To(Equal(expectedRouteList))
			})
		})
		Context("getPhase1Binding", func() {
			BeforeEach(func() {
				mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
				mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
				mockNetwork.EXPECT().RouteList(primaryPodInterface, netlink.FAMILY_V4).Return(routeList, nil)
			})

			Context("for Bridge", func() {
				It("should populate MAC address", func() {
					vmi := newVMIBridgeInterface("testnamespace", "testVmName")
					vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
					podnic, err := newPhase1PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, &pid)
					Expect(err).ToNot(HaveOccurred())
					Expect(podnic.infraConfigurator.DiscoverPodNetworkInterface(primaryPodInterfaceName)).NotTo(HaveOccurred())
					Expect(podnic.infraConfigurator.GenerateNonRecoverableDHCPConfig().MAC.String()).To(Equal("de:ad:00:00:be:af"))
				})
			})
		})
		Context("Masquerade Plug", func() {
			It("should define a bridge in pod and forward all traffic to VM using iptables", func() {
				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(fmt.Errorf("no nft"))
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
				}
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a bridge in pod and forward specific ports to VM using iptables", func() {
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(fmt.Errorf("no nft"))
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
					mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
						"KUBEVIRT_POSTINBOUND",
						"-p",
						"tcp",
						"--dport",
						"80",
						"--source", infraconfigurators.GetLoopbackAdrress(proto),
						"-j", "SNAT", "--to-source", GetMasqueradeGwIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
						"KUBEVIRT_PREINBOUND",
						"-p",
						"tcp",
						"--dport",
						"80", "-j", "DNAT", "--to-destination", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
						"OUTPUT",
						"-p",
						"tcp",
						"--dport",
						"80", "--destination", infraconfigurators.GetLoopbackAdrress(proto),
						"-j", "DNAT", "--to-destination", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
				}

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)
				vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a bridge in pod and forward all traffic to VM using nftables", func() {
				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(nil)
				}
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a bridge in pod and forward specific ports to VM using nftables", func() {
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(nil)

					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_POSTINBOUND",
						"tcp",
						"dport",
						"80",
						GetNFTIPString(proto), "saddr", "{ "+infraconfigurators.GetLoopbackAdrress(proto)+" }",
						"counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_PREINBOUND",
						"tcp",
						"dport",
						"80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"output",
						GetNFTIPString(proto), "daddr", "{ "+infraconfigurators.GetLoopbackAdrress(proto)+" }",
						"tcp",
						"dport",
						"80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
				}

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)
				vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a bridge in pod with Istio proxy and forward all traffic to VM using nftables", func() {
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(nil)

					for _, chain := range []string{"output", "KUBEVIRT_POSTINBOUND"} {
						mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
							chain, "tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(istio.ReservedPorts(), ", ")),
							GetNFTIPString(proto), "saddr", infraconfigurators.GetLoopbackAdrress(proto), "counter", "return").Return(nil)
					}

					mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return(fakeAddr.IP.String(), "", nil)
					mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)

					srcAddressesToSnat := []string{infraconfigurators.GetLoopbackAdrress(proto)}
					dstAddressesToDnat := []string{infraconfigurators.GetLoopbackAdrress(proto)}
					if proto == iptables.ProtocolIPv4 {
						srcAddressesToSnat = append(srcAddressesToSnat, istio.GetLoopbackAddress())
						dstAddressesToDnat = append(dstAddressesToDnat, fakeAddr.IP.String())
					}
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_POSTINBOUND", GetNFTIPString(proto), "saddr", fmt.Sprintf("{ %s }", strings.Join(srcAddressesToSnat, ", ")),
						"counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil)
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"output", GetNFTIPString(proto), "daddr", fmt.Sprintf("{ %s }", strings.Join(dstAddressesToDnat, ", ")),
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_PREINBOUND",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).Times(0)
				}

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)
				vm.Annotations = map[string]string{
					istio.ISTIO_INJECT_ANNOTATION: "true",
				}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a bridge in pod with Istio proxy and forward specific ports to VM using nftables", func() {
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				for _, proto := range [2]iptables.Protocol{iptables.ProtocolIPv4, iptables.ProtocolIPv6} {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(nil)

					mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return(fakeAddr.IP.String(), "", nil)
					mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)

					srcAddressesToSnat := []string{infraconfigurators.GetLoopbackAdrress(proto)}
					dstAddressesToDnat := []string{infraconfigurators.GetLoopbackAdrress(proto)}
					if proto == iptables.ProtocolIPv4 {
						srcAddressesToSnat = append(srcAddressesToSnat, istio.GetLoopbackAddress())
						dstAddressesToDnat = append(dstAddressesToDnat, fakeAddr.IP.String())
					}
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_POSTINBOUND",
						"tcp", "dport", "80",
						GetNFTIPString(proto), "saddr", fmt.Sprintf("{ %s }", strings.Join(srcAddressesToSnat, ", ")),
						"counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil)
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"output",
						GetNFTIPString(proto), "daddr", fmt.Sprintf("{ %s }", strings.Join(dstAddressesToDnat, ", ")),
						"tcp", "dport", "80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_PREINBOUND",
						"tcp", "dport", "80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).Times(0)
				}

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)
				vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}
				vm.Annotations = map[string]string{
					istio.ISTIO_INJECT_ANNOTATION: "true",
				}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})

		})
		Context("Slirp Plug", func() {
			It("Should create an interface in the qemu command line and remove it from the interfaces", func() {
				domain := NewDomainWithSlirpInterface()
				vmi := newVMISlirpInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				podnic, err := newPhase2PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, domain)
				Expect(err).ToNot(HaveOccurred())
				driver := podnic.newLibvirtSpecGenerator(domain)
				Expect(driver.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
			})
			It("Should append MAC address to qemu arguments if set", func() {
				domain := NewDomainWithSlirpInterface()
				vmi := newVMISlirpInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
				podnic, err := newPhase2PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, domain)
				Expect(err).ToNot(HaveOccurred())
				driver := podnic.newLibvirtSpecGenerator(domain)
				Expect(driver.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default,mac=de-ad-00-00-be-af"}))
			})
			It("Should create an interface in the qemu command line, remove it from the interfaces and leave the other interfaces inplace", func() {
				domain := NewDomainWithSlirpInterface()
				vmi := newVMISlirpInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

				domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, api.Interface{
					Model: &api.Model{
						Type: "virtio",
					},
					Type: "bridge",
					Source: api.InterfaceSource{
						Bridge: api.DefaultBridgeName,
					},
					Alias: api.NewUserDefinedAlias("default"),
				})
				podnic, err := newPhase1PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, &pid)
				Expect(err).ToNot(HaveOccurred())
				driver := podnic.newLibvirtSpecGenerator(domain)
				Expect(driver.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
			})
		})
		Context("Macvtap plug", func() {
			var (
				podnic *podNIC
			)

			BeforeEach(func() {
				vmi := newVMIMacvtapInterface("testnamespace", "testVmName", "default")
				var err error
				podnic, err = newPhase1PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, &pid)
				Expect(err).ToNot(HaveOccurred())
				macvtapInterface := &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: podnic.podInterfaceName, MTU: mtu, HardwareAddr: fakeMac}}
				mockNetwork.EXPECT().LinkByName(podnic.podInterfaceName).Return(macvtapInterface, nil)
			})

			It("Should pass a non-privileged macvtap interface to qemu", func() {
				domain := NewDomainWithMacvtapInterface("default")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				specGenerator := podnic.newLibvirtSpecGenerator(domain)

				Expect(specGenerator.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1), "should have a single interface")
				Expect(domain.Spec.Devices.Interfaces[0].Target).To(Equal(&api.InterfaceTarget{Device: podnic.podInterfaceName, Managed: "no"}), "should have an unmanaged interface")
				Expect(domain.Spec.Devices.Interfaces[0].MAC).To(Equal(&api.MAC{MAC: fakeMac.String()}), "should have the expected MAC address")
				Expect(domain.Spec.Devices.Interfaces[0].MTU).To(Equal(&api.MTU{Size: "1410"}), "should have the expected MTU")

			})
		})
	})
})
