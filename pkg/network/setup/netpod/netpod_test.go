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
	"fmt"
	"net"
	"os"
	"sync"

	vishnetlink "github.com/vishvananda/netlink"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	kfs "kubevirt.io/kubevirt/pkg/os/fs"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/pkg/pointer"

	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	"kubevirt.io/kubevirt/pkg/network/driver/procsys"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod"
)

const (
	defaultPodNetworkName = "default"

	vmiUID = "12345"

	primaryIPv4Address = "10.222.222.1"
	primaryIPv6Address = "2001::1"
)

var (
	ipDisabled = nmstate.IP{Enabled: pointer.P(false)}
)

var _ = Describe("netpod", func() {

	var (
		baseCacheCreator tempCacheCreator
		state            *netpod.State
	)

	BeforeEach(dutils.MockDefaultOwnershipManager)

	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})

	BeforeEach(func() {
		DeferCleanup(os.Setenv, "MY_POD_IP", os.Getenv("MY_POD_IP"))
		Expect(os.Setenv("MY_POD_IP", "10.10.10.10")).To(Succeed())
	})

	BeforeEach(func() {
		cache := newConfigStateCacheStub()
		state = netpod.NewState(cache, netnsStub{})
	})

	It("fails setup when reading nmstate status fails", func() {
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			}},
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstateStub{readErr: errNMStateRead}),
			netpod.WithCacheCreator(&baseCacheCreator),
		)
		Expect(netPod.Setup()).To(MatchError(errNMStateRead))
	})

	It("fails setup when applying nmstate status fails", func() {
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			}},
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstateStub{
				applyErr: errNMStateApply,
				status: nmstate.Status{
					Interfaces: []nmstate.Interface{{
						Name:       "eth0",
						Index:      0,
						TypeName:   nmstate.TypeVETH,
						State:      nmstate.IfaceStateUp,
						MacAddress: "12:34:56:78:90:ab",
						MTU:        1500,
					}},
				},
			}),
			netpod.WithCacheCreator(&baseCacheCreator),
		)
		err := netPod.Setup()
		Expect(err).To(MatchError(errNMStateApply))

		var criticalNetErr *neterrors.CriticalNetworkError
		Expect(errors.As(err, &criticalNetErr)).To(BeTrue())
	})

	It("fails setup when applying nmstate status with undefined binding", func() {
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{Name: defaultPodNetworkName}},
			vmiUID, 0, 0, 0, state,
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
			netpod.WithCacheCreator(&baseCacheCreator),
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
			vmiUID, 0, 0, 0, state,
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
			netpod.WithCacheCreator(&baseCacheCreator),
		)
		Expect(netPod.Setup()).To(MatchError(errMasqueradeSetup))
	})

	DescribeTable("fails setup discovery when pod interface is missing", func(binding v1.InterfaceBindingMethod) {
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{{Name: defaultPodNetworkName, InterfaceBindingMethod: binding}},
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstateStub{status: nmstate.Status{
				Interfaces: []nmstate.Interface{{Name: "other0"}},
			}}),
			netpod.WithCacheCreator(&baseCacheCreator),
		)
		err := netPod.Setup()
		Expect(err.Error()).To(HavePrefix("pod link (eth0) is missing"))
	},
		Entry("bridge", v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}),
		Entry("masquerade", v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}),
		Entry("passt", v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}}),
		Entry("slirp", v1.InterfaceBindingMethod{Slirp: &v1.InterfaceSlirp{}}),
	)

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
						IP:        primaryIPv4Address,
						PrefixLen: 30,
					}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        primaryIPv6Address,
						PrefixLen: 64,
					}},
				},
			}},
		}}
		masqstub := masqueradeStub{}

		vmiIface := v1.Interface{
			Name:                   defaultPodNetworkName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
		}
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{vmiIface},
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstatestub),
			netpod.WithMasqueradeAdapter(&masqstub),
			netpod.WithCacheCreator(&baseCacheCreator),
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
		Expect(cache.ReadPodInterfaceCache(&baseCacheCreator, vmiUID, defaultPodNetworkName)).To(Equal(&cache.PodIfaceCacheData{
			Iface:  &vmiIface,
			PodIP:  primaryIPv4Address,
			PodIPs: []string{primaryIPv4Address, primaryIPv6Address},
		}))
	})

	It("setup bridge binding with IP and a static route", func() {
		const (
			defaultGatewayIP4Address = "10.222.222.254"

			podIfaceOrignalMAC = "12:34:56:78:90:ab"
		)
		nmstatestub := nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: podIfaceOrignalMAC,
				MTU:        1500,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        primaryIPv4Address,
						PrefixLen: 30,
					}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        primaryIPv6Address,
						PrefixLen: 64,
					}},
				},
			}},
			Routes: nmstate.Routes{Running: []nmstate.Route{
				// Default Route
				{
					Destination:      "0.0.0.0/0",
					NextHopInterface: "eth0",
					NextHopAddress:   defaultGatewayIP4Address,
					TableID:          0,
				},
				// Local Route (should be ignored)
				{
					Destination:      "10.222.222.0/30",
					NextHopInterface: "eth0",
					NextHopAddress:   primaryIPv4Address,
					TableID:          0,
				},
				// Static Route
				{
					Destination:      "192.168.1.0/24",
					NextHopInterface: "eth0",
					NextHopAddress:   defaultGatewayIP4Address,
					TableID:          0,
				},
			}},
		}}

		vmiIface := v1.Interface{
			Name:                   defaultPodNetworkName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
		}
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{vmiIface},
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstatestub),
			netpod.WithCacheCreator(&baseCacheCreator),
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
								IP:        primaryIPv4Address,
								PrefixLen: 30,
							}},
						},
						IPv6: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{
								IP:        primaryIPv6Address,
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
		Expect(cache.ReadPodInterfaceCache(&baseCacheCreator, vmiUID, defaultPodNetworkName)).To(Equal(&cache.PodIfaceCacheData{
			Iface:  &vmiIface,
			PodIP:  primaryIPv4Address,
			PodIPs: []string{primaryIPv4Address, primaryIPv6Address},
		}))

		expDHCPConfig, err := expectedDHCPConfig(
			"10.222.222.1/30",
			podIfaceOrignalMAC,
			defaultGatewayIP4Address,
			"192.168.1.0/24",
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(cache.ReadDHCPInterfaceCache(&baseCacheCreator, "0", "eth0")).To(Equal(expDHCPConfig))
		Expect(cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", defaultPodNetworkName)).To(Equal(&api.Interface{
			MAC: &api.MAC{MAC: podIfaceOrignalMAC},
		}))
	})

	It("setup bridge binding without IP", func() {
		const podIfaceOrignalMAC = "12:34:56:78:90:ab"
		const linklocalIPv6Address = "fe80::1"
		nmstatestub := nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: podIfaceOrignalMAC,
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
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstatestub),
			netpod.WithCacheCreator(&baseCacheCreator),
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
		// When there are no IP/s, the pod interface data is not stored.
		_, err := cache.ReadPodInterfaceCache(&baseCacheCreator, vmiUID, defaultPodNetworkName)
		Expect(err).To(HaveOccurred())

		Expect(cache.ReadDHCPInterfaceCache(&baseCacheCreator, "0", "eth0")).To(
			Equal(&cache.DHCPConfig{IPAMDisabled: true}))

		Expect(cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", defaultPodNetworkName)).To(Equal(&api.Interface{
			MAC: &api.MAC{MAC: podIfaceOrignalMAC},
		}))
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
								IP:        primaryIPv4Address,
								PrefixLen: 30,
							}},
						},
						IPv6: nmstate.IP{
							Enabled: pointer.P(true),
							Address: []nmstate.IPAddress{{
								IP:        primaryIPv6Address,
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
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
				netpod.WithCacheCreator(&baseCacheCreator),
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
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
				netpod.WithCacheCreator(&baseCacheCreator),
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
			Expect(cache.ReadPodInterfaceCache(&baseCacheCreator, vmiUID, defaultPodNetworkName)).To(Equal(&cache.PodIfaceCacheData{
				Iface:  &specInterfaces[0],
				PodIP:  primaryIPv4Address,
				PodIPs: []string{primaryIPv4Address, primaryIPv6Address},
			}))
			// When there are no IP/s, the pod interface data is not stored.
			_, err := cache.ReadPodInterfaceCache(&baseCacheCreator, vmiUID, secondaryNetworkName)
			Expect(err).To(HaveOccurred())
		},
			Entry("with two setup invokes", !hotplugEnabled),
			Entry("with hotplug (second invoke adds a network)", hotplugEnabled),
		)

		It("setup secondary bridge binding with hashed pod interfaces and absent set", func() {
			specInterfaces[1].State = v1.InterfaceStateAbsent
			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
				netpod.WithCacheCreator(&baseCacheCreator),
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
						// Secondary network with `absent` marking.
						{
							Name:     "k6t-914f438d88d",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateAbsent,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:       "tap914f438d88d",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateAbsent,
							MTU:        1500,
							Controller: "k6t-914f438d88d",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: secondaryNetworkName},
						},
						{
							Name:     secondaryPodInterfaceName,
							TypeName: nmstate.TypeDummy,
							State:    nmstate.IfaceStateAbsent,
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

		It("setup secondary bridge binding with ordered pod interfaces", func() {
			nmstatestub.status.Interfaces[1].Name = secondaryPodInterfaceOrderedName
			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(&nmstatestub),
				netpod.WithMasqueradeAdapter(&masqstub),
				netpod.WithCacheCreator(&baseCacheCreator),
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
						IP:        primaryIPv4Address,
						PrefixLen: 30,
					}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        primaryIPv6Address,
						PrefixLen: 64,
					}},
				},
			}},
		}}

		vmiIface := v1.Interface{
			Name:                   defaultPodNetworkName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}},
		}
		netPod := netpod.NewNetPod(
			[]v1.Network{*v1.DefaultPodNetwork()},
			[]v1.Interface{vmiIface},
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstatestub),
			netpod.WithCacheCreator(&baseCacheCreator),
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
		Expect(cache.ReadPodInterfaceCache(&baseCacheCreator, vmiUID, defaultPodNetworkName)).To(Equal(&cache.PodIfaceCacheData{
			Iface:  &vmiIface,
			PodIP:  primaryIPv4Address,
			PodIPs: []string{primaryIPv4Address, primaryIPv6Address},
		}))
	})

	DescribeTable("setup unhandled bindings", func(binding v1.InterfaceBindingMethod, expNmstateSpec nmstate.Spec) {
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
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			},
			[]v1.Interface{{Name: "somenet", InterfaceBindingMethod: binding}},
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(&nmstatestub),
			netpod.WithCacheCreator(&baseCacheCreator),
		)
		Expect(netPod.Setup()).To(Succeed())
		Expect(nmstatestub.spec).To(Equal(expNmstateSpec))
	},
		// Not processed by the discovery & config steps.
		Entry("SR-IOV", v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}, nmstate.Spec{}),
		Entry("Macvtap", v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}, nmstate.Spec{}),
		// Processed by the discovery but not by the config step.
		// When processed by the config step, the nmstate structure will be initialized (e.g. to an empty interface list).
		// Interfaces will not get populated because the specific binding (slirp) is not treated there.
		Entry("Slirp", v1.InterfaceBindingMethod{Slirp: &v1.InterfaceSlirp{}}, nmstate.Spec{Interfaces: []nmstate.Interface{}}),
	)

	Context("setup with plugged networks marked for removal", func() {
		const (
			testNet1 = "testnet1"
			testNet2 = "testnet2"
		)
		var (
			specNetworks   []v1.Network
			specInterfaces []v1.Interface
			nmstatestub    *nmstateStub
		)

		BeforeEach(func() {
			specNetworks = []v1.Network{
				{Name: defaultPodNetworkName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
				{Name: testNet1, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
				{Name: testNet2, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
			}
			specInterfaces = []v1.Interface{
				{
					Name:                   defaultPodNetworkName,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				},
				{Name: testNet1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				{Name: testNet2, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
			}

			nmstatestub = &nmstateStub{status: nmstate.Status{
				Interfaces: []nmstate.Interface{
					{
						Name:       "eth0",
						Index:      0,
						TypeName:   nmstate.TypeVETH,
						State:      nmstate.IfaceStateUp,
						MacAddress: "12:34:56:78:90:ab",
						MTU:        1500,
					},
					{
						Name:       "pod7087ef4cd1f",
						Index:      0,
						TypeName:   nmstate.TypeVETH,
						State:      nmstate.IfaceStateUp,
						MacAddress: "22:34:56:78:90:ab",
						MTU:        1500,
					},
					{
						Name:       "podbc6cc93fa1e",
						Index:      0,
						TypeName:   nmstate.TypeVETH,
						State:      nmstate.IfaceStateUp,
						MacAddress: "32:34:56:78:90:ab",
						MTU:        1500,
					},
				},
			}}

			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(nmstatestub),
				netpod.WithCacheCreator(&baseCacheCreator),
			)

			Expect(netPod.Setup()).To(Succeed())

			pending, started, finished, err := state.PendingStartedFinished(specNetworks)
			Expect(err).NotTo(HaveOccurred())
			Expect(pending).To(BeEmpty())
			Expect(started).To(BeEmpty())
			Expect(finished).To(Equal(specNetworks))

			Expect(cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet1)).NotTo(BeNil())
			Expect(cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet2)).NotTo(BeNil())
		})

		It("unplug 1 out of 2 secondary bridge binding networks", func() {
			specInterfaces[1].State = v1.InterfaceStateAbsent
			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(nmstatestub),
				netpod.WithCacheCreator(&baseCacheCreator),
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
							IPv4:       nmstate.IP{Enabled: pointer.P(false)},
							IPv6:       nmstate.IP{Enabled: pointer.P(false)},
							LinuxStack: nmstate.LinuxIfaceStack{},
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
						// Secondary network with `absent` marking.
						{
							Name:     "k6t-7087ef4cd1f",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateAbsent,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						{
							Name:       "tap7087ef4cd1f",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateAbsent,
							MTU:        1500,
							Controller: "k6t-7087ef4cd1f",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						{
							Name:     "pod7087ef4cd1f",
							TypeName: nmstate.TypeDummy,
							State:    nmstate.IfaceStateAbsent,
							MTU:      1500,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						// Third network.
						{
							Name:     "k6t-bc6cc93fa1e",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateUp,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
						{
							Name:        "bc6cc93fa1e-nic",
							CopyMacFrom: "k6t-bc6cc93fa1e",
							Controller:  "k6t-bc6cc93fa1e",
							IPv4:        ipDisabled,
							IPv6:        ipDisabled,
							LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
							Metadata:    &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
						{
							Name:       "tapbc6cc93fa1e",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateUp,
							MTU:        1500,
							Controller: "k6t-bc6cc93fa1e",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
						{
							Name:     "podbc6cc93fa1e",
							TypeName: nmstate.TypeDummy,
							MTU:      1500,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
					},
					LinuxStack: nmstate.LinuxStack{},
				},
			))

			_, _, finished, err := state.PendingStartedFinished(specNetworks)
			Expect(err).NotTo(HaveOccurred())
			Expect(finished).To(Equal([]v1.Network{
				{Name: defaultPodNetworkName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
				{Name: testNet2, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
			}))

			// testNet1 is not expected to exist anymore.
			_, err = cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet1)
			Expect(err).To(HaveOccurred())

			Expect(cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet2)).NotTo(BeNil())
		})

		It("unplug 2 out of 2 secondary bridge binding networks", func() {
			specInterfaces[1].State = v1.InterfaceStateAbsent
			specInterfaces[2].State = v1.InterfaceStateAbsent
			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(nmstatestub),
				netpod.WithCacheCreator(&baseCacheCreator),
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
							IPv4:       nmstate.IP{Enabled: pointer.P(false)},
							IPv6:       nmstate.IP{Enabled: pointer.P(false)},
							LinuxStack: nmstate.LinuxIfaceStack{},
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
						// Secondary network with `absent` marking.
						{
							Name:     "k6t-7087ef4cd1f",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateAbsent,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						{
							Name:       "tap7087ef4cd1f",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateAbsent,
							MTU:        1500,
							Controller: "k6t-7087ef4cd1f",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						{
							Name:     "pod7087ef4cd1f",
							TypeName: nmstate.TypeDummy,
							State:    nmstate.IfaceStateAbsent,
							MTU:      1500,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						// Third network.
						{
							Name:     "k6t-bc6cc93fa1e",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateAbsent,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
						{
							Name:       "tapbc6cc93fa1e",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateAbsent,
							MTU:        1500,
							Controller: "k6t-bc6cc93fa1e",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
						{
							Name:     "podbc6cc93fa1e",
							TypeName: nmstate.TypeDummy,
							State:    nmstate.IfaceStateAbsent,
							MTU:      1500,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
					},
					LinuxStack: nmstate.LinuxStack{},
				},
			))

			_, _, finished, err := state.PendingStartedFinished(specNetworks)
			Expect(err).NotTo(HaveOccurred())
			Expect(finished).To(Equal([]v1.Network{
				{Name: defaultPodNetworkName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			}))

			// testNet1 and testNet2 are not expected to exist anymore.
			_, err = cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet1)
			Expect(err).To(HaveOccurred())

			_, err = cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet2)
			Expect(err).To(HaveOccurred())
		})

		It("unplug secondary bridge binding network that is still in pending state", func() {
			// This test is placed as a stand alone to emphasize an unexpected side effect.
			// The composed configuration includes removal of interfaces that do not even exist yet.
			// With the nmstate backend implementation, this is acceptable, as the interface deletion
			// is conditional to its existence. The composed configuration is expressing the desire
			// for the interface to be removed, therefore, if it is already absent, it will silently do nothing.

			By("Unplug the 3rd network")
			specInterfaces[2].State = v1.InterfaceStateAbsent
			netPod := netpod.NewNetPod(
				specNetworks,
				specInterfaces,
				vmiUID, 0, 0, 0, state,
				netpod.WithNMStateAdapter(nmstatestub),
				netpod.WithCacheCreator(&baseCacheCreator),
			)
			Expect(netPod.Setup()).To(Succeed())

			By("Unplug the 3rd network again")
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
							IPv4:       nmstate.IP{Enabled: pointer.P(false)},
							IPv6:       nmstate.IP{Enabled: pointer.P(false)},
							LinuxStack: nmstate.LinuxIfaceStack{},
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
						// Secondary network.
						{
							Name:     "k6t-7087ef4cd1f",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateUp,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						{
							Name:        "7087ef4cd1f-nic",
							CopyMacFrom: "k6t-7087ef4cd1f",
							Controller:  "k6t-7087ef4cd1f",
							IPv4:        ipDisabled,
							IPv6:        ipDisabled,
							LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
							Metadata:    &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						{
							Name:       "tap7087ef4cd1f",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateUp,
							MTU:        1500,
							Controller: "k6t-7087ef4cd1f",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						{
							Name:     "pod7087ef4cd1f",
							TypeName: nmstate.TypeDummy,
							MTU:      1500,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
						},
						// Third network.
						{
							Name:     "k6t-bc6cc93fa1e",
							TypeName: nmstate.TypeBridge,
							State:    nmstate.IfaceStateAbsent,
							Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
						{
							Name:       "tapbc6cc93fa1e",
							TypeName:   nmstate.TypeTap,
							State:      nmstate.IfaceStateAbsent,
							MTU:        1500,
							Controller: "k6t-bc6cc93fa1e",
							Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
							Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
						{
							Name:     "podbc6cc93fa1e",
							TypeName: nmstate.TypeDummy,
							State:    nmstate.IfaceStateAbsent,
							MTU:      1500,
							Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
						},
					},
					LinuxStack: nmstate.LinuxStack{},
				},
			))

			_, _, finished, err := state.PendingStartedFinished(specNetworks)
			Expect(err).NotTo(HaveOccurred())
			Expect(finished).To(Equal([]v1.Network{
				{Name: defaultPodNetworkName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
				{Name: testNet1, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
			}))

			Expect(cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet1)).NotTo(BeNil())

			// testNet2 is not expected to exist anymore.
			_, err = cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet2)
			Expect(err).To(HaveOccurred())
		})
	})

	It("unplug secondary bridge binding network that is still in started state", func() {
		// This test is placed as a stand alone to emphasize an unexpected side effect.
		// The composed configuration includes removal of interfaces that do not even exist yet.
		// With the nmstate backend implementation, this is acceptable, as the interface deletion
		// is conditional to its existence. The composed configuration is expressing the desire
		// for the interface to be removed, therefore, if it is already absent, it will silently do nothing.

		const (
			testNet1 = "testnet1"
			testNet2 = "testnet2"
		)

		specNetworks := []v1.Network{
			{Name: defaultPodNetworkName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			{Name: testNet1, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
			{Name: testNet2, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
		}
		specInterfaces := []v1.Interface{
			{
				Name:                   defaultPodNetworkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			},
			{Name: testNet1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
			{Name: testNet2, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
		}

		nmstatestub := &nmstateStub{status: nmstate.Status{
			Interfaces: []nmstate.Interface{
				{
					Name:       "eth0",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "12:34:56:78:90:ab",
					MTU:        1500,
				},
				{
					Name:       "pod7087ef4cd1f",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "22:34:56:78:90:ab",
					MTU:        1500,
				},
				{
					Name:       "podbc6cc93fa1e",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "32:34:56:78:90:ab",
					MTU:        1500,
				},
			},
		}}

		By("Plug 2 networks")
		netPod := netpod.NewNetPod(
			specNetworks[:2],
			specInterfaces[:2],
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(nmstatestub),
			netpod.WithCacheCreator(&baseCacheCreator),
		)
		Expect(netPod.Setup()).To(Succeed())

		By("Plug an additional network that fails the config step")
		nmstatestub.applyErr = errNMStateApply
		netPod = netpod.NewNetPod(
			specNetworks[2:],
			specInterfaces[2:],
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(nmstatestub),
			netpod.WithCacheCreator(&baseCacheCreator),
		)
		err := netPod.Setup()
		Expect(err).To(MatchError(errNMStateApply))

		pending, started, finished, err := state.PendingStartedFinished(specNetworks)
		Expect(err).NotTo(HaveOccurred())
		Expect(pending).To(BeEmpty())
		Expect(started).To(Equal(specNetworks[2:]))
		Expect(finished).To(Equal(specNetworks[:2]))

		By("Unplug the 3rd network (that is in started state)")
		nmstatestub.applyErr = nil
		specInterfaces[2].State = v1.InterfaceStateAbsent
		netPod = netpod.NewNetPod(
			specNetworks,
			specInterfaces,
			vmiUID, 0, 0, 0, state,
			netpod.WithNMStateAdapter(nmstatestub),
			netpod.WithCacheCreator(&baseCacheCreator),
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
						IPv4:       nmstate.IP{Enabled: pointer.P(false)},
						IPv6:       nmstate.IP{Enabled: pointer.P(false)},
						LinuxStack: nmstate.LinuxIfaceStack{},
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
					// Secondary network.
					{
						Name:     "k6t-7087ef4cd1f",
						TypeName: nmstate.TypeBridge,
						State:    nmstate.IfaceStateUp,
						Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
					},
					{
						Name:        "7087ef4cd1f-nic",
						CopyMacFrom: "k6t-7087ef4cd1f",
						Controller:  "k6t-7087ef4cd1f",
						IPv4:        ipDisabled,
						IPv6:        ipDisabled,
						LinuxStack:  nmstate.LinuxIfaceStack{PortLearning: pointer.P(false)},
						Metadata:    &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
					},
					{
						Name:       "tap7087ef4cd1f",
						TypeName:   nmstate.TypeTap,
						State:      nmstate.IfaceStateUp,
						MTU:        1500,
						Controller: "k6t-7087ef4cd1f",
						Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
						Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
					},
					{
						Name:     "pod7087ef4cd1f",
						TypeName: nmstate.TypeDummy,
						MTU:      1500,
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet1},
					},
					// Third network.
					{
						Name:     "k6t-bc6cc93fa1e",
						TypeName: nmstate.TypeBridge,
						State:    nmstate.IfaceStateAbsent,
						Ethtool:  nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}},
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
					},
					{
						Name:       "tapbc6cc93fa1e",
						TypeName:   nmstate.TypeTap,
						State:      nmstate.IfaceStateAbsent,
						MTU:        1500,
						Controller: "k6t-bc6cc93fa1e",
						Tap:        &nmstate.TapDevice{Queues: 0, UID: 0, GID: 0},
						Metadata:   &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
					},
					{
						Name:     "podbc6cc93fa1e",
						TypeName: nmstate.TypeDummy,
						State:    nmstate.IfaceStateAbsent,
						MTU:      1500,
						Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: testNet2},
					},
				},
				LinuxStack: nmstate.LinuxStack{},
			},
		))

		_, _, finished, err = state.PendingStartedFinished(specNetworks)
		Expect(err).NotTo(HaveOccurred())
		Expect(finished).To(Equal([]v1.Network{
			{Name: defaultPodNetworkName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			{Name: testNet1, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
		}))

		Expect(cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet1)).NotTo(BeNil())

		// testNet2 is not expected to exist anymore.
		_, err = cache.ReadDomainInterfaceCache(&baseCacheCreator, "0", testNet2)
		Expect(err).To(HaveOccurred())
	})
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

type tempCacheCreator struct {
	once   sync.Once
	tmpDir string
}

func (c *tempCacheCreator) New(filePath string) *cache.Cache {
	c.once.Do(func() {
		tmpDir, err := os.MkdirTemp("", "temp-cache")
		if err != nil {
			panic("Unable to create temp cache directory")
		}
		c.tmpDir = tmpDir
	})
	return cache.NewCustomCache(filePath, kfs.NewWithRootPath(c.tmpDir))
}

func expectedDHCPConfig(podIfaceCIDR, podIfaceMAC, defaultGW, staticRouteDst string) (*cache.DHCPConfig, error) {
	ipv4, err := vishnetlink.ParseAddr(podIfaceCIDR)
	if err != nil {
		return nil, err
	}
	mac, err := net.ParseMAC(podIfaceMAC)
	if err != nil {
		return nil, err
	}
	destAddr, err := vishnetlink.ParseAddr(staticRouteDst)
	if err != nil {
		return nil, err
	}
	routes := []vishnetlink.Route{
		{Gw: net.ParseIP(defaultGW)},
		{Dst: destAddr.IPNet, Gw: net.ParseIP(defaultGW)},
	}
	return &cache.DHCPConfig{
		IP:           *ipv4,
		MAC:          mac,
		Routes:       &routes,
		IPAMDisabled: false,
		Gateway:      net.ParseIP(defaultGW),
		Subdomain:    "",
	}, nil
}

type netnsStub struct {
	shouldFail bool
}

func (n netnsStub) Do(f func() error) error {
	if n.shouldFail {
		return fmt.Errorf("do-netns failure")
	}
	return f()
}

type configStateCacheStub struct {
	stateCache map[string]cache.PodIfaceState
	readErr    error
	writeErr   error
	deleteErr  error
}

func newConfigStateCacheStub() configStateCacheStub {
	return configStateCacheStub{map[string]cache.PodIfaceState{}, nil, nil, nil}
}

func (c configStateCacheStub) Read(key string) (cache.PodIfaceState, error) {
	return c.stateCache[key], c.readErr
}

func (c configStateCacheStub) Write(key string, state cache.PodIfaceState) error {
	if c.writeErr != nil {
		return c.writeErr
	}
	c.stateCache[key] = state
	return nil
}

func (c configStateCacheStub) Delete(key string) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	delete(c.stateCache, key)
	return nil
}
