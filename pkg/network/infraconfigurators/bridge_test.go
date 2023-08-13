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

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/api"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

type Option func(handler *netdriver.MockNetworkHandler)

func newMockedBridgeConfigurator(
	vmi *v1.VirtualMachineInstance,
	iface *v1.Interface,
	handler *netdriver.MockNetworkHandler,
	launcherPID int,
	options ...Option) *BridgePodNetworkConfigurator {
	configurator := NewBridgePodNetworkConfigurator(vmi, iface, launcherPID, handler)
	for _, option := range options {
		option(handler)
	}
	return configurator
}

var _ = Describe("Bridge infrastructure configurator", func() {
	var (
		ctrl    *gomock.Controller
		handler *netdriver.MockNetworkHandler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		handler = netdriver.NewMockNetworkHandler(ctrl)
	})

	const (
		bridgeIfaceName = "br0"
	)

	Context("discover link information", func() {
		const (
			ifaceName       = "eth0"
			bridgeIfaceName = "k6t-eth0"
			tapDeviceName   = "tap0"
			launcherPID     = 1000
		)

		var (
			defaultGwRoute netlink.Route
			iface          *v1.Interface
			podLink        *netlink.GenericLink
			podIP          netlink.Addr
			vmi            *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			iface = v1.DefaultBridgeNetworkInterface()
			vmi = newVMIWithBridgeInterface("default", "vm1")
			podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 1000}}
			podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}}
			defaultGwRoute = netlink.Route{Dst: nil, Gw: net.IPv4(10, 35, 0, 1)}
		})

		It("succeeds reading the pod link, and generate bridge iface and tap device names", func() {
			bridgeConfigurator := newMockedBridgeConfigurator(
				vmi,
				iface,
				handler,
				launcherPID,
				withLink(podLink),
				withIPOnLink(podLink, podIP),
				withRoutesOnLink(podLink, defaultGwRoute))
			Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
			Expect(bridgeConfigurator.podNicLink).To(Equal(podLink))
			Expect(bridgeConfigurator.bridgeInterfaceName).To(Equal(bridgeIfaceName))
			Expect(bridgeConfigurator.tapDeviceName).To(Equal(tapDeviceName))
		})

		It("succeeds reading the pod link, pod IP, and routes", func() {
			bridgeConfigurator := newMockedBridgeConfigurator(
				vmi,
				iface,
				handler,
				launcherPID,
				withLink(podLink),
				withIPOnLink(podLink, podIP),
				withRoutesOnLink(podLink, defaultGwRoute))
			Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
			Expect(bridgeConfigurator.podNicLink).To(Equal(podLink))
			Expect(bridgeConfigurator.podIfaceIP).To(Equal(podIP))
			Expect(bridgeConfigurator.podIfaceRoutes).To(ConsistOf(defaultGwRoute))
		})

		It("fails to discover pod information when the pod does not feature routes", func() {
			bridgeConfigurator := newMockedBridgeConfigurator(
				vmi,
				iface,
				handler,
				launcherPID,
				withLink(podLink),
				withIPOnLink(podLink, podIP),
				withRoutesOnLink(podLink))
			Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(
				MatchError(fmt.Sprintf("no gateway address found in routes for %s", ifaceName)))
		})

		It("fails to discover pod information when retrieving the pod link errors", func() {
			const errorString = "failed to read link"
			bridgeConfigurator := newMockedBridgeConfigurator(
				vmi,
				iface,
				handler,
				launcherPID,
				withErrorOnLinkRetrieval(podLink, errorString))
			Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(MatchError(errorString))
		})

		It("fails to discover pod information when retrieving the pod IP address errors", func() {
			const errorString = "failed to read IPs"
			bridgeConfigurator := newMockedBridgeConfigurator(
				vmi,
				iface,
				handler,
				launcherPID,
				withLink(podLink),
				withErrorOnIPRetrieval(podLink, errorString))
			Expect(bridgeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(MatchError(errorString))
		})

		When("the pod does not report an IP address", func() {
			var bridgeConfigurator *BridgePodNetworkConfigurator

			BeforeEach(func() {
				bridgeConfigurator = newMockedBridgeConfigurator(
					vmi,
					iface,
					handler,
					launcherPID,
					withLink(podLink),
					withIPOnLink(podLink))
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
	})

	Context("DHCP configuration generation", func() {
		const (
			ifaceName   = "eth0"
			launcherPID = 1000
		)

		var (
			iface *v1.Interface
			vmi   *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			iface = v1.DefaultBridgeNetworkInterface()
			vmi = newVMIWithBridgeInterface("default", "vm1")
		})

		When("IPAM is not enabled", func() {
			createBridgeConfiguratorWithoutIPAM := func() *BridgePodNetworkConfigurator {
				bc := NewBridgePodNetworkConfigurator(vmi, iface, launcherPID, handler)
				bc.podNicLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName}}
				return bc
			}

			It("should generate a minimal dhcp configuration", func() {
				expectedDhcpConfig := cache.DHCPConfig{IPAMDisabled: true}
				Expect(createBridgeConfiguratorWithoutIPAM().GenerateNonRecoverableDHCPConfig()).To(Equal(&expectedDhcpConfig))
			})
		})

		When("IPAM is enabled", func() {
			var (
				mac   net.HardwareAddr
				podIP netlink.Addr
			)

			createBridgeConfiguratorWithIPAM := func(mac net.HardwareAddr, podIP netlink.Addr, routes ...netlink.Route) *BridgePodNetworkConfigurator {
				bc := NewBridgePodNetworkConfigurator(vmi, iface, launcherPID, handler)
				bc.podNicLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName}}
				bc.vmMac = &mac
				bc.ipamEnabled = true
				bc.podIfaceIP = podIP
				bc.podIfaceRoutes = routes
				return bc
			}

			BeforeEach(func() {
				podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}}
				mac, _ = net.ParseMAC("AF:B3:1F:78:2A:CA")
			})

			When("routes are installed", func() {
				var (
					defaultGwRoute netlink.Route
				)

				BeforeEach(func() {
					defaultGwRoute = netlink.Route{Gw: net.IPv4(10, 35, 0, 1)}
				})

				It("generate a DHCP config also featuring the routes", func() {
					expectedDhcpConfig := cache.DHCPConfig{
						IPAMDisabled: false,
						MAC:          mac,
						IP:           podIP,
						Gateway:      defaultGwRoute.Gw,
					}
					Expect(createBridgeConfiguratorWithIPAM(
						mac, podIP, defaultGwRoute).GenerateNonRecoverableDHCPConfig()).To(Equal(&expectedDhcpConfig))
				})
			})

			It("generate a DHCP config only with MAC / IP address information, when the pod does not feature routes", func() {
				expectedDhcpConfig := cache.DHCPConfig{
					IPAMDisabled: false,
					MAC:          mac,
					IP:           podIP,
				}
				Expect(createBridgeConfiguratorWithIPAM(
					mac, podIP).GenerateNonRecoverableDHCPConfig()).To(Equal(&expectedDhcpConfig))
			})
		})
	})
})

func newVMIWithBridgeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := api.NewMinimalVMIWithNS(namespace, name)
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func withLink(link netlink.Link) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkByName(link.Attrs().Name).Return(link, nil)
	}
}

func withIPOnLink(link netlink.Link, ips ...netlink.Addr) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().AddrList(link, netlink.FAMILY_V4).Return(ips, nil)
	}
}

func withRoutesOnLink(link netlink.Link, routes ...netlink.Route) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().RouteList(link, netlink.FAMILY_V4).Return(routes, nil)
	}
}

func withErrorOnLinkRetrieval(link netlink.Link, expectedErrorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkByName(link.Attrs().Name).Return(nil, fmt.Errorf(expectedErrorString))
	}
}

func withErrorOnIPRetrieval(link netlink.Link, expectedErrorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().AddrList(link, netlink.FAMILY_V4).Return([]netlink.Addr{}, fmt.Errorf(expectedErrorString))
	}
}
