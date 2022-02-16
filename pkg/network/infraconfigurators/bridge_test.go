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

	"kubevirt.io/client-go/api"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

const (
	bridgeIPStr = "169.254.75.10/32"
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

	Context("prepare the pod networking infrastructure", func() {
		const (
			ifaceName     = "eth0"
			launcherPID   = 1000
			macStr        = "AF:B3:1F:78:2A:CA"
			mtu           = 1000
			queueCount    = uint32(0)
			tapDeviceName = "tap0"
		)

		var (
			bridgeIPAddr *netlink.Addr
			iface        *v1.Interface
			inPodBridge  *netlink.Bridge
			mac          net.HardwareAddr
			podLink      *netlink.GenericLink
			podIP        netlink.Addr
			vmi          *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			iface = v1.DefaultBridgeNetworkInterface()
			vmi = newVMIWithBridgeInterface("default", "vm1")
			mac, _ = net.ParseMAC(macStr)
			podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
		})

		When("the pod features an L3 configured network (IPAM)", func() {
			var (
				dummySwap              *netlink.Dummy
				podLinkAfterNameChange *netlink.GenericLink
			)

			BeforeEach(func() {
				bridgeIPAddr, _ = netlink.ParseAddr(bridgeIPStr)
				dummySwap = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: ifaceName}}
				inPodBridge = &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeIfaceName}}
				podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}}
				podLinkAfterNameChange = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: generateDummyIfaceName(ifaceName), MTU: mtu}}
			})

			newMockedBridgeConfiguratorForPreparePhase := func(vmi *v1.VirtualMachineInstance,
				iface *v1.Interface,
				handler *netdriver.MockNetworkHandler,
				bridgeIfaceName string,
				launcherPID int,
				link netlink.Link,
				podIP netlink.Addr,
				options ...Option) *BridgePodNetworkConfigurator {
				configurator := newMockedBridgeConfigurator(vmi, iface, handler, bridgeIfaceName, launcherPID, options...)
				configurator.podNicLink = link
				configurator.tapDeviceName = tapDeviceName
				configurator.ipamEnabled = true
				configurator.podIfaceIP = podIP
				return configurator
			}

			It("network preparation succeeds", func() {
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLinkAfterNameChange),
					withPodPrimaryLinkSwapped(podLink, podLinkAfterNameChange, dummySwap, podIP),
					withPodLinkRandomMac(podLinkAfterNameChange, mac),
					withARPIgnore(),
					withCreatedTapDevice(tapDeviceName, bridgeIfaceName, launcherPID, mtu, queueCount),
					withDisabledTxOffloadChecksum(bridgeIfaceName),
					withLinkLearningOff(podLinkAfterNameChange),
					withLinkUp(podLinkAfterNameChange))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(Succeed())
			})

			It("network preparation fails when setting the link down errors", func() {
				const errorString = "failed to set link down"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withErrorSettingDownPodLink(podLink, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when renaming the original pod link errors", func() {
				const errorString = "failed to rename the interface"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withErrorSwitchingIfaceName(podLink, podIP, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when creating the in-pod bridge errors", func() {
				const errorString = "failed to create the in-pod bridge"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withPodPrimaryLinkSwapped(podLink, podLinkAfterNameChange, dummySwap, podIP),
					withPodLinkRandomMac(podLinkAfterNameChange, mac),
					withARPIgnore(),
					withErrorCreatingBridge(*inPodBridge, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when adding the dummy device to perform the switcharoo errors", func() {
				const errorString = "failed to create the dummy device to hold the pod original IP"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withErrorAddingDummyDevice(podLink, podLinkAfterNameChange, dummySwap, podIP, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when setting the pod's IP address in the dummy errors", func() {
				const errorString = "failed to set the pod's original IP on the newly create dummy interface"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withErrorMovingPodIPAddressToDummy(podLink, podLinkAfterNameChange, dummySwap, podIP, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when deleting the pod's IP from the pod link errors", func() {
				const errorString = "failed todelete the original IP address from the pod link"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withErrorDeletingIPAddressFromPod(podLink, podIP, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when setting a random MAC in the pod link errors", func() {
				const errorString = "failed to set a random mac in the renamed pod link"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withPodPrimaryLinkSwapped(podLink, podLinkAfterNameChange, dummySwap, podIP),
					withARPIgnore(),
					withErrorRandomizingPodLinkMac(podLinkAfterNameChange, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when creating the tap device errors", func() {
				const errorString = "failed to create the tap device"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLinkAfterNameChange),
					withPodPrimaryLinkSwapped(podLink, podLinkAfterNameChange, dummySwap, podIP),
					withPodLinkRandomMac(podLinkAfterNameChange, mac),
					withARPIgnore(),
					withDisabledTxOffloadChecksum(bridgeIfaceName),
					withErrorCreatingTapDevice(tapDeviceName, mtu, launcherPID, queueCount, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when configuring ARP ignore errors", func() {
				const errorString = "failed to set bridge ARP ignore"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withPodPrimaryLinkSwapped(podLink, podLinkAfterNameChange, dummySwap, podIP),
					withErrorARPIgnore(errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when setting the pod link back up errors", func() {
				const errorString = "failed to set link back up"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLinkAfterNameChange),
					withPodPrimaryLinkSwapped(podLink, podLinkAfterNameChange, dummySwap, podIP),
					withPodLinkRandomMac(podLinkAfterNameChange, mac),
					withARPIgnore(),
					withCreatedTapDevice(tapDeviceName, bridgeIfaceName, launcherPID, mtu, queueCount),
					withDisabledTxOffloadChecksum(bridgeIfaceName),
					withErrorSettingPodLinkUp(podLinkAfterNameChange, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when setting the pod link learning off errors", func() {
				const errorString = "failed to set link learning off"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					podLink,
					podIP,
					withOriginalPodLinkDown(podLink),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLinkAfterNameChange),
					withPodPrimaryLinkSwapped(podLink, podLinkAfterNameChange, dummySwap, podIP),
					withPodLinkRandomMac(podLinkAfterNameChange, mac),
					withARPIgnore(),
					withCreatedTapDevice(tapDeviceName, bridgeIfaceName, launcherPID, mtu, queueCount),
					withDisabledTxOffloadChecksum(bridgeIfaceName),
					withLinkUp(podLinkAfterNameChange),
					withErrorSettingLinkLearningOff(podLinkAfterNameChange, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})
		})

		When("the pod features an L2 network (no IPAM)", func() {
			BeforeEach(func() {
				bridgeIPAddr, _ = netlink.ParseAddr(bridgeIPStr)
				inPodBridge = &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeIfaceName}}
				podIP = netlink.Addr{IPNet: &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}}
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
			})

			newMockedBridgeConfiguratorForPreparePhase := func(vmi *v1.VirtualMachineInstance,
				iface *v1.Interface,
				handler *netdriver.MockNetworkHandler,
				bridgeIfaceName string,
				launcherPID int,
				options ...Option) *BridgePodNetworkConfigurator {
				configurator := newMockedBridgeConfigurator(vmi, iface, handler, bridgeIfaceName, launcherPID, options...)
				configurator.podNicLink = podLink
				configurator.tapDeviceName = tapDeviceName
				return configurator
			}

			It("network preparation succeeds", func() {
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLink),
					withCreatedTapDevice(tapDeviceName, bridgeIfaceName, launcherPID, mtu, queueCount),
					withDisabledTxOffloadChecksum(bridgeIfaceName),
					withLinkLearningOff(podLink),
					withLinkUp(podLink))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(Succeed())
			})

			It("network preparation fails when setting the pod link down errors", func() {
				const errorString = "failed to set link down"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withErrorSettingDownPodLink(podLink, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when creating the in-pod bridge errors", func() {
				const errorString = "failed to create the in-pod bridge"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withErrorCreatingBridge(*inPodBridge, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when connecting the pod link to the in-pod bridge errors", func() {
				const errorString = "failed to connect the pod link to the in-pod bridge"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withErrorAddingPodLinkToBridge(inPodBridge, podLink, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when setting the in-pod bridge UP errors", func() {
				const errorString = "failed to set the in-pod bridge up"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withErrorSettingBridgeUp(inPodBridge, podLink, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when configuring the in-pod bridge IP address errors", func() {
				const errorString = "failed to set the in-pod bridge IP address"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withErrorSettingBridgeIPAddress(inPodBridge, podLink, bridgeIPAddr, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when disabling transaction checksum offloading on the errors", func() {
				const errorString = "failed to disable transaction checksum offloading"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLink),
					withErrorDisablingTXOffloadChecksum(inPodBridge.Name, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when creating the tap device errors", func() {
				const errorString = "failed to create the tap device"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLink),
					withDisabledTxOffloadChecksum(inPodBridge.Name),
					withErrorCreatingTapDevice(tapDeviceName, mtu, launcherPID, queueCount, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when setting the pod link back up errors", func() {
				const errorString = "failed to set pod link UP"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLink),
					withDisabledTxOffloadChecksum(inPodBridge.Name),
					withCreatedTapDevice(tapDeviceName, inPodBridge.Name, launcherPID, mtu, queueCount),
					withErrorSettingPodLinkUp(podLink, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
			})

			It("network preparation fails when setting the pod link learning off errors", func() {
				const errorString = "failed to set link learning off"
				bridgeConfigurator := newMockedBridgeConfiguratorForPreparePhase(
					vmi,
					iface,
					handler,
					bridgeIfaceName,
					launcherPID,
					withSwitchedPodLinkMac(podLink, mac),
					withCreatedInPodBridge(inPodBridge, bridgeIPAddr),
					withLinkAsBridgePort(inPodBridge, podLink),
					withDisabledTxOffloadChecksum(inPodBridge.Name),
					withCreatedTapDevice(tapDeviceName, inPodBridge.Name, launcherPID, mtu, queueCount),
					withLinkUp(podLink),
					withErrorSettingLinkLearningOff(podLink, errorString))
				Expect(bridgeConfigurator.PreparePodNetworkInterface()).To(MatchError(errorString))
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
				bc := NewBridgePodNetworkConfigurator(vmi, iface, bridgeIfaceName, launcherPID, handler)
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
				bc := NewBridgePodNetworkConfigurator(vmi, iface, bridgeIfaceName, launcherPID, handler)
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

func withSwitchedPodLinkMac(link netlink.Link, mac net.HardwareAddr) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetDown(link)
		handler.EXPECT().SetRandomMac(link.Attrs().Name).Return(mac, nil)
	}
}

func withLinkUp(link netlink.Link) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetUp(link)
	}
}

func withErrorSettingDownPodLink(link netlink.Link, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetDown(link).Return(fmt.Errorf(errorString))
	}
}

func withErrorCreatingBridge(bridge netlink.Bridge, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkAdd(&bridge).Return(fmt.Errorf(errorString))
	}
}

func withCreatedInPodBridge(bridge *netlink.Bridge, bridgeIP *netlink.Addr) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkAdd(bridge)
		handler.EXPECT().LinkSetUp(bridge)
		handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIP, nil)
		handler.EXPECT().AddrAdd(bridge, bridgeIP)
	}
}

func withLinkAsBridgePort(bridge *netlink.Bridge, link netlink.Link) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetMaster(link, bridge)
	}
}

