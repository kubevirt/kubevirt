package services_test

import (
	. "kubevirt.io/kubevirt/pkg/virt-controller/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	coreapi "k8s.io/client-go/pkg/api"
	corev1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Template", func() {

	svc, err := NewTemplateService("kubevirt/virt-launcher")

	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {

				Expect(err).To(BeNil())
				pod, err := svc.RenderLaunchManifest(&v1.VM{ObjectMeta: corev1.ObjectMeta{Name: "testvm", UID: "1234"}, Spec: v1.VMSpec{Domain: &v1.DomainSpec{}}})

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.UIDLabel:    "1234",
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
				vm := v1.VM{ObjectMeta: corev1.ObjectMeta{Name: "testvm", UID: "1234"}, Spec: v1.VMSpec{NodeSelector: nodeSelector, Domain: &v1.DomainSpec{}}}

				pod, err := svc.RenderLaunchManifest(&vm)

				Expect(err).To(BeNil())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:    "virt-launcher",
					v1.DomainLabel: "testvm",
					v1.UIDLabel:    "1234",
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
				srcHost    = corev1.NodeAddress{}
				srcIp      = corev1.NodeAddress{}
				destHost   = corev1.NodeAddress{}
				destIp     = corev1.NodeAddress{}
				srcNode    = corev1.Node{}
				srcNodeIp  = corev1.Node{}
				destNodeIp = corev1.Node{}
				destNode   = corev1.Node{}
			)

			BeforeEach(func() {
				srcHost = corev1.NodeAddress{
					Type:    corev1.NodeHostName,
					Address: "src-node.kubevirt.io",
				}
				srcIp = corev1.NodeAddress{
					Type:    corev1.NodeInternalIP,
					Address: "127.0.0.2",
				}
				destHost = corev1.NodeAddress{
					Type:    corev1.NodeHostName,
					Address: "dest-node.kubevirt.io",
				}
				destIp = corev1.NodeAddress{
					Type:    corev1.NodeInternalIP,
					Address: "127.0.0.3",
				}
				// Note: the IP's are listed before the hostnames on srcNode and destNode
				// so that we can ensure we test the priority order of selection
				srcNode = corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{srcIp, srcHost},
					},
				}
				destNode = corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{destIp, destHost},
					},
				}
				srcNodeIp = corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{srcIp},
					},
				}
				destNodeIp = corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{destIp},
					},
				}
			})

			Context("migration template with correct parameters", func() {
				It("should never restart", func() {
					vm := v1.NewMinimalVM("testvm")

					job, err := svc.RenderMigrationJob(vm, &srcNode, &destNode)
					Expect(err).ToNot(HaveOccurred())
					Expect(job.Spec.Template.Spec.RestartPolicy).To(Equal(coreapi.RestartPolicyNever))
				})
				It("should prefer DNS name over IP for source", func() {
					vm := v1.NewMinimalVM("testvm")
					job, err := svc.RenderMigrationJob(vm, &srcNode, &destNodeIp)
					Expect(err).ToNot(HaveOccurred())
					refCommand := []string{
						"virsh", "migrate", "testvm",
						"qemu+tcp://127.0.0.3", "qemu+tcp://src-node.kubevirt.io"}
					Expect(job.Spec.Template.Spec.Containers[0].Command).To(Equal(refCommand))
				})
				It("should prefer DNS name over IP for target", func() {
					vm := v1.NewMinimalVM("testvm")
					job, err := svc.RenderMigrationJob(vm, &srcNodeIp, &destNode)
					Expect(err).ToNot(HaveOccurred())
					refCommand := []string{
						"virsh", "migrate", "testvm",
						"qemu+tcp://dest-node.kubevirt.io", "qemu+tcp://127.0.0.2"}
					Expect(job.Spec.Template.Spec.Containers[0].Command).To(Equal(refCommand))
				})
				It("should use the first address it finds", func() {
					vm := v1.NewMinimalVM("testvm")
					// These are contrived nodes with conflicting addresses, so not
					// defined at the context scope to ensure they're not re-used
					node1 := corev1.Node{
						Status: corev1.NodeStatus{
							Addresses: []corev1.NodeAddress{srcHost, destHost},
						},
					}
					node2 := corev1.Node{
						Status: corev1.NodeStatus{
							Addresses: []corev1.NodeAddress{srcIp, destIp},
						},
					}
					job, err := svc.RenderMigrationJob(vm, &node1, &node2)
					Expect(err).ToNot(HaveOccurred())
					refCommand := []string{
						"virsh", "migrate", "testvm",
						"qemu+tcp://127.0.0.2", "qemu+tcp://src-node.kubevirt.io"}
					Expect(job.Spec.Template.Spec.Containers[0].Command).To(Equal(refCommand))
				})
			})
			Context("migration template with incorrect parameters", func() {
				It("should error on missing source address", func() {
					vm := v1.NewMinimalVM("testvm")
					node1 := corev1.Node{}
					node2 := destNodeIp
					job, err := svc.RenderMigrationJob(vm, &node1, &node2)
					Expect(err).To(HaveOccurred())
					Expect(job).To(BeNil())
				})
				It("should error on missing destination address", func() {
					vm := v1.NewMinimalVM("testvm")
					node1 := srcNodeIp
					node2 := corev1.Node{}
					job, err := svc.RenderMigrationJob(vm, &node1, &node2)
					Expect(err).To(HaveOccurred())
					Expect(job).To(BeNil())
				})
			})
		})
	})

})
