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
	"flag"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vishnetlink "github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	"kubevirt.io/kubevirt/pkg/network/driver/procsys"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var runIntegrationTests bool

func init() {
	flag.BoolVar(&runIntegrationTests, "run-integration-tests", false, "run integration tests")
}

var _ = Describe("NMState", integrationLabel, func() {
	BeforeEach(func() {
		if !runIntegrationTests {
			Skip("integration tests are not set to run")
		}
	})

	var nmState = nmstate.New()

	It("reports actual network devices", func() {
		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())

		Expect(status.Interfaces).NotTo(BeEmpty())
		Expect(status.Routes.Running).NotTo(BeEmpty())
	})

	It("create a new bridge (without IP)", func() {
		const (
			bridgeName = "brtest99"
			bridgeMac  = "02:ff:ff:ff:ff:01"
			bridgeMTU  = 1000
		)
		ethtoolWithTxChecksumOff := nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}}
		spec := nmstate.Spec{Interfaces: []nmstate.Interface{
			{
				Name:       bridgeName,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MTU:        bridgeMTU,
				MacAddress: bridgeMac,
				Ethtool:    ethtoolWithTxChecksumOff,
			},
		}}

		Expect(nmState.Apply(&spec)).To(Succeed())

		DeferCleanup(func() error {
			spec.Interfaces[0].State = nmstate.IfaceStateAbsent
			return nmState.Apply(&spec)
		})

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())

		bridgeIface := lookupIface(status.Interfaces, bridgeName)
		resetIfaceStatusForComparison(bridgeIface)
		Expect(bridgeIface).To(Equal(&nmstate.Interface{
			Name:       bridgeName,
			TypeName:   nmstate.TypeBridge,
			MacAddress: bridgeMac,
			MTU:        bridgeMTU,
			Ethtool:    ethtoolWithTxChecksumOff,
			IPv4:       nmstate.IP{Enabled: pointer.P(false)},
			IPv6:       nmstate.IP{Enabled: pointer.P(false)},
			LinuxStack: nmstate.LinuxIfaceStack{
				IP4RouteLocalNet: pointer.P(false),
				PortLearning:     pointer.P(false),
			},
		}))
	})

	When("given an existing bridge", func() {
		const (
			bridgeName = "brtest99"
			bridgeMac  = "02:ff:ff:ff:ff:01"
			bridgeMTU  = 1000
		)
		var (
			ethtoolWithTxChecksumOff nmstate.Ethtool
			spec                     nmstate.Spec
		)

		BeforeEach(func() {
			ethtoolWithTxChecksumOff = nmstate.Ethtool{Feature: nmstate.Feature{TxChecksum: pointer.P(false)}}
			spec = nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:       bridgeName,
					TypeName:   nmstate.TypeBridge,
					State:      nmstate.IfaceStateUp,
					MTU:        bridgeMTU,
					MacAddress: bridgeMac,
					Ethtool:    ethtoolWithTxChecksumOff,
				},
			}}
			Expect(nmState.Apply(&spec)).To(Succeed())

			DeferCleanup(func() error {
				spec.Interfaces[0].State = nmstate.IfaceStateAbsent
				return nmState.Apply(&spec)
			})
		})

		It("add IP to the bridge", func() {
			spec.Interfaces[0].IPv4 = nmstate.IP{
				Enabled: pointer.P(true),
				Address: []nmstate.IPAddress{primaryIP4Address()},
			}
			Expect(nmState.Apply(&spec)).To(Succeed())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())

			bridgeIface := lookupIface(status.Interfaces, bridgeName)
			resetIfaceStatusForComparison(bridgeIface)
			Expect(bridgeIface).To(Equal(&nmstate.Interface{
				Name:       bridgeName,
				TypeName:   nmstate.TypeBridge,
				MacAddress: bridgeMac,
				MTU:        bridgeMTU,
				Ethtool:    ethtoolWithTxChecksumOff,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{primaryIP4Address()},
				},
				IPv6: nmstate.IP{Enabled: pointer.P(false)},
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			}))
		})

		It("add a secondary IP to the bridge", func() {
			spec.Interfaces[0].IPv4 = nmstate.IP{
				Enabled: pointer.P(true),
				Address: []nmstate.IPAddress{primaryIP4Address()},
			}
			Expect(nmState.Apply(&spec)).To(Succeed())

			spec.Interfaces[0].IPv4.Address = append(spec.Interfaces[0].IPv4.Address, secondaryIP4Address())
			Expect(nmState.Apply(&spec)).To(Succeed())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())

			bridgeIface := lookupIface(status.Interfaces, bridgeName)
			resetIfaceStatusForComparison(bridgeIface)
			Expect(bridgeIface).To(Equal(&nmstate.Interface{
				Name:       bridgeName,
				TypeName:   nmstate.TypeBridge,
				MacAddress: bridgeMac,
				MTU:        bridgeMTU,
				Ethtool:    ethtoolWithTxChecksumOff,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{primaryIP4Address(), secondaryIP4Address()},
				},
				IPv6: nmstate.IP{Enabled: pointer.P(false)},
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			}))
		})

		It("remove a secondary IP from the bridge", func() {
			spec.Interfaces[0].IPv4 = nmstate.IP{
				Enabled: pointer.P(true),
				Address: []nmstate.IPAddress{primaryIP4Address(), secondaryIP4Address()},
			}
			Expect(nmState.Apply(&spec)).To(Succeed())

			spec.Interfaces[0].IPv4.Address = spec.Interfaces[0].IPv4.Address[:1]
			Expect(nmState.Apply(&spec)).To(Succeed())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())

			bridgeIface := lookupIface(status.Interfaces, bridgeName)
			resetIfaceStatusForComparison(bridgeIface)
			Expect(bridgeIface).To(Equal(&nmstate.Interface{
				Name:       bridgeName,
				TypeName:   nmstate.TypeBridge,
				MacAddress: bridgeMac,
				MTU:        bridgeMTU,
				Ethtool:    ethtoolWithTxChecksumOff,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{primaryIP4Address()},
				},
				IPv6: nmstate.IP{Enabled: pointer.P(false)},
				LinuxStack: nmstate.LinuxIfaceStack{
					IP4RouteLocalNet: pointer.P(false),
					PortLearning:     pointer.P(false),
				},
			}))
		})

		It("fails to change the interface type", func() {
			currentType, requestType := spec.Interfaces[0].TypeName, nmstate.TypeVETH
			spec.Interfaces[0].TypeName = requestType
			Expect(nmState.Apply(&spec)).To(MatchError(
				fmt.Sprintf("type collision on link %s: actual %s, requested %s", spec.Interfaces[0].Name, currentType, requestType),
			))
		})

		It("attach a veth to the bridge", func() {
			const vethName = "testveth99"
			Expect(createVeth(vethName)).To(Succeed())
			DeferCleanup(func() error {
				return nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
					{Name: vethName, State: nmstate.IfaceStateAbsent},
				}})
			})

			bridgeIface := spec.Interfaces[0]
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:       vethName,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					Controller: bridgeIface.Name,
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())

			vethIface := lookupIface(status.Interfaces, vethName)
			Expect(vethIface.Controller).To(Equal(bridgeIface.Name))
			Expect(vethIface.LinuxStack).To(Equal(nmstate.LinuxIfaceStack{
				IP4RouteLocalNet: pointer.P(false),
				PortLearning:     pointer.P(true),
			}))
		})

		// The backend implementation to create a tap device requires the `virt-chroot` executable.
		// The test is marked as pending until the required executable is placed in the test environment.
		PIt("attach a new created tap to the bridge", func() {
			const (
				tapName = "testTap99"
			)
			ownerID := os.Getpid()

			bridgeIface := spec.Interfaces[0]
			err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{
					Name:       tapName,
					TypeName:   nmstate.TypeTap,
					State:      nmstate.IfaceStateUp,
					Controller: bridgeIface.Name,
					Tap: &nmstate.TapDevice{
						Queues: 2,
						UID:    ownerID,
						GID:    ownerID,
					},
					LinuxStack: nmstate.LinuxIfaceStack{
						PortLearning: pointer.P(false),
					},
				},
			}})
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() error {
				return nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
					{Name: tapName, State: nmstate.IfaceStateAbsent},
				}})
			})

			status, err := nmState.Read()
			Expect(err).NotTo(HaveOccurred())

			tapIface := lookupIface(status.Interfaces, tapName)
			Expect(tapIface.TypeName).To(Equal(nmstate.TypeTap))
			Expect(tapIface.Controller).To(Equal(bridgeIface.Name))
			Expect(tapIface.LinuxStack).To(Equal(nmstate.LinuxIfaceStack{
				IP4RouteLocalNet: pointer.P(false),
				PortLearning:     pointer.P(false),
			}))
		})
	})

	It("delete a non existing interface", func() {
		const vethName = "testveth99"
		Expect(
			nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
				{Name: vethName, State: nmstate.IfaceStateAbsent},
			}}),
		).To(Succeed())
	})

	It("rename (dummy) interface name", func() {
		const dummyName = "dummytest99"
		spec := nmstate.Spec{Interfaces: []nmstate.Interface{{Name: dummyName, TypeName: nmstate.TypeDummy}}}
		Expect(nmState.Apply(&spec)).To(Succeed())

		DeferCleanup(func() error {
			spec.Interfaces[0].State = nmstate.IfaceStateAbsent
			return nmState.Apply(&spec)
		})

		dummyLink, err := vishnetlink.LinkByName(dummyName)
		Expect(err).NotTo(HaveOccurred())

		const newDummyName = "new" + dummyName
		spec.Interfaces[0].Name = newDummyName
		// Specifying the index allows the link name to be modified.
		spec.Interfaces[0].Index = dummyLink.Attrs().Index
		Expect(nmState.Apply(&spec)).To(Succeed())

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())

		previousDummyIface := lookupIface(status.Interfaces, dummyName)
		Expect(previousDummyIface).To(BeNil())

		dummyIface := lookupIface(status.Interfaces, newDummyName)
		Expect(dummyIface).NotTo(BeNil())
		Expect(dummyIface.TypeName).To(Equal(nmstate.TypeDummy))
	})

	It("change Linux stack parameters", func() {
		const (
			dummyName = "testDummy99"

			pingGroupRangeFrom    = 1
			pingGroupRangeTo      = 1
			unprivilegedPortStart = 2000
		)
		spec := nmstate.Spec{
			Interfaces: []nmstate.Interface{
				{
					Name:     bridgeName,
					TypeName: nmstate.TypeBridge,
					State:    nmstate.IfaceStateUp,
				},
				{
					Name:       dummyName,
					TypeName:   nmstate.TypeDummy,
					State:      nmstate.IfaceStateUp,
					Controller: bridgeName,
					LinuxStack: nmstate.LinuxIfaceStack{
						IP4RouteLocalNet: pointer.P(true),
						PortLearning:     pointer.P(false),
					},
				},
			},
			LinuxStack: nmstate.LinuxStack{
				IPv4: nmstate.LinuxStackIP4{
					ArpIgnore:             pointer.P(procsys.ARPReplyMode1),
					Forwarding:            pointer.P(true),
					PingGroupRange:        []int{pingGroupRangeFrom, pingGroupRangeTo},
					UnprivilegedPortStart: pointer.P(unprivilegedPortStart),
				},
				IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
			},
		}

		originalStatus, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())

		Expect(nmState.Apply(&spec)).To(Succeed())

		DeferCleanup(func() error {
			return nmState.Apply(&nmstate.Spec{
				Interfaces: []nmstate.Interface{
					{Name: dummyName, State: nmstate.IfaceStateAbsent},
					{Name: bridgeName, State: nmstate.IfaceStateAbsent},
				},
				LinuxStack: originalStatus.LinuxStack,
			})
		})

		status, err := nmState.Read()
		Expect(err).NotTo(HaveOccurred())

		dummyIface := lookupIface(status.Interfaces, dummyName)
		Expect(dummyIface.LinuxStack).To(Equal(nmstate.LinuxIfaceStack{
			IP4RouteLocalNet: pointer.P(true),
			PortLearning:     pointer.P(false),
		}))
		Expect(status.LinuxStack).To(Equal(nmstate.LinuxStack{
			IPv4: nmstate.LinuxStackIP4{
				ArpIgnore:             pointer.P(procsys.ARPReplyMode1),
				Forwarding:            pointer.P(true),
				PingGroupRange:        []int{pingGroupRangeFrom, pingGroupRangeTo},
				UnprivilegedPortStart: pointer.P(unprivilegedPortStart),
			},
			IPv6: nmstate.LinuxStackIP6{Forwarding: pointer.P(true)},
		}))
	})

})

func lookupIface(ifaces []nmstate.Interface, name string) *nmstate.Interface {
	for i := range ifaces {
		if ifaces[i].Name == name {
			return &ifaces[i]
		}
	}
	return nil
}

func primaryIP4Address() nmstate.IPAddress {
	return nmstate.IPAddress{
		IP:        "10.222.111.1",
		PrefixLen: 30,
	}
}

func secondaryIP4Address() nmstate.IPAddress {
	return nmstate.IPAddress{
		IP:        "10.222.222.1",
		PrefixLen: 30,
	}
}

// resetIfaceStatusForComparison clears the interface status in order to
// allow stable comparison between the actual and expected statuses.
//
// Some interface types (e.g. bridge) have intermediate fluctuation in the
// operational state (from unknown to down/up).
func resetIfaceStatusForComparison(iface *nmstate.Interface) {
	iface.State = ""
	iface.Index = 0
}

func createVeth(name string) error {
	link := &vishnetlink.Veth{
		LinkAttrs: vishnetlink.LinkAttrs{Name: name},
		PeerName:  name + "peer",
	}
	return vishnetlink.LinkAdd(link)
}
