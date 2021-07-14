package infraconfigurators

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/consts"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

type netfilterBackend int

const (
	nft netfilterBackend = iota
	ipTables
)

type masqueradeMockMachine struct {
	backend      netfilterBackend
	configurator *MasqueradePodNetworkConfigurator
	dhcpConfig   cache.DHCPConfig
	gwIP         string
	handler      *netdriver.MockNetworkHandler
	protocols    []iptables.Protocol
	vmIP         string
	vmi          *v1.VirtualMachineInstance
}

func newMasqueradeMockMachine(
	backend netfilterBackend,
	configurator *MasqueradePodNetworkConfigurator,
	dhcpConfig cache.DHCPConfig,
	optionalProtocols ...iptables.Protocol) masqueradeMockMachine {

	return masqueradeMockMachine{
		backend:      backend,
		configurator: configurator,
		dhcpConfig:   dhcpConfig,
		handler:      configurator.handler.(*netdriver.MockNetworkHandler),
		vmi:          configurator.vmi,
		protocols:    getProtocols(optionalProtocols...),
	}
}

func (mmm *masqueradeMockMachine) mockVML3Config(podIface string, inPodBridge *netlink.Bridge) {
	hasIPv6Config := len(mmm.protocols) > 1
	mmm.handler.EXPECT().IsIpv6Enabled(podIface).Return(hasIPv6Config, nil).Times(2) // once on create bridge, another on prepare pod network

	for _, l3Protocol := range mmm.protocols {
		gatewayAddr := mmm.configurator.vmGatewayAddr
		if l3Protocol == iptables.ProtocolIPv6 {
			gatewayAddr = mmm.configurator.vmGatewayIpv6Addr
		}
		mmm.handler.EXPECT().AddrAdd(inPodBridge, gatewayAddr).Return(nil)
	}
}

func (mmm *masqueradeMockMachine) mockNATNetfilterRules() {
	getNFTIPString := func(proto iptables.Protocol) string {
		ipString := "ip"
		if proto == iptables.ProtocolIPv6 {
			ipString = "ip6"
		}
		return ipString
	}

	for _, proto := range mmm.protocols {
		vmIP := mmm.dhcpConfig.IP.IP.String()
		gwIP := mmm.dhcpConfig.AdvertisingIPAddr.String()
		if proto == iptables.ProtocolIPv6 {
			vmIP = mmm.dhcpConfig.IPv6.IP.String()
			gwIP = mmm.dhcpConfig.AdvertisingIPv6Addr.String()
		}

		portList := mmm.getVMPrimaryInterfacePortList()
		mmm.mockNetfilterBackend(proto, getNFTIPString(proto), vmIP, gwIP, portList)
	}
}

func (mmm *masqueradeMockMachine) getVMPrimaryInterfacePortList() []int {
	var portList []int
	for _, port := range mmm.vmi.Spec.Domain.Devices.Interfaces[0].Ports {
		portList = append(portList, int(port.Port))
	}
	return portList
}

func (mmm *masqueradeMockMachine) mockNetfilterBackend(proto iptables.Protocol, nftIPString string, vmIP string, gwIP string, portList []int) {
	mmm.handler.EXPECT().ConfigureIpForwarding(proto).Return(nil)

	switch mmm.backend {
	case nft:
		mmm.handler.EXPECT().NftablesLoad(proto).Return(nil)
		mmm.handler.EXPECT().HasNatIptables(proto).Return(true).Times(0)
		mmm.handler.EXPECT().HasNatIptables(proto).Return(false).Times(0)
		nftMockMachine := nftBackendMockMachine{handler: mmm.handler, gwIP: gwIP, vmIP: vmIP, portList: portList, proto: proto, backendL3Prefix: nftIPString, annotations: mmm.vmi.Annotations}
		nftMockMachine.configureHandler()
	case ipTables:
		mmm.handler.EXPECT().NftablesLoad(proto).Return(fmt.Errorf("nft not found"))
		mmm.handler.EXPECT().HasNatIptables(proto).Return(true)
		iptablesMockMachine := iptablesBackendMockMachine{handler: mmm.handler, gwIP: gwIP, vmIP: vmIP, portList: portList, proto: proto, backendL3Prefix: nftIPString}
		iptablesMockMachine.configureHandler()
	}
}

