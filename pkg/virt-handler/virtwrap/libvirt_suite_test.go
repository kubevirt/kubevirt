package virtwrap

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"testing"
	"time"
)

var _ = Describe("Libvirt Suite", func() {
	Context("Upon attempt to connect to Libvirt", func() {
		It("should time out while waiting for libvirt", func() {
			err := waitForLibvirt("http://", "", "", 1*time.Microsecond)
			msg := fmt.Sprintf("%v", err)
			Expect(err).To(HaveOccurred())
			Expect(msg).To(Equal("timed out waiting for the condition"))
		})
	})
})

func TestLibvirt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Libvirt Suite")
}
