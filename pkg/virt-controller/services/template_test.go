package services_test

import (
	. "kubevirt.io/kubevirt/pkg/virt-controller/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

var _ = Describe("Template", func() {

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)
	svc, err := NewTemplateService("kubevirt/virt-launcher")

	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {

				Expect(err).To(BeNil())
				pod, err := svc.RenderLaunchManifest(&v1.VM{ObjectMeta: kubev1.ObjectMeta{Name: "testvm", UID: "1234"}, Spec: v1.VMSpec{Domain: &v1.DomainSpec{}}})

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.VMUIDLabel:  "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvm-----"))
				Expect(pod.Spec.NodeSelector).To(BeEmpty())
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/virt-launcher", "-qemu-timeout", "60s"}))
			})
		})
		Context("with node selectors", func() {
			It("should add node selectors to template", func() {

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
				}
				vm := v1.VM{ObjectMeta: kubev1.ObjectMeta{Name: "testvm", UID: "1234"}, Spec: v1.VMSpec{NodeSelector: nodeSelector, Domain: &v1.DomainSpec{}}}

				pod, err := svc.RenderLaunchManifest(&vm)

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.VMUIDLabel:  "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvm-----"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					"kubernetes.io/hostname": "master",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/virt-launcher", "-qemu-timeout", "60s"}))
			})
		})
		Context("migration", func() {
			var (
				srcIp      = kubev1.NodeAddress{}
				destIp     = kubev1.NodeAddress{}
				srcNodeIp  = kubev1.Node{}
				destNodeIp = kubev1.Node{}
				srcNode    kubev1.Node
				targetNode kubev1.Node
			)

			BeforeEach(func() {
				srcIp = kubev1.NodeAddress{
					Type:    kubev1.NodeInternalIP,
					Address: "127.0.0.2",
				}
				destIp = kubev1.NodeAddress{
					Type:    kubev1.NodeInternalIP,
					Address: "127.0.0.3",
				}
				srcNodeIp = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{srcIp},
					},
				}
				destNodeIp = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{destIp},
					},
				}
				srcNode = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{srcIp, destIp},
					},
				}
				targetNode = kubev1.Node{
					Status: kubev1.NodeStatus{
						Addresses: []kubev1.NodeAddress{destIp, srcIp},
					},
				}
			})

			Context("migration template with correct parameters", func() {
				It("should never restart", func() {
					vm := v1.NewMinimalVM("testvm")

					job, err := svc.RenderMigrationJob(vm, &srcNodeIp, &destNodeIp)
					Expect(err).ToNot(HaveOccurred())
					Expect(job.Spec.RestartPolicy).To(Equal(kubev1.RestartPolicyNever))
				})
				It("should use the first ip it finds", func() {
					vm := v1.NewMinimalVM("testvm")
					job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode)
					Expect(err).ToNot(HaveOccurred())
					refCommand := []string{
						"virsh", "-c", "qemu+tcp://127.0.0.2/system", "migrate", "--tunnelled", "--p2p", "testvm",
						"qemu+tcp://127.0.0.3/system"}
					Expect(job.Spec.Containers[0].Command).To(Equal(refCommand))
				})
			})
			Context("migration template with incorrect parameters", func() {
				It("should error on missing source address", func() {
					vm := v1.NewMinimalVM("testvm")
					srcNode.Status.Addresses = []kubev1.NodeAddress{}
					job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode)
					Expect(err).To(HaveOccurred())
					Expect(job).To(BeNil())
				})
				It("should error on missing destination address", func() {
					vm := v1.NewMinimalVM("testvm")
					targetNode.Status.Addresses = []kubev1.NodeAddress{}
					job, err := svc.RenderMigrationJob(vm, &srcNode, &targetNode)
					Expect(err).To(HaveOccurred())
					Expect(job).To(BeNil())
				})
			})
		})
	})

})
