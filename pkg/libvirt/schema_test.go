package libvirt

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
        <source network='kubevirt-net'/>
      </interface>
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
               "network":"kubevirt-net"
            }
         },
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
			newDomain := Domain{}
			err = xml.Unmarshal(buf, &newDomain)
			Expect(err).To(BeNil())
			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
		It("Generate expected libvirt json", func() {
			domain := NewMinimalVM("testvm")
			buf, err := json.Marshal(domain)
			Expect(err).To(BeNil())
			newDomain := Domain{}
			err = json.Unmarshal(buf, &newDomain)
			Expect(err).To(BeNil())
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With example schema in xml", func() {
		It("Unmarshal into struct", func() {
			domain := NewMinimalVM("testvm")
			newDomain := Domain{}
			err := xml.Unmarshal([]byte(exampleXML), &newDomain)
			Expect(err).To(BeNil())
			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With example schema in json", func() {
		It("Unmarshal into struct", func() {
			domain := NewMinimalVM("testvm")
			newDomain := Domain{}
			err := json.Unmarshal([]byte(exampleJSON), &newDomain)
			Expect(err).To(BeNil())
			Expect(newDomain).To(Equal(*domain))
		})
	})
})
