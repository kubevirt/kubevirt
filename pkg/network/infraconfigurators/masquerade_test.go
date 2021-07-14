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

type netfilterBackend int

const (
	nft netfilterBackend = iota
	ipTables
)

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

	newVMIMasqueradeInterface := func(namespace string, name string, ports ...int) *v1.VirtualMachineInstance {
		vmi := v1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
		var portList []v1.Port
		for i, port := range ports {
			portList = append(portList, v1.Port{
				Name:     fmt.Sprintf("port%d", i),
				Protocol: "tcp",
				Port:     int32(port),
			})
		}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
				Ports: portList,
			},
		}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}

	Context("discover link information", func() {
		const (
			ifaceName   = "eth0"
			launcherPID = 1000
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
				When("and is missing and IPv6 address", func() {
					BeforeEach(func() {
						handler.EXPECT().IsIpv6Enabled(ifaceName).Return(false, nil)
					})

					It("should succeed discovering the pod link info", func() {
						Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
						Expect(masqueradeConfigurator.podNicLink).To(Equal(podLink))
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

		When("the podnic link has invalid information", func() {
			BeforeEach(func() {
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 100000}}
				handler.EXPECT().LinkByName(ifaceName).Return(podLink, nil)
			})

			It("should fail to discover the pod's link information", func() {
				Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
			})
		})
	})

	Context("preparing network infrastructure", func() {
		const (
			ifaceName   = "eth0"
			ipv6GwStr   = "fd10:0:2::1/120"
			launcherPID = 1000
			namespace   = "testns"
			vmIPv6Str   = "fd10:0:2::2/120"
			vmName      = "vm1"
		)

		var (
			inPodBridge   *netlink.Bridge
			mtu           int
			podLink       *netlink.GenericLink
			gatewayAddr   *netlink.Addr
			podIP         netlink.Addr
			queueCount    uint32
			tapDeviceName string
			dhcpConfig    *cache.DHCPConfig
			ipv6VmAddr    *netlink.Addr
			ipv6GwAddr    *netlink.Addr
		)

		newConfigurator := func(vmi *v1.VirtualMachineInstance, link *netlink.GenericLink, vmIpAddr *netlink.Addr, gwAddr *netlink.Addr, ipv6VmAddr *netlink.Addr, ipv6GwAddr *netlink.Addr) *MasqueradePodNetworkConfigurator {
			masqueradeConfigurator := NewMasqueradePodNetworkConfigurator(
				vmi, &vmi.Spec.Domain.Devices.Interfaces[0], bridgeIfaceName, &vmi.Spec.Networks[0], launcherPID, handler)

			masqueradeConfigurator.podNicLink = podLink
			masqueradeConfigurator.vmGatewayAddr = gwAddr
			masqueradeConfigurator.vmIPv4Addr = *vmIpAddr
			masqueradeConfigurator.vmGatewayIpv6Addr = ipv6GwAddr
			masqueradeConfigurator.vmIPv6Addr = *ipv6VmAddr
			return masqueradeConfigurator
		}

		BeforeEach(func() {
			mtu = 1000
			queueCount = uint32(0)
			tapDeviceName = "tap0"
			gatewayAddr = &netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 0, 2, 1), Mask: net.CIDRMask(24, 32)}}
			podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 0, 2, 2), Mask: net.CIDRMask(24, 32)}}
		})

		BeforeEach(func() {
			inPodBridgeMAC, _ := net.ParseMAC("02:00:00:00:00:00")
			inPodBridge = &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeIfaceName, MTU: mtu, HardwareAddr: inPodBridgeMAC}}
			podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
		})

		BeforeEach(func() {
			ipv6GwAddr, _ = netlink.ParseAddr(ipv6GwStr)
			ipv6VmAddr, _ = netlink.ParseAddr(vmIPv6Str)
			dhcpConfig = &cache.DHCPConfig{
				Name:                ifaceName,
				IP:                  podIP,
				IPv6:                *ipv6VmAddr,
				Mtu:                 uint16(mtu),
				AdvertisingIPAddr:   gatewayAddr.IP.To4(),
				AdvertisingIPv6Addr: ipv6GwAddr.IP,
			}
		})

		When("the pod features a properly configured primary link", func() {
			BeforeEach(func() {
				handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
				handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
				handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
				handler.EXPECT().CreateTapDevice(tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
				handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, inPodBridge.Name).Return(nil)
			})

			table.DescribeTable("should work with", func(vmi *v1.VirtualMachineInstance, backend netfilterBackend, additionalIPProtocol ...iptables.Protocol) {
				masqueradeConfigurator := newConfigurator(vmi, podLink, &podIP, gatewayAddr, ipv6VmAddr, ipv6GwAddr)
				mockVML3Config(*masqueradeConfigurator, ifaceName, inPodBridge, additionalIPProtocol...)
				mockNATNetfilterRules(*masqueradeConfigurator, *dhcpConfig, backend, additionalIPProtocol...)

				Expect(masqueradeConfigurator.PreparePodNetworkInterface()).To(Succeed())
			},
				table.Entry("NFTables backend on an IPv4 cluster",
					newVMIMasqueradeInterface(namespace, vmName),
					nft),
				table.Entry("IPTables backend on an IPv4 cluster",
					newVMIMasqueradeInterface(namespace, vmName),
					ipTables),
				table.Entry("NFTables backend on an IPv4 cluster when specific ports are specified",
					newVMIMasqueradeInterface(namespace, vmName, 15000, 18000),
					nft),
				table.Entry("IPTables backend on an IPv4 cluster when specific ports are specified",
					newVMIMasqueradeInterface(namespace, vmName, 15000, 18000),
					ipTables),
				table.Entry("NFTables backend on a dual stack cluster",
					newVMIMasqueradeInterface(namespace, vmName),
					nft,
					iptables.ProtocolIPv6),
				table.Entry("IPTables backend on a dual stack cluster",
					newVMIMasqueradeInterface(namespace, vmName),
					ipTables,
					iptables.ProtocolIPv6),
				table.Entry("NFTables backend on a dual stack cluster when specific ports are specified",
					newVMIMasqueradeInterface(namespace, vmName, 15000, 18000),
					nft,
					iptables.ProtocolIPv6),
				table.Entry("IPTables backend on a dual stack cluster when specific ports are specified",
					newVMIMasqueradeInterface(namespace, vmName, 15000, 18000),
					ipTables,
					iptables.ProtocolIPv6),
			)
		})
	})
})

