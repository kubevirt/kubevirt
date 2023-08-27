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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package netpod_test

import (
	"errors"

	"kubevirt.io/kubevirt/pkg/network/driver/procsys"

	"kubevirt.io/kubevirt/pkg/pointer"

	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod"
)

const (
	defaultPodNetworkName = "default"
)

var (
	ipDisabled = nmstate.IP{Enabled: pointer.P(false)}
)

var _ = Describe("netpod", func() {
	It("fails setup when reading nmstate status fails", func() {
		netPod := netpod.NewNetPod(
			nil, nil, 0, 0, 0,
			netpod.WithNMStateAdapter(&nmstateStub{readErr: errNMStateRead}),
		)
		Expect(netPod.Setup()).To(MatchError(errNMStateRead))
	})

	It("fails setup when applying nmstate status fails", func() {
		netPod := netpod.NewNetPod(
			nil, nil, 0, 0, 0,
			netpod.WithNMStateAdapter(&nmstateStub{applyErr: errNMStateApply}),
		)
		Expect(netPod.Setup()).To(MatchError(errNMStateApply))
	})

	It("fails setup when applying nmstate status with undefined binding", func() {
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{Name: defaultPodNetworkName}},
			0, 0, 0,
			netpod.WithNMStateAdapter(&nmstateStub{status: nmstate.Status{
				Interfaces: []nmstate.Interface{{
					Name:       "eth0",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "12:34:56:78:90:ab",
					MTU:        1500,
				}},
			}}),
		)
		err := netPod.Setup()
		Expect(err.Error()).To(HavePrefix("undefined binding method"))
	})

	It("fails setup when masquerade (nft) setup fails", func() {
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			}},
			0, 0, 0,
			netpod.WithNMStateAdapter(&nmstateStub{status: nmstate.Status{
				Interfaces: []nmstate.Interface{{
					Name:       "eth0",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "12:34:56:78:90:ab",
					MTU:        1500,
				}},
			}}),
			netpod.WithMasqueradeAdapter(&masqueradeStub{setupErr: errMasqueradeSetup}),
		)
		Expect(netPod.Setup()).To(MatchError(errMasqueradeSetup))
	})

	It("setup masquerade binding", func() {
		nmstatestub := nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "12:34:56:78:90:ab",
				MTU:        1500,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "10.222.222.1",
						PrefixLen: 30,
					}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "2001::1",
						PrefixLen: 64,
					}},
				},
			}},
		}}
		masqstub := masqueradeStub{}

		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			}},
			0, 0, 0,
			netpod.WithNMStateAdapter(&nmstatestub),
			netpod.WithMasqueradeAdapter(&masqstub),
		)
		Expect(netPod.Setup()).To(Succeed())
		Expect(nmstatestub.spec).To(Equal(
			nmstate.Spec{
				Interfaces: []nmstate.Interface{
					{
						Name:       "k6t-eth0",
						TypeName:   nmstate.TypeBridge,
						State:      nmstate.IfaceStateUp,
						MacAddress: "02:00:00:00:00:00",
						MTU:        1500,
						Ethtool:    nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
						IPv4: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
						},
						IPv6: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
						},
						LinuxStack: nmstate.LinuxIfaceStack{IP4RouteLocalNet: pointer.P(true)},
						Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
					{
						Name:       "tap0",
						TypeName:   nmstate.TypeTap,
						State:      nmstate.IfaceStateUp,
						MTU:        1500,
						Controller: "k6t-eth0",
						Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
						Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
				},
				LinuxStack: nmstate.LinuxStack{
					IPv4: nmstate.LinuxStackIP4{Forwarding: pointer.P(true)},
					IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
				},
			}),
		)
		Expect(masqstub.bridgeIfaceSpec.Name).To(Equal("k6t-eth0"))
		Expect(masqstub.podIfaceSpec.Name).To(Equal("eth0"))
		Expect(masqstub.vmiIfaceSpec.Name).To(Equal(defaultPodNetworkName))
	})

	It("setup bridge binding with IP", func() {
		nmstatestub := nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "12:34:56:78:90:ab",
				MTU:        1500,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "10.222.222.1",
						PrefixLen: 30,
					}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "2001::1",
						PrefixLen: 64,
					}},
				},
			}},
		}}

		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			}},
			0, 0, 0,
			netpod.WithNMStateAdapter(&nmstatestub),
		)
		Expect(netPod.Setup()).To(Succeed())
		Expect(nmstatestub.spec).To(Equal(
			nmstate.Spec{
				Interfaces: []nmstate.Interface{
					{
						Name:     "k6t-eth0",
						TypeName: nmstate.TypeBridge,
						State:    nmstate.IfaceStateUp,
						Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
						IPv4: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{IP: "169.254.75.10", PrefixLen: 32}},
						},
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
					{
						Name:        "eth0-nic",
						Index:       0,
						CopyMacFrom: "k6t-eth0",
						Controller:  "k6t-eth0",
						IPv4:        ipDisabled,
						IPv6:        ipDisabled,
						LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
						Metadata:    &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
					{
						Name:       "tap0",
						TypeName:   nmstate.TypeTap,
						State:      nmstate.IfaceStateUp,
						MTU:        1500,
						Controller: "k6t-eth0",
						Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
						Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
					{
						Name:     "eth0",
						TypeName: nmstate.TypeDummy,
						MTU:      1500,
						IPv4: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{
								IP:        "10.222.222.1",
								PrefixLen: 30,
							}},
						},
						IPv6: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{
								IP:        "2001::1",
								PrefixLen: 64,
							}},
						},
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
				},
				LinuxStack: nmstate.LinuxStack{IPv4: nmstate.LinuxStackIP4{
					ArpIgnore: pointer.P(procsys.ARPReplyMode1),
				}},
			}),
		)
	})

	It("setup bridge binding without IP", func() {
		const linklocalIPv6Address = "fe80::1"
		nmstatestub := nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "12:34:56:78:90:ab",
				MTU:        1500,
				IPv4:       ipDisabled,
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        linklocalIPv6Address,
						PrefixLen: 64,
					}},
				},
			}},
		}}

		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			}},
			0, 0, 0,
			netpod.WithNMStateAdapter(&nmstatestub),
		)
		Expect(netPod.Setup()).To(Succeed())
		Expect(nmstatestub.spec).To(Equal(
			nmstate.Spec{
				Interfaces: []nmstate.Interface{
					{
						Name:     "k6t-eth0",
						TypeName: nmstate.TypeBridge,
						State:    nmstate.IfaceStateUp,
						Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
					{
						Name:        "eth0-nic",
						Index:       0,
						CopyMacFrom: "k6t-eth0",
						Controller:  "k6t-eth0",
						IPv4:        ipDisabled,
						IPv6:        ipDisabled,
						LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
						Metadata:    &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
					{
						Name:       "tap0",
						TypeName:   nmstate.TypeTap,
						State:      nmstate.IfaceStateUp,
						MTU:        1500,
						Controller: "k6t-eth0",
						Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
						Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
					{
						Name:     "eth0",
						TypeName: nmstate.TypeDummy,
						MTU:      1500,
						IPv4:     ipDisabled,
						IPv6: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{
								IP:        linklocalIPv6Address,
								PrefixLen: 64,
							}},
						},
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
					},
				},
			}),
		)
	})

	When("using secondary network", func() {

		const (
			secondaryPodInterfaceName        = "pod914f438d88d"
			secondaryPodInterfaceOrderedName = "net1"
			secondaryPodInterfaceIndex       = 1

			secondaryNetworkName = "secondnetwork"

			hotplugEnabled = true
		)
		var (
			specNetworks   []v1.Network
			specInterfaces []v1.Interface

			nmstatestub nmstateStub
			masqstub    masqueradeStub
		)

		BeforeEach(func() {
			nmstatestub = nmstateStub{status: nmstate.Status{
				Interfaces: []nmstate.Interface{
					{
						Name:       "eth0",
						Index:      0,
						TypeName:   nmstate.TypeVETH,
						State:      nmstate.IfaceStateUp,
						MacAddress: "12:34:56:78:90:ab",
						MTU:        1500,
						IPv4: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{
								IP:        "10.222.222.1",
								PrefixLen: 30,
							}},
						},
						IPv6: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{
								IP:        "2001::1",
								PrefixLen: 64,
							}},
						},
					},
					{
						Name:       secondaryPodInterfaceName,
						Index:      secondaryPodInterfaceIndex,
						TypeName:   nmstate.TypeVETH,
						State:      nmstate.IfaceStateUp,
						MacAddress: "12:34:56:78:90:cd",
						MTU:        1500,
						IPv4:       ipDisabled,
						IPv6:       ipDisabled,
					},
				},
			}}

			specNetworks = []v1.Network{
				*v1.DefaultPodNetwork(),
				{
					Name: secondaryNetworkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "somenad"},
					},
				},
			}
			specInterfaces = []v1.Interface{
				{
					Name:                   defaultPodNetworkName,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				},
				{
					Name:                   secondaryNetworkName,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
				},
			}
		})

		DescribeTable("setup masquerade (primary) and bridge (secondary) binding", func(asHotPlug bool) {
			initialNetworksToPlug, initialInterfacesToPlug := specNetworks, specInterfaces
			if asHotPlug {
				initialNetworksToPlug, initialInterfacesToPlug = specNetworks[:1], specInterfaces[:1]
			}
			netPod := netpod.NewNetPod(
				initialNetworksToPlug,
				initialInterfacesToPlug,
				0, 0, 0,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
			)
			Expect(netPod.Setup()).To(Succeed())

			expectedPrimaryNetIfaces := []nmstate.Interface{
				{
					Name:       "k6t-eth0",
					TypeName:   nmstate.TypeBridge,
					State:      nmstate.IfaceStateUp,
					MacAddress: "02:00:00:00:00:00",
					MTU:        1500,
					Ethtool:    nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
					IPv4: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
					},
					IPv6: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
					},
					LinuxStack: nmstate.LinuxIfaceStack{IP4RouteLocalNet: pointer.P(true)},
					Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
				},
				{
					Name:       "tap0",
					TypeName:   nmstate.TypeTap,
					State:      nmstate.IfaceStateUp,
					MTU:        1500,
					Controller: "k6t-eth0",
					Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
					Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
				},
			}
			if asHotPlug {
				Expect(nmstatestub.spec).To(Equal(
					nmstate.Spec{
						Interfaces: expectedPrimaryNetIfaces,
						LinuxStack: nmstate.LinuxStack{
							IPv4: nmstate.LinuxStackIP4{Forwarding: pointer.P(true)},
							IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
						},
					},
				))
			}

			netPod = netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				0, 0, 0,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
			)
			Expect(netPod.Setup()).To(Succeed())

			Expect(nmstatestub.spec).To(Equal(
				nmstate.Spec{
					Interfaces: []nmstate.Interface{
						expectedPrimaryNetIfaces[0],
						expectedPrimaryNetIfaces[1],
						// Secondary network
						{
							Name:     "k6t-914f438d88d",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateUp,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:        "914f438d88d-nic",
							Index:       secondaryPodInterfaceIndex,
							CopyMacFrom: "k6t-914f438d88d",
							Controller:  "k6t-914f438d88d",
							IPv4:        ipDisabled,
							IPv6:        ipDisabled,
							LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
							Metadata:    &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:       "tap914f438d88d",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateUp,
							MTU:        1500,
							Controller: "k6t-914f438d88d",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:     secondaryPodInterfaceName,
							TypeName: nmstate.TypeDummy,
							MTU:      1500,
							IPv4:     ipDisabled,
							IPv6:     ipDisabled,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
					},
					LinuxStack: nmstate.LinuxStack{
						IPv4: nmstate.LinuxStackIP4{Forwarding: pointer.P(true)},
						IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
					},
				}),
			)
			Expect(masqstub.bridgeIfaceSpec.Name).To(Equal("k6t-eth0"))
			Expect(masqstub.podIfaceSpec.Name).To(Equal("eth0"))
			Expect(masqstub.vmiIfaceSpec.Name).To(Equal(defaultPodNetworkName))
		},
			Entry("with two setup invokes", !hotplugEnabled),
			Entry("with hotplug (second invoke adds a network)", hotplugEnabled),
		)

		It("setup secondary bridge binding with hashed pod interfaces and absent set", func() {
			specInterfaces[1].State = v1.InterfaceStateAbsent
			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				0, 0, 0,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
			)
			Expect(netPod.Setup()).To(Succeed())
			Expect(nmstatestub.spec).To(Equal(
				nmstate.Spec{
					Interfaces: []nmstate.Interface{
						// Primary network
						{
							Name:       "k6t-eth0",
							TypeName:   nmstate.TypeBridge,
							State:      nmstate.IfaceStateUp,
							MacAddress: "02:00:00:00:00:00",
							MTU:        1500,
							Ethtool:    nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							IPv4: nmstate.IP{
								Enabled: pointer.P(true),
								Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
							},
							IPv6: nmstate.IP{
								Enabled: pointer.P(true),
								Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
							},
							LinuxStack: nmstate.LinuxIfaceStack{IP4RouteLocalNet: pointer.P(true)},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
						},
						{
							Name:       "tap0",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateUp,
							MTU:        1500,
							Controller: "k6t-eth0",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
						},
						// Secondary network is ignored due to the `absent` marking.
					},
					LinuxStack: nmstate.LinuxStack{
						IPv4: nmstate.LinuxStackIP4{Forwarding: pointer.P(true)},
						IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
					},
				}),
			)
			Expect(masqstub.bridgeIfaceSpec.Name).To(Equal("k6t-eth0"))
			Expect(masqstub.podIfaceSpec.Name).To(Equal("eth0"))
			Expect(masqstub.vmiIfaceSpec.Name).To(Equal(defaultPodNetworkName))
		})

		It("setup secondary bridge binding with ordered pod interfaces", func() {
			nmstatestub.status.Interfaces[1].Name = secondaryPodInterfaceOrderedName
			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				0, 0, 0,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
			)
			Expect(netPod.Setup()).To(Succeed())
			Expect(nmstatestub.spec).To(Equal(
				nmstate.Spec{
					Interfaces: []nmstate.Interface{
						// Primary network
						{
							Name:       "k6t-eth0",
							TypeName:   nmstate.TypeBridge,
							State:      nmstate.IfaceStateUp,
							MacAddress: "02:00:00:00:00:00",
							MTU:        1500,
							Ethtool:    nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							IPv4: nmstate.IP{
								Enabled: pointer.P(true),
								Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
							},
							IPv6: nmstate.IP{
								Enabled: pointer.P(true),
								Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
							},
							LinuxStack: nmstate.LinuxIfaceStack{IP4RouteLocalNet: pointer.P(true)},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
						},
						{
							Name:       "tap0",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateUp,
							MTU:        1500,
							Controller: "k6t-eth0",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: defaultPodNetworkName},
						},
						// Secondary network
						{
							Name:     "k6t-net1",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateUp,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:        "net1-nic",
							Index:       secondaryPodInterfaceIndex,
							CopyMacFrom: "k6t-net1",
							Controller:  "k6t-net1",
							IPv4:        ipDisabled,
							IPv6:        ipDisabled,
							LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
							Metadata:    &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:       "tap1",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateUp,
							MTU:        1500,
							Controller: "k6t-net1",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:     secondaryPodInterfaceOrderedName,
							TypeName: nmstate.TypeDummy,
							MTU:      1500,
							IPv4:     ipDisabled,
							IPv6:     ipDisabled,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
					},
					LinuxStack: nmstate.LinuxStack{
						IPv4: nmstate.LinuxStackIP4{Forwarding: pointer.P(true)},
						IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
					},
				}),
			)
			Expect(masqstub.bridgeIfaceSpec.Name).To(Equal("k6t-eth0"))
			Expect(masqstub.podIfaceSpec.Name).To(Equal("eth0"))
			Expect(masqstub.vmiIfaceSpec.Name).To(Equal(defaultPodNetworkName))
		})
	})

	It("setup Passt binding", func() {
		nmstatestub := nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "12:34:56:78:90:ab",
				MTU:        1500,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "10.222.222.1",
						PrefixLen: 30,
					}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "2001::1",
						PrefixLen: 64,
					}},
				},
			}},
		}}
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}},
			}},
			0, 0, 0,
			netpod.WithNMStateAdapter(&nmstatestub),
		)
		Expect(netPod.Setup()).To(Succeed())
		Expect(nmstatestub.spec).To(Equal(
			nmstate.Spec{
				Interfaces: []nmstate.Interface{},
				LinuxStack: nmstate.LinuxStack{IPv4: nmstate.LinuxStackIP4{
					PingGroupRange:        []int{107, 107},
					UnprivilegedPortStart: pointer.P(0),
				}},
			},
		))
	})

	DescribeTable("setup unhandled bindings", func(binding v1.InterfaceBindingMethod) {
		nmstatestub := nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{
				{
					Name:       "eth0",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "12:34:56:78:90:ab",
					MTU:        1500,
				},
			},
		}}
		netPod := netpod.NewNetPod(
			[]v1.Network{
				{
					Name:          "somenet",
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "somenad"}},
				},
			},
			[]v1.Interface{{Name: "somenet", InterfaceBindingMethod: binding}},
			0, 0, 0,
			netpod.WithNMStateAdapter(&nmstatestub),
		)
		Expect(netPod.Setup()).To(Succeed())
		Expect(nmstatestub.spec).To(Equal(nmstate.Spec{Interfaces: []nmstate.Interface{}}))
	},
		Entry("SR-IOV", v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}),
		Entry("Slirp", v1.InterfaceBindingMethod{Slirp: &v1.InterfaceSlirp{}}),
		Entry("Macvtap", v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}),
	)
})

type nmstateStub struct {
	applyErr error
	readErr  error
	spec     nmstate.Spec
	status   nmstate.Status
}

var (
	errNMStateApply = errors.New("nmstate Apply Test Error")
	errNMStateRead  = errors.New("nmstate Real Test Error")
)

func (n *nmstateStub) Apply(spec *nmstate.Spec) error {
	if n.applyErr != nil {
		return n.applyErr
	}
	n.spec = *spec
	return nil
}

func (n *nmstateStub) Read() (*nmstate.Status, error) {
	return &n.status, n.readErr
}

type masqueradeStub struct {
	setupErr        error
	bridgeIfaceSpec *nmstate.Interface
	podIfaceSpec    *nmstate.Interface
	vmiIfaceSpec    v1.Interface
}

var errMasqueradeSetup = errors.New("masquerade Setup Test Error")

func (m *masqueradeStub) Setup(bridgeIfaceSpec, podIfaceSpec *nmstate.Interface, vmiIfaceSpec v1.Interface) error {
	if m.setupErr != nil {
		return m.setupErr
	}
	m.bridgeIfaceSpec = bridgeIfaceSpec
	m.podIfaceSpec = podIfaceSpec
	m.vmiIfaceSpec = vmiIfaceSpec
	return nil
}
