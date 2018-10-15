package dns

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resolveconf", func() {
	Context("Function ParseNameservers()", func() {
		It("should return a byte array of nameservers", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "nameserver 8.8.8.8\nnameserver 8.8.4.4\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(nameservers).To(Equal([][]uint8{ns1, ns2}))
			Expect(err).To(BeNil())
		})

		It("should ignore non-nameserver lines and malformed nameserver lines", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "search example.com\nnameserver 8.8.8.8\nnameserver 8.8.4.4\nnameserver mynameserver\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(nameservers).To(Equal([][]uint8{ns1, ns2}))
			Expect(err).To(BeNil())
		})

		It("should return a default nameserver if none is parsed", func() {
			nameservers, err := ParseNameservers("")
			expectedDNS := net.ParseIP(defaultDNS).To4()
			Expect(nameservers).To(Equal([][]uint8{expectedDNS}))
			Expect(err).To(BeNil())
		})
	})

	Context("Function ParseSearchDomains()", func() {
		It("should return a string of search domains", func() {
			resolvConf := "search cluster.local svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should handle multi-line search domains", func() {
			resolvConf := "search cluster.local\nsearch svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should clean up extra whitespace between search domains", func() {
			resolvConf := "search cluster.local\tsvc.cluster.local    example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should handle non-presence of search domains by returning default search domain", func() {
			resolvConf := fmt.Sprintf("nameserver %s\n", defaultDNS)
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{defaultSearchDomain}))
			Expect(err).To(BeNil())
		})

		It("should allow partial search domains", func() {
			resolvConf := "search local\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"local"}))
			Expect(err).To(BeNil())
		})
	})
})