type nftBackendMockMachine struct {
	annotations     map[string]string
	backendL3Prefix string
	gwIP            string
	handler         *netdriver.MockNetworkHandler
	portList        []int
	proto           iptables.Protocol
	vmIP            string
}

func (nbmm *nftBackendMockMachine) configureHandler() {
	nbmm.handler.EXPECT().GetNFTIPString(nbmm.proto).Return(nbmm.backendL3Prefix).AnyTimes()
	nbmm.handler.EXPECT().NftablesNewChain(nbmm.proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
	nbmm.handler.EXPECT().NftablesNewChain(nbmm.proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)

	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat", "postrouting", nbmm.backendL3Prefix, "saddr", nbmm.vmIP, "counter", "masquerade").Return(nil)
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat", "prerouting", "iifname", "eth0", "counter", "jump", "KUBEVIRT_PREINBOUND").Return(nil)
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat", "postrouting", "oifname", "k6t-eth0", "counter", "jump", "KUBEVIRT_POSTINBOUND").Return(nil)

	for _, chain := range []string{"output", "KUBEVIRT_POSTINBOUND"} {
		nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat", chain, "tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(PortsUsedByLiveMigration(), ", ")), nbmm.backendL3Prefix, "saddr", GetLoopbackAdrress(nbmm.proto), "counter", "return").Return(nil)
	}

	if len(nbmm.portList) > 0 {
		nbmm.mockNFTablesBackendSpecificPorts()
	} else {
		if isIstioAware(nbmm.annotations) {
			nbmm.mockIstioNetfilterCalls()
		} else {
			nbmm.mockNFTablesBackendAllPorts()
		}
	}
}

func (nbmm *nftBackendMockMachine) mockNFTablesBackendSpecificPorts() {
	for _, port := range nbmm.portList {
		nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat",
			"KUBEVIRT_POSTINBOUND",
			"tcp",
			"dport",
			fmt.Sprintf("%d", port),
			nbmm.backendL3Prefix, "saddr", "{ "+GetLoopbackAdrress(nbmm.proto)+" }",
			"counter", "snat", "to", nbmm.gwIP).Return(nil)
		nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat",
			"KUBEVIRT_PREINBOUND",
			"tcp",
			"dport",
			fmt.Sprintf("%d", port),
			"counter", "dnat", "to", nbmm.vmIP).Return(nil)
		nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat",
			"output",
			nbmm.backendL3Prefix, "daddr", "{ "+GetLoopbackAdrress(nbmm.proto)+" }",
			"tcp",
			"dport",
			fmt.Sprintf("%d", port),
			"counter", "dnat", "to", nbmm.vmIP).Return(nil)
	}
}

func (nbmm *nftBackendMockMachine) mockIstioNetfilterCalls() {
	for _, chain := range []string{"output", "KUBEVIRT_POSTINBOUND"} {
		nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat",
			chain, "tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(PortsUsedByIstio(), ", ")),
			nbmm.backendL3Prefix, "saddr", GetLoopbackAdrress(nbmm.proto), "counter", "return").Return(nil)
	}

	podIP := netlink.Addr{IPNet: &net.IPNet{IP: net.ParseIP("10.35.0.2"), Mask: net.CIDRMask(24, 32)}}
	srcAddressesToSnat := getSrcAddressesToSNAT(nbmm.proto)
	dstAddressesToDnat := getDstAddressesToDNAT(nbmm.proto, podIP)
	if nbmm.proto == iptables.ProtocolIPv4 {
		nbmm.handler.EXPECT().ReadIPAddressesFromLink("eth0").Return(podIP.IP.String(), "", nil)
	}
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat",
		"KUBEVIRT_POSTINBOUND", nbmm.backendL3Prefix, "saddr", fmt.Sprintf("{ %s }", strings.Join(srcAddressesToSnat, ", ")),
		"counter", "snat", "to", nbmm.gwIP).Return(nil)
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat",
		"output", nbmm.backendL3Prefix, "daddr", fmt.Sprintf("{ %s }", strings.Join(dstAddressesToDnat, ", ")),
		"counter", "dnat", "to", nbmm.vmIP).Return(nil)
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat",
		"KUBEVIRT_PREINBOUND",
		"counter", "dnat", "to", nbmm.vmIP).Return(nil).Times(0)
}

