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

package dhcp

import (
	"net"

	"github.com/krolaw/dhcp4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("DHCP", func() {

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
			Expect(err).To(BeNil())
		})

		It("should reject invalid domains", func() {
			searchDomains := []string{"foo,com"}
			_, err := convertSearchDomainsToBytes(searchDomains)
			Expect(err).NotTo(BeNil())
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
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(HavePrefix(errorSearchDomainTooLong))
		})
	})

	Context("function getDomainName", func() {
		It("should return the longest search domain entry", func() {
			searchDomains := []string{
				"pix3ob5ymm5jbsjessf0o4e84uvij588rz23iz0o.com",
				"3wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"t4lanpt7z4ix58nvxl4d.com",
				"14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"4wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
			}
			domain := getDomainName(searchDomains)
			Expect(domain).To(Equal("14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com"))
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

		It("should reject domains that start with '-'", func() {
			dom := "-foo.com"
			Expect(isValidSearchDomain(dom)).To(BeFalse())
		})

		It("should reject domains with full length greater than 253 chars", func() {
			b := append(createBytes(250), []byte(".com")...)
			Expect(isValidSearchDomain(string(b))).To(BeFalse())
		})

		It("should accept domains with full length of 253 chars", func() {
			b := append(createBytes(249), []byte(".com")...)
			Expect(isValidSearchDomain(string(b))).To(BeFalse())
		})

		It("should reject domains that have labels longer than 63 chars", func() {
			b := append(createBytes(64), []byte(".com")...)
			Expect(isValidSearchDomain(string(b))).To(BeFalse())
		})

		It("should accept domains that have 63 character labels", func() {
			b := append(createBytes(63), []byte(".com")...)
			Expect(isValidSearchDomain(string(b))).To(BeTrue())
		})

		It("should reject domains with invalid characters", func() {
			dom := "foo\n.com"
			Expect(isValidSearchDomain(dom)).To(BeFalse())
		})

		It("should accept a valid domain", func() {
			dom := "example.default.svc.cluster.local"
			Expect(isValidSearchDomain(dom)).To(BeTrue())
		})

		It("should accept a valid FQDN", func() {
			dom := "example.default.svc.cluster.local."
			Expect(isValidSearchDomain(dom)).To(BeTrue())
		})

		It("should accept a partial search domain", func() {
			Expect(isValidSearchDomain("local")).To(BeTrue())
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
			options, err := prepareDHCPOptions(ip.DefaultMask(), ip, nil, nil, searchDomains, 1500, "myhost")
			Expect(err).ToNot(HaveOccurred())
			Expect(options[dhcp4.OptionDomainName]).To(Equal([]byte("14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com")))
		})
	})
})
