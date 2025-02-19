package apply_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Instancetype.Spec.Annotations and Preference.Spec.Annotations", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec

		vmiApplier = apply.NewVMIApplier()
		field      = k8sfield.NewPath("spec", "template", "spec")
	)

	BeforeEach(func() {
		vmi = libvmi.New()

		instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
			Annotations: map[string]string{
				"annotation-1": "1",
				"annotation-2": "2",
			},
		}
	})

	Context("Instancetype.Spec.Annotations", func() {
		It("should apply to VMI", func() {
			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Annotations).To(Equal(instancetypeSpec.Annotations))
		})

		It("should not detect conflict when annotation with the same value already exists", func() {
			vmi.Annotations = instancetypeSpec.Annotations

			Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Annotations).To(Equal(instancetypeSpec.Annotations))
		})

		It("should detect conflict when annotation with different value already exists", func() {
			vmi.Annotations = map[string]string{
				"annotation-1": "conflict",
			}

			conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)
			Expect(conflicts).To(HaveLen(1))
			Expect(conflicts[0].String()).To(Equal("annotations.annotation-1"))
		})
	})
})
