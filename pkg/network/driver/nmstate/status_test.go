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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package nmstate_test

import (
	"net"

	"kubevirt.io/kubevirt/pkg/network/driver/procsys"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vishnetlink "github.com/vishvananda/netlink"

	nlfake "kubevirt.io/kubevirt/pkg/network/driver/netlink/fake"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	psfake "kubevirt.io/kubevirt/pkg/network/driver/procsys/fake"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	vethName   = "red"
	bridgeName = "bridge-blue"

	defaultMTU = 1500
)

// The test strategy is to setup the state of the test-adapter using the various drivers adapters directly.
// Then, assert that the returned (nmstate) status is indeed as expected.
//
// The test-adapter is not updated using the nmstate spec (Apply()) intentionally. The interest is to check
// that the returned status is interpreted correctly from the driver adapters API.

var _ = Describe("NMState Status interfaces", func() {
	var driversAdapter *testAdapter
	var nmState nmstate.NMState

	BeforeEach(func() {
		driversAdapter = newTestAdapter()
		nmState = nmstate.New(nmstate.WithAdapter(driversAdapter))
	})

	BeforeEach(func() {
		Expect(driversAdapter.LinkAdd(newVethLink(vethName))).To(Succeed())
		driversAdapter.txChecksum[vethName] = pointer.P(true)

		Expect(driversAdapter.LinkAdd(newBridgeLink(bridgeName))).To(Succeed())
		driversAdapter.txChecksum[bridgeName] = pointer.P(true)
	})

	It("reports 2 interfaces without IPs", func() {
		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Interfaces).To(Equal([]nmstate.Interface{
			{
				Name:       vethName,
				Index:      1,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				IPv4:       nmstate.IP{Enabled: pointer.P(false)},
				IPv6:       nmstate.IP{Enabled: pointer.P(false)},
				Ethtool:    defaultEthtool(),
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			},
			{
				Name:       bridgeName,
				Index:      2,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				IPv4:       nmstate.IP{Enabled: pointer.P(false)},
				IPv6:       nmstate.IP{Enabled: pointer.P(false)},
				Ethtool:    defaultEthtool(),
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			},
		}))
	})

	It("reports an interfaces connected to a bridge", func() {
		portLink, err := driversAdapter.LinkByName(vethName)
		Expect(err).NotTo(HaveOccurred())
		bridgeLink, err := driversAdapter.LinkByName(bridgeName)
		Expect(err).NotTo(HaveOccurred())
		concreteBridgeLink := bridgeLink.(*vishnetlink.Bridge)

		Expect(driversAdapter.LinkSetMaster(portLink, concreteBridgeLink)).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Interfaces).To(Equal([]nmstate.Interface{
			{
				Name:       vethName,
				Index:      1,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				Controller: bridgeName,
				IPv4:       nmstate.IP{Enabled: pointer.P(false)},
				IPv6:       nmstate.IP{Enabled: pointer.P(false)},
				Ethtool:    defaultEthtool(),
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			},
			{
				Name:       bridgeName,
				Index:      2,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				IPv4:       nmstate.IP{Enabled: pointer.P(false)},
				IPv6:       nmstate.IP{Enabled: pointer.P(false)},
				Ethtool:    defaultEthtool(),
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			},
		}))
	})

	It("reports 2 interfaces, one with IPs", func() {
		ip4CIDR, ip6CIDR := "1.2.3.4/24", "2001::1/64"
		Expect(driversAdapter.setIPConfigOnLink(0, ip4CIDR, ip6CIDR)).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())

		ip4, prefix4 := parseCIDR(ip4CIDR)
		ip6, prefix6 := parseCIDR(ip6CIDR)
		Expect(status.Interfaces).To(Equal([]nmstate.Interface{
			{
				Name:       vethName,
				Index:      1,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: ip4, PrefixLen: prefix4}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: ip6, PrefixLen: prefix6}},
				},
				Ethtool: defaultEthtool(),
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			},
			{
				Name:       bridgeName,
				Index:      2,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MacAddress: macAddress0,
				MTU:        defaultMTU,
				IPv4:       nmstate.IP{Enabled: pointer.P(false)},
				IPv6:       nmstate.IP{Enabled: pointer.P(false)},
				Ethtool:    defaultEthtool(),
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			},
		}))
	})

	It("report interface with tx checksum off", func() {
		Expect(driversAdapter.TXChecksumOff(vethName)).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Interfaces[0]).To(Equal(nmstate.Interface{
			Name:       vethName,
			Index:      1,
			TypeName:   nmstate.TypeVETH,
			State:      nmstate.IfaceStateUp,
			MacAddress: macAddress0,
			MTU:        defaultMTU,
			IPv4:       nmstate.IP{Enabled: pointer.P(false)},
			IPv6:       nmstate.IP{Enabled: pointer.P(false)},
			Ethtool: nmstate.Ethtool{
				Feature: nmstate.Feature{
					TxChecksum: pointer.P(false),
				},
			},
			LinuxStack: nmstate.LinuxIfaceStack{
				IP4RouteLocalNet: pointer.P(false),
				PortLearning:     pointer.P(false),
			},
		}))
	})

	It("report interface with route-local-net and port learning", func() {
		Expect(driversAdapter.IPv4EnableRouteLocalNet(vethName)).To(Succeed())

		link, err := driversAdapter.LinkByName(vethName)
		Expect(err).NotTo(HaveOccurred())
		// Setting learning-off is done to allocate the structure correctly.
		// Then it is enabled directly to allow the test to show it is reflected.
		Expect(driversAdapter.LinkSetLearningOff(link)).To(Succeed())
		link.Attrs().Protinfo.Learning = true

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Interfaces[0]).To(Equal(nmstate.Interface{
			Name:       vethName,
			Index:      1,
			TypeName:   nmstate.TypeVETH,
			State:      nmstate.IfaceStateUp,
			MacAddress: macAddress0,
			MTU:        defaultMTU,
			IPv4:       nmstate.IP{Enabled: pointer.P(false)},
			IPv6:       nmstate.IP{Enabled: pointer.P(false)},
			Ethtool:    defaultEthtool(),
			LinuxStack: nmstate.LinuxIfaceStack{
				IP4RouteLocalNet: pointer.P(true),
				PortLearning:     pointer.P(true),
			},
		}))
	})
})

