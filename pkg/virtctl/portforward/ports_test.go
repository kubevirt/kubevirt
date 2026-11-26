package portforward

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ports", func() {

	DescribeTable("parsePort", func(arg string, port forwardedPort, success bool) {
		result, err := parsePort(arg)
		if success {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
		Expect(result).To(Equal(port))
	},
		Entry("only one port", "8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolTCP}, true),
		Entry("local and remote port", "8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolTCP}, true),
		Entry("tcp protocol and one port", "tcp/8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolTCP}, true),
		Entry("tcp protocol and both ports", "tcp/8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolTCP}, true),
		Entry("udp protocol and one port", "udp/8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolUDP}, true),
		Entry("udp protocol and both ports", "udp/8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolUDP}, true),

		Entry("only protocol no slash", "udp", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
		Entry("only protocol with slash", "udp/", forwardedPort{local: 0, remote: 0, protocol: protocolUDP}, false),
		Entry("invalid symbol in port", "80C0:8X90", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
		Entry("invalid remote port", "8080:8X90", forwardedPort{local: 8080, remote: 0, protocol: protocolTCP}, false),
	)

	DescribeTable("parsePort rejects extra colon-separated ports",
		func(arg string) {
			_, err := parsePort(arg)
			Expect(err).To(HaveOccurred())
		},
		Entry("three ports", "8080:9090:9999"),
		Entry("protocol and three ports", "udp/8080:9090:9999"),
		Entry("trailing colon after two ports", "8080:9090:"),
	)

	Context("parsePorts", func() {
		It("parses multiple valid ports", func() {
			ports, err := parsePorts([]string{"8080", "tcp/8443:443", "udp/5353:53"})
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]forwardedPort{
				{local: 8080, remote: 8080, protocol: protocolTCP},
				{local: 8443, remote: 443, protocol: protocolTCP},
				{local: 5353, remote: 53, protocol: protocolUDP},
			}))
		})

		It("handles empty input slice", func() {
			ports, err := parsePorts([]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(BeEmpty())
		})

		It("returns an error when any port argument is invalid", func() {
			_, err := parsePorts([]string{"8080", "invalid"})
			Expect(err).To(HaveOccurred())
		})
	})
})
