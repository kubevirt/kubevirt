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

var _ = Describe("Preference - Apply to vmi - clock", func() {
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
			Clock: &v1beta1.ClockPreferences{
				PreferredClockOffset: &virtv1.ClockOffset{
					UTC: &virtv1.ClockOffsetUTC{
						OffsetSeconds: pointer.P(30),
					},
				},
				PreferredTimer: &virtv1.Timer{
					Hyperv: &virtv1.HypervTimer{},
				},
			},
		}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Clock.ClockOffset).To(Equal(*preferenceSpec.Clock.PreferredClockOffset))
		Expect(vmi.Spec.Domain.Clock.Timer).To(HaveValue(Equal(*preferenceSpec.Clock.PreferredTimer)))
	})
})
