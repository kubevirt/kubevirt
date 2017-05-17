package convert_test

import (
	. "kubevirt.io/kubevirt/pkg/virtctl/convert"

	"bytes"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Guess", func() {
	table.DescribeTable("Should detect stream type", func(data string, detected Type) {
		b := []byte(data)
		reader, t := GuessStreamType(bytes.NewReader(b), 2048)
		Expect(t).To(Equal(detected))
		Expect(ioutil.ReadAll(reader)).To(Equal(b))
	},
		table.Entry("Should detect a XML stream", "    <xml></xml>", XML),
		table.Entry("Should detect a JSON stream", `    {"a": "b"}`, JSON),
		table.Entry("Should fall back to YAML if it is not a JSON or XML stream", `    a: "b"`, YAML),
	)
})
