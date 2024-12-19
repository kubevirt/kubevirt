package virtctl_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Root", func() {

	It("returns virtctl", func() {
		Expect(virtctl.GetProgramName("virtctl")).To(BeEquivalentTo("virtctl"))
	})

	It("returns virtctl as default", func() {
		Expect(virtctl.GetProgramName("42")).To(BeEquivalentTo("virtctl"))
	})

	It("returns kubectl", func() {
		Expect(virtctl.GetProgramName("kubectl-virt")).To(BeEquivalentTo("kubectl virt"))
	})

	It("returns oc", func() {
		Expect(virtctl.GetProgramName("oc-virt")).To(BeEquivalentTo("oc virt"))
	})

	DescribeTable("the log verbosity flag should be supported", func(arg string) {
		Expect(clientcmd.NewRepeatableVirtctlCommand(arg)()).To(Succeed())
	},
		Entry("regular flag", "--v=2"),
		Entry("shorthand flag", "-v=2"),
	)

})
