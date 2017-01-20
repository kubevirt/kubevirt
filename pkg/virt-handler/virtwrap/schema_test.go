package virtwrap

import (
	"encoding/json"
	"encoding/xml"
	"github.com/jeevatkm/go-model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

var exampleXML = `
<domain type="qemu">
  <name>testvm</name>
  <memory unit="KiB">8192</memory>
  <os>
    <type>hvm</type>
  </os>
    <devices>
      <emulator>/usr/local/bin/qemu-x86_64</emulator>
      <interface type='network'>
        <source network='default'/>
      </interface>
    </devices>
</domain>
`

var exampleJSON = `
{
   "name":"testvm",
   "memory":{
      "value":8192,
      "unit":"KiB"
   },
   "type":"qemu",
   "os":{
      "type":{
         "os":"hvm"
      },
      "bootOrder":null
   },
   "devices":{
      "emulator":"/usr/local/bin/qemu-x86_64",
      "interfaces":[
         {
            "type":"network",
            "source":{
               "network":"default"
            }
         }
      ]
   }
}
`

var _ = Describe("Schema", func() {
	Context("With schema", func() {
		It("Generate expected libvirt xml", func() {
			domain := NewMinimalVM("testvm")
			buf, err := xml.Marshal(domain)
			Expect(err).To(BeNil())
			newDomain := DomainSpec{}
			err = xml.Unmarshal(buf, &newDomain)
			Expect(err).To(BeNil())
			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
		It("Generate expected libvirt json", func() {
			domain := NewMinimalVM("testvm")
			buf, err := json.Marshal(domain)
			Expect(err).To(BeNil())
			newDomain := DomainSpec{}
			err = json.Unmarshal(buf, &newDomain)
			Expect(err).To(BeNil())
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With example schema in xml", func() {
		It("Unmarshal into struct", func() {
			domain := NewMinimalVM("testvm")
			newDomain := DomainSpec{}
			err := xml.Unmarshal([]byte(exampleXML), &newDomain)
			Expect(err).To(BeNil())
			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With example schema in json", func() {
		It("Unmarshal into struct", func() {
			domain := NewMinimalVM("testvm")
			newDomain := DomainSpec{}
			err := json.Unmarshal([]byte(exampleJSON), &newDomain)
			Expect(err).To(BeNil())
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With v1.DomainSpec", func() {
		It("converts to libvirt.DomainSpec", func() {
			v1DomainSpec := v1.NewMinimalDomainSpec("testvm")
			libvirtDomainSpec := DomainSpec{}
			errs := model.Copy(&libvirtDomainSpec, v1DomainSpec)
			Expect(libvirtDomainSpec).To(Equal(*NewMinimalVM("testvm")))
			Expect(errs).To(BeEmpty())
		})
	})
})
