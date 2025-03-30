package tests_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe("[sig-compute]Controller devices", decorators.SigCompute, func() {
	Context("with ephemeral disk", func() {
		DescribeTable("a scsi controller", func(enabled bool) {
			vmi := libvmifact.NewCirros()
			vmi.Spec.Domain.Devices.DisableHotplug = !enabled
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 30)
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
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