func mockVML3Config(configurator MasqueradePodNetworkConfigurator, podIface string, inPodBridge *netlink.Bridge, optionalIPProtocol ...iptables.Protocol) {
	protocols := getProtocols(optionalIPProtocol...)
	hasIPv6Config := len(protocols) > 1
	mockedHandler := configurator.handler.(*netdriver.MockNetworkHandler)
	mockedHandler.EXPECT().IsIpv6Enabled(podIface).Return(hasIPv6Config, nil).Times(2) // once on create bridge, another on prepare pod network

	for _, l3Protocol := range protocols {
		gatewayAddr := configurator.vmGatewayAddr
		if l3Protocol == iptables.ProtocolIPv6 {
			gatewayAddr = configurator.vmGatewayIpv6Addr
		}
		mockedHandler.EXPECT().AddrAdd(inPodBridge, gatewayAddr).Return(nil)
	}
}

func mockNATNetfilterRules(configurator MasqueradePodNetworkConfigurator, dhcpConfig cache.DHCPConfig, netfilterBackend netfilterBackend, optionalIPProtocol ...iptables.Protocol) {
	getNFTIPString := func(proto iptables.Protocol) string {
		ipString := "ip"
		if proto == iptables.ProtocolIPv6 {
			ipString = "ip6"
		}
		return ipString
	}

	handler := configurator.handler.(*netdriver.MockNetworkHandler)
	portList := getVMPrimaryInterfacePortList(*configurator.vmi)
	for _, proto := range getProtocols(optionalIPProtocol...) {
		vmIP := dhcpConfig.IP.IP.String()
		gwIP := dhcpConfig.AdvertisingIPAddr.String()
		if proto == iptables.ProtocolIPv6 {
			vmIP = dhcpConfig.IPv6.IP.String()
			gwIP = dhcpConfig.AdvertisingIPv6Addr.String()
		}

		mockNetfilterBackend(handler, proto, netfilterBackend, getNFTIPString(proto), vmIP, gwIP, portList)
	}
}

func getVMPrimaryInterfacePortList(vmi v1.VirtualMachineInstance) []int {
	var portList []int
	for _, port := range vmi.Spec.Domain.Devices.Interfaces[0].Ports {
		portList = append(portList, int(port.Port))
	}
	return portList
}

func mockNetfilterBackend(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, backendType netfilterBackend, nftIPString string, vmIP string, gwIP string, portList []int) {
	handler.EXPECT().ConfigureIpForwarding(proto).Return(nil)

	switch backendType {
	case nft:
		handler.EXPECT().NftablesLoad(proto).Return(nil)
		handler.EXPECT().HasNatIptables(proto).Return(true).Times(0)
		handler.EXPECT().HasNatIptables(proto).Return(false).Times(0)
		mockNFTablesBackend(handler, proto, nftIPString, vmIP, gwIP, portList)
	case ipTables:
		handler.EXPECT().NftablesLoad(proto).Return(fmt.Errorf("nft not found"))
		handler.EXPECT().HasNatIptables(proto).Return(true)
		mockIPTablesBackend(handler, proto, nftIPString, vmIP, gwIP, portList)
	}
}