func withErrorAddingPodLinkToBridge(bridge *netlink.Bridge, link netlink.Link, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkAdd(bridge)
		handler.EXPECT().LinkSetMaster(link, bridge).Return(fmt.Errorf(errorString))
	}
}

func withErrorSettingBridgeUp(bridge *netlink.Bridge, link netlink.Link, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetMaster(link, bridge)
		handler.EXPECT().LinkAdd(bridge)
		handler.EXPECT().LinkSetUp(bridge).Return(fmt.Errorf(errorString))
	}
}

func withErrorSettingBridgeIPAddress(bridge *netlink.Bridge, link netlink.Link, bridgeIP *netlink.Addr, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetMaster(link, bridge)
		handler.EXPECT().LinkAdd(bridge)
		handler.EXPECT().LinkSetUp(bridge)
		handler.EXPECT().ParseAddr(bridgeIPStr).Return(bridgeIP, nil)
		handler.EXPECT().AddrAdd(bridge, bridgeIP).Return(fmt.Errorf(errorString))
	}
}

func withErrorDisablingTXOffloadChecksum(bridgeName string, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().DisableTXOffloadChecksum(bridgeName).Return(fmt.Errorf(errorString))
	}
}

func withCreatedTapDevice(tapDeviceName string, bridgeName string, launcherPID int, mtu int, queueCount uint32) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().CreateTapDevice(tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId)
		handler.EXPECT().BindTapDeviceToBridge(tapDeviceName, bridgeName)
	}
}

