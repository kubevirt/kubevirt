package nodelabeller

import (
	"encoding/xml"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"
)

var _ = Describe("Capabilities", func() {

	It("should be able to read the TSC timer freqemency from the host", func() {
		f, err := os.Open("testdata/capabilities.xml")
		Expect(err).ToNot(HaveOccurred())
		defer f.Close()
		capabilities := &api.Capabilities{}
		Expect(xml.NewDecoder(f).Decode(capabilities)).To(Succeed())
		Expect(capabilities.Host.CPU.Counter).To(HaveLen(1))
		Expect(capabilities.Host.CPU.Counter[0].Name).To(Equal("tsc"))
		Expect(capabilities.Host.CPU.Counter[0].Frequency).To(BeNumerically("==", 4008012000))
		Expect(bool(capabilities.Host.CPU.Counter[0].Scaling)).To(BeFalse())
		counter, err := capabilities.GetTSCCounter()
		Expect(err).ToNot(HaveOccurred())
		Expect(counter.Frequency).To(BeNumerically("==", 4008012000))
		Expect(bool(counter.Scaling)).To(BeFalse())
	})
})
