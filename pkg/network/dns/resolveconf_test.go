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
 * Copyright The KubeVirt Authors.
 */

package dns

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resolveconf", func() {
	Context("Function ParseNameservers()", func() {
		It("should return a byte array of nameservers", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "nameserver 8.8.8.8\nnameserver 8.8.4.4\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(err).ToNot(HaveOccurred())
			Expect(nameservers.IPv4).To(Equal([][]byte{ns1, ns2}))
			Expect(nameservers.IPv6).To(BeEmpty())
		})

		It("should return a byte array of IPv6 nameservers", func() {
			ipv6_1 := net.ParseIP("2001:db8::1").To16()
			ipv6_2 := net.ParseIP("::1").To16()
			resolvConf := "nameserver 2001:db8::1\nnameserver ::1\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(err).ToNot(HaveOccurred())
			Expect(nameservers.IPv4).To(BeEmpty())
			Expect(nameservers.IPv6).To(Equal([][]byte{ipv6_1, ipv6_2}))
		})

		It("should handle mixed IPv4 and IPv6 nameservers", func() {
			ipv4 := []uint8{8, 8, 8, 8}
			ipv6 := net.ParseIP("2001:db8::1").To16()
			resolvConf := "nameserver 8.8.8.8\nnameserver 2001:db8::1\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(err).ToNot(HaveOccurred())
			Expect(nameservers.IPv4).To(Equal([][]byte{ipv4}))
			Expect(nameservers.IPv6).To(Equal([][]byte{ipv6}))
		})

		It("should ignore non-nameserver lines and malformed nameserver lines", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "search example.com\nnameserver 8.8.8.8\nnameserver 8.8.4.4\nnameserver mynameserver\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(err).ToNot(HaveOccurred())
			Expect(nameservers.IPv4).To(Equal([][]byte{ns1, ns2}))
			Expect(nameservers.IPv6).To(BeEmpty())
		})

		It("should return a default nameserver if none is parsed", func() {
			nameservers, err := ParseNameservers("")
			expectedDNS := net.ParseIP(defaultDNS).To4()
			Expect(err).ToNot(HaveOccurred())
			Expect(nameservers.IPv4).To(Equal([][]byte{expectedDNS}))
			Expect(nameservers.IPv6).To(BeEmpty())
		})
	})

	Context("Function ParseSearchDomains()", func() {
		It("should return a string of search domains", func() {
			resolvConf := "search cluster.local svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle multi-line search domains", func() {
			resolvConf := "search cluster.local\nsearch svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should clean up extra whitespace between search domains", func() {
			resolvConf := "search cluster.local\tsvc.cluster.local    example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle non-presence of search domains by returning default search domain", func() {
			resolvConf := fmt.Sprintf("nameserver %s\n", defaultDNS)
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{defaultSearchDomain}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow partial search domains", func() {
			resolvConf := "search local\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"local"}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should normalize search domains to lower-case", func() {
			resolvConf := "search LoCaL\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"local"}))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("function GetDomainName", func() {
		It("should return the longest search domain entry", func() {
			searchDomains := []string{
				"pix3ob5ymm5jbsjessf0o4e84uvij588rz23iz0o.com",
				"3wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"t4lanpt7z4ix58nvxl4d.com",
				"14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
				"4wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com",
			}
			domain := GetDomainName(searchDomains)
			Expect(domain).To(Equal("14wg5xngig6vzfqjww4kocnky3c9dqjpwkewzlwpf.com"))
		})
	})

	Context("subdomain", func() {
		It("should be added to the longest service domain", func() {
			searchDomains := []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"}

			const subdomain = "subdomain"
			domain := DomainNameWithSubdomain(searchDomains, subdomain)
			Expect(domain).To(Equal(subdomain + "." + searchDomains[0]))
		})

		It("should not be added if subdomain is empty", func() {
			searchDomains := []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"}

			const subdomain = ""
			domain := DomainNameWithSubdomain(searchDomains, subdomain)
			Expect(domain).To(Equal(""))
		})

		It("should be added even if the longest existing service domain isn't the first", func() {
			searchDomains := []string{"svc.cluster.local", "cluster.local", "default.svc.cluster.local"}

			const subdomain = "subdomain"
			domain := DomainNameWithSubdomain(searchDomains, subdomain)
			Expect(domain).To(Equal(subdomain + "." + searchDomains[2]))
		})

		It("should not be added if the longest existing service domain already has it", func() {
			searchDomains := []string{"svc.cluster.local", "cluster.local", "subdomain.default.svc.cluster.local"}

			const subdomain = "subdomain"
			domain := DomainNameWithSubdomain(searchDomains, subdomain)
			Expect(domain).To(Equal(""))
		})

		It("should be added to the right entry if the longest entry is not a service entry", func() {
			searchDomains := []string{"default.svc.cluster.local", "svc.cluster.local",
				"cluster.local", "this.is.a.very.very.very.long.entry"}

			const subdomain = "subdomain"
			domain := DomainNameWithSubdomain(searchDomains, subdomain)
			Expect(domain).To(Equal(subdomain + "." + searchDomains[0]))
		})

		It("should not be added if there is no service entry", func() {
			searchDomains := []string{"example.com"}

			const subdomain = "subdomain"
			domain := DomainNameWithSubdomain(searchDomains, subdomain)
			Expect(domain).To(Equal(""))
		})
	})
})
