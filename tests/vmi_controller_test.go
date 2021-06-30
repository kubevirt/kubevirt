package tests_test

import (
	"encoding/xml"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[sig-compute]Controller devices", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("with ephemeral disk", func() {
		table.DescribeTable("a scsi controller", func(enabled bool) {
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			randomVMI.Spec.Domain.Devices.DisableHotplug = !enabled
			vmi, apiErr := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())
			found := false
			for _, controller := range domSpec.Devices.Controllers {
				if controller.Type == "scsi" {
					found = true
					Expect(controller.Index).To(Equal("0"))
					Expect(controller.Model).To(Equal("virtio-non-transitional"))
				}
			}
			Expect(found).To(Equal(enabled))
		},
			table.Entry("should appear if enabled", true),
			table.Entry("should NOT appear if disabled", false),
		)
	})
})
