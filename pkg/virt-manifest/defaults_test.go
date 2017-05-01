package virt_manifest

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Defaults", func() {
	Context("Default Domains", func() {
		var dom *v1.DomainSpec
		BeforeEach(func() {
			dom = new(v1.DomainSpec)
		})

		It("Shouldn't add graphics devices", func() {
			AddMinimalDomainSpec(dom)
			Expect(len(dom.Devices.Graphics)).To(Equal(0))
		})

		It("Should ensure spice devices have proper defaults", func() {
			dom.Devices.Graphics = []v1.Graphics{v1.Graphics{Type: "spice"}}
			// Ensure there's no listen address info
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal(""))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal(""))

			AddMinimalDomainSpec(dom)
			// Ensure the listen address has been added
			Expect(dom.Devices.Graphics[0].Type).To(Equal("spice"))
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal("address"))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal("0.0.0.0"))
		})

		It("Should ensure spice devices have proper defaults", func() {
			dom.Devices.Graphics = []v1.Graphics{v1.Graphics{Type: "spice",
				Listen: v1.Listen{Type: "address"}}}
			// Ensure there's no listen address info
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal("address"))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal(""))

			AddMinimalDomainSpec(dom)
			// Ensure the listen address has been added
			Expect(dom.Devices.Graphics[0].Type).To(Equal("spice"))
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal("address"))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal("0.0.0.0"))
		})
	})

	Context("Default VMs", func() {
		var vm *v1.VM
		BeforeEach(func() {
			vm = tests.NewRandomVM()
		})

		It("should assign a random name to domain spec", func() {
			vm.Spec.Domain = nil
			AddMinimalVMSpec(vm)
			Expect(vm.Spec.Domain.Name).ToNot(BeNil())
			Expect(vm.ObjectMeta.Name).ToNot(Equal(vm.Spec.Domain.Name))
		})

		It("should create missing domain", func() {
			vm.Spec.Domain = nil
			AddMinimalVMSpec(vm)

			Expect(vm.Spec.Domain).ToNot(BeNil())
		})

		It("should not modify an existing domain", func() {
			vm.Spec.Domain = &v1.DomainSpec{Type: "invalid"}

			AddMinimalVMSpec(vm)
			Expect(vm.Spec.Domain.Type).To(Equal("invalid"))
		})
	})
})

func TestDefaults(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Defaults")
}
