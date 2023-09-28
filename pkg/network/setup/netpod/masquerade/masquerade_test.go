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

package masquerade_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/driver/nft"
	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod/masquerade"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("masquerade (NAT)", func() {
	It("setup fails", func() {
		testErr := errors.New("test error")
		masqPod := masquerade.New(masquerade.WithNftableAdapter(&nftableStub{
			addTableErr: testErr,
		}))

		ifaceSpec := nmstate.Interface{IPv4: nmstate.IP{Enabled: pointer.P(true)}}
		Expect(masqPod.Setup(&ifaceSpec, &ifaceSpec, v1.Interface{})).To(MatchError(testErr))
	})

	It("setup with IPv4, no ports", func() {
		nftStub := &nftableStub{}
		masqPod := masquerade.New(masquerade.WithNftableAdapter(nftStub))

		err := masqPod.Setup(
			&nmstate.Interface{
				Name:       "k6t-eth0",
				Index:      1,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MacAddress: "bb:bb:bb:bb:bb:bb",
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
				},
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			&nmstate.Interface{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "aa:aa:aa:aa:aa:aa",
				MTU:        1500,
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "10.222.222.1",
						PrefixLen: 30,
					}},
				},
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		expectedConfig := `tables:
family ip name nat
chains:
family ip table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip table nat name input chainspec [{ type nat hook input priority 100; }]
family ip table nat name output chainspec [{ type nat hook output priority -100; }]
family ip table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip table nat name KUBEVIRT_PREINBOUND chainspec []
family ip table nat name KUBEVIRT_POSTINBOUND chainspec []
rules:
family ip table nat chain postrouting rulespec [ip saddr 10.0.2.2 counter masquerade]
family ip table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip table nat chain KUBEVIRT_PREINBOUND rulespec [counter dnat to 10.0.2.2]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [ip saddr { 127.0.0.1 } counter snat to 10.0.2.1]
family ip table nat chain output rulespec [ip daddr { 127.0.0.1 } counter dnat to 10.0.2.2]
`
		Expect(nftStub.String()).To(Equal(expectedConfig), fmt.Sprintf("actual:\n%s\n\nexpected:\n%s", nftStub.String(), expectedConfig))
	})

	It("setup with IPv6, no ports", func() {
		nftStub := &nftableStub{}
		masqPod := masquerade.New(masquerade.WithNftableAdapter(nftStub))

		err := masqPod.Setup(
			&nmstate.Interface{
				Name:       "k6t-eth0",
				Index:      1,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MacAddress: "bb:bb:bb:bb:bb:bb",
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
				},
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			&nmstate.Interface{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "aa:aa:aa:aa:aa:aa",
				MTU:        1500,
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{
						IP:        "2001::1",
						PrefixLen: 64,
					}},
				},
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		expectedConfig := `tables:
family ip6 name nat
chains:
family ip6 table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip6 table nat name input chainspec [{ type nat hook input priority 100; }]
family ip6 table nat name output chainspec [{ type nat hook output priority -100; }]
family ip6 table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip6 table nat name KUBEVIRT_PREINBOUND chainspec []
family ip6 table nat name KUBEVIRT_POSTINBOUND chainspec []
rules:
family ip6 table nat chain postrouting rulespec [ip6 saddr fd10:0:2::2 counter masquerade]
family ip6 table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip6 table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip6 table nat chain KUBEVIRT_PREINBOUND rulespec [counter dnat to fd10:0:2::2]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [ip6 saddr { ::1 } counter snat to fd10:0:2::1]
family ip6 table nat chain output rulespec [ip6 daddr { ::1 } counter dnat to fd10:0:2::2]
`
		Expect(nftStub.String()).To(Equal(expectedConfig), fmt.Sprintf("actual:\n%s\n\nexpected:\n%s", nftStub.String(), expectedConfig))
	})

	It("setup with IPv4 and IPv6, no ports", func() {
		nftStub := &nftableStub{}
		masqPod := masquerade.New(masquerade.WithNftableAdapter(nftStub))

		err := masqPod.Setup(
			&nmstate.Interface{
				Name:       "k6t-eth0",
				Index:      1,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MacAddress: "bb:bb:bb:bb:bb:bb",
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
				},
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			&nmstate.Interface{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "aa:aa:aa:aa:aa:aa",
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
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		expectedConfig := `tables:
family ip name nat
family ip6 name nat
chains:
family ip table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip table nat name input chainspec [{ type nat hook input priority 100; }]
family ip table nat name output chainspec [{ type nat hook output priority -100; }]
family ip table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip table nat name KUBEVIRT_PREINBOUND chainspec []
family ip table nat name KUBEVIRT_POSTINBOUND chainspec []
family ip6 table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip6 table nat name input chainspec [{ type nat hook input priority 100; }]
family ip6 table nat name output chainspec [{ type nat hook output priority -100; }]
family ip6 table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip6 table nat name KUBEVIRT_PREINBOUND chainspec []
family ip6 table nat name KUBEVIRT_POSTINBOUND chainspec []
rules:
family ip table nat chain postrouting rulespec [ip saddr 10.0.2.2 counter masquerade]
family ip table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip table nat chain KUBEVIRT_PREINBOUND rulespec [counter dnat to 10.0.2.2]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [ip saddr { 127.0.0.1 } counter snat to 10.0.2.1]
family ip table nat chain output rulespec [ip daddr { 127.0.0.1 } counter dnat to 10.0.2.2]
family ip6 table nat chain postrouting rulespec [ip6 saddr fd10:0:2::2 counter masquerade]
family ip6 table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip6 table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip6 table nat chain KUBEVIRT_PREINBOUND rulespec [counter dnat to fd10:0:2::2]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [ip6 saddr { ::1 } counter snat to fd10:0:2::1]
family ip6 table nat chain output rulespec [ip6 daddr { ::1 } counter dnat to fd10:0:2::2]
`
		Expect(nftStub.String()).To(Equal(expectedConfig), fmt.Sprintf("actual:\n%s\n\nexpected:\n%s", nftStub.String(), expectedConfig))
	})

	It("setup with IPv4 and IPv6, including ports", func() {
		nftStub := &nftableStub{}
		masqPod := masquerade.New(masquerade.WithNftableAdapter(nftStub))

		err := masqPod.Setup(
			&nmstate.Interface{
				Name:       "k6t-eth0",
				Index:      1,
				TypeName:   nmstate.TypeBridge,
				State:      nmstate.IfaceStateUp,
				MacAddress: "bb:bb:bb:bb:bb:bb",
				IPv4: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
				},
				IPv6: nmstate.IP{
					Enabled: pointer.P(true),
					Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
				},
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			&nmstate.Interface{
				Name:       "eth0",
				Index:      0,
				TypeName:   nmstate.TypeVETH,
				State:      nmstate.IfaceStateUp,
				MacAddress: "aa:aa:aa:aa:aa:aa",
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
				Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
			},
			v1.Interface{
				Name:                   "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				Ports: []v1.Port{
					{Name: "http", Protocol: "tcp", Port: 80},
					{Name: "http", Protocol: "tcp", Port: 8080},
				},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		expectedConfig := `tables:
family ip name nat
family ip6 name nat
chains:
family ip table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip table nat name input chainspec [{ type nat hook input priority 100; }]
family ip table nat name output chainspec [{ type nat hook output priority -100; }]
family ip table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip table nat name KUBEVIRT_PREINBOUND chainspec []
family ip table nat name KUBEVIRT_POSTINBOUND chainspec []
family ip6 table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip6 table nat name input chainspec [{ type nat hook input priority 100; }]
family ip6 table nat name output chainspec [{ type nat hook output priority -100; }]
family ip6 table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip6 table nat name KUBEVIRT_PREINBOUND chainspec []
family ip6 table nat name KUBEVIRT_POSTINBOUND chainspec []
rules:
family ip table nat chain postrouting rulespec [ip saddr 10.0.2.2 counter masquerade]
family ip table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip table nat chain KUBEVIRT_PREINBOUND rulespec [tcp dport { 80 } counter dnat to 10.0.2.2]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 80 ip saddr { 127.0.0.1 } counter snat to 10.0.2.1]
family ip table nat chain output rulespec [ip daddr { 127.0.0.1 } tcp dport 80 counter dnat to 10.0.2.2]
family ip table nat chain KUBEVIRT_PREINBOUND rulespec [tcp dport { 8080 } counter dnat to 10.0.2.2]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 8080 ip saddr { 127.0.0.1 } counter snat to 10.0.2.1]
family ip table nat chain output rulespec [ip daddr { 127.0.0.1 } tcp dport 8080 counter dnat to 10.0.2.2]
family ip6 table nat chain postrouting rulespec [ip6 saddr fd10:0:2::2 counter masquerade]
family ip6 table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip6 table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip6 table nat chain KUBEVIRT_PREINBOUND rulespec [tcp dport { 80 } counter dnat to fd10:0:2::2]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 80 ip6 saddr { ::1 } counter snat to fd10:0:2::1]
family ip6 table nat chain output rulespec [ip6 daddr { ::1 } tcp dport 80 counter dnat to fd10:0:2::2]
family ip6 table nat chain KUBEVIRT_PREINBOUND rulespec [tcp dport { 8080 } counter dnat to fd10:0:2::2]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 8080 ip6 saddr { ::1 } counter snat to fd10:0:2::1]
family ip6 table nat chain output rulespec [ip6 daddr { ::1 } tcp dport 8080 counter dnat to fd10:0:2::2]
`
		Expect(nftStub.String()).To(Equal(expectedConfig), fmt.Sprintf("actual:\n%s\n\nexpected:\n%s", nftStub.String(), expectedConfig))
	})

	Context("with ISTIO", func() {
		It("setup with IPv4 and IPv6, no ports", func() {
			nftStub := &nftableStub{}
			masqPod := masquerade.New(masquerade.WithNftableAdapter(nftStub), masquerade.WithIstio(true))

			err := masqPod.Setup(
				&nmstate.Interface{
					Name:       "k6t-eth0",
					Index:      1,
					TypeName:   nmstate.TypeBridge,
					State:      nmstate.IfaceStateUp,
					MacAddress: "bb:bb:bb:bb:bb:bb",
					IPv4: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
					},
					IPv6: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
					},
					Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
				},
				&nmstate.Interface{
					Name:       "eth0",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "aa:aa:aa:aa:aa:aa",
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
					Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
				},
				v1.Interface{
					Name:                   "default",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			expectedConfig := `tables:
family ip name nat
family ip6 name nat
chains:
family ip table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip table nat name input chainspec [{ type nat hook input priority 100; }]
family ip table nat name output chainspec [{ type nat hook output priority -100; }]
family ip table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip table nat name KUBEVIRT_PREINBOUND chainspec []
family ip table nat name KUBEVIRT_POSTINBOUND chainspec []
family ip6 table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip6 table nat name input chainspec [{ type nat hook input priority 100; }]
family ip6 table nat name output chainspec [{ type nat hook output priority -100; }]
family ip6 table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip6 table nat name KUBEVIRT_PREINBOUND chainspec []
family ip6 table nat name KUBEVIRT_POSTINBOUND chainspec []
rules:
family ip table nat chain postrouting rulespec [ip saddr 10.0.2.2 counter masquerade]
family ip table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip table nat chain output rulespec [tcp dport { 15000, 15001, 15004, 15006, 15008, 15009, 15020, 15021, 15053, 15090 } ip saddr 127.0.0.1 counter return]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport { 15000, 15001, 15004, 15006, 15008, 15009, 15020, 15021, 15053, 15090 } ip saddr 127.0.0.1 counter return]
family ip table nat chain KUBEVIRT_PREINBOUND rulespec [tcp dport { 22 } counter dnat to 10.0.2.2]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [ip saddr { 127.0.0.1, 127.0.0.6 } counter snat to 10.0.2.1]
family ip table nat chain output rulespec [ip daddr { 127.0.0.1, 10.222.222.1 } counter dnat to 10.0.2.2]
family ip6 table nat chain postrouting rulespec [ip6 saddr fd10:0:2::2 counter masquerade]
family ip6 table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip6 table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip6 table nat chain output rulespec [tcp dport { 15000, 15001, 15004, 15006, 15008, 15009, 15020, 15021, 15053, 15090 } ip6 saddr ::1 counter return]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport { 15000, 15001, 15004, 15006, 15008, 15009, 15020, 15021, 15053, 15090 } ip6 saddr ::1 counter return]
family ip6 table nat chain KUBEVIRT_PREINBOUND rulespec [tcp dport { 22 } counter dnat to fd10:0:2::2]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [ip6 saddr { ::1 } counter snat to fd10:0:2::1]
family ip6 table nat chain output rulespec [ip6 daddr { ::1 } counter dnat to fd10:0:2::2]
`
			Expect(nftStub.String()).To(Equal(expectedConfig), fmt.Sprintf("actual:\n%s\n\nexpected:\n%s", nftStub.String(), expectedConfig))
		})

		It("setup with IPv4 and IPv6, including ports and legacy migration set", func() {
			nftStub := &nftableStub{}
			masqPod := masquerade.New(
				masquerade.WithNftableAdapter(nftStub),
				masquerade.WithIstio(true),
				masquerade.WithLegacyMigrationPorts(),
			)

			err := masqPod.Setup(
				&nmstate.Interface{
					Name:       "k6t-eth0",
					Index:      1,
					TypeName:   nmstate.TypeBridge,
					State:      nmstate.IfaceStateUp,
					MacAddress: "bb:bb:bb:bb:bb:bb",
					IPv4: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{IP: "10.0.2.1", PrefixLen: 24}},
					},
					IPv6: nmstate.IP{
						Enabled: pointer.P(true),
						Address: []nmstate.IPAddress{{IP: "fd10:0:2::1", PrefixLen: 120}},
					},
					Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
				},
				&nmstate.Interface{
					Name:       "eth0",
					Index:      0,
					TypeName:   nmstate.TypeVETH,
					State:      nmstate.IfaceStateUp,
					MacAddress: "aa:aa:aa:aa:aa:aa",
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
					Metadata: &nmstate.IfaceMetadata{Pid: 0, NetworkName: "default"},
				},
				v1.Interface{
					Name:                   "default",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
					Ports: []v1.Port{
						{Name: "http", Protocol: "tcp", Port: 80},
						{Name: "http", Protocol: "tcp", Port: 8080},
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			expectedConfig := `tables:
family ip name nat
family ip6 name nat
chains:
family ip table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip table nat name input chainspec [{ type nat hook input priority 100; }]
family ip table nat name output chainspec [{ type nat hook output priority -100; }]
family ip table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip table nat name KUBEVIRT_PREINBOUND chainspec []
family ip table nat name KUBEVIRT_POSTINBOUND chainspec []
family ip6 table nat name prerouting chainspec [{ type nat hook prerouting priority -100; }]
family ip6 table nat name input chainspec [{ type nat hook input priority 100; }]
family ip6 table nat name output chainspec [{ type nat hook output priority -100; }]
family ip6 table nat name postrouting chainspec [{ type nat hook postrouting priority 100; }]
family ip6 table nat name KUBEVIRT_PREINBOUND chainspec []
family ip6 table nat name KUBEVIRT_POSTINBOUND chainspec []
rules:
family ip table nat chain postrouting rulespec [ip saddr 10.0.2.2 counter masquerade]
family ip table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip table nat chain output rulespec [tcp dport { 49152, 49153 } ip saddr 127.0.0.1 counter return]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport { 49152, 49153 } ip saddr 127.0.0.1 counter return]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 80 ip saddr { 127.0.0.1, 127.0.0.6 } counter snat to 10.0.2.1]
family ip table nat chain output rulespec [ip daddr { 127.0.0.1, 10.222.222.1 } tcp dport 80 counter dnat to 10.0.2.2]
family ip table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 8080 ip saddr { 127.0.0.1, 127.0.0.6 } counter snat to 10.0.2.1]
family ip table nat chain output rulespec [ip daddr { 127.0.0.1, 10.222.222.1 } tcp dport 8080 counter dnat to 10.0.2.2]
family ip6 table nat chain postrouting rulespec [ip6 saddr fd10:0:2::2 counter masquerade]
family ip6 table nat chain prerouting rulespec [iifname eth0 counter jump KUBEVIRT_PREINBOUND]
family ip6 table nat chain postrouting rulespec [oifname k6t-eth0 counter jump KUBEVIRT_POSTINBOUND]
family ip6 table nat chain output rulespec [tcp dport { 49152, 49153 } ip6 saddr ::1 counter return]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport { 49152, 49153 } ip6 saddr ::1 counter return]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 80 ip6 saddr { ::1 } counter snat to fd10:0:2::1]
family ip6 table nat chain output rulespec [ip6 daddr { ::1 } tcp dport 80 counter dnat to fd10:0:2::2]
family ip6 table nat chain KUBEVIRT_POSTINBOUND rulespec [tcp dport 8080 ip6 saddr { ::1 } counter snat to fd10:0:2::1]
family ip6 table nat chain output rulespec [ip6 daddr { ::1 } tcp dport 8080 counter dnat to fd10:0:2::2]
`
			Expect(nftStub.String()).To(Equal(expectedConfig), fmt.Sprintf("actual:\n%s\n\nexpected:\n%s", nftStub.String(), expectedConfig))
		})
	})
})

type nftableStub struct {
	addTableErr error
	Tables      []tableData `json:"tables"`
	Chains      []chainData `json:"chains"`
	Rules       []ruleData  `json:"rules"`
}

type tableData struct {
	Family nft.IPFamily `json:"family"`
	Name   string       `json:"name"`
}

type chainData struct {
	Table     tableData `json:"table"`
	Name      string    `json:"name"`
	Chainspec []string  `json:"chainspec,omitempty"`
}

type ruleData struct {
	Chain    chainData `json:"chain"`
	Rulespec []string  `json:"rulespec,omitempty"`
}

func (n *nftableStub) AddTable(family nft.IPFamily, name string) error {
	if n.addTableErr != nil {
		return n.addTableErr
	}
	n.Tables = append(n.Tables, tableData{family, name})
	return nil
}

func (n *nftableStub) AddChain(family nft.IPFamily, table string, name string, chainspec ...string) error {
	n.Chains = append(n.Chains, chainData{
		tableData{family, table},
		name,
		chainspec,
	})
	return nil
}

func (n *nftableStub) AddRule(family nft.IPFamily, table string, chain string, rulespec ...string) error {
	n.Rules = append(n.Rules, ruleData{
		Chain: chainData{
			Table: tableData{Family: family, Name: table},
			Name:  chain,
		},
		Rulespec: rulespec,
	})
	return nil
}

func (n *nftableStub) String() string {
	var out string

	out += "tables:\n"
	for _, t := range n.Tables {
		out += fmt.Sprintf("family %s name %s\n", t.Family, t.Name)
	}
	out += "chains:\n"
	for _, c := range n.Chains {
		out += fmt.Sprintf("family %s table %s name %s chainspec %s\n", c.Table.Family, c.Table.Name, c.Name, c.Chainspec)
	}
	out += "rules:\n"
	for _, r := range n.Rules {
		out += fmt.Sprintf("family %s table %s chain %s rulespec %s\n", r.Chain.Table.Family, r.Chain.Table.Name, r.Chain.Name, r.Rulespec)
	}
	return out
}
