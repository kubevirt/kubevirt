package tests_test

import (
	"encoding/xml"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = Describe("[sig-compute]Controller devices", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("with ephemeral disk", func() {
		DescribeTable("a scsi controller", func(enabled bool) {
			vmi := libvmifact.NewCirros()
			vmi.Spec.Domain.Devices.DisableHotplug = !enabled
			vmi = libclient.RunVMIAndExpectLaunch(vmi, 30)
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
			Entry("should appear if enabled", true),
			Entry("should NOT appear if disabled", false),
		)
	})
})
