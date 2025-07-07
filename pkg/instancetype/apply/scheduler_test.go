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

var _ = Describe("instancetype.spec.SchedulerName", func() {
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
			SchedulerName: "ultra-scheduler",
		}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.SchedulerName).To(Equal(instancetypeSpec.SchedulerName))
	})

	It("should be no-op if vmi.Spec.SchedulerName is already set but instancetype.SchedulerName is empty", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{}
		vmi.Spec.SchedulerName = "super-fast-scheduler"

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.SchedulerName).To(Equal("super-fast-scheduler"))
	})

	It("should return a conflict if vmi.Spec.SchedulerName is already set and instancetype.SchedulerName is defined", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			SchedulerName: "ultra-fast-scheduler",
		}
		vmi.Spec.SchedulerName = "slow-scheduler"

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.schedulerName"))
	})
})
