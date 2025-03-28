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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package server

import (
	"net"

	"github.com/krolaw/dhcp4"
	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("DHCP Server", func() {
	Context("check routes", func() {
		It("verify should form correctly", func() {
			expected := []byte{4, 224, 0, 0, 0, 0, 24, 192, 168, 1, 192, 168, 2, 1}
			routes := []netlink.Route{
				{
					LinkIndex: 3,
					Dst: &net.IPNet{
						IP:   net.IPv4(224, 0, 0, 0),
						Mask: net.CIDRMask(4, 32),
					},
				},
				{
					LinkIndex: 12,
					Dst: &net.IPNet{
						IP:   net.IPv4(192, 168, 1, 0),
						Mask: net.CIDRMask(24, 32),
					},
					Src: nil,
					Gw:  net.IPv4(192, 168, 2, 1),
				},
			}

			dhcpRoutes := formClasslessRoutes(&routes)
			Expect(dhcpRoutes).To(Equal(expected))
		})

		It("should not panic", func() {
			dhcpRoutes := formClasslessRoutes(nil)
			Expect(dhcpRoutes).To(Equal([]byte{}))
		})

		It("should build OpenShift routes correctly", func() {
			expected := []byte{14, 10, 128, 0, 0, 0, 0, 4, 224, 0, 0, 0, 0, 0, 10, 129, 0, 1}
			gatewayRoute := netlink.Route{Gw: net.IPv4(10, 129, 0, 1)}
			staticRoute1 := netlink.Route{
				Dst: &net.IPNet{
					IP:   net.IPv4(10, 128, 0, 0),
					Mask: net.CIDRMask(14, 32),
				},
			}
			staticRoute2 := netlink.Route{
				Dst: &net.IPNet{
					IP:   net.IPv4(224, 0, 0, 0),
					Mask: net.CIDRMask(4, 32),
				},
			}
			routes := []netlink.Route{gatewayRoute, staticRoute1, staticRoute2}
			routeBytes := formClasslessRoutes(&routes)
			Expect(routeBytes).To(Equal(expected))
		})

		It("should build Calico routes correctly", func() {
			expected := []byte{32, 169, 254, 1, 1, 0, 0, 0, 0, 0, 169, 254, 1, 1}
			gatewayRoute := netlink.Route{Gw: net.IPv4(169, 254, 1, 1)}
			staticRoute1 := netlink.Route{
				Dst: &net.IPNet{
					IP:   net.IPv4(169, 254, 1, 1),
					Mask: net.CIDRMask(32, 32),
				},
			}

			routes := []netlink.Route{gatewayRoute, staticRoute1}
			routeBytes := formClasslessRoutes(&routes)
			Expect(routeBytes).To(Equal(expected))
		})
	})

	Context("function convertSearchDomainsToBytes(searchDomainStrings []string) ([]byte, error)", func() {
		It("should return RFC3397 compatible DNS search data", func() {
			searchDomains := []string{"foo.com", "foo.local"}
			expected := []byte{3, 'f', 'o', 'o', 3, 'c', 'o', 'm', 0, 3, 'f', 'o', 'o', 5, 'l', 'o', 'c', 'a', 'l', 0}
			result, err := convertSearchDomainsToBytes(searchDomains)
			Expect(result).To(Equal(expected))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject invalid domains", func() {
			searchDomains := []string{"foo,com"}
			_, err := convertSearchDomainsToBytes(searchDomains)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix(errorSearchDomainNotValid))
		})

		It("should reject search domains that exceed max length", func() {
			// should result in 256 byte slice
			searchDomains := []string{
				"pix3ob5ymm5jbsjessf0o4e84uvij588rz23iz0o.com",
				"3wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"38rfuqbyvkjg4z1f3aogul55wtgxrd9dlwzewqo0.com",
				"yza01ojnzi0tkyjeusmlg728nqdqvz3domymifvq.com",
				"m130lhs7a8yjgpn6almqggkqc222otedms6vslcd.com",
				"t4lanpt7z4ix58nvxl4d.com",
			}

			_, err := convertSearchDomainsToBytes(searchDomains)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(HavePrefix(errorSearchDomainTooLong))
		})
	})

	Context("function isValidSearchDomain(domain string) bool", func() {
		createBytes := func(size int) []byte {
			b := make([]byte, size)
			for i := range b {
				b[i] = 'a'
			}
			return b
		}

		DescribeTable("should *REJECT* domains", func(domaintToTest string) {
			Expect(isValidSearchDomain(domaintToTest)).To(BeFalse())
		},
			Entry("that start with '-'", "-foo.com"),
			Entry("that end with '-'", "foo.com-"),
			Entry("with full length greater than 253 chars", string(append(createBytes(250), []byte(".com")...))),
			Entry("that have labels longer than 63 chars", string(append(createBytes(64), []byte(".com")...))),
			Entry("with invalid characters", "foo\n.com"),
		)

		DescribeTable("should *ACCEPT* domains", func(domaintToTest string) {
			Expect(isValidSearchDomain(domaintToTest)).To(BeTrue())
		},
			Entry("that have 63 character labels", string(append(createBytes(63), []byte(".com")...))),
			Entry("with a valid domain name", "example.default.svc.cluster.local"),
			Entry("with a valid FQDN", "example.default.svc.cluster.local."),
			Entry("with a partial search domain", "local"),
		)

		XIt("should accept domains with full length of 253 chars", func() {
			b := append(createBytes(249), []byte(".com")...)
			Expect(isValidSearchDomain(string(b))).To(BeTrue())
		})
	})

	Context("Options returned by prepareDHCPOptions", func() {
		It("should contain the domain name option", func() {
			searchDomains := []string{
				"pix3ob5ymm5jbsjessf0o4e84uvij588rz23iz0o.com",
				"3wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"t4lanpt7z4ix58nvxl4d.com",
				"14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"4wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
			}
			ip := net.ParseIP("192.168.2.1")
			options, err := prepareDHCPOptions(ip.DefaultMask(), ip, nil, nil, searchDomains, 1500, "myhost", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(options[dhcp4.OptionDomainName]).To(Equal([]byte("14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com")))
		})

		It("should contain custom options", func() {
			searchDomains := []string{
				"pix3ob5ymm5jbsjessf0o4e84uvij588rz23iz0o.com",
				"3wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"t4lanpt7z4ix58nvxl4d.com",
				"14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"4wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
			}
			ip := net.ParseIP("192.168.2.1")

			dhcpOptions := &v1.DHCPOptions{
				BootFileName:   "config",
				TFTPServerName: "tftp.kubevirt.io",
				NTPServers: []string{
					"192.168.2.2", "192.168.2.3",
				},
				PrivateOptions: []v1.DHCPPrivateOptions{{Option: 240, Value: "private.options.kubevirt.io"}},
			}

			options, err := prepareDHCPOptions(ip.DefaultMask(), ip, nil, nil, searchDomains, 1500, "myhost", dhcpOptions)

			Expect(err).ToNot(HaveOccurred())
			Expect(options[dhcp4.OptionBootFileName]).To(Equal([]byte("config")))
			Expect(options[dhcp4.OptionTFTPServerName]).To(Equal([]byte("tftp.kubevirt.io")))
			Expect(options[dhcp4.OptionNetworkTimeProtocolServers]).To(Equal([]byte{
				192, 168, 2, 2, 192, 168, 2, 3,
			}))
			Expect(options[240]).To(Equal([]byte("private.options.kubevirt.io")))
		})

		It("expects the gateway as an IPv4 addresses", func() {
			gw := net.ParseIP("192.168.2.1")
			options, err := prepareDHCPOptions(gw.DefaultMask(), gw, nil, nil, nil, 1500, "myhost", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(options[dhcp4.OptionRouter]).To(Equal([]byte{192, 168, 2, 1}))
		})

		Context("Options set to invalid value", func() {
			var (
				err           error
				clientMask    []byte
				routerIP      net.IP
				dnsIPs        [][]byte
				routes        *[]netlink.Route
				hostname      string
				searchDomains []string
				dhcpOptions   *v1.DHCPOptions
				options       dhcp4.Options
			)
			BeforeEach(func() {
				options, err = prepareDHCPOptions(clientMask, routerIP, dnsIPs, routes, searchDomains, 1500, hostname, dhcpOptions)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should omit RouterIP Option", func() {
				_, ok := options[dhcp4.OptionRouter]
				Expect(ok).To(BeFalse())
			})
			It("should omit Classless Static Route Option", func() {
				_, ok := options[dhcp4.OptionClasslessRouteFormat]
				Expect(ok).To(BeFalse())
			})
			It("should omit Subnet Mask Option", func() {
				_, ok := options[dhcp4.OptionSubnetMask]
				Expect(ok).To(BeFalse())
			})
		})
	})
})
