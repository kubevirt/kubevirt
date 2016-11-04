package services_test

import (
	. "kubevirt/core/pkg/virt-controller/services"

	"bytes"
	"github.com/go-kit/kit/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"kubevirt/core/pkg/api"
	"os"
)

var _ = Describe("Template", func() {

	logger := log.NewLogfmtLogger(os.Stderr)
	svc, err := NewTemplateService(logger, "../../../cmd/virt-controller/templates/manifest-template.yaml", "kubevirt", "virt-launcher")
	launchArgs := `
    command:
        - "/virt-launcher"
        - "-downward-api-path"
        - "/etc/kubeapi/annotations"
`
	migrationArgs := `
    command:
        - "/virt-launcher"
        - "-receive-only"`
	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {

				Expect(err).To(BeNil())
				buffer := new(bytes.Buffer)
				err = svc.RenderLaunchManifest(&api.VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm"}}, []byte("<domain>\n</domain>"), buffer)

				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("image: kubevirt/virt-launcher"))
				Expect(buffer.String()).To(ContainSubstring("domain: testvm"))
				Expect(buffer.String()).To(ContainSubstring("generateName: virt-launcher-testvm"))
				Expect(buffer.String()).To(ContainSubstring("&lt;domain&gt;\\n&lt;/domain&gt;"))
				Expect(buffer.String()).To(Not(ContainSubstring("  nodeSelector:")))
				Expect(buffer.String()).To(ContainSubstring(launchArgs))
			})
		})
		Context("migration template with correct parameters", func() {
			It("should work", func() {

				Expect(err).To(BeNil())
				buffer := new(bytes.Buffer)
				err = svc.RenderMigrationManifest(&api.VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm"}}, buffer)
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("image: kubevirt/virt-launcher"))
				Expect(buffer.String()).To(ContainSubstring("domain: testvm"))
				Expect(buffer.String()).To(ContainSubstring("generateName: virt-launcher-testvm"))
				Expect(buffer.String()).To(Not(ContainSubstring("annotations:")))
				Expect(buffer.String()).To(Not(ContainSubstring("  nodeSelector:")))
				Expect(buffer.String()).To(ContainSubstring(migrationArgs))
			})
		})
		Context("with wrong template path", func() {
			It("should fail", func() {
				_, err := NewTemplateService(logger, "templates/manifest-template.yaml", "kubevirt", "virt-launcher")
				Expect(err).To(Not(BeNil()))
			})
		})
		Context("with node selectors", func() {
			It("should add node selectors to template", func() {

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
				}
				vm := api.VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm"}, Spec: api.VMSpec{NodeSelector: nodeSelector}}
				buffer := new(bytes.Buffer)

				err = svc.RenderLaunchManifest(&vm, []byte("<domain>\n</domain>"), buffer)

				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("image: kubevirt/virt-launcher"))
				Expect(buffer.String()).To(ContainSubstring("domain: testvm"))
				Expect(buffer.String()).To(ContainSubstring("generateName: virt-launcher-testvm"))
				Expect(buffer.String()).To(ContainSubstring("&lt;domain&gt;\\n&lt;/domain&gt;"))
				Expect(buffer.String()).To(ContainSubstring("  nodeSelector:"))
				Expect(buffer.String()).To(ContainSubstring("    kubernetes.io/hostname: master"))
			})
		})
	})

})
