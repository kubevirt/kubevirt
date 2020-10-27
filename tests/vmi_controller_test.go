package tests_test

import (
	"encoding/xml"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("Controller devices", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)
	})

	Context("with ephemeral disk", func() {
		It("Should create a valid VMI and appropriate libvirt domain, with scsi controller", func() {
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "scsi",
				Index: "0",
			})
			vmi, apiErr := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
			Expect(len(domSpec.Devices.Controllers)).To(Equal(11))
			found := false
			for _, controller := range domSpec.Devices.Controllers {
				if controller.Type == "scsi" {
					found = true
					Expect(controller.Index).To(Equal("0"))
					Expect(controller.Model).To(Equal("virtio-scsi"))
				}
			}
			Expect(found).To(BeTrue(), "Did not find virtio-scsi controller in domain.xml")
		})

		It("Should create a valid VMI and appropriate libvirt domain, with pci controller", func() {
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "pci",
				Index: "20",
			})
			vmi, apiErr := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
			Expect(len(domSpec.Devices.Controllers)).To(Equal(24))
			found := false
			for _, controller := range domSpec.Devices.Controllers {
				if controller.Type == "pci" && controller.Index == "20" {
					found = true
					Expect(controller.Model).To(Equal("pcie-root-port"))
				}
			}
			Expect(found).To(BeTrue(), "Did not find pcie-root-port at appropriate index in domain.xml")
		})

		It("Should create a valid VMI and appropriate libvirt domain, with pci controller and scsi controller", func() {
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "pci",
				Index: "20",
			})
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "scsi",
				Index: "0",
			})
			vmi, apiErr := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
			Expect(len(domSpec.Devices.Controllers)).To(Equal(25))
			foundPci := false
			foundScsi := false
			for _, controller := range domSpec.Devices.Controllers {
				if controller.Type == "pci" && controller.Index == "20" {
					foundPci = true
					Expect(controller.Model).To(Equal("pcie-root-port"))
				}
				if controller.Type == "scsi" {
					foundScsi = true
					Expect(controller.Index).To(Equal("0"))
					Expect(controller.Model).To(Equal("virtio-scsi"))
				}
			}
			Expect(foundScsi).To(BeTrue(), "Did not find virtio-scsi controller in domain.xml")
			Expect(foundPci).To(BeTrue(), "Did not find pcie-root-port at appropriate index in domain.xml")
		})

		It("Should create a valid VMI and appropriate libvirt domain, with multiple pci controllers", func() {
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "pci",
				Index: "20",
			})
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "pci",
				Index: "21",
			})
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "pci",
				Index: "22",
			})
			vmi, apiErr := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
			Expect(len(domSpec.Devices.Controllers)).To(Equal(26))
			found1 := false
			found2 := false
			found3 := false
			for _, controller := range domSpec.Devices.Controllers {
				if controller.Type == "pci" && controller.Index == "20" {
					found1 = true
					Expect(controller.Model).To(Equal("pcie-root-port"))
				}
				if controller.Type == "pci" && controller.Index == "21" {
					found2 = true
					Expect(controller.Model).To(Equal("pcie-root-port"))
				}
				if controller.Type == "pci" && controller.Index == "22" {
					found3 = true
					Expect(controller.Model).To(Equal("pcie-root-port"))
				}
			}
			Expect(found1).To(BeTrue(), "Did not find pcie-root-port at appropriate index in domain.xml")
			Expect(found2).To(BeTrue(), "Did not find pcie-root-port at appropriate index in domain.xml")
			Expect(found3).To(BeTrue(), "Did not find pcie-root-port at appropriate index in domain.xml")
		})

		It("Should not sync domain xml, with pci type, and index=0", func() {
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			randomVMI.Spec.Domain.Devices.Controllers = append(randomVMI.Spec.Domain.Devices.Controllers, v1.Controller{
				Type:  "pci",
				Index: "0",
			})
			vmi, apiErr := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			// Verify the VMI is not started, the Synchronized condition should be false
			vmi, apiErr = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(randomVMI.Name, &metav1.GetOptions{})
			Expect(apiErr).ToNot(HaveOccurred())
			for _, c := range vmi.Status.Conditions {
				if c.Type == v1.VirtualMachineInstanceSynchronized {
					Expect(c.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(c.Reason).To(Equal("Synchronizing with the Domain failed."))
					Expect(c.Message).To(ContainSubstring("The PCI controller with index=''0'' must be model=''pcie-root''"))
				}
			}
		})
	})
})
