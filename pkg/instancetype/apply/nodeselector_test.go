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

var _ = Describe("Apply to vmi - node selector", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		vmiApplier = apply.NewVMIApplier()
		field      = k8sfield.NewPath("spec", "template", "spec")
	)

	BeforeEach(func() {
		vmi = libvmi.New()
	})

	It("should apply to VMI", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			NodeSelector: map[string]string{"key": "value"},
		}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.NodeSelector).To(Equal(instancetypeSpec.NodeSelector))
	})

	It("should be no-op if vmi.Spec.NodeSelector is already set but instancetype.NodeSelector is empty", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{}
		vmi.Spec.NodeSelector = map[string]string{"key": "value"}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.NodeSelector).To(Equal(map[string]string{"key": "value"}))
	})

	It("should return a conflict if vmi.Spec.NodeSelector is already set and instancetype.NodeSelector is defined", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			NodeSelector: map[string]string{"key": "value"},
		}
		vmi.Spec.NodeSelector = map[string]string{"key": "value"}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.nodeSelector"))
	})
})
