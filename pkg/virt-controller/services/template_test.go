package services_test

import (
	. "kubevirt/core/pkg/virt-controller/services"

	"bytes"
	"github.com/go-kit/kit/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"kubevirt/core/pkg/api"
	"os"
)

var _ = Describe("Template", func() {

	logger := log.NewLogfmtLogger(os.Stderr)

	Describe("Rendering", func() {
		Context("with correct parameters", func() {
			It("should work", func() {
				svc, err := NewTemplateService(logger, "../../../cmd/virt-controller/templates/manifest-template.yaml", "kubevirt", "virt-launcher")
				Expect(err).To(BeNil())

				buffer := new(bytes.Buffer)

				err = svc.RenderManifest(&api.VM{Name: "testvm"}, []byte("<domain>\n</domain>"), buffer)

				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("image: kubevirt/virt-launcher"))
				Expect(buffer.String()).To(ContainSubstring("domain: testvm"))
				Expect(buffer.String()).To(ContainSubstring("name: virt-launcher-testvm"))
				Expect(buffer.String()).To(ContainSubstring("&lt;domain&gt;\\n&lt;/domain&gt;"))
			})
		})
		Context("with wrong template path", func() {
			It("should fail", func() {
				_, err := NewTemplateService(logger, "templates/manifest-template.yaml", "kubevirt", "virt-launcher")
				Expect(err).To(Not(BeNil()))
			})
		})
	})

})