func mockNFTablesBackend(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIPString string, vmIP string, gwIP string, portList []int) {
	handler.EXPECT().GetNFTIPString(proto).Return(nftIPString).AnyTimes()
	handler.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
	handler.EXPECT().NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)

	handler.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", nftIPString, "saddr", vmIP, "counter", "masquerade").Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "prerouting", "iifname", "eth0", "counter", "jump", "KUBEVIRT_PREINBOUND").Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "postrouting", "oifname", "k6t-eth0", "counter", "jump", "KUBEVIRT_POSTINBOUND").Return(nil)

	for _, chain := range []string{"output", "KUBEVIRT_POSTINBOUND"} {
		handler.EXPECT().NftablesAppendRule(proto, "nat", chain, "tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(PortsUsedByLiveMigration(), ", ")), nftIPString, "saddr", GetLoopbackAdrress(proto), "counter", "return").Return(nil)
	}

	if len(portList) > 0 {
		mockNFTablesBackendSpecificPorts(handler, proto, nftIPString, vmIP, gwIP, portList)
	} else {
		mockNFTablesBackendAllPorts(handler, proto, nftIPString, vmIP, gwIP)
	}
}

func mockNFTablesBackendSpecificPorts(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIpString string, vmIP string, gwIP string, portList []int) {
	for _, port := range portList {
		handler.EXPECT().NftablesAppendRule(proto, "nat",
			"KUBEVIRT_POSTINBOUND",
			"tcp",
			"dport",
			fmt.Sprintf("%d", port),
			nftIpString, "saddr", "{ "+GetLoopbackAdrress(proto)+" }",
			"counter", "snat", "to", gwIP).Return(nil)
		handler.EXPECT().NftablesAppendRule(proto, "nat",
			"KUBEVIRT_PREINBOUND",
			"tcp",
			"dport",
			fmt.Sprintf("%d", port),
			"counter", "dnat", "to", vmIP).Return(nil)
		handler.EXPECT().NftablesAppendRule(proto, "nat",
			"output",
			nftIpString, "daddr", "{ "+GetLoopbackAdrress(proto)+" }",
			"tcp",
			"dport",
			fmt.Sprintf("%d", port),
			"counter", "dnat", "to", vmIP).Return(nil)
	}
}

func mockNFTablesBackendAllPorts(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIPString string, vmIP string, gwIP string) {
	handler.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND", "counter", "dnat", "to", vmIP).Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND", nftIPString, "saddr", fmt.Sprintf("{ %s }", GetLoopbackAdrress(proto)), "counter", "snat", "to", gwIP).Return(nil)
	handler.EXPECT().NftablesAppendRule(proto, "nat", "output", nftIPString, "daddr", fmt.Sprintf("{ %s }", GetLoopbackAdrress(proto)), "counter", "dnat", "to", vmIP).Return(nil)
}

func mockIPTablesBackend(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, nftIPString string, vmIP string, gwIP string, portList []int) {
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

	for _, chain := range []string{"OUTPUT", "KUBEVIRT_POSTINBOUND"} {
		handler.EXPECT().IptablesAppendRule(proto, "nat", chain,
			"-p", "tcp", "--match", "multiport",
			"--dports", fmt.Sprintf("%s", strings.Join(PortsUsedByLiveMigration(), ",")),
			"--source", GetLoopbackAdrress(proto), "-j", "RETURN").Return(nil)
	}

	if len(portList) > 0 {
		mockIPTablesBackendSpecificPorts(handler, proto, vmIP, gwIP, portList)
	} else {
		mockIPTablesBackendAllPorts(handler, proto, vmIP, gwIP)
	}
}

func mockIPTablesBackendSpecificPorts(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, vmIP string, gwIP string, portList []int) {
	for _, port := range portList {
		handler.EXPECT().IptablesAppendRule(proto, "nat",
			"KUBEVIRT_POSTINBOUND",
			"-p",
			"tcp",
			"--dport",
			fmt.Sprintf("%d", port),
			"--source", GetLoopbackAdrress(proto),
			"-j", "SNAT", "--to-source", gwIP).Return(nil)
		handler.EXPECT().IptablesAppendRule(proto, "nat",
			"KUBEVIRT_PREINBOUND",
			"-p",
			"tcp",
			"--dport",
			fmt.Sprintf("%d", port), "-j", "DNAT", "--to-destination", vmIP).Return(nil)
		handler.EXPECT().IptablesAppendRule(proto, "nat",
			"OUTPUT",
			"-p",
			"tcp",
			"--dport",
			fmt.Sprintf("%d", port), "--destination", GetLoopbackAdrress(proto),
			"-j", "DNAT", "--to-destination", vmIP).Return(nil)
	}
}

func mockIPTablesBackendAllPorts(handler *netdriver.MockNetworkHandler, proto iptables.Protocol, vmIP string, gwIP string) {
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
}

func getProtocols(optionalIPProtocol ...iptables.Protocol) []iptables.Protocol {
	return append(
		[]iptables.Protocol{iptables.ProtocolIPv4},
		optionalIPProtocol...)
}
