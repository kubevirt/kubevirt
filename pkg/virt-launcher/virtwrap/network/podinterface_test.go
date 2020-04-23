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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"

	"kubevirt.io/kubevirt/pkg/util"

	"github.com/coreos/go-iptables/iptables"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Pod Network", func() {
	var mockNetwork *MockNetworkHandler
	var ctrl *gomock.Controller
	var dummy *netlink.Dummy
	var addrList []netlink.Addr
	var routeList []netlink.Route
	var routeAddr netlink.Route
	var fakeMac net.HardwareAddr
	var fakeAddr netlink.Addr
	var updateFakeMac net.HardwareAddr
	var bridgeTest *netlink.Bridge
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
	var isTapDeviceMultiqueued bool

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		tmpDir, _ := ioutil.TempDir("", "networktest")
		setInterfaceCacheFile(tmpDir + "/cache-iface-%s.json")
		setVifCacheFile(tmpDir + "/cache-vif-%s.json")

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = NewMockNetworkHandler(ctrl)
		Handler = mockNetwork
		testMac := "12:34:56:78:9A:BC"
		updateTestMac := "AF:B3:1F:78:2A:CA"
		dummy = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1, MTU: 1410}}
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

		bridgeAddr, _ = netlink.ParseAddr(fmt.Sprintf(bridgeFakeIP, 0))
		tapDeviceName = "tap0"
		testNic = &VIF{Name: podInterface,
			IP:        fakeAddr,
			MAC:       fakeMac,
			Mtu:       1410,
			Gateway:   gw,
			TapDevice: tapDeviceName,
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
		masqueradeDummy = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: masqueradeDummyName}}
		masqueradeTestNic = &VIF{Name: podInterface,
			IP:          *masqueradeVmAddr,
			IPv6:        *masqueradeIpv6VmAddr,
			MAC:         fakeMac,
			Mtu:         1410,
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

	isTapDeviceMultiqueued = false

	TestPodInterfaceIPBinding := func(vm *v1.VirtualMachineInstance, domain *api.Domain) {

		//For Bridge tests
		mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil).Times(2)
		mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
		mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().RouteList(dummy, netlink.FAMILY_V4).Return(routeList, nil)
		mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)
		mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(nil)
		mockNetwork.EXPECT().LinkSetDown(dummy).Return(nil)
		mockNetwork.EXPECT().SetRandomMac(podInterface).Return(updateFakeMac, nil)
		mockNetwork.EXPECT().LinkSetUp(dummy).Return(nil)
		mockNetwork.EXPECT().LinkSetLearningOff(dummy).Return(nil)
		mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
		mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
		mockNetwork.EXPECT().ParseAddr(fmt.Sprintf(bridgeFakeIP, 0)).Return(bridgeAddr, nil)
		mockNetwork.EXPECT().LinkSetMaster(dummy, bridgeTest).Return(nil)
		mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(testNic, bridgeAddr, api.DefaultBridgeName, nil)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, isTapDeviceMultiqueued).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)

		// For masquerade tests
		mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
		mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeGwStr).Return(masqueradeGwAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeIpv6GwStr).Return(masqueradeIpv6GwAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeVmStr).Return(masqueradeVmAddr, nil)
		mockNetwork.EXPECT().ParseAddr(masqueradeIpv6VmStr).Return(masqueradeIpv6VmAddr, nil)
		mockNetwork.EXPECT().LinkAdd(masqueradeDummy).Return(nil)
		mockNetwork.EXPECT().LinkByName(masqueradeDummyName).Return(masqueradeDummy, nil)
		mockNetwork.EXPECT().LinkSetUp(masqueradeDummy).Return(nil)
		mockNetwork.EXPECT().GenerateRandomMac().Return(fakeMac, nil)
		mockNetwork.EXPECT().LinkSetMaster(masqueradeDummy, bridgeTest).Return(nil)
		mockNetwork.EXPECT().AddrAdd(bridgeTest, masqueradeGwAddr).Return(nil)
		mockNetwork.EXPECT().AddrAdd(bridgeTest, masqueradeIpv6GwAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(masqueradeTestNic, masqueradeGwAddr, api.DefaultBridgeName, nil)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMCIDR).Return(masqueradeGwStr, masqueradeVmStr, nil)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMIpv6CIDR).Return(masqueradeIpv6GwStr, masqueradeIpv6VmStr, nil)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, isTapDeviceMultiqueued).Return(nil)
		// Global nat rules using iptables
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
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, isTapDeviceMultiqueued).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)

		err := SetupPodNetworkPhase1(vm, pid)
		Expect(err).To(BeNil())

		// Calling SetupPodNetworkPhase1 a second time should result in
		// no mockNetwork function calls, as confirmed by mock object
		// limited number of calls expected for each mocked entry point.
		err = SetupPodNetworkPhase1(vm, pid)
		Expect(err).To(BeNil())
	}

	TestRunPlug := func(driver BindMechanism) {
		err := driver.discoverPodNetworkInterface()
		Expect(err).ToNot(HaveOccurred())

		err = driver.preparePodNetworkInterfaces(false)
		Expect(err).ToNot(HaveOccurred())

		err = driver.decorateConfig()
		Expect(err).ToNot(HaveOccurred())
	}

	Context("on successful setup", func() {
		It("should define a new VIF bind to a bridge", func() {

			domain := NewDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			TestPodInterfaceIPBinding(vm, domain)
		})
		It("phase1 should return a CriticalNetworkError if pod networking fails to setup", func() {

			domain := NewDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
			mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
			mockNetwork.EXPECT().LinkSetDown(dummy).Return(nil)
			mockNetwork.EXPECT().SetRandomMac(podInterface).Return(updateFakeMac, nil)
			mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
			mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkSetUp(dummy).Return(nil)
			mockNetwork.EXPECT().ParseAddr(fmt.Sprintf(bridgeFakeIP, 0)).Return(bridgeAddr, nil)
			mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
			mockNetwork.EXPECT().RouteList(dummy, netlink.FAMILY_V4).Return(routeList, nil)
			mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)
			mockNetwork.EXPECT().LinkSetMaster(dummy, bridgeTest).Return(nil)
			mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(errors.New("device is busy"))
			mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, isTapDeviceMultiqueued).Return(nil)
			mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)

			err := SetupPodNetworkPhase1(vm, pid)
			Expect(err).To(HaveOccurred(), "SetupPodNetworkPhase1 should return an error")

			_, ok := err.(*CriticalNetworkError)
			Expect(ok).To(BeTrue(), "SetupPodNetworkPhase1 should return an error of type CriticalNetworkError")
		})
		It("should return an error if the MTU is out or range", func() {
			dummy = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1, MTU: 65536}}

			domain := NewDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil).Times(2)
			mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)

			err := SetupPodNetworkPhase1(vm, pid)
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
				mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to open file"))
				SetupPodNetworkPhase2(vm, domain)
			}
			Expect(testDhcpPanic).To(Panic())
		})
		Context("getPhase1Binding", func() {
			Context("for Bridge", func() {
				It("should populate MAC address", func() {
					vmi := newVMIBridgeInterface("testnamespace", "testVmName")
					vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
					driver, err := getPhase1Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], podInterface)
					Expect(err).ToNot(HaveOccurred())
					bridge, ok := driver.(*BridgePodInterface)
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
				podiface := PodInterface{}
				err := podiface.PlugPhase1(vmi, iface, net, "fakeiface", pid)
				Expect(err).ToNot(HaveOccurred())

				err = podiface.PlugPhase2(vmi, iface, net, domain, "fakeiface")
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Masquerade Plug", func() {
			It("should define a new VIF bind to a bridge and create a default nat rule using iptables", func() {

				// forward all the traffic
				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
				}
				mockNetwork.EXPECT().IsIpv6Enabled().Return(true).Times(3)

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a specific nat rule using iptables", func() {
				// Forward a specific port
				mockNetwork.EXPECT().IsIpv6Enabled().Return(true).Times(3)

				for _, proto := range ipProtocols() {
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
					mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
				}
				mockNetwork.EXPECT().IsIpv6Enabled().Return(true).Times(3)

				domain := NewDomainWithBridgeInterface()
				vm := newVMIMasqueradeInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				TestPodInterfaceIPBinding(vm, domain)
			})
			It("should define a new VIF bind to a bridge and create a specific nat rule using nftables", func() {
				// Forward a specific port
				mockNetwork.EXPECT().IsIpv6Enabled().Return(true).Times(3)

				for _, proto := range ipProtocols() {
					mockNetwork.EXPECT().HasNatIptables(proto).Return(false).Times(2)

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

				driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
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

				driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
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
					Alias: &api.Alias{
						Name: "default",
					}})

				driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
				Expect(err).ToNot(HaveOccurred())
				TestRunPlug(driver)
				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
			})
		})
	})

	Context("Masquerade startDHCP", func() {
		It("should succeed when DHCP server started", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradePodInterface)
			Expect(ok).To(BeTrue())

			masq.vif.Gateway = masqueradeGwAddr.IP.To4()
			masq.vif.GatewayIpv6 = masqueradeIpv6GwAddr.IP.To16()
			mockNetwork.EXPECT().StartDHCP(masq.vif, gomock.Any(), masq.bridgeInterfaceName, nil).Return(nil)

			err = masq.startDHCP(vmi)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should fail when DHCP server failed", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradePodInterface)
			Expect(ok).To(BeTrue())

			masq.vif.Gateway = masqueradeGwAddr.IP.To4()
			masq.vif.GatewayIpv6 = masqueradeIpv6GwAddr.IP.To16()

			err = fmt.Errorf("failed to start DHCP server")
			mockNetwork.EXPECT().StartDHCP(masq.vif, gomock.Any(), masq.bridgeInterfaceName, nil).Return(err)

			err = masq.startDHCP(vmi)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Bridge startDHCP", func() {
		It("should succeed when DHCP server started", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgePodInterface)
			Expect(ok).To(BeTrue())

			mockNetwork.EXPECT().StartDHCP(bridge.vif, gomock.Any(), api.DefaultBridgeName, nil).Return(nil)

			err = bridge.startDHCP(vmi)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should fail when DHCP server failed", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgePodInterface)
			Expect(ok).To(BeTrue())

			err = fmt.Errorf("failed to start DHCP server")
			mockNetwork.EXPECT().StartDHCP(bridge.vif, gomock.Any(), api.DefaultBridgeName, nil).Return(err)

			err = bridge.startDHCP(vmi)
			Expect(err).To(HaveOccurred())
		})
		It("should succeed when DHCP server started and isLayer2 = true", func() {
			domain := NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgePodInterface)
			Expect(ok).To(BeTrue())

			bridge.vif.IPAMDisabled = true
			err = fmt.Errorf("failed to start DHCP server")
			mockNetwork.EXPECT().StartDHCP(bridge.vif, gomock.Any(), api.DefaultBridgeName, nil).Return(err)

			err = bridge.startDHCP(vmi)
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Context("Slirp startDHCP", func() {
		It("should succeed when DHCP server started", func() {
			domain := NewDomainWithSlirpInterface()
			vmi := newVMISlirpInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			driver, err := getPhase2Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], domain, podInterface)
			Expect(err).ToNot(HaveOccurred())
			slirp, ok := driver.(*SlirpPodInterface)
			Expect(ok).To(BeTrue())

			err = slirp.startDHCP(vmi)
			Expect(err).ToNot(HaveOccurred())
		})
		// slirp never fails to start DHCP because it doesn't need it at all
	})

	Context("Bridge loadCachedVIF", func() {
		It("should fail when nothing to load", func() {
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase1Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], podInterface)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgePodInterface)
			Expect(ok).To(BeTrue())

			succ, err := bridge.loadCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).To(HaveOccurred())
			Expect(succ).To(BeFalse())
		})
		It("should succeed when cache file present", func() {
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase1Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], podInterface)
			Expect(err).ToNot(HaveOccurred())
			bridge, ok := driver.(*BridgePodInterface)
			Expect(ok).To(BeTrue())

			err = bridge.setCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).ToNot(HaveOccurred())

			succ, err := bridge.loadCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).ToNot(HaveOccurred())
			Expect(succ).To(BeTrue())
		})
	})

	Context("Slirp loadCachedVIF", func() {
		It("should succeed", func() {
			vmi := newVMISlirpInterface("testnamespace", "testVmName")

			driver, err := getPhase1Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], podInterface)
			Expect(err).ToNot(HaveOccurred())
			slirp, ok := driver.(*SlirpPodInterface)
			Expect(ok).To(BeTrue())

			// it doesn't fail regardless, whether called without setCachedVIF...
			succ, err := slirp.loadCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).ToNot(HaveOccurred())
			Expect(succ).To(BeTrue())

			// ...or after it
			err = slirp.setCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).ToNot(HaveOccurred())

			succ, err = slirp.loadCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).ToNot(HaveOccurred())
			Expect(succ).To(BeTrue())
		})
	})

	Context("Masquerade loadCachedVIF", func() {
		It("should fail when nothing to load", func() {
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase1Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], podInterface)
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradePodInterface)
			Expect(ok).To(BeTrue())

			succ, err := masq.loadCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).To(HaveOccurred())
			Expect(succ).To(BeFalse())
		})
		It("should succeed when cache file present", func() {
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := getPhase1Binding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], &vmi.Spec.Networks[0], podInterface)
			Expect(err).ToNot(HaveOccurred())
			masq, ok := driver.(*MasqueradePodInterface)
			Expect(ok).To(BeTrue())

			err = masq.setCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).ToNot(HaveOccurred())

			succ, err := masq.loadCachedVIF(fmt.Sprintf("%d", pid), "fakename")
			Expect(err).ToNot(HaveOccurred())
			Expect(succ).To(BeTrue())
		})
	})

	It("should write interface to cache file", func() {
		uid := "test-1234"
		address1 := &net.IPNet{IP: net.IPv4(1, 2, 3, 4)}
		address2 := &net.IPNet{IP: net.IPv4(169, 254, 0, 0)}
		fakeAddr1 := netlink.Addr{IPNet: address1}
		fakeAddr2 := netlink.Addr{IPNet: address2}
		addrList := []netlink.Addr{fakeAddr1, fakeAddr2}
		err := os.MkdirAll(fmt.Sprintf(util.VMIInterfaceDir, uid), 0755)
		Expect(err).ToNot(HaveOccurred())

		iface := &v1.Interface{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
		mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
		mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_ALL).Return(addrList, nil)

		err = setPodInterfaceCache(iface, podInterface, uid)
		Expect(err).ToNot(HaveOccurred())

		data, err := ioutil.ReadFile(fmt.Sprintf(util.VMIInterfacepath, uid, iface.Name))
		Expect(err).ToNot(HaveOccurred())
		var podData *PodCacheInterface
		err = json.Unmarshal(data, &podData)
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
		Alias: &api.Alias{
			Name: "default",
		}},
	}
	return domain
}

func NewDomainWithSlirpInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: "e1000",
		},
		Type: "user",
		Alias: &api.Alias{
			Name: "default",
		}},
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
