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

package nmstate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/driver/procsys"

	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	dummyName = "dummy-red"

	ip4Addr0   = "10.10.10.10"
	ip4Prefix0 = 24

	ip6Addr0   = "2001::1"
	ip6Prefix0 = 64
)

// The test strategy is to setup and read the state of the network configuration through the nmstate API.
// Then, assert that the returned (nmstate) status is indeed as expected.
//
// Unlike the strategy taken with the nmstate status unit tests, to use the drivers adapters,
// the nmstate spec unit tests do not act on the drivers directly.

var _ = Describe("NMState Spec interfaces", func() {
	var nmState nmstate.NMState

	BeforeEach(func() {
		nmState = nmstate.New(nmstate.WithAdapter(newTestAdapter()))
	})

	DescribeTable("setup a new interfaces with", func(ipv4, ipv6 nmstate.IP) {
		err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
			{
				Name:       dummyName,
				TypeName:   nmstate.TypeDummy,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				IPv4:       ipv4,
				IPv6:       ipv6,
			},
		}})
		Expect(err).NotTo(HaveOccurred())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Interfaces).To(Equal([]nmstate.Interface{
			{
				Name:       dummyName,
				Index:      1,
				TypeName:   nmstate.TypeDummy,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				IPv4:       ipv4,
				IPv6:       ipv6,
				Ethtool:    defaultEthtool(),
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			},
		}))
	},
		Entry("no IP", nmstate.IP{Enabled: pointer.P(false)}, nmstate.IP{Enabled: pointer.P(false)}),
		Entry("IPv4 (only)",
			nmstate.IP{
				Enabled: pointer.P(true),
				Address: []nmstate.IPAddress{{
					IP:        ip4Addr0,
					PrefixLen: ip4Prefix0,
				}},
			},
			nmstate.IP{Enabled: pointer.P(false)},
		),
		Entry("IPv6 (only)",
			nmstate.IP{Enabled: pointer.P(false)},
			nmstate.IP{
				Enabled: pointer.P(true),
				Address: []nmstate.IPAddress{{
					IP:        ip6Addr0,
					PrefixLen: ip6Prefix0,
				}},
			},
		),
		Entry("IPv4 & IPv6",
			nmstate.IP{
				Enabled: pointer.P(true),
				Address: []nmstate.IPAddress{{
					IP:        ip4Addr0,
					PrefixLen: ip4Prefix0,
				}},
			},
			nmstate.IP{
				Enabled: pointer.P(true),
				Address: []nmstate.IPAddress{{
					IP:        ip6Addr0,
					PrefixLen: ip6Prefix0,
				}},
			},
		),
	)

	Context("given an existing interface", func() {
		BeforeEach(func() {
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:       dummyName,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					IPv4: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip4Addr0,
							PrefixLen: ip4Prefix0,
						}},
					},
					IPv6: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip6Addr0,
							PrefixLen: ip6Prefix0,
						}},
					},
				},
			}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes the interface", func() {
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:  dummyName,
					State: nmstate.IfaceStateAbsent,
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(BeEmpty())
		})

		It("creates a new interface, copying the mac of the current", func() {
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:        bridgeName,
					TypeName:    nmstate.TypeBridge,
					State:       nmstate.IfaceStateUp,
					MTU:         defaultMTU,
					CopyMacFrom: dummyName,
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(Equal([]nmstate.Interface{
				{
					Name:       dummyName,
					Index:      1,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip4Addr0,
							PrefixLen: ip4Prefix0,
						}},
					},
					IPv6: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip6Addr0,
							PrefixLen: ip6Prefix0,
						}},
					},
				},
				{
					Name:       bridgeName,
					Index:      2,
					TypeName:   nmstate.TypeBridge,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
			}))
		})

		It("deletes the IP addresses", func() {
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name: dummyName,
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(Equal([]nmstate.Interface{
				{
					Name:       dummyName,
					Index:      1,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
			}))
		})

		It("modifies the IP addresses", func() {
			const (
				ip4Addr1   = ip4Addr0 + "1"
				ip4Prefix1 = ip4Prefix0 + 1
				ip6Addr1   = ip6Addr0 + "1"
				ip6Prefix1 = ip6Prefix0 + 1
			)
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name: dummyName,
					IPv4: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip4Addr1,
							PrefixLen: ip4Prefix1,
						}},
					},
					IPv6: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip6Addr1,
							PrefixLen: ip6Prefix1,
						}},
					},
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(Equal([]nmstate.Interface{
				{
					Name:       dummyName,
					Index:      1,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip4Addr1,
							PrefixLen: ip4Prefix1,
						}},
					},
					IPv6: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{
							IP:        ip6Addr1,
							PrefixLen: ip6Prefix1,
						}},
					},
				},
			}))
		})

		It("modifies its name (and removes the IP/s)", func() {
			const newDummyName = dummyName + "-new"
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:  newDummyName,
					Index: 1,
					IPv4:  nmstate.IP{Enabled: pointer.P(false)},
					IPv6:  nmstate.IP{Enabled: pointer.P(false)},
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(Equal([]nmstate.Interface{
				{
					Name:       newDummyName,
					Index:      1,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
			}))
		})

		It("connects it to a bridge (and removes the IP/s)", func() {
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:       bridgeName,
					TypeName:   nmstate.TypeBridge,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
				},
				{
					Name:       dummyName,
					Controller: bridgeName,
					IPv4:       nmstate.IP{Enabled: pointer.P(false)},
					IPv6:       nmstate.IP{Enabled: pointer.P(false)},
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(Equal([]nmstate.Interface{
				{
					Name:       dummyName,
					Index:      1,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					Controller: bridgeName,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
				{
					Name:       bridgeName,
					Index:      2,
					TypeName:   nmstate.TypeBridge,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
			}))
		})

		It("disables tx-checksum (and removes the IP/s)", func() {
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name: dummyName,
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
					Ethtool: nmstate.Ethtool{
						Feature: nmstate.Feature{
							TxChecksum: pointer.P(false),
						},
					},
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(Equal([]nmstate.Interface{
				{
					Name:       dummyName,
					Index:      1,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool: nmstate.Ethtool{
						Feature: nmstate.Feature{
							TxChecksum: pointer.P(false),
						},
					},
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(false),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
			}))
		})

		It("enable route-local-net and disable learning (and removes the IP/s)", func() {
			Expect(nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name: dummyName,
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(true),
						PortLearning:     pointer.P(false),
					},
				},
			}})).To(Succeed())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Interfaces).To(Equal([]nmstate.Interface{
				{
					Name:       dummyName,
					Index:      1,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					MacAddress: macAddress0,
					MTU:        defaultMTU,
					Ethtool:    defaultEthtool(),
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(true),
						PortLearning:     pointer.P(false),
					},
					IPv4: nmstate.IP{Enabled: pointer.P(false)},
					IPv6: nmstate.IP{Enabled: pointer.P(false)},
				},
			}))
		})
	})
})