var _ = Describe("NMState Status Linux Stack", func() {
	var driversAdapter *testAdapter
	var nmState nmstate.NMState

	BeforeEach(func() {
		driversAdapter = newTestAdapter()
		nmState = nmstate.New(nmstate.WithAdapter(driversAdapter))
	})

	It("reports arp-ignore enabled on all interfaces", func() {
		Expect(driversAdapter.IPv4SetArpIgnore("all", procsys.ARPReplyMode1)).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.LinuxStack).To(Equal(nmstate.LinuxStack{
			IPv4: nmstate.LinuxStackIP4{
				ArpIgnore:             pointer.P(procsys.ARPReplyMode1),
				Forwarding:            pointer.P(false),
				PingGroupRange:        []int{0, 0},
				UnprivilegedPortStart: pointer.P(0),
			},
			IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(false)},
		}))
	})

	It("reports IPv4 Forwarding enabled", func() {
		Expect(driversAdapter.IPv4EnableForwarding()).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.LinuxStack).To(Equal(nmstate.LinuxStack{
			IPv4: nmstate.LinuxStackIP4{
				ArpIgnore:             pointer.P(procsys.ARPReplyMode0),
				Forwarding:            pointer.P(true),
				PingGroupRange:        []int{0, 0},
				UnprivilegedPortStart: pointer.P(0),
			},
			IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(false)},
		}))
	})

	It("reports IPv6 Forwarding enabled", func() {
		Expect(driversAdapter.IPv6EnableForwarding()).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.LinuxStack).To(Equal(nmstate.LinuxStack{
			IPv4: nmstate.LinuxStackIP4{
				ArpIgnore:             pointer.P(procsys.ARPReplyMode0),
				Forwarding:            pointer.P(false),
				PingGroupRange:        []int{0, 0},
				UnprivilegedPortStart: pointer.P(0),
			},
			IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
		}))
	})

	It("reports ping group range", func() {
		const groupID = 107
		Expect(driversAdapter.IPv4SetPingGroupRange(groupID, groupID)).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.LinuxStack).To(Equal(nmstate.LinuxStack{
			IPv4: nmstate.LinuxStackIP4{
				ArpIgnore:             pointer.P(procsys.ARPReplyMode0),
				Forwarding:            pointer.P(false),
				PingGroupRange:        []int{groupID, groupID},
				UnprivilegedPortStart: pointer.P(0),
			},
			IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(false)},
		}))
	})

	It("reports unprivileged port start", func() {
		const unprivPortStart = 1234
		Expect(driversAdapter.IPv4SetUnprivilegedPortStart(unprivPortStart)).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.LinuxStack).To(Equal(nmstate.LinuxStack{
			IPv4: nmstate.LinuxStackIP4{
				ArpIgnore:             pointer.P(procsys.ARPReplyMode0),
				Forwarding:            pointer.P(false),
				PingGroupRange:        []int{0, 0},
				UnprivilegedPortStart: pointer.P(unprivPortStart),
			},
			IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(false)},
		}))
	})
})

