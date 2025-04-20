/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

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

var _ = Describe("Preference.Features", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		field      = k8sfield.NewPath("spec", "template", "spec")
		vmiApplier = apply.NewVMIApplier()
	)

	BeforeEach(func() {
		vmi = libvmi.New()

		spinLockRetries := uint32(32)
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Features: &v1beta1.FeaturePreferences{
				PreferredAcpi: &virtv1.FeatureState{},
				PreferredApic: &virtv1.FeatureAPIC{
					Enabled:        pointer.P(true),
					EndOfInterrupt: false,
				},
				PreferredHyperv: &virtv1.FeatureHyperv{
					Relaxed: &virtv1.FeatureState{},
					VAPIC:   &virtv1.FeatureState{},
					Spinlocks: &virtv1.FeatureSpinlocks{
						Enabled: pointer.P(true),
						Retries: &spinLockRetries,
					},
					VPIndex: &virtv1.FeatureState{},
					Runtime: &virtv1.FeatureState{},
					SyNIC:   &virtv1.FeatureState{},
					SyNICTimer: &virtv1.SyNICTimer{
						Enabled: pointer.P(true),
						Direct:  &virtv1.FeatureState{},
					},
					Reset: &virtv1.FeatureState{},
					VendorID: &virtv1.FeatureVendorID{
						Enabled:  pointer.P(true),
						VendorID: "1234",
					},
					Frequencies:     &virtv1.FeatureState{},
					Reenlightenment: &virtv1.FeatureState{},
					TLBFlush:        &virtv1.FeatureState{},
					IPI:             &virtv1.FeatureState{},
					EVMCS:           &virtv1.FeatureState{},
				},
				PreferredKvm: &virtv1.FeatureKVM{
					Hidden: true,
				},
				PreferredPvspinlock: &virtv1.FeatureState{},
				PreferredSmm:        &virtv1.FeatureState{},
			},
		}
	})

	It("should apply to VMI", func() {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Features.ACPI).To(Equal(*preferenceSpec.Features.PreferredAcpi))
		Expect(vmi.Spec.Domain.Features.APIC).To(HaveValue(Equal(*preferenceSpec.Features.PreferredApic)))
		Expect(vmi.Spec.Domain.Features.Hyperv).To(HaveValue(Equal(*preferenceSpec.Features.PreferredHyperv)))
		Expect(vmi.Spec.Domain.Features.KVM).To(HaveValue(Equal(*preferenceSpec.Features.PreferredKvm)))
		Expect(vmi.Spec.Domain.Features.Pvspinlock).To(HaveValue(Equal(*preferenceSpec.Features.PreferredPvspinlock)))
		Expect(vmi.Spec.Domain.Features.SMM).To(HaveValue(Equal(*preferenceSpec.Features.PreferredSmm)))
	})

	It("should apply when some HyperV features already defined in the VMI", func() {
		vmi.Spec.Domain.Features = &virtv1.Features{
			Hyperv: &virtv1.FeatureHyperv{
				EVMCS: &virtv1.FeatureState{
					Enabled: pointer.P(false),
				},
			},
		}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.Features.Hyperv.EVMCS.Enabled).To(HaveValue(BeFalse()))
	})
})