func (nbmm *nftBackendMockMachine) mockNFTablesBackendAllPorts() {
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat", "KUBEVIRT_PREINBOUND", "counter", "dnat", "to", nbmm.vmIP).Return(nil)
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat", "KUBEVIRT_POSTINBOUND", nbmm.backendL3Prefix, "saddr", fmt.Sprintf("{ %s }", GetLoopbackAdrress(nbmm.proto)), "counter", "snat", "to", nbmm.gwIP).Return(nil)
	nbmm.handler.EXPECT().NftablesAppendRule(nbmm.proto, "nat", "output", nbmm.backendL3Prefix, "daddr", fmt.Sprintf("{ %s }", GetLoopbackAdrress(nbmm.proto)), "counter", "dnat", "to", nbmm.vmIP).Return(nil)
}

type iptablesBackendMockMachine struct {
	backendL3Prefix string
	gwIP            string
	handler         *netdriver.MockNetworkHandler
	portList        []int
	proto           iptables.Protocol
	vmIP            string
}

func (ibmm *iptablesBackendMockMachine) configureHandler() {
	ibmm.handler.EXPECT().GetNFTIPString(ibmm.proto).Return(ibmm.backendL3Prefix).AnyTimes()
	ibmm.handler.EXPECT().IptablesNewChain(ibmm.proto, "nat", "KUBEVIRT_PREINBOUND").Return(nil)
	ibmm.handler.EXPECT().IptablesNewChain(ibmm.proto, "nat", "KUBEVIRT_POSTINBOUND").Return(nil)

	ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
		"POSTROUTING",
		"-s",
		ibmm.vmIP,
		"-j",
		"MASQUERADE").Return(nil)
	ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
		"PREROUTING",
		"-i",
		"eth0",
		"-j",
		"KUBEVIRT_PREINBOUND").Return(nil)
	ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
		"POSTROUTING",
		"-o",
		"k6t-eth0",
		"-j",
		"KUBEVIRT_POSTINBOUND").Return(nil)

	for _, chain := range []string{"OUTPUT", "KUBEVIRT_POSTINBOUND"} {
		ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat", chain,
			"-p", "tcp", "--match", "multiport",
			"--dports", fmt.Sprintf("%s", strings.Join(PortsUsedByLiveMigration(), ",")),
			"--source", GetLoopbackAdrress(ibmm.proto), "-j", "RETURN").Return(nil)
	}

	if len(ibmm.portList) > 0 {
		ibmm.mockIPTablesBackendSpecificPorts()
	} else {
		ibmm.mockIPTablesBackendAllPorts()
	}
}

func (ibmm *iptablesBackendMockMachine) mockIPTablesBackendSpecificPorts() {
	for _, port := range ibmm.portList {
		ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
			"KUBEVIRT_POSTINBOUND",
			"-p",
			"tcp",
			"--dport",
			fmt.Sprintf("%d", port),
			"--source", GetLoopbackAdrress(ibmm.proto),
			"-j", "SNAT", "--to-source", ibmm.gwIP).Return(nil)
		ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
			"KUBEVIRT_PREINBOUND",
			"-p",
			"tcp",
			"--dport",
			fmt.Sprintf("%d", port), "-j", "DNAT", "--to-destination", ibmm.vmIP).Return(nil)
		ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
			"OUTPUT",
			"-p",
			"tcp",
			"--dport",
			fmt.Sprintf("%d", port), "--destination", GetLoopbackAdrress(ibmm.proto),
			"-j", "DNAT", "--to-destination", ibmm.vmIP).Return(nil)
	}
}

