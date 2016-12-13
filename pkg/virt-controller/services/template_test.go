package services_test

import (
	. "kubevirt.io/kubevirt/pkg/virt-controller/services"

	"github.com/go-kit/kit/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"os"
)

var _ = Describe("Template", func() {

	logger := log.NewLogfmtLogger(os.Stderr)
	svc, err := NewTemplateService(logger, "kubevirt", "virt-launcher")

	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {

				Expect(err).To(BeNil())
				pod, err := svc.RenderLaunchManifest(&v1.VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm", UID: "1234"}})

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.UIDLabel:    "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvm"))
				Expect(pod.Spec.NodeSelector).To(BeEmpty())
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/virt-launcher", "-qemu-timeout", "60s"}))
			})
		})
		Context("with node selectors", func() {
			It("should add node selectors to template", func() {

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
				}
				vm := v1.VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm", UID: "1234"}, Spec: v1.VMSpec{NodeSelector: nodeSelector}}

				pod, err := svc.RenderLaunchManifest(&vm)

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.UIDLabel:    "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvm"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					"kubernetes.io/hostname": "master",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/virt-launcher", "-qemu-timeout", "60s"}))
			})
		})
	})

})
