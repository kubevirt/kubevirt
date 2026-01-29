package vgpuhook

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Premigration Hook Server", func() {
	Context("vGPU Dedicated Hook", func() {
		It("should update vGPU mdev uuid according to target node config", func() {
			targetUUID := "05b59010-d19c-47d2-9477-33b4579edc90"
			domXML :=
				`<domain type="kvm" id="1">
				<name>kubevirt</name>
				<devices>
					<hostdev mode="subsystem" type="mdev" managed="no" model="vfio-pci" display="on" ramfb="on">
						<source>
							<address uuid="bb4a98d8-60c1-40c6-b39b-866b1e82bd8c"/>
						</source>
						<alias name="ua-gpu-gpu1"/>
						<address type="pci" domain="0x0000" bus="0x09" slot="0x00" function="0x0"/>
					</hostdev>
				</devices>
			</domain>`
			expectedXML := fmt.Sprintf(
				`<domain type="kvm" id="1">
				<name>kubevirt</name>
				<devices>
					<hostdev mode="subsystem" type="mdev" managed="no" model="vfio-pci" display="on" ramfb="on">
						<source>
							<address uuid="%s"/>
						</source>
						<alias name="ua-gpu-gpu1"/>
						<address type="pci" domain="0x0000" bus="0x09" slot="0x00" function="0x0"/>
					</hostdev>
				</devices>
			</domain>`, targetUUID)

			By("creating a VMI with a single vGPU")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "kubevirt",
					Namespace:   "testns",
					Annotations: map[string]string{},
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								DeviceName: "nvidia.com/test",
								Name:       "gpu1",
							}},
						},
					},
				},
			}

			By("annotating the vmi with the target mdev uuid")
			vmi.Annotations["kubevirt.io/target-mdev-uuid"] = targetUUID

			By("parsing the input domain XML")
			var domain libvirtxml.Domain
			err := domain.Unmarshal(domXML)
			Expect(err).NotTo(HaveOccurred(), "failed to parse input domain XML")

			By("running the vGPU dedicated hook")
			err = VGPUDedicatedHook(vmi, &domain)
			Expect(err).NotTo(HaveOccurred(), "failed to modify domain")

			By("marshaling the modified domain back to XML")
			newXML, err := domain.Marshal()
			Expect(err).NotTo(HaveOccurred(), "failed to marshal modified domain")

			By("ensuring the generated XML is accurate")
			Expect(newXML).To(MatchXML(expectedXML), "the target XML is not as expected")
		})

		It("should fail if there is more than 1 vGPU", func() {
			targetUUID := "3ad93aea-4e81-49a9-b1af-9d020373a357"
			domXML :=
				`<domain type="kvm" id="1">
				<name>kubevirt</name>
				<devices>
					<hostdev mode="subsystem" type="mdev" managed="no" model="vfio-pci" display="on" ramfb="on">
						<source>
							<address uuid="bb4a98d8-60c1-40c6-b39b-866b1e82bd8c"/>
						</source>
						<alias name="ua-gpu-gpu1"/>
						<address type="pci" domain="0x0000" bus="0x09" slot="0x00" function="0x0"/>
					</hostdev>
					<hostdev mode="subsystem" type="mdev" managed="no" model="vfio-pci" display="on" ramfb="on">
						<source>
							<address uuid="19dcdf19-0ef0-496b-be3b-c591109ca572"/>
						</source>
						<alias name="ua-gpu-gpu2"/>
						<address type="pci" domain="0x0000" bus="0x09" slot="0x01" function="0x0"/>
					</hostdev>
				</devices>
			</domain>`

			By("creating a VMI with 2 vGPUs")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "kubevirt",
					Namespace:   "testns",
					Annotations: map[string]string{},
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{
									DeviceName: "nvidia.com/test",
									Name:       "gpu1",
								},
								{
									DeviceName: "nvidia.com/test",
									Name:       "gpu2",
								},
							},
						},
					},
				},
			}

			By("annotating the vmi with the target mdev uuid")
			vmi.Annotations["kubevirt.io/target-mdev-uuid"] = targetUUID

			By("parsing the input domain XML")
			var domain libvirtxml.Domain
			err := domain.Unmarshal(domXML)
			Expect(err).NotTo(HaveOccurred(), "failed to parse input domain XML")

			By("running the vGPU dedicated hook")
			err = VGPUDedicatedHook(vmi, &domain)
			Expect(err).To(MatchError("the migrating vmi can only have one vGPU"))
		})
	})
})
