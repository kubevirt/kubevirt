//nolint:dupl
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

var _ = Describe("Apply to vmi - gpu", func() {
	var (
		vmi            *virtv1.VirtualMachineInstance
		preferenceSpec *v1beta1.VirtualMachinePreferenceSpec

		vmiApplier       = apply.NewVMIApplier()
		field            = k8sfield.NewPath("spec", "template", "spec")
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			GPUs: []virtv1.GPU{
				{
					Name:       "barfoo",
					DeviceName: "vendor.com/gpu_name",
				},
			},
		}
	)

	BeforeEach(func() {
		vmi = libvmi.New()
	})

	It("should apply to VMI", func() {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Devices.GPUs).To(Equal(instancetypeSpec.GPUs))
	})

	It("should detect GPU conflict", func() {
		vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
			{
				Name:       "foobar",
				DeviceName: "vendor.com/gpu_name",
			},
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.devices.gpus"))
	})
})
