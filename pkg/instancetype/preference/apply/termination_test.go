package apply_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Preference.PreferredTerminationGracePeriodSeconds", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		field      = k8sfield.NewPath("spec", "template", "spec")
		vmiApplier = apply.NewVMIApplier()
	)

	BeforeEach(func() {
		vmi = libvmi.New()
		// delete spec.TerminationGracePeriodSeconds set in VM
		vmi.Spec.TerminationGracePeriodSeconds = nil
	})

	It("should apply to VMI", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			PreferredTerminationGracePeriodSeconds: pointer.P(int64(180)),
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.TerminationGracePeriodSeconds).To(HaveValue(Equal(*preferenceSpec.PreferredTerminationGracePeriodSeconds)))
	})

	It("should not overwrite user defined value", func() {
		const userDefinedValue = int64(100)
		vmi.Spec.TerminationGracePeriodSeconds = pointer.P(userDefinedValue)
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			PreferredTerminationGracePeriodSeconds: pointer.P(int64(180)),
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.TerminationGracePeriodSeconds).To(HaveValue(Equal(userDefinedValue)))
	})
})
