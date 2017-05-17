package convert

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Convert", func() {

	vm := v1.NewMinimalVM("testvm")
	var buffer *bytes.Buffer
	var cmd *Convert
	var flags *pflag.FlagSet

	BeforeEach(func() {
		buffer = bytes.NewBuffer(nil)
		cmd = &Convert{ioutil.NopCloser(buffer), buffer}
		flags = cmd.FlagSet()
	})

	Context("Document conversions", func() {

		table.DescribeTable("converter invocation given, it", func(s *v1.VM, toStream Encoder, arguments string, fromStream Decoder, t *v1.VM) {
			flags.Parse(strings.Split(arguments, " "))

			Expect(toStream(s, buffer)).To(Succeed())
			cmd.Run(flags)
			Expect(fromStream(buffer)).To(Equal(t))
		},
			table.Entry("should take YAML and convert it explicit to JSON", vm, toYAML, "-f - -o json", fromJSON, vm),
			table.Entry("should take YAML and convert it implicit to JSON", vm, toYAML, "-f - ", fromJSON, vm),
			table.Entry("should take YAML and output YAML", vm, toJSON, "-f - -o yaml", fromYAML, vm),
			table.Entry("should take YAML and output XML", vm, toYAML, "-f - -o xml", fromXML, vm),
			table.Entry("should take XML and output YAML", vm, toXML, "-f - -o yaml", fromYAML, vm),
		)

	})

	Context("Document sources", func() {
		It("should read the source from stdin", func() {
			Expect(toJSON(vm, buffer)).To(Succeed())
			flags.Parse(strings.Split("-f -", " "))
			Expect(fromJSON(buffer)).To(Equal(vm))

		})
		It("should read the source from a file", func() {
			// Create temporary file
			tmpfile, err := ioutil.TempFile("", "conversiontest")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tmpfile.Name())

			// Write content
			Expect(toJSON(vm, tmpfile)).To(Succeed())
			tmpfile.Close()

			// Reopen for reading the content
			tmpfile, err = os.Open(tmpfile.Name())
			flags.Parse(strings.Split("-f "+tmpfile.Name(), " "))
			cmd.Run(flags)
			Expect(fromJSON(buffer)).To(Equal(vm))
		})
		It("should read the source from a http resource", func() {
			server := ghttp.NewServer()
			defer server.Close()
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/spec/vm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)
			flags.Parse(strings.Split("-f "+server.URL()+"/spec/vm", " "))
			cmd.Run(flags)
			Expect(server.ReceivedRequests()).To(HaveLen(1))
			Expect(fromJSON(buffer)).To(Equal(vm))
		})

	})
})

func fromYAML(reader io.Reader) (*v1.VM, error) {
	r, t := GuessStreamType(reader, 2048)
	Expect(t).To(Equal(YAML))
	return fromYAMLOrJSON(r)
}

func fromJSON(reader io.Reader) (*v1.VM, error) {
	r, t := GuessStreamType(reader, 2048)
	Expect(t).To(Equal(JSON))
	return fromYAMLOrJSON(r)
}
