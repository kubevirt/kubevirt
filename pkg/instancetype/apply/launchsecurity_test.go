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
)

var _ = Describe("instancetype.Spec.LaunchSecurity", func() {
	var (
		vmi            *virtv1.VirtualMachineInstance
		preferenceSpec *v1beta1.VirtualMachinePreferenceSpec

		vmiApplier       = apply.NewVMIApplier()
		field            = k8sfield.NewPath("spec", "template", "spec")
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			LaunchSecurity: &virtv1.LaunchSecurity{
				SEV: &virtv1.SEV{},
			},
		}
	)

	BeforeEach(func() {
		vmi = libvmi.New()
	})

	It("should apply to VMI", func() {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.LaunchSecurity).To(HaveValue(Equal(*instancetypeSpec.LaunchSecurity)))
	})

	It("should detect LaunchSecurity conflict", func() {
		vmi.Spec.Domain.LaunchSecurity = &virtv1.LaunchSecurity{
			SEV: &virtv1.SEV{},
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.launchSecurity"))
	})
})
