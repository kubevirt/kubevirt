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

var _ = Describe("Preference - Apply to vmi - subdomain", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		field      = k8sfield.NewPath("spec", "template", "spec")
		vmiApplier = apply.NewVMIApplier()
	)

	BeforeEach(func() {
		vmi = libvmi.New()
	})

	It("should apply to VMI", func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			PreferredSubdomain: pointer.P("kubevirt.io"),
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Subdomain).To(Equal(*preferenceSpec.PreferredSubdomain))
	})

	It("should not overwrite user defined value", func() {
		const userDefinedValue = "foo.com"
		vmi.Spec.Subdomain = userDefinedValue
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			PreferredSubdomain: pointer.P("kubevirt.io"),
		}
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Subdomain).To(Equal(userDefinedValue))
	})
})