func withErrorCreatingTapDevice(tapDeviceName string, mtu int, launcherPID int, queueCount uint32, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().CreateTapDevice(
			tapDeviceName, queueCount, launcherPID, mtu, netdriver.LibvirtUserAndGroupId).Return(
			fmt.Errorf(errorString))
	}
}

func withDisabledTxOffloadChecksum(bridgeName string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().DisableTXOffloadChecksum(bridgeName)
	}
}

func withLinkLearningOff(link netlink.Link) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetLearningOff(link)
	}
}

func withErrorSettingPodLinkUp(link netlink.Link, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetUp(link).Return(fmt.Errorf(errorString))
	}
}

func withErrorSettingLinkLearningOff(link netlink.Link, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetLearningOff(link).Return(fmt.Errorf(errorString))
	}
}

func withOriginalPodLinkDown(link netlink.Link) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetDown(link)
	}
}

func withPodPrimaryLinkSwapped(oldPodLink netlink.Link, renamedPodLink netlink.Link, newDummy netlink.Link, ip netlink.Addr) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkAdd(newDummy)
		handler.EXPECT().AddrReplace(newDummy, &ip)
		handler.EXPECT().AddrDel(oldPodLink, &ip)
		handler.EXPECT().LinkSetName(oldPodLink, renamedPodLink.Attrs().Name)
		handler.EXPECT().LinkByName(renamedPodLink.Attrs().Name).Return(renamedPodLink, nil)
	}
}

