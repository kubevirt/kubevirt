package apply_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("instancetype.Spec.ioThreadsPolicy", func() {
	var (
		vmi            *virtv1.VirtualMachineInstance
		preferenceSpec *v1beta1.VirtualMachinePreferenceSpec

		vmiApplier         = apply.NewVMIApplier()
		field              = k8sfield.NewPath("spec", "template", "spec")
		instancetypePolicy = virtv1.IOThreadsPolicyShared
		instancetypeSpec   = &v1beta1.VirtualMachineInstancetypeSpec{
			IOThreadsPolicy: &instancetypePolicy,
		}
	)

	BeforeEach(func() {
		vmi = libvmi.New()
	})

	It("should apply to VMI", func() {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.IOThreadsPolicy).To(HaveValue(Equal(*instancetypeSpec.IOThreadsPolicy)))
	})

	It("should detect IOThreadsPolicy conflict", func() {
		vmi.Spec.Domain.IOThreadsPolicy = &instancetypePolicy

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.ioThreadsPolicy"))
	})
})
