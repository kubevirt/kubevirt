package virtctl_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var _ = Describe("Root", func() {

	It("returns virtctl", func() {
		Expect(templates.GetProgramName("virtctl")).To(BeEquivalentTo("virtctl"))
	})

	It("returns virtctl as default", func() {
		Expect(templates.GetProgramName("42")).To(BeEquivalentTo("virtctl"))
	})

	It("returns kubectl", func() {
		Expect(templates.GetProgramName("kubectl-virt")).To(BeEquivalentTo("kubectl virt"))
	})

	It("returns oc", func() {
		Expect(templates.GetProgramName("oc-virt")).To(BeEquivalentTo("oc virt"))
	})

})
