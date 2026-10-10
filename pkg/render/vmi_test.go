package render_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/render"
)

var _ = Describe("NewVMI", func() {
	It("copies name, namespace, owner refs and a stable firmware UUID", func() {
		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace("ns"),
			libvmi.WithName("myvm"),
		))

		vmi := render.NewVMI(vm)

		Expect(vmi.Name).To(Equal("myvm"))
		Expect(vmi.Namespace).To(Equal("ns"))
		Expect(vmi.OwnerReferences).To(HaveLen(1))
		Expect(vmi.OwnerReferences[0].Name).To(Equal(vm.Name))
		Expect(vmi.Spec.Domain.Firmware).NotTo(BeNil())
		Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(render.FirmwareUUID("myvm")))
	})

	It("does not alias the VM template labels", func() {
		vm := libvmi.NewVirtualMachine(libvmi.New(
			libvmi.WithNamespace("ns"),
			libvmi.WithName("labeled"),
			libvmi.WithLabel("keep", "vm"),
		))

		vmi := render.NewVMI(vm)
		vmi.Labels["keep"] = "vmi"

		Expect(vm.Spec.Template.ObjectMeta.Labels["keep"]).To(Equal("vm"))
	})
})

var _ = Describe("AutoAttachInputDevice", func() {
	It("adds a default input when requested and none exist", func() {
		vmi := libvmi.New()
		vmi.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)

		render.AutoAttachInputDevice(vmi)

		Expect(vmi.Spec.Domain.Devices.Inputs).To(ConsistOf(v1.Input{Name: "default-0"}))
	})

	It("does not add an input when one is already present", func() {
		vmi := libvmi.New()
		vmi.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)
		vmi.Spec.Domain.Devices.Inputs = []v1.Input{{Name: "existing"}}

		render.AutoAttachInputDevice(vmi)

		Expect(vmi.Spec.Domain.Devices.Inputs).To(ConsistOf(v1.Input{Name: "existing"}))
	})
})