var _ = Describe("NMState Status Routes", func() {
	var driversAdapter *testAdapter
	var nmState nmstate.NMState

	BeforeEach(func() {
		driversAdapter = newTestAdapter()
		nmState = nmstate.New(nmstate.WithAdapter(driversAdapter))
	})

	DescribeTable("reports routes", func(netlinkFamily int, destinationNetwork, gwIP string) {
		Expect(driversAdapter.LinkAdd(newVethLink(vethName))).To(Succeed())
		link, err := driversAdapter.LinkByName(vethName)
		Expect(err).NotTo(HaveOccurred())

		_, ipNet, _ := net.ParseCIDR(destinationNetwork)
		route := vishnetlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       ipNet,
			Gw:        net.ParseIP(gwIP),
		}
		Expect(driversAdapter.RouteAdd(&route)).To(Succeed())
		Expect(driversAdapter.RouteList(link, netlinkFamily)).To(Equal([]vishnetlink.Route{route}))

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Routes).To(Equal(nmstate.Routes{Running: []nmstate.Route{
			{
				Destination:      destinationNetwork,
				NextHopInterface: vethName,
				NextHopAddress:   gwIP,
				TableID:          0,
			},
		}}))
	},
		Entry("with ipv4", vishnetlink.FAMILY_V4, "10.10.10.0/24", "1.1.1.1"),
		Entry("with ipv6", vishnetlink.FAMILY_V6, "2001::/64", "2001::1"),
	)
})

func newTestAdapter() *testAdapter {
	return &testAdapter{
		NetLink:    *nlfake.New(),
		ProcSys:    *psfake.New(),
		txChecksum: map[string]*bool{},
	}
}

func newVethLink(name string) vishnetlink.Link {
	return &vishnetlink.Veth{
		LinkAttrs: vishnetlink.LinkAttrs{
			Name:         name,
			HardwareAddr: []byte("123456"),
			MTU:          defaultMTU,
			OperState:    vishnetlink.OperUp,
		},
	}
}

func newBridgeLink(name string) vishnetlink.Link {
	return &vishnetlink.Bridge{
		LinkAttrs: vishnetlink.LinkAttrs{
			Name:         name,
			HardwareAddr: []byte("123456"),
			MTU:          defaultMTU,
			OperState:    vishnetlink.OperUp,
		},
	}
}

func parseCIDR(s string) (string, int) {
	ip, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	prefixLen, _ := ipNet.Mask.Size()
	return ip.String(), prefixLen
}