var _ = Describe("NMState Spec Linux Stack", func() {
	var nmState nmstate.NMState

	BeforeEach(func() {
		nmState = nmstate.New(nmstate.WithAdapter(newTestAdapter()))
	})

	DescribeTable("setup with", func(lsSpec, lsStatus nmstate.LinuxStack) {
		Expect(nmState.Apply(&nmstate.Spec{LinuxStack: lsSpec})).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.LinuxStack).To(Equal(lsStatus))
	},
		Entry("arp-ignore enabled on all interfaces",
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{ArpIgnore: pointer.P(procsys.ARPReplyMode1)},
			},
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{
					ArpIgnore:             pointer.P(procsys.ARPReplyMode1),
					Forwarding:            pointer.P(false),
					PingGroupRange:        []int{0, 0},
					UnprivilegedPortStart: pointer.P(0),
				},
				IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(false)},
			},
		),
		Entry("IPv4 & IPv6 forwarding enabled",
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{Forwarding: pointer.P(true)},
				IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
			},
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{
					ArpIgnore:             pointer.P(procsys.ARPReplyMode0),
					Forwarding:            pointer.P(true),
					PingGroupRange:        []int{0, 0},
					UnprivilegedPortStart: pointer.P(0),
				},
				IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
			},
		),
		Entry("ping group range",
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{PingGroupRange: []int{123, 321}},
			},
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{
					ArpIgnore:             pointer.P(procsys.ARPReplyMode0),
					Forwarding:            pointer.P(false),
					PingGroupRange:        []int{123, 321},
					UnprivilegedPortStart: pointer.P(0),
				},
				IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(false)},
			},
		),
		Entry("unprivileged port start",
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{UnprivilegedPortStart: pointer.P(1000)},
			},
			nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{
					ArpIgnore:             pointer.P(procsys.ARPReplyMode0),
					Forwarding:            pointer.P(false),
					PingGroupRange:        []int{0, 0},
					UnprivilegedPortStart: pointer.P(1000),
				},
				IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(false)},
			},
		),
	)
})
