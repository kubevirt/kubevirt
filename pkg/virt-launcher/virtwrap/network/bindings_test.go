package network

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/vishvananda/netlink"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("BindingMechanism", func() {
	var (
		mockNetwork                        *netdriver.MockNetworkHandler
		ctrl                               *gomock.Controller
		dummySwap                          *netlink.Dummy
		cacheFactory                       cache.InterfaceCacheFactory
		primaryPodInterface                *netlink.GenericLink
		primaryPodInterfaceAfterNameChange *netlink.GenericLink
		addrList                           []netlink.Addr
		newPodInterfaceName                string
		routeList                          []netlink.Route
		routeAddr                          netlink.Route
		fakeMac                            net.HardwareAddr
		fakeAddr                           netlink.Addr
		updateFakeMac                      net.HardwareAddr
		bridgeTest                         *netlink.Bridge
		masqueradeBridgeTest               *netlink.Bridge
		bridgeAddr                         *netlink.Addr
		testNic                            *cache.DhcpConfig
		//tmpDir                             string
		masqueradeTestNic    *cache.DhcpConfig
		masqueradeDummyName  string
		masqueradeDummy      *netlink.Dummy
		masqueradeGwStr      string
		masqueradeGwAddr     *netlink.Addr
		masqueradeGwIp       string
		masqueradeVmStr      string
		masqueradeVmAddr     *netlink.Addr
		masqueradeVmIp       string
		masqueradeIpv6GwStr  string
		masqueradeIpv6GwAddr *netlink.Addr
		masqueradeGwIpv6     string
		masqueradeIpv6VmStr  string
		masqueradeIpv6VmAddr *netlink.Addr
		masqueradeVmIpv6     string
		pid                  int
		tapDeviceName        string
		queueNumber          uint32
		mtu                  int
		libvirtUser          string
	)

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

	testDiscoverAndPrepare := func(driver BindMechanism, podInterfaceName string) {
		err := driver.discoverPodNetworkInterface(podInterfaceName)
		Expect(err).ToNot(HaveOccurred())

		Expect(driver.preparePodNetworkInterface()).To(Succeed())
		Expect(driver.decorateConfig(driver.generateDomainIfaceSpec())).To(Succeed())
	}
	testDiscoverAndPrepareWithoutIfaceName := func(driver BindMechanism) {
		testDiscoverAndPrepare(driver, "")
	}
	newVMI := func(namespace, name string) *kubevirtv1.VirtualMachineInstance {
		vmi := kubevirtv1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []kubevirtv1.Network{*v1.DefaultPodNetwork()}
		return vmi
	}
	ipProtocols := func() [2]iptables.Protocol {
		return [2]iptables.Protocol{iptables.ProtocolIPv4, iptables.ProtocolIPv6}
	}
	newDomainWithBridgeInterface := func() *api.Domain {
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
	newVMIMasqueradeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := newVMI(namespace, name)
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}
	newVMIBridgeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := newVMI(namespace, name)
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}
	testPodInterfaceIPBinding := func(vm *v1.VirtualMachineInstance, domain *api.Domain) {
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
		mockNetwork.EXPECT().StartDHCP(masqueradeTestNic, masqueradeGwAddr, api.DefaultBridgeName, nil, false)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMCIDR).Return(masqueradeGwStr, masqueradeVmStr, nil)
		mockNetwork.EXPECT().GetHostAndGwAddressesFromCIDR(api.DefaultVMIpv6CIDR).Return(masqueradeIpv6GwStr, masqueradeIpv6VmStr, nil)
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
		mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
		// Global nat rules using iptables
		for _, proto := range ipProtocols() {
			for _, chain := range []string{"OUTPUT", "KUBEVIRT_POSTINBOUND"} {
				mockNetwork.EXPECT().IptablesAppendRule(proto, "nat", chain,
					"-p", "tcp", "--match", "multiport",
					"--dports", fmt.Sprintf("%s", strings.Join(portsUsedByLiveMigration(), ",")),
					"--source", getLoopbackAdrress(proto), "-j", "RETURN").Return(nil)
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
				getLoopbackAdrress(proto),
				"-j",
				"SNAT",
				"--to-source",
				GetMasqueradeGwIp(proto)).Return(nil)
			mockNetwork.EXPECT().IptablesAppendRule(proto, "nat",
				"OUTPUT",
				"--destination",
				getLoopbackAdrress(proto),
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
				mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", chain, "tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(portsUsedByLiveMigration(), ", ")), GetNFTIPString(proto), "saddr", getLoopbackAdrress(proto), "counter", "return").Return(nil)
			}
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND", "counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND", GetNFTIPString(proto), "saddr", getLoopbackAdrress(proto), "counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil)
			mockNetwork.EXPECT().NftablesAppendRule(proto, "nat", "output", GetNFTIPString(proto), "daddr", getLoopbackAdrress(proto), "counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil)
		}
		mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
		mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
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

		masqueradeBridgeMAC, _ := net.ParseMAC(network.StaticMasqueradeBridgeMAC)
		masqueradeBridgeTest = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name:         api.DefaultBridgeName,
				MTU:          mtu,
				HardwareAddr: masqueradeBridgeMAC,
			},
		}

		bridgeAddr, _ = netlink.ParseAddr(fmt.Sprintf(bridgeFakeIP, 0))
		tapDeviceName = "tap0"
		testNic = &cache.DhcpConfig{Name: primaryPodInterfaceName,
			IP:                fakeAddr,
			MAC:               fakeMac,
			Mtu:               uint16(mtu),
			AdvertisingIPAddr: gw,
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
		masqueradeTestNic = &cache.DhcpConfig{Name: primaryPodInterfaceName,
			IP:                  *masqueradeVmAddr,
			IPv6:                *masqueradeIpv6VmAddr,
			MAC:                 fakeMac,
			Mtu:                 uint16(mtu),
			AdvertisingIPAddr:   masqueradeGwAddr.IP.To4(),
			AdvertisingIPv6Addr: masqueradeIpv6GwAddr.IP.To16()}

	})

	Context("when masquerade mechanism is selected", func() {
		It("should define a new DhcpConfig bind to a bridge and create a default nat rule using iptables", func() {

			// forward all the traffic
			for _, proto := range ipProtocols() {
				mockNetwork.EXPECT().NftablesLoad(proto).Return(fmt.Errorf("no nft"))
				mockNetwork.EXPECT().HasNatIptables(proto).Return(true).Times(2)
			}
			mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			domain := newDomainWithBridgeInterface()
			vm := newVMIMasqueradeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			testPodInterfaceIPBinding(vm, domain)
		})
		It("should define a new DhcpConfig bind to a bridge and create a specific nat rule using iptables", func() {
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

			domain := newDomainWithBridgeInterface()
			vm := newVMIMasqueradeInterface("testnamespace", "testVmName")
			vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			testPodInterfaceIPBinding(vm, domain)
		})
		It("should define a new DhcpConfig bind to a bridge and create a default nat rule using nftables", func() {
			// forward all the traffic
			for _, proto := range ipProtocols() {
				mockNetwork.EXPECT().NftablesLoad(proto).Return(nil)
			}
			mockNetwork.EXPECT().IsIpv6Enabled(primaryPodInterfaceName).Return(true, nil).Times(3)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			domain := newDomainWithBridgeInterface()
			vm := newVMIMasqueradeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			testPodInterfaceIPBinding(vm, domain)
		})
		It("should define a new DhcpConfig bind to a bridge and create a specific nat rule using nftables", func() {
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
					GetNFTIPString(proto), "saddr", "{ "+getLoopbackAdrress(proto)+" }",
					"counter", "snat", "to", GetMasqueradeGwIp(proto)).Return(nil).AnyTimes()
				mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
					"KUBEVIRT_PREINBOUND",
					"tcp",
					"dport",
					"80",
					"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
				mockNetwork.EXPECT().NftablesAppendRule(proto, "nat",
					"output",
					GetNFTIPString(proto), "daddr", "{ "+getLoopbackAdrress(proto)+" }",
					"tcp",
					"dport",
					"80",
					"counter", "dnat", "to", GetMasqueradeVmIp(proto)).Return(nil).AnyTimes()
			}

			domain := newDomainWithBridgeInterface()
			vm := newVMIMasqueradeInterface("testnamespace", "testVmName")
			vm.Spec.Domain.Devices.Interfaces[0].Ports = []v1.Port{{Name: "test", Port: 80, Protocol: "TCP"}}

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			testPodInterfaceIPBinding(vm, domain)
		})
	})
	Context("when bridge binding mechanism is selected", func() {
		It("should define a new DhcpConfig bind to a bridge", func() {
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			domain := newDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			testPodInterfaceIPBinding(vm, domain)
		})
		It("should populate MAC address", func() {
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			driver, err := newBridgeBinding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], nil, primaryPodInterfaceName, cacheFactory, nil, mockNetwork)
			Expect(err).ToNot(HaveOccurred())
			driver.ipamEnabled = true
			driver.podNicLink = primaryPodInterface
			Expect(driver.generateDhcpConfig().MAC.String()).To(Equal("de:ad:00:00:be:af"))
		})
	})
	Context("when slirp binding mechanism is selected", func() {
		var (
			domain *api.Domain
			driver SlirpBindMechanism
			vmi    *kubevirtv1.VirtualMachineInstance
		)
		NewDomainWithSlirpInterface := func() *api.Domain {
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
		newVMISlirpInterface := func(namespace string, name string) *kubevirtv1.VirtualMachineInstance {
			vmi := newVMI(namespace, name)
			vmi.Spec.Domain.Devices.Interfaces = []kubevirtv1.Interface{*kubevirtv1.DefaultSlirpNetworkInterface()}
			kubevirtv1.SetObjectDefaults_VirtualMachineInstance(vmi)
			return vmi
		}
		BeforeEach(func() {
			domain = NewDomainWithSlirpInterface()
			vmi = newVMISlirpInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			driver = SlirpBindMechanism{iface: &vmi.Spec.Domain.Devices.Interfaces[0], domain: domain}
		})
		It("Should create an interface in the qemu command line and remove it from the interfaces", func() {
			testDiscoverAndPrepareWithoutIfaceName(&driver)
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
			Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
		})
		It("Should append MAC address to qemu arguments if set", func() {
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			testDiscoverAndPrepareWithoutIfaceName(&driver)
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
			Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default,mac=de-ad-00-00-be-af"}))
		})
		It("Should create an interface in the qemu command line, remove it from the interfaces and leave the other interfaces inplace", func() {
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
			testDiscoverAndPrepareWithoutIfaceName(&driver)
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
			Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
		})
	})
	Context("when macvtap binding mechanism is selected", func() {
		var (
			domain           *api.Domain
			driver           *MacvtapBindMechanism
			vmi              *kubevirtv1.VirtualMachineInstance
			ifaceName        string
			macvtapInterface *netlink.GenericLink
			mtu              int
			cacheFactory     cache.InterfaceCacheFactory
			mockNetwork      *netdriver.MockNetworkHandler
		)
		NewDomainWithMacvtapInterface := func(macvtapName string) *api.Domain {
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
		newVMIMacvtapInterface := func(namespace string, vmiName string, ifaceName string) *kubevirtv1.VirtualMachineInstance {
			vmi := newVMI(namespace, vmiName)
			iface := kubevirtv1.DefaultMacvtapNetworkInterface(ifaceName)
			iface.MacAddress = "12:34:56:78:9a:bc"
			vmi.Spec.Domain.Devices.Interfaces = []kubevirtv1.Interface{*iface}
			kubevirtv1.SetObjectDefaults_VirtualMachineInstance(vmi)
			return vmi
		}
		BeforeEach(func() {
			cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
			ctrl := gomock.NewController(GinkgoT())
			mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
			mtu = 1410
			var err error
			ifaceName = "macvtap0"
			domain = NewDomainWithMacvtapInterface(ifaceName)
			vmi = newVMIMacvtapInterface("testnamespace", "default", ifaceName)
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			driver, err = newMacvtapBinding(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], domain, cacheFactory, nil, mockNetwork)
			Expect(err).ToNot(HaveOccurred())
			macvtapInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu, HardwareAddr: *driver.mac}}
		})
		It("Should pass a non-privileged macvtap interface to qemu", func() {
			mockNetwork.EXPECT().LinkByName(ifaceName).Return(macvtapInterface, nil)
			testDiscoverAndPrepare(driver, ifaceName)
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(domain.Spec.Devices.Interfaces[0].Target).To(Equal(&api.InterfaceTarget{Device: ifaceName, Managed: "no"}), "should have an unmanaged interface")
			Expect(domain.Spec.Devices.Interfaces[0].MAC).To(Equal(&api.MAC{MAC: vmi.Spec.Domain.Devices.Interfaces[0].MacAddress}), "should have the expected MAC address")
			Expect(domain.Spec.Devices.Interfaces[0].MTU).To(Equal(&api.MTU{Size: fmt.Sprintf("%d", mtu)}), "should have the expected MTU")

		})
	})
})
