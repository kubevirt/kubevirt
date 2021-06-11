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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

type Option func(handler *netdriver.MockNetworkHandler)

func newMockedBridgeConfigurator(
	vmi *v1.VirtualMachineInstance,
	iface *v1.Interface,
	handler *netdriver.MockNetworkHandler,
	bridgeIfaceName string,
	launcherPID int,
	options ...Option) *BridgePodNetworkConfigurator {
	configurator := NewBridgePodNetworkConfigurator(vmi, iface, bridgeIfaceName, launcherPID, handler)
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

	AfterEach(func() {
		ctrl.Finish()
	})

	const (
		bridgeIfaceName = "br0"
	)

	Context("discover link information", func() {
		const (
			ifaceName   = "eth0"
			launcherPID = 1000
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

		It("succeeds reading the pod link, pod IP, and routes", func() {
			bridgeConfigurator := newMockedBridgeConfigurator(
				vmi,
				iface,
				handler,
				bridgeIfaceName,
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
				bridgeIfaceName,
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
				bridgeIfaceName,
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
				bridgeIfaceName,
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
					bridgeIfaceName,
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
})

func newVMIWithBridgeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMIWithNS(namespace, name)
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
