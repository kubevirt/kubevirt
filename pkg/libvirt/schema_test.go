package libvirt

import (
	"encoding/xml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var example = `
<domain type="qemu">
  <name>testvm</name>
  <memory unit="KiB">8192</memory>
  <os>
    <type>hvm</type>
  </os>
    <devices>
      <emulator>/usr/local/bin/qemu-x86_64</emulator>
      <interface type='network'>
        <source network='kubevirt-net'/>
      </interface>
      <interface type='network'>
        <source network='default'/>
      </interface>
    </devices>
</domain>
`

var _ = Describe("Schema", func() {
	Context("With schema", func() {
		It("Generate expected libvirt xml", func() {
			domain := NewMinimalVM("testvm")
			buf, err := xml.Marshal(domain)
			Expect(err).To(BeNil())
			newDomain := Domain{}
			err = xml.Unmarshal(buf, &newDomain)
			Expect(err).To(BeNil())
			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With example schema in xml", func() {
		It("Unmarshal into struct", func() {
			domain := NewMinimalVM("testvm")
			newDomain := Domain{}
			err := xml.Unmarshal([]byte(example), &newDomain)
			Expect(err).To(BeNil())
			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
	})
})
