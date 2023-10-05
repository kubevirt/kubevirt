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
			Expect(nameservers).To(Equal([][]byte{ns1, ns2}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should ignore non-nameserver lines and malformed nameserver lines", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "search example.com\nnameserver 8.8.8.8\nnameserver 8.8.4.4\nnameserver mynameserver\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(nameservers).To(Equal([][]byte{ns1, ns2}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return a default nameserver if none is parsed", func() {
			nameservers, err := ParseNameservers("")
			expectedDNS := net.ParseIP(defaultDNS).To4()
			Expect(nameservers).To(Equal([][]byte{expectedDNS}))
			Expect(err).ToNot(HaveOccurred())
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
