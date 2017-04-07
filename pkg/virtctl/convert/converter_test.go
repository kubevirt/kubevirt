package convert

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Converter", func() {

	vm := v1.NewMinimalVM("testvm")

	table.DescribeTable("", func(s *v1.VM, toStream Encoder, fromStream Decoder, t *v1.VM) {
		b := bytes.NewBuffer(nil)
		Expect(toStream(s, b)).To(Succeed())
		Expect(fromStream(b)).To(Equal(t))
	},
		table.Entry("Should convert to YAML and back to struct", vm, toYAML, fromYAMLOrJSON, vm),
		table.Entry("Should convert to JSON and back to struct", vm, toJSON, fromYAMLOrJSON, vm),
		table.Entry("Should convert to XML and back to struct", vm, toXML, fromXML, vm),
	)
})
