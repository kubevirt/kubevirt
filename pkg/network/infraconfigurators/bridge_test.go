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
})
