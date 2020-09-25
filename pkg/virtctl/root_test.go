package virtctl_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl"
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

})
