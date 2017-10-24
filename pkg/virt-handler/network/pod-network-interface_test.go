package network

import (
	"net"

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pod Network Interface", func() {
	Context("Loopback Interface", func() {
		mac, _ := net.ParseMAC("00:11:22:33:44:55")

		var podNetworkInterface PodNetworkInterface

		BeforeEach(func() {
			podNetworkInterface = PodNetworkInterface{
				Name: "test",
				IPAddr: net.IPNet{IP: net.ParseIP("127.0.0.1"),
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
				Mac: mac,
			}
		})

		It("Should create a podNetworkINterface", func() {
			iface, err := podNetworkInterface.GetConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(iface.Type).To(Equal("direct"))
			Expect(iface.TrustGuestRxFilters).To(Equal("yes"))
		})

		It("Should add interface attributes", func() {
			origIP := podNetworkInterface.IPAddr.IP
			err := podNetworkInterface.SetInterfaceAttributes("99:88:77:66:55:44", "1.2.3.4", "fake")
			Expect(err).ToNot(HaveOccurred())
			Expect(podNetworkInterface.IPAddr.IP).ToNot(Equal(origIP))
		})

		It("Should decorate interface metadata", func() {
			meta := podNetworkInterface.DecorateInterfaceMetadata()
			Expect(meta.IP).To(Equal("127.0.0.1"))
			Expect(meta.Device).To(Equal("test"))
		})
	})
})

func TestPodNetworkInterface(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pod Network Interface")
}
