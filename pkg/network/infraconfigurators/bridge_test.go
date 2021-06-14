package infraconfigurators

import (
	"fmt"
	"net"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

var _ = Describe("Bridge infrastructure configurator", func() {
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
		bridgeIfaceName = "br0"
	)

	newVMIBridgeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := v1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}

	Context("discover link information", func() {
		const (
			ifaceName   = "eth0"
			launcherPID = 1000
		)

		var (
			bridgeConfigurator *BridgePodNetworkConfigurator
			defaultGwRoute     netlink.Route
			iface              *v1.Interface
			podLink            *netlink.GenericLink
			podIP              netlink.Addr
			vmi                *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			iface = v1.DefaultBridgeNetworkInterface()
			vmi = newVMIBridgeInterface("default", "vm1")

			bridgeConfigurator = NewBridgePodNetworkConfigurator(vmi, iface, bridgeIfaceName, launcherPID, handler)
		})

		When("the pod link is defined", func() {
			BeforeEach(func() {
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 1000}}
				handler.EXPECT().LinkByName(ifaceName).Return(podLink, nil)
			})

			When("the pod's IP addresses are defined", func() {
				BeforeEach(func() {
					podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}}
					handler.EXPECT().AddrList(podLink, netlink.FAMILY_V4).Return([]netlink.Addr{podIP}, nil)
				})

				When("the pod's routes are defined", func() {
					BeforeEach(func() {
						defaultGwRoute = netlink.Route{Dst: nil, Gw: net.IPv4(10, 35, 0, 1)}
						handler.EXPECT().RouteList(podLink, netlink.FAMILY_V4).Return([]netlink.Route{defaultGwRoute}, nil)
					})

					It("should succeed reading the pod link info", func() {
						Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
						Expect(bridgeConfigurator.podNicLink).To(Equal(podLink))
					})

					It("should succeed reading the pod IP addresses", func() {
						Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
						Expect(bridgeConfigurator.podIfaceIP).To(Equal(podIP))
					})

					It("should succeed reading the routes", func() {
						Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
						Expect(bridgeConfigurator.podIfaceRoutes).To(ConsistOf(defaultGwRoute))
					})
				})

				When("the pod does not feature routes", func() {
					BeforeEach(func() {
						handler.EXPECT().RouteList(podLink, netlink.FAMILY_V4).Return([]netlink.Route{}, nil)
					})

					It("should fail", func() {
						Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
					})
				})
			})

			When("the pod does not report IP addresses", func() {
				BeforeEach(func() {
					handler.EXPECT().AddrList(podLink, netlink.FAMILY_V4).Return([]netlink.Addr{}, nil)
				})

				It("should report disabled IPAM and miss the IP address field", func() {
					Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
					Expect(bridgeConfigurator.podIfaceIP).To(Equal(netlink.Addr{}))
					Expect(bridgeConfigurator.ipamEnabled).To(BeFalse())
				})

				It("should not care about missing routes when an IP was not found", func() {
					handler.EXPECT().RouteList(podLink, netlink.FAMILY_V4).Return([]netlink.Route{}, nil).Times(0)
					Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
				})
			})

			When("the pod errors while attempting to retrieve the IP address", func() {
				BeforeEach(func() {
					podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 1000}}
					handler.EXPECT().AddrList(podLink, netlink.FAMILY_V4).Return([]netlink.Addr{}, fmt.Errorf("failed to read IPs"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
				})
			})
		})

		When("the pod link cannot be read", func() {
			BeforeEach(func() {
				handler.EXPECT().LinkByName(ifaceName).Return(nil, fmt.Errorf("failed to read link"))
			})

			It("should fail when pod link cannot be read", func() {
				Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
			})
		})

		When("the pod link features an invalid MTU", func() {
			BeforeEach(func() {
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 700000}}
				handler.EXPECT().LinkByName(ifaceName).Return(podLink, nil)
			})

			It("should fail", func() {
				Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
			})
		})
	})

	Context("prepre the pod networking infrastructure", func() {
		const (
			ifaceName   = "eth0"
			launcherPID = 1000
		)

		var (
			bridgeConfigurator *BridgePodNetworkConfigurator
			bridgeIPAddr       *netlink.Addr
			bridgeIPStr        string
			iface              *v1.Interface
			inPodBridge        *netlink.Bridge
			mac                net.HardwareAddr
			mtu                int
			podLink            *netlink.GenericLink
			podIP              netlink.Addr
			queueCount         uint32
			tapDeviceName      string
			vmi                *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			iface = v1.DefaultBridgeNetworkInterface()
			vmi = newVMIBridgeInterface("default", "vm1")

			bridgeConfigurator = NewBridgePodNetworkConfigurator(vmi, iface, bridgeIfaceName, launcherPID, handler)
		})

		BeforeEach(func() {
			macStr := "AF:B3:1F:78:2A:CA"
			mac, _ = net.ParseMAC(macStr)
			mtu = 1000
			podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
			queueCount = uint32(0)
			tapDeviceName = "tap0"
		})

		When("the pod features an L3 configured network (IPAM)", func() {
			var (
				dummySwap              *netlink.Dummy
				podLinkAfterNameChange *netlink.GenericLink
			)

			BeforeEach(func() {
				bridgeIPStr = fmt.Sprintf(bridgeFakeIP, 0)
				bridgeIPAddr, _ = netlink.ParseAddr(bridgeIPStr)
				dummySwap = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: ifaceName}}
				inPodBridge = &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeIfaceName}}
				podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}}
				podLinkAfterNameChange = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: generateDummyIfaceName(ifaceName), MTU: mtu}}
			})

			BeforeEach(func() {
				bridgeConfigurator.podNicLink = podLink
				bridgeConfigurator.tapDeviceName = "tap0"
				bridgeConfigurator.ipamEnabled = true
				bridgeConfigurator.podIfaceIP = podIP
			})

			When("all network driver calls succeed", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetMaster(podLinkAfterNameChange, inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
					handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, inPodBridge.Name).Return(nil)
					handler.EXPECT().LinkSetUp(podLinkAfterNameChange).Return(nil)
					handler.EXPECT().LinkSetLearningOff(podLinkAfterNameChange).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName))
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(nil)
					handler.EXPECT().SetRandomMac(generateDummyIfaceName(ifaceName)).Return(mac, nil)
					handler.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)
				})

				It("should work", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(Succeed())
				})
			})

			When("we fail to set the link down", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(fmt.Errorf("failed to set link down"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to rename the original pod link", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(fmt.Errorf("failed to rename the interface"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to create the in-pod bridge", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(nil)
					handler.EXPECT().SetRandomMac(generateDummyIfaceName(ifaceName)).Return(mac, nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(fmt.Errorf("failed to create the in pod bridge"))
					handler.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to add the dummy device to perform the switcharoo", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(fmt.Errorf("failed to create the dummy interface"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to set the pod's IP address in the dummy", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(fmt.Errorf("failed to configure the dummy's IP"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to replace the pod link", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(nil, fmt.Errorf("failed to retrieve the renamed link"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to delete the pod's IP from the pod link", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(fmt.Errorf("failed to delete the pod IP from the pod link"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to set a random MAC in the pod link", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(nil)
					handler.EXPECT().SetRandomMac(generateDummyIfaceName(ifaceName)).Return(nil, fmt.Errorf("failed to set a random mac in the pod link"))
					handler.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to create the tap device", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(nil)
					handler.EXPECT().SetRandomMac(generateDummyIfaceName(ifaceName)).Return(mac, nil)
					handler.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetMaster(podLinkAfterNameChange, inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(
						tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(
						fmt.Errorf("failed to create the tap device"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to configure ARP ignore", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(nil)
					handler.EXPECT().ConfigureIpv4ArpIgnore().Return(fmt.Errorf("failed to configure ARP ignore"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to set the pod link back up", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(nil)
					handler.EXPECT().SetRandomMac(generateDummyIfaceName(ifaceName)).Return(mac, nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetMaster(podLinkAfterNameChange, inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(
						tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
					handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, inPodBridge.Name).Return(nil)
					handler.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)
					handler.EXPECT().LinkSetUp(podLinkAfterNameChange).Return(fmt.Errorf("failed to set pod link UP"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to set the pod link learning off", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().AddrDel(podLink, &podIP).Return(nil)
					handler.EXPECT().LinkSetName(podLink, generateDummyIfaceName(ifaceName)).Return(nil)
					handler.EXPECT().LinkByName(generateDummyIfaceName(ifaceName)).Return(podLinkAfterNameChange, nil)
					handler.EXPECT().LinkAdd(dummySwap).Return(nil)
					handler.EXPECT().AddrReplace(dummySwap, &podIP).Return(nil)
					handler.EXPECT().SetRandomMac(generateDummyIfaceName(ifaceName)).Return(mac, nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetMaster(podLinkAfterNameChange, inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(
						tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
					handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, inPodBridge.Name).Return(nil)
					handler.EXPECT().ConfigureIpv4ArpIgnore().Return(nil)
					handler.EXPECT().LinkSetUp(podLinkAfterNameChange).Return(nil)
					handler.EXPECT().LinkSetLearningOff(podLinkAfterNameChange).Return(fmt.Errorf("failed to set link learning off"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})
		})

		When("the pod features an L2 network (no IPAM)", func() {
			BeforeEach(func() {
				bridgeIPStr = fmt.Sprintf(bridgeFakeIP, 0)
				bridgeIPAddr, _ = netlink.ParseAddr(bridgeIPStr)
				inPodBridge = &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeIfaceName}}
				podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}}
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
			})

			BeforeEach(func() {
				bridgeConfigurator.podNicLink = podLink
				bridgeConfigurator.tapDeviceName = "tap0"
			})

			When("all network calls succeed", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
					handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, inPodBridge.Name).Return(nil)
					handler.EXPECT().LinkSetUp(podLink).Return(nil)
					handler.EXPECT().LinkSetLearningOff(podLink).Return(nil)
				})

				It("should work", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(Succeed())
				})
			})

			When("we fail to set the link down", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(fmt.Errorf("failed to set link down"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to create the in-pod bridge", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(fmt.Errorf("failed to create the in pod bridge"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to connect the pod link to the in pod bridge", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(fmt.Errorf("failed to connect the pod link to the in pod bridge"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to set the in pod bridge UP", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(fmt.Errorf("failed to set the in pod bridge up"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail the the in pod bridge IP address", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(fmt.Errorf("failed to set the in pod bridge IP address"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to disable transaction checksum offloading on the bridge", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(fmt.Errorf("failed to disable transaction checksum offloading"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to create the tap device", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(
						tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(
						fmt.Errorf("failed to create the tap device"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to set the pod link back up", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(
						tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
					handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, inPodBridge.Name).Return(nil)
					handler.EXPECT().LinkSetUp(podLink).Return(fmt.Errorf("failed to set pod link UP"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})

			When("we fail to set the pod link learning off", func() {
				BeforeEach(func() {
					handler.EXPECT().LinkSetDown(podLink).Return(nil)
					handler.EXPECT().SetRandomMac(ifaceName).Return(mac, nil)
					handler.EXPECT().LinkSetMaster(podLink, inPodBridge).Return(nil)
					handler.EXPECT().LinkAdd(inPodBridge).Return(nil)
					handler.EXPECT().LinkSetUp(inPodBridge).Return(nil)
					handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIPAddr, nil)
					handler.EXPECT().AddrAdd(inPodBridge, bridgeIPAddr).Return(nil)
					handler.EXPECT().DisableTXOffloadChecksum(inPodBridge.Name).Return(nil)
					handler.EXPECT().CreateTapDevice(
						tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(nil)
					handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, inPodBridge.Name).Return(nil)
					handler.EXPECT().LinkSetUp(podLink).Return(nil)
					handler.EXPECT().LinkSetLearningOff(podLink).Return(fmt.Errorf("failed to set link learning off"))
				})

				It("should fail", func() {
					Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(HaveOccurred())
				})
			})
		})
	})
})

func generateDummyIfaceName(ifaceName string) string {
	return ifaceName + "-nic"
}
