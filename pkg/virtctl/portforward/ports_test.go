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
		Entry("protocol and one port", "udp/8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolUDP}, true),

		Entry("protocol and both ports", "udp/8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolUDP}, true),

		Entry("only protocol no slash", "udp", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
		Entry("only protocol with slash", "udp/", forwardedPort{local: 0, remote: 0, protocol: protocolUDP}, false),
		Entry("invalid symbol in port", "80C0:8X90", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
		Entry("valid port boundary minimum", "1", forwardedPort{local: 1, remote: 1, protocol: protocolTCP}, true),
		Entry("valid port boundary maximum", "65535", forwardedPort{local: 65535, remote: 65535, protocol: protocolTCP}, true),
		Entry("uppercase protocol TCP", "TCP/8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolTCP}, true),
		Entry("uppercase protocol UDP", "UDP/8080:9090", forwardedPort{local: 8080, remote: 9090, protocol: protocolUDP}, true),
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

	DescribeTable("parsePort rejects invalid protocol or port range",
		func(arg string) {
			_, err := parsePort(arg)
			Expect(err).To(HaveOccurred())
		},
		Entry("unknown protocol", "sctp/8080"),
		Entry("invalid protocol format", "http/8080"),
		Entry("zero local port", "0"),
		Entry("negative local port", "-1"),
		Entry("local port above 65535", "65536"),
		Entry("zero remote port", "8080:0"),
		Entry("negative remote port", "8080:-1"),
		Entry("remote port above 65535", "8080:65536"),
	)
})
