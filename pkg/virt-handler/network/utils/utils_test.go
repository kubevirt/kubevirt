package utils

import (
	"strings"
	"testing"

	lmf "github.com/subgraph/libmacouflage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network Utils", func() {
	Context("Defaults", func() {
		It("Should generate a random tap name", func() {
			tap, err := GenerateRandomTapName()
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.HasPrefix(tap, tapPrefix)).To(BeTrue())
		})

		It("Should find loopback interface", func() {
			iface, err := GetInterfaceByIP("127.0.0.1")
			Expect(err).ToNot(HaveOccurred())
			Expect(iface.Name).To(Equal("lo"))
		})

		It("Should fail to find nonsense address", func() {
			iface, err := GetInterfaceByIP("0.0.0.0")
			Expect(err).To(HaveOccurred())
			Expect(iface).To(BeNil())
			Expect(err.Error()).To(Equal("failed to find an interface with ip: 0.0.0.0"))
		})

		It("Should error trying to find loopback mac", func() {
			mac, err := GetMacDetails("lo")
			Expect(err).To(HaveOccurred())
			Expect(mac).To(BeNil())
			Expect(err.Error()).To(Equal("Invalid interface type: lo"))
		})

		It("", func() {
			ifaces, err := lmf.GetInterfaces()
			Expect(err).ToNot(HaveOccurred())
			if len(ifaces) < 1 {
				Skip("No interfaces present")
			}
			mac, err := GetMacDetails(ifaces[0].Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(mac).ToNot(BeNil())
		})
	})
})

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Utils")
}
