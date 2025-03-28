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
	)
})