func withPodLinkRandomMac(link netlink.Link, mac net.HardwareAddr) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().SetRandomMac(link.Attrs().Name).Return(mac, nil)
	}
}

func withErrorRandomizingPodLinkMac(link netlink.Link, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().SetRandomMac(link.Attrs().Name).Return(nil, fmt.Errorf(errorString))
	}
}

func withARPIgnore() Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().ConfigureIpv4ArpIgnore()
	}
}

func withErrorARPIgnore(errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().ConfigureIpv4ArpIgnore().Return(fmt.Errorf(errorString))
	}
}

func withErrorSwitchingIfaceName(link netlink.Link, ip netlink.Addr, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().LinkSetDown(link)
		handler.EXPECT().AddrDel(link, &ip)
		handler.EXPECT().LinkSetName(link, generateDummyIfaceName(link.Attrs().Name)).Return(fmt.Errorf(errorString))
	}
}

func withErrorDeletingIPAddressFromPod(link netlink.Link, ip netlink.Addr, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().AddrDel(link, &ip).Return(fmt.Errorf(errorString))
	}
}

func withErrorAddingDummyDevice(oldLink netlink.Link, newLink netlink.Link, dummyLink netlink.Link, ip netlink.Addr, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().AddrDel(oldLink, &ip)
		handler.EXPECT().LinkSetName(oldLink, generateDummyIfaceName(oldLink.Attrs().Name))
		handler.EXPECT().LinkByName(newLink.Attrs().Name).Return(newLink, nil)
		handler.EXPECT().LinkAdd(dummyLink).Return(fmt.Errorf(errorString))
	}
}

func withErrorMovingPodIPAddressToDummy(oldLink netlink.Link, newLink netlink.Link, dummy netlink.Link, ip netlink.Addr, errorString string) Option {
	return func(handler *netdriver.MockNetworkHandler) {
		handler.EXPECT().AddrDel(oldLink, &ip)
		handler.EXPECT().LinkSetName(oldLink, newLink.Attrs().Name)
		handler.EXPECT().LinkByName(newLink.Attrs().Name).Return(newLink, nil)
		handler.EXPECT().LinkAdd(dummy)
		handler.EXPECT().AddrReplace(dummy, &ip).Return(fmt.Errorf(errorString))
	}
}

func generateDummyIfaceName(ifaceName string) string {
	return ifaceName + "-nic"
}
