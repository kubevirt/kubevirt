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

	"github.com/coreos/go-iptables/iptables"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache/fake"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Pod Network", func() {
	var mockNetwork *MockNetworkHandler
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
	var testNic *VIF
	var tmpDir string
	var masqueradeTestNic *VIF
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
	var cacheFactory cache.InterfaceCacheFactory
	var libvirtUser string
	var newPodNIC = func(vmi *v1.VirtualMachineInstance) podNIC {
		return podNIC{
			cacheFactory: cacheFactory,
			handler:      mockNetwork,
			vmi:          vmi,
		}
	}
	var createDefaultPodNIC = func(vmi *v1.VirtualMachineInstance) podNIC {
		podnic := newPodNIC(vmi)
		podnic.iface = &vmi.Spec.Domain.Devices.Interfaces[0]
		podnic.network = &vmi.Spec.Networks[0]
		podnic.podInterfaceName = primaryPodInterfaceName
		return podnic
	}
	BeforeEach(func() {
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
		tmpDir, _ := ioutil.TempDir("", "networktest")
		setVifCacheFile(tmpDir + "/cache-vif-%s.json")

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = NewMockNetworkHandler(ctrl)
		testMac := "12:34:56:78:9A:BC"
		updateTestMac := "AF:B3:1F:78:2A:CA"
		mtu = 1410
		newPodInterfaceName = fmt.Sprintf("%s-nic", primaryPodInterfaceName)
		dummySwap = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: primaryPodInterfaceName}}
		primaryPodInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: primaryPodInterfaceName, MTU: mtu}}
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
		libvirtUser = libvirtUserAndGroupId

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

		bridgeAddr, _ = netlink.ParseAddr(fmt.Sprintf(bridgeFakeIP, 0))
		tapDeviceName = "tap0"
		testNic = &VIF{Name: primaryPodInterfaceName,
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
		masqueradeTestNic = &VIF{Name: primaryPodInterfaceName,
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
		mockNetwork.EXPECT().StartDHCP(testNic, bridgeAddr, api.DefaultBridgeName, nil, true)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
		mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
		mockNetwork.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)

		// For masquerade tests
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
		mockNetwork.EXPECT().StartDHCP(masqueradeTestNic, masqueradeGwAddr, api.DefaultBridgeName, nil, false)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMCIDR).Return(masqueradeGwStr, masqueradeVmStr, nil)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMIpv6CIDR).Return(masqueradeIpv6GwStr, masqueradeIpv6VmStr, nil)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
		mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
		// Global nat rules using iptables
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
			//Global net rules using nftable
			mockNetwork.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", GetNFTIPString(proto), "saddr", GetMasqueradeVmIp(proto), "counter", "masquerade").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "prerouting", "iifname", "eth0", "counter", "jump", "KUBEVIRT_PREINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", "oifname", "k6t-eth0", "counter", "jump", "KUBEVIRT_POSTINBOUND").Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND", "counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)

		}
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
		podnic := createDefaultPodNIC(vm)
		podnic.launcherPID = &pid
		err := podnic.PlugPhase1()
		Expect(err).To(BeNil())

		// Calling SetupPhase1 a second time should result in
		// no mockNetwork function calls, as confirmed by mock object
		// limited number of calls expected for each mocked entry point.
		podnic = createDefaultPodNIC(vm)
		podnic.launcherPID = &pid
		err = podnic.PlugPhase1()
		Expect(err).To(BeNil())
	}

	TestRunPlug := func(driver BindMechanism) {
		err := driver.discoverPodNetworkInterface()
		Expect(err).ToNot(HaveOccurred())

		err = driver.preparePodNetworkInterfaces()
		Expect(err).ToNot(HaveOccurred())

		err = driver.decorateConfig()
		Expect(err).ToNot(HaveOccurred())
	}

	Context("on successful setup", func() {
		It("should define a new VIF bind to a bridge", func() {
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			domain := NewDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			TestPodInterfaceIPBinding(vm, domain)
		})
		It("phase1 should return a CriticalNetworkError if pod networking fails to setup", func() {

			domain := NewDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
			mockNetwork.EXPECT().LinkSetDown(primaryPodInterface).Return(nil)
			mockNetwork.EXPECT().SetRandomMac(primaryPodInterfaceName).Return(updateFakeMac, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
			mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkSetUp(primaryPodInterface).Return(nil)
			mockNetwork.EXPECT().ParseAddr(fmt.Sprintf(bridgeFakeIP, 0)).Return(bridgeAddr, nil)
			mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
			mockNetwork.EXPECT().RouteList(primaryPodInterface, netlink.FAMILY_V4).Return(routeList, nil)
			mockNetwork.EXPECT().GetMacDetails(primaryPodInterfaceName).Return(fakeMac, nil)
			mockNetwork.EXPECT().LinkSetMaster(primaryPodInterface, bridgeTest).Return(nil)
			mockNetwork.EXPECT().AddrDel(primaryPodInterface, &fakeAddr).Return(errors.New("device is busy"))
			mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
			mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
			mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			podnic := createDefaultPodNIC(vm)
			podnic.launcherPID = &pid
			err := podnic.PlugPhase1()
			Expect(err).To(HaveOccurred(), "SetupPhase1 should return an error")

			_, ok := err.(*CriticalNetworkError)
			Expect(ok).To(BeTrue(), "SetupPhase1 should return an error of type CriticalNetworkError")
		})
		It("should return an error if the MTU is out or range", func() {
			primaryPodInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Index: 1, MTU: 65536}}
			domain := NewDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil).Times(2)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().GetMacDetails(primaryPodInterfaceName).Return(fakeMac, nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			podnic := createDefaultPodNIC(vm)
			podnic.launcherPID = &pid
			err := podnic.PlugPhase1()
			Expect(err).To(HaveOccurred())
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
				Expect(filterPodNetworkRoutes(staticRouteList, testNic)).To(Equal(expectedRouteList))
			})
		})
		It("phase2 should panic if DHCP startup fails", func() {
			testDhcpPanic := func() {
				domain := NewDomainWithBridgeInterface()
				vm := newVMIBridgeInterface("testnamespace", "testVmName")
				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to open file"))
				podnic := createDefaultPodNIC(vm)
				Expect(podnic.PlugPhase2(domain)).To(Succeed())
			}
			Expect(testDhcpPanic).To(Panic())
		})
		Context("getPhase1Binding", func() {
			Context("for Bridge", func() {
				It("should populate MAC address", func() {
					vmi := newVMIBridgeInterface("testnamespace", "testVmName")
					vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
					podnic := createDefaultPodNIC(vmi)
					podnic.launcherPID = &pid
					driver, err := podnic.getPhase1Binding()
					Expect(err).ToNot(HaveOccurred())
					bridge, ok := driver.(*BridgeBindMechanism)
					Expect(ok).To(BeTrue())
					Expect(bridge.vif.MAC.String()).To(Equal("de:ad:00:00:be:af"))
				})
			})
		})
		Context("SRIOV Plug", func() {
			It("Does not crash", func() {
				// Plug doesn't do anything for sriov so it's enough to pass an empty domain
				domain := &api.Domain{}
				// Same for network
				net := &v1.Network{}

				iface := &v1.Interface{
					Name: "sriov",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						SRIOV: &v1.InterfaceSRIOV{},
					},
				}
				vmi := newVMI("testnamespace", "testVmName")
				podnic := newPodNIC(vmi)
				podnic.iface = iface
				podnic.network = net
				podnic.podInterfaceName = "fakeiface"
				podnic.launcherPID = &pid
				err := podnic.PlugPhase1()
				Expect(err).ToNot(HaveOccurred())

				err = podnic.PlugPhase2(domain)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Masquerade Plug", func() {
			It("should define a new VIF bind to a bridge and create a default nat rule using iptables", func() {

				// forward all the traffic
				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(fmt.Errorf("no nft"))
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
				}
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a specific nat rule using iptables", func() {
				// Forward a specific port
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
						"--source", getLoopbackAdrress(proto),
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
						"80", "--destination", getLoopbackAdrress(proto),
						"-j", "DNAT", "--to-destination", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
				}

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName")
				vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a default nat rule using nftables", func() {
				// forward all the traffic
				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(nil)
				}
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a specific nat rule using nftables", func() {
				// Forward a specific port
				mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().NftablesLoad(proto).Return(nil)

					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_POSTINBOUND",
						"tcp",
						"dport",
						"80",
						GetNFTIPString(proto), "saddr", getLoopbackAdrress(proto),
						"counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"KUBEVIRT_PREINBOUND",
						"tcp",
						"dport",
						"80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
					mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
						"output",
						GetNFTIPString(proto), "daddr", getLoopbackAdrress(proto),
						"tcp",
						"dport",
						"80",
						"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
				}

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName")
				vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})

		})
		Context("Slirp Plug", func() {
			It("Should create an interface in the qemu command line and remove it from the interfaces", func() {
				domain := NewDomainWithSlirpInterface()
				vmi := newVMISlirpInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				podnic := createDefaultPodNIC(vmi)
				driver, err := podnic.getPhase2Binding(domain)
				Expect(err).ToNot(HaveOccurred())
				TestRunPlug(driver)
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
				podnic := createDefaultPodNIC(vmi)
				driver, err := podnic.getPhase2Binding(domain)
				Expect(err).ToNot(HaveOccurred())
				TestRunPlug(driver)
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
				podnic := createDefaultPodNIC(vmi)
				podnic.launcherPID = &pid
				driver, err := podnic.getPhase2Binding(domain)
				Expect(err).ToNot(HaveOccurred())
				TestRunPlug(driver)
				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
			})
		})
		Context("Macvtap plug", func() {
			const ifaceName = "macvtap0"
			var macvtapInterface *netlink.GenericLink

			BeforeEach(func() {
				macvtapInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu, HardwareAddr: fakeMac}}
			})

			It("Should pass a non-privileged macvtap interface to qemu", func() {
				domain := NewDomainWithMacvtapInterface(ifaceName)
				vmi := newVMIMacvtapInterface("testnamespace", "default", ifaceName)

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				podnic := createDefaultPodNIC(vmi)
				podnic.launcherPID = &pid
				podnic.podInterfaceName = ifaceName
				driver, err := podnic.getPhase2Binding(domain)
				mockNetwork.EXPECT().LinkByName(ifaceName).Return(macvtapInterface, nil)
				Expect(err).ToNot(HaveOccurred(), "should have identified the correct binding mechanism")
				TestRunPlug(driver)
				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1), "should have a single interface")
				Expect(domain.Spec.Devices.Interfaces[0].Target).To(Equal(&api.InterfaceTarget{Device: ifaceName, Managed: "no"}), "should have an unmanaged interface")
				Expect(domain.Spec.Devices.Interfaces[0].MAC).To(Equal(&api.MAC{MAC: fakeMac.String()}), "should have the expected MAC address")
				Expect(domain.Spec.Devices.Interfaces[0].MTU).To(Equal(&api.MTU{Size: "1410"}), "should have the expected MTU")

			})
		})
	})

	Context("Masquerade startDHCP", func() {
		It("should succeed when DHCP server started", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase2Binding(domain)
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradeBindMechanism)
			Expect(ok).To(BeTrue())

			masq.vif.Gateway = masqueradeGwAddr.IP.To4()
			masq.vif.GatewayIpv6 = masqueradeIpv6GwAddr.IP.To16()
			mockNetwork.EXPECT().StartDHCP(masq.vif, gomock.Any(), masq.bridgeInterfaceName, nil, false).Return(nil)

			Expect(masq.startDHCP()).To(Succeed())
		})
		It("should fail when DHCP server failed", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase2Binding(domain)
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradeBindMechanism)
			Expect(ok).To(BeTrue())

			masq.vif.Gateway = masqueradeGwAddr.IP.To4()
			masq.vif.GatewayIpv6 = masqueradeIpv6GwAddr.IP.To16()

			err = fmt.Errorf("failed to start DHCP server")
			mockNetwork.EXPECT().StartDHCP(masq.vif, gomock.Any(), masq.bridgeInterfaceName, nil, false).Return(err)

			Expect(masq.startDHCP()).To(HaveOccurred())
		})
	})
	Context("Bridge startDHCP", func() {
		It("should succeed when DHCP server started", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase2Binding(domain)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgeBindMechanism)
			Expect(ok).To(BeTrue())

			mockNetwork.EXPECT().StartDHCP(bridge.vif, gomock.Any(), api.DefaultBridgeName, nil, true).Return(nil)

			Expect(bridge.startDHCP()).To(Succeed())
		})
		It("should fail when DHCP server failed", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase2Binding(domain)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgeBindMechanism)
			Expect(ok).To(BeTrue())

			err = fmt.Errorf("failed to start DHCP server")
			mockNetwork.EXPECT().StartDHCP(bridge.vif, gomock.Any(), api.DefaultBridgeName, nil, true).Return(err)

			Expect(bridge.startDHCP()).To(HaveOccurred())
		})
		It("should succeed when DHCP server started and isLayer2 = true", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase2Binding(domain)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgeBindMechanism)
			Expect(ok).To(BeTrue())

			bridge.vif.IPAMDisabled = true
			err = fmt.Errorf("failed to start DHCP server")
			mockNetwork.EXPECT().StartDHCP(bridge.vif, gomock.Any(), api.DefaultBridgeName, nil, true).Return(err)

			Expect(bridge.startDHCP()).To(Succeed())
		})
	})
	Context("Slirp startDHCP", func() {
		It("should succeed when DHCP server started", func() {
			domain := NewDomainWithSlirpInterface()
			vmi := newVMISlirpInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			podnic := createDefaultPodNIC(vmi)
			driver, err := podnic.getPhase2Binding(domain)
			Expect(err).ToNot(HaveOccurred())
			slirp, ok := driver.(*SlirpBindMechanism)
			Expect(ok).To(BeTrue())

			Expect(slirp.startDHCP()).To(Succeed())
		})
		// slirp never fails to start DHCP because it doesn't need it at all
	})

	Context("Bridge loadCachedVIF", func() {
		It("should fail when nothing to load", func() {
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase1Binding()
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgeBindMechanism)
			Expect(ok).To(BeTrue())

			Expect(bridge.loadCachedVIF(fmt.Sprintf("%d", pid))).To(HaveOccurred())
		})
		It("should succeed when cache file present", func() {
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase1Binding()
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgeBindMechanism)
			Expect(ok).To(BeTrue())

			pidStr := fmt.Sprintf("%d", pid)
			Expect(bridge.setCachedVIF(pidStr)).ToNot(HaveOccurred())
			Expect(bridge.loadCachedVIF(pidStr)).ToNot(HaveOccurred())
		})
	})

	Context("Slirp loadCachedVIF", func() {
		It("should succeed", func() {
			vmi := newVMISlirpInterface("testnamespace", "testVmName")

			podnic := createDefaultPodNIC(vmi)

			driver, err := podnic.getPhase1Binding()
			Expect(err).ToNot(HaveOccurred())
			slirp, ok := driver.(*SlirpBindMechanism)
			Expect(ok).To(BeTrue())

			// it doesn't fail regardless, whether called without setCachedVIF...
			Expect(slirp.loadCachedVIF(fmt.Sprintf("%d", pid))).NotTo(HaveOccurred())

			// ...or after it
			err = slirp.setCachedVIF(fmt.Sprintf("%d", pid))
			Expect(err).ToNot(HaveOccurred())

			Expect(slirp.loadCachedVIF(fmt.Sprintf("%d", pid))).NotTo(HaveOccurred())
		})
	})

	Context("Masquerade loadCachedVIF", func() {
		It("should fail when nothing to load", func() {
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase1Binding()
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradeBindMechanism)
			Expect(ok).To(BeTrue())

			Expect(masq.loadCachedVIF(fmt.Sprintf("%d", pid))).To(HaveOccurred())
		})
		It("should succeed when cache file present", func() {
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			podnic := createDefaultPodNIC(vmi)
			podnic.launcherPID = &pid
			driver, err := podnic.getPhase1Binding()
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradeBindMechanism)
			Expect(ok).To(BeTrue())

			pidStr := fmt.Sprintf("%d", pid)
			Expect(masq.setCachedVIF(pidStr)).ToNot(HaveOccurred())
			Expect(masq.loadCachedVIF(pidStr)).ToNot(HaveOccurred())
		})
	})

	It("should write interface to cache file", func() {
		uid := types.UID("test-1234")
		vmi := &v1.VirtualMachineInstance{ObjectMeta: v12.ObjectMeta{UID: uid}}
		address1 := &net.IPNet{IP: net.IPv4(1, 2, 3, 4)}
		address2 := &net.IPNet{IP: net.IPv4(169, 254, 0, 0)}
		fakeAddr1 := netlink.Addr{IPNet: address1}
		fakeAddr2 := netlink.Addr{IPNet: address2}
		addrList := []netlink.Addr{fakeAddr1, fakeAddr2}

		iface := &v1.Interface{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
		mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)
		podnic := newPodNIC(vmi)
		podnic.iface = iface
		podnic.podInterfaceName = primaryPodInterfaceName
		err := podnic.setPodInterfaceCache()
		Expect(err).ToNot(HaveOccurred())

		podData, err := cacheFactory.CacheForVMI(vmi).Read(iface.Name)
		Expect(err).ToNot(HaveOccurred())
		Expect(podData.PodIP).To(Equal("1.2.3.4"))
	})
})

