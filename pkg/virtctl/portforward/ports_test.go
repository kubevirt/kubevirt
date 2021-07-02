package portforward

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ports", func() {

	table.DescribeTable("parsePort", func(arg string, port forwardedPort, success bool) {
		result, err := parsePort(arg)
		if success {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
		Expect(result).To(Equal(port))
	},
		table.Entry("only one port", "8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolTCP}, true),
		table.Entry("local and remote port", "8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolTCP}, true),
		table.Entry("protocol and one port", "udp/8080", forwardedPort{local: 8080, remote: 8080, protocol: protocolUDP}, true),

		table.Entry("protocol and both ports", "udp/8080:8090", forwardedPort{local: 8080, remote: 8090, protocol: protocolUDP}, true),

		table.Entry("only protocol no slash", "udp", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
		table.Entry("only protocol with slash", "udp/", forwardedPort{local: 0, remote: 0, protocol: protocolUDP}, false),
		table.Entry("invalid symbol in port", "80C0:8X90", forwardedPort{local: 0, remote: 0, protocol: protocolTCP}, false),
	)
})