func (ibmm *iptablesBackendMockMachine) mockIPTablesBackendAllPorts() {
	ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
		"KUBEVIRT_PREINBOUND",
		"-j",
		"DNAT",
		"--to-destination",
		ibmm.vmIP).Return(nil)
	ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
		"KUBEVIRT_POSTINBOUND",
		"--source",
		GetLoopbackAdrress(ibmm.proto),
		"-j",
		"SNAT",
		"--to-source",
		ibmm.gwIP).Return(nil)
	ibmm.handler.EXPECT().IptablesAppendRule(ibmm.proto, "nat",
		"OUTPUT",
		"--destination",
		GetLoopbackAdrress(ibmm.proto),
		"-j",
		"DNAT",
		"--to-destination",
		ibmm.vmIP).Return(nil)
}

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

	newIstioAwareVMIWithSingleInterface := func(namespace string, name string, ports ...int) *v1.VirtualMachineInstance {
		vmi := newVMIMasqueradeInterface(namespace, name, ports...)
		vmi.Annotations = map[string]string{
			consts.ISTIO_INJECT_ANNOTATION: "true",
		}
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
				mockMachine := newMasqueradeMockMachine(backend, masqueradeConfigurator, *dhcpConfig, additionalIPProtocol...)
				mockMachine.mockVML3Config(ifaceName, inPodBridge)
				mockMachine.mockNATNetfilterRules()

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
				table.Entry("NFTables backend on an IPv4 cluster when *reserved* ports are specified",
					newVMIMasqueradeInterface(namespace, vmName, getReservedPortList()...),
					nft),
				table.Entry("NFTables backend on an IPv4 cluster when using an ISTIO aware VMI",
					newIstioAwareVMIWithSingleInterface(namespace, vmName),
					nft),
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
				table.Entry("NFTables backend on a dual stack cluster when *reserved* ports are specified",
					newVMIMasqueradeInterface(namespace, vmName, getReservedPortList()...),
					nft,
					iptables.ProtocolIPv6),
				table.Entry("NFTables backend on a dual stack cluster when using an ISTIO aware VMI",
					newIstioAwareVMIWithSingleInterface(namespace, vmName),
					nft,
					iptables.ProtocolIPv6),
			)
		})
	})
})

func isIstioAware(vmiAnnotations map[string]string) bool {
	istioAnnotationValue, ok := vmiAnnotations[consts.ISTIO_INJECT_ANNOTATION]
	return ok && strings.ToLower(istioAnnotationValue) == "true"
}

func getSrcAddressesToSNAT(proto iptables.Protocol) []string {
	srcAddressesToSnat := []string{GetLoopbackAdrress(proto)}
	if proto == iptables.ProtocolIPv4 {
		srcAddressesToSnat = append(srcAddressesToSnat, GetEnvoyLoopbackAddress())
	}
	return srcAddressesToSnat
}

func getDstAddressesToDNAT(proto iptables.Protocol, podIP netlink.Addr) []string {
	dstAddressesToDnat := []string{GetLoopbackAdrress(proto)}
	if proto == iptables.ProtocolIPv4 {
		dstAddressesToDnat = append(dstAddressesToDnat, podIP.IP.String())
	}
	return dstAddressesToDnat
}

func getProtocols(optionalIPProtocol ...iptables.Protocol) []iptables.Protocol {
	return append(
		[]iptables.Protocol{iptables.ProtocolIPv4},
		optionalIPProtocol...)
}

func getReservedPortList() []int {
	var portList []int
	for _, port := range PortsUsedByLiveMigration() {
		intPort, err := strconv.ParseInt(port, 10, 64)
		if err != nil {
			Panic()
		}
		portList = append(portList, int(intPort))
	}
	return portList
}