func ipProtocols() [2]iptables.Protocol {
	return [2]iptables.Protocol{iptables.ProtocolIPv4, iptables.ProtocolIPv6}
}

func newVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMIWithNS(namespace, name)
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func newVMIBridgeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := newVMI(namespace, name)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func newVMIMasqueradeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := newVMI(namespace, name)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func newVMISlirpInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := newVMI(namespace, name)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultSlirpNetworkInterface()}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func newVMIMacvtapInterface(namespace string, vmiName string, ifaceName string) *v1.VirtualMachineInstance {
	vmi := newVMI(namespace, vmiName)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMacvtapNetworkInterface(ifaceName)}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewDomainWithBridgeInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: "virtio",
		},
		Type: "bridge",
		Source: api.InterfaceSource{
			Bridge: api.DefaultBridgeName,
		},
		Alias: api.NewUserDefinedAlias("default"),
	},
	}
	return domain
}

func NewDomainWithSlirpInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: "e1000",
		},
		Type:  "user",
		Alias: api.NewUserDefinedAlias("default"),
	},
	}

	// Create network interface
	if domain.Spec.QEMUCmd == nil {
		domain.Spec.QEMUCmd = &api.Commandline{}
	}

	if domain.Spec.QEMUCmd.QEMUArg == nil {
		domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
	}

	return domain
}

func NewDomainWithMacvtapInterface(macvtapName string) *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Alias: api.NewUserDefinedAlias(macvtapName),
		Model: &api.Model{
			Type: "virtio",
		},
		Type: "ethernet",
	}}
	return domain
}
