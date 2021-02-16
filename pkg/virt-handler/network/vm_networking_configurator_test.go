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
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"

	networkdriver "kubevirt.io/kubevirt/pkg/network"
	network_test_utils "kubevirt.io/kubevirt/pkg/network/test_utils"

	"github.com/coreos/go-iptables/iptables"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func MockVirtLauncherCachedPattern(path string) {
	networkdriver.VirtLauncherCachedPattern = path
}

var _ = Describe("Pod Network", func() {
	var mockNetwork *networkdriver.MockNetworkHandler
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
	var testNic *networkdriver.VIF
	var tmpDir string
	var masqueradeTestNic *networkdriver.VIF
	var masqueradeDummyName string
	var masqueradeDummy *netlink.Dummy
	var masqueradeGwStr string
	var masqueradeGwAddr *netlink.Addr
	var masqueradeGwIp string
	var masqueradeVmStr string
	var masqueradeVmAddr *netlink.Addr
	var masqueradeVmIp string
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

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		tmpDir, _ := ioutil.TempDir("", "networktest")
		MockVirtLauncherCachedPattern(tmpDir + "/cache-iface-%s.json")
		networkdriver.SetVifCacheFile(tmpDir + "/cache-vif-%s.json")

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = networkdriver.NewMockNetworkHandler(ctrl)
		networkdriver.Handler = mockNetwork
		testMac := "12:34:56:78:9A:BC"
		updateTestMac := "AF:B3:1F:78:2A:CA"
		mtu = 1410
		newPodInterfaceName = fmt.Sprintf("%s-nic", networkdriver.PrimaryPodInterfaceName)
		dummySwap = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: networkdriver.PrimaryPodInterfaceName}}
		primaryPodInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: networkdriver.PrimaryPodInterfaceName, MTU: mtu}}
		primaryPodInterfaceAfterNameChange = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: newPodInterfaceName}}
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

		// Create a bridge
		bridgeTest = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: api.DefaultBridgeName,
			},
		}

		masqueradeBridgeTest = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: api.DefaultBridgeName,
				MTU:  mtu,
			},
		}

		bridgeAddr, _ = netlink.ParseAddr(fmt.Sprintf(networkdriver.BridgeFakeIP, 0))
		tapDeviceName = "tap0"
		testNic = &networkdriver.VIF{Name: networkdriver.PrimaryPodInterfaceName,
			IP:      fakeAddr,
			MAC:     fakeMac,
			Mtu:     uint16(mtu),
			Gateway: gw,
		}

		masqueradeGwStr = "10.0.2.1/30"
		masqueradeGwAddr, _ = netlink.ParseAddr(masqueradeGwStr)
		masqueradeGwIp = masqueradeGwAddr.IP.String()
		masqueradeVmStr = "10.0.2.2/30"
		masqueradeVmAddr, _ = netlink.ParseAddr(masqueradeVmStr)
		masqueradeVmIp = masqueradeVmAddr.IP.String()
		masqueradeIpv6GwStr = "fd10:0:2::1/120"
		masqueradeIpv6GwAddr, _ = netlink.ParseAddr(masqueradeIpv6GwStr)
		masqueradeGwIpv6 = masqueradeIpv6GwAddr.IP.String()
		masqueradeIpv6VmStr = "fd10:0:2::2/120"
		masqueradeIpv6VmAddr, _ = netlink.ParseAddr(masqueradeIpv6VmStr)
		masqueradeVmIpv6 = masqueradeIpv6VmAddr.IP.String()
		masqueradeDummyName = fmt.Sprintf("%s-nic", api.DefaultBridgeName)
		masqueradeDummy = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: masqueradeDummyName, MTU: mtu}}
		masqueradeTestNic = &networkdriver.VIF{Name: networkdriver.PrimaryPodInterfaceName,
			IP:          *masqueradeVmAddr,
			IPv6:        *masqueradeIpv6VmAddr,
			MAC:         fakeMac,
			Mtu:         uint16(mtu),
			Gateway:     masqueradeGwAddr.IP.To4(),
			GatewayIpv6: masqueradeIpv6GwAddr.IP.To16()}
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

	queueNumber = uint32(0)

	TestPodInterfaceIPBinding := func(vm *v1.VirtualMachineInstance, domain *api.Domain) {

		//For Bridge tests
		mockNetwork.EXPECT().LinkSetName(primaryPodInterface, newPodInterfaceName).Return(nil)
		mockNetwork.EXPECT().LinkByName(networkdriver.PrimaryPodInterfaceName).Return(primaryPodInterface, nil)
		mockNetwork.EXPECT().LinkByName(newPodInterfaceName).Return(primaryPodInterfaceAfterNameChange, nil)
		mockNetwork.EXPECT().LinkAdd(dummySwap).Return(nil)
		mockNetwork.EXPECT().AddrReplace(dummySwap, &fakeAddr).Return(nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().RouteList(primaryPodInterface, netlink.FAMILY_V4).Return(routeList, nil)
		mockNetwork.EXPECT().GetMacDetails(networkdriver.PrimaryPodInterfaceName).Return(fakeMac, nil)
		mockNetwork.EXPECT().AddrDel(primaryPodInterface, &fakeAddr).Return(nil)
		mockNetwork.EXPECT().LinkSetDown(primaryPodInterface).Return(nil)
		mockNetwork.EXPECT().SetRandomMac(newPodInterfaceName).Return(updateFakeMac, nil)
		mockNetwork.EXPECT().LinkSetUp(primaryPodInterfaceAfterNameChange).Return(nil)
		mockNetwork.EXPECT().LinkSetLearningOff(primaryPodInterfaceAfterNameChange).Return(nil)
		mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
		mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
		mockNetwork.EXPECT().ParseAddr(fmt.Sprintf(networkdriver.BridgeFakeIP, 0)).Return(bridgeAddr, nil)
		mockNetwork.EXPECT().LinkSetMaster(primaryPodInterfaceAfterNameChange, bridgeTest).Return(nil)
		mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(testNic, bridgeAddr, api.DefaultBridgeName, nil, true)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
		mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
		mockNetwork.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)

		// For masquerade tests
		mockNetwork.EXPECT().LinkByName(networkdriver.PrimaryPodInterfaceName).Return(primaryPodInterface, nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeGwStr).Return(masqueradeGwAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeIpv6GwStr).Return(masqueradeIpv6GwAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeVmStr).Return(masqueradeVmAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeIpv6VmStr).Return(masqueradeIpv6VmAddr, nil)
		mockNetwork.EXPECT().LinkAdd(masqueradeDummy).Return(nil)
		mockNetwork.EXPECT().LinkByName(masqueradeDummyName).Return(masqueradeDummy, nil)
		mockNetwork.EXPECT().LinkSetUp(masqueradeDummy).Return(nil)
		mockNetwork.EXPECT().GenerateRandomMac().Return(fakeMac, nil)
		mockNetwork.EXPECT().LinkAdd(masqueradeBridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkSetUp(masqueradeBridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkSetMaster(masqueradeDummy, masqueradeBridgeTest).Return(nil)
		mockNetwork.EXPECT().AddrAdd(masqueradeBridgeTest, masqueradeGwAddr).Return(nil)
		mockNetwork.EXPECT().AddrAdd(masqueradeBridgeTest, masqueradeIpv6GwAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(masqueradeTestNic, masqueradeGwAddr, api.DefaultBridgeName, nil, false)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMCIDR).Return(masqueradeGwStr, masqueradeVmStr, nil)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMIpv6CIDR).Return(masqueradeIpv6GwStr, masqueradeIpv6VmStr, nil)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu).Return(nil)
		mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
		// Global nat rules using iptables
		mockNetwork.EXPECT().ConfigureIpv4Forwarding().Return(nil)
		mockNetwork.EXPECT().ConfigureIpv6Forwarding().Return(nil)
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
			//Global net rules using nftable
			ipVersionNum := "4"
			if proto == iptables.ProtocolIPv6 {
				ipVersionNum = "6"
			}
			mockNetwork.EXPECT().NftablesLoad(fmt.Sprintf("ipv%s-nat", ipVersionNum)).Return(nil)
			mockNetwork.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", GetNFTIPString(proto), "saddr", GetMasqueradeVmIp(proto), "counter", "masquerade").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "prerouting", "iifname", "eth0", "counter", "jump", "KUBEVIRT_PREINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", "oifname", "k6t-eth0", "counter", "jump", "KUBEVIRT_POSTINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND", "counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)

		}
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)

		networkingConfigurator := NewVMNetworkConfigurator(vm, pid)
		Expect(networkingConfigurator.SetupPodInfrastructure()).To(BeNil())

		// Calling SetupPodNetworkPhase1 a second time should result in
		// no mockNetwork function calls, as confirmed by mock object
		// limited number of calls expected for each mocked entry point.
		Expect(networkingConfigurator.SetupPodInfrastructure()).To(BeNil())
	}

	Context("on successful setup", func() {
		It("should define a new VIF bind to a bridge", func() {
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			domain := network_test_utils.NewDomainWithBridgeInterface()
			vm := network_test_utils.NewVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			TestPodInterfaceIPBinding(vm, domain)
		})
		It("phase1 should return a CriticalNetworkError if pod networking fails to setup", func() {

			domain := network_test_utils.NewDomainWithBridgeInterface()
			vm := network_test_utils.NewVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().LinkByName(networkdriver.PrimaryPodInterfaceName).Return(primaryPodInterface, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().LinkByName(networkdriver.PrimaryPodInterfaceName).Return(primaryPodInterface, nil)
			mockNetwork.EXPECT().LinkSetDown(primaryPodInterface).Return(nil)
			mockNetwork.EXPECT().SetRandomMac(networkdriver.PrimaryPodInterfaceName).Return(updateFakeMac, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
			mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkSetUp(primaryPodInterface).Return(nil)
			mockNetwork.EXPECT().ParseAddr(fmt.Sprintf(networkdriver.BridgeFakeIP, 0)).Return(bridgeAddr, nil)
			mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
			mockNetwork.EXPECT().RouteList(primaryPodInterface, netlink.FAMILY_V4).Return(routeList, nil)
			mockNetwork.EXPECT().GetMacDetails(networkdriver.PrimaryPodInterfaceName).Return(fakeMac, nil)
			mockNetwork.EXPECT().LinkSetMaster(primaryPodInterface, bridgeTest).Return(nil)
			mockNetwork.EXPECT().AddrDel(primaryPodInterface, &fakeAddr).Return(errors.New("device is busy"))
			mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu).Return(nil)
			mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
			mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			networkConfigurator := NewVMNetworkConfigurator(vm, pid)
			err := networkConfigurator.SetupPodInfrastructure()
			Expect(err).To(HaveOccurred(), "SetupPodNetworkPhase1 should return an error")

			_, ok := err.(*networkdriver.CriticalNetworkError)
			Expect(ok).To(BeTrue(), "SetupPodNetworkPhase1 should return an error of type CriticalNetworkError")
		})
		It("should return an error if the MTU is out or range", func() {
			primaryPodInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Index: 1, MTU: 65536}}
			domain := network_test_utils.NewDomainWithBridgeInterface()
			vm := network_test_utils.NewVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().LinkByName(networkdriver.PrimaryPodInterfaceName).Return(primaryPodInterface, nil).Times(2)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().GetMacDetails(networkdriver.PrimaryPodInterfaceName).Return(fakeMac, nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			networkConfigurator := NewVMNetworkConfigurator(vm, pid)
			Expect(networkConfigurator.SetupPodInfrastructure()).To(HaveOccurred())
		})
		Context("func FilterPodNetworkRoutes()", func() {
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
				Expect(networkdriver.FilterPodNetworkRoutes(staticRouteList, testNic)).To(Equal(expectedRouteList))
			})
		})
		Context("getPhase1Binding", func() {
			Context("for Bridge", func() {
				It("should populate MAC address", func() {
					vmi := network_test_utils.NewVMIBridgeInterface("testnamespace", "testVmName")
					vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
					driver := NewVMNetworkConfigurator(vmi, pid)
					vmConfigurator, err := driver.GenerateVMNetworkingConfiguratorForIface(&vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], networkdriver.PrimaryPodInterfaceName)
					Expect(err).ToNot(HaveOccurred())
					bridge, ok := vmConfigurator.(*BridgedNetworkingVMConfigurator)
					Expect(ok).To(BeTrue())
					Expect(bridge.vif.MAC.String()).To(Equal("de:ad:00:00:be:af"))
				})
			})
		})
		Context("SRIOV Plug", func() {
			It("Does not crash", func() {
				// Same for network
				net := &v1.Network{}

				iface := &v1.Interface{
					Name: "sriov",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						SRIOV: &v1.InterfaceSRIOV{},
					},
				}
				vmi := network_test_utils.NewVMI("testnamespace", "testVmName")
				vmNetworkingConfigurator := NewVMNetworkConfigurator(vmi, pid)
				Expect(vmNetworkingConfigurator.GenerateVMNetworkingConfiguratorForIface(iface, net, "fakeiface")).ToNot(HaveOccurred())
			})
		})
		Context("Masquerade Plug", func() {
			It("should define a new VIF bind to a bridge and create a default nat rule using iptables", func() {

				// forward all the traffic
				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
				}
				mockNetwork.EXPECT().IsIpv6Enabled(networkdriver.PrimaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				domain := network_test_utils.NewDomainWithBridgeInterface()
				vm := network_test_utils.NewVMIMasqueradeInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a specific nat rule using iptables", func() {
				// Forward a specific port
				mockNetwork.EXPECT().IsIpv6Enabled(networkdriver.PrimaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
					mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
						"KUBEVIRT_POSTINBOUND",
						"-p",
						"tcp",
						"--dport",
						"80",
						"--source", networkdriver.GetLoopbackAdrress(proto),
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
						"80", "--destination", networkdriver.GetLoopbackAdrress(proto),
						"-j", "DNAT", "--to-destination", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
				}

				domain := network_test_utils.NewDomainWithBridgeInterface()
				vm := network_test_utils.NewVMIMasqueradeInterface("testnamespace", "testVmName")
				vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a default nat rule using nftables", func() {
				// forward all the traffic
				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
				}
				mockNetwork.EXPECT().IsIpv6Enabled(networkdriver.PrimaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				domain := network_test_utils.NewDomainWithBridgeInterface()
				vm := network_test_utils.NewVMIMasqueradeInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a specific nat rule using nftables", func() {
				// Forward a specific port
				mockNetwork.EXPECT().IsIpv6Enabled(networkdriver.PrimaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().HasNatIptables(proto).Return(false).Times(2)

					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_POSTINBOUND",
						"tcp",
						"dport",
						"80",
						GetNFTIPString(proto), "saddr", networkdriver.GetLoopbackAdrress(proto),
						"counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_PREINBOUND",
						"tcp",
						"dport",
						"80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"output",
						GetNFTIPString(proto), "daddr", networkdriver.GetLoopbackAdrress(proto),
						"tcp",
						"dport",
						"80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
				}

				domain := network_test_utils.NewDomainWithBridgeInterface()
				vm := network_test_utils.NewVMIMasqueradeInterface("testnamespace", "testVmName")
				vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})

		})
	})

	It("should write interface to cache file", func() {
		uid := types.UID("test-1234")
		address1 := &net.IPNet{IP: net.IPv4(1, 2, 3, 4)}
		address2 := &net.IPNet{IP: net.IPv4(169, 254, 0, 0)}
		fakeAddr1 := netlink.Addr{IPNet: address1}
		fakeAddr2 := netlink.Addr{IPNet: address2}
		addrList := []netlink.Addr{fakeAddr1, fakeAddr2}
		err := networkdriver.CreateVirtHandlerCacheDir(uid)
		Expect(err).ToNot(HaveOccurred())

		iface := &v1.Interface{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
		mockNetwork.EXPECT().LinkByName(networkdriver.PrimaryPodInterfaceName).Return(primaryPodInterface, nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

		err = setPodInterfaceCache(iface, networkdriver.PrimaryPodInterfaceName, string(uid))
		Expect(err).ToNot(HaveOccurred())

		var podData PodCacheInterface
		err = networkdriver.ReadFromVirtHandlerCachedFile(&podData, uid, iface.Name)
		Expect(err).ToNot(HaveOccurred())
		Expect(podData.PodIP).To(Equal("1.2.3.4"))
	})
})

func ipProtocols() [2]iptables.Protocol {
	return [2]iptables.Protocol{iptables.ProtocolIPv4, iptables.ProtocolIPv6}
}
