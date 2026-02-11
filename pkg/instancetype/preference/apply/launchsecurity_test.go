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
 *
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

var _ = Describe("Preference.LaunchSecurity", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		field      = k8sfield.NewPath("spec", "template", "spec")
		vmiApplier = apply.NewVMIApplier()

		vmiLaunchSecurity = &virtv1.LaunchSecurity{
			SEV: &virtv1.SEV{},
		}
		preferenceLaunchSecurity = &virtv1.LaunchSecurity{
			SEV: &virtv1.SEV{
				Policy: &virtv1.SEVPolicy{},
			},
		}
		instancetypeLaunchSecurity = &virtv1.LaunchSecurity{
			TDX: &virtv1.TDX{},
		}
	)

	BeforeEach(func() {
		vmi = libvmi.New()
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{}
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{}
	})

	DescribeTable("should apply correct LaunchSecurity",
		func(
			vmiLaunchSecurity *virtv1.LaunchSecurity,
			preferenceLaunchSecurity *virtv1.LaunchSecurity,
			instancetypeLaunchSecurity *virtv1.LaunchSecurity,
			expectError bool,
			expectedLaunchSecurity *virtv1.LaunchSecurity,
		) {
			vmi.Spec.Domain.LaunchSecurity = vmiLaunchSecurity
			preferenceSpec.PreferredLaunchSecurity = preferenceLaunchSecurity
			instancetypeSpec.LaunchSecurity = instancetypeLaunchSecurity

			conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)

			if expectError {
				Expect(conflicts).ToNot(BeEmpty())
				Expect(conflicts[0].String()).To(ContainSubstring("launchSecurity"))
			} else {
				Expect(conflicts).To(BeEmpty())
				if expectedLaunchSecurity == nil {
					Expect(vmi.Spec.Domain.LaunchSecurity).To(BeNil())
				} else {
					Expect(vmi.Spec.Domain.LaunchSecurity).To(Equal(expectedLaunchSecurity))
				}
			}
		},
		Entry("all nil", nil, nil, nil, false, nil),
		Entry("preference set", nil, preferenceLaunchSecurity, nil, false, preferenceLaunchSecurity),
		Entry("vmi set", vmiLaunchSecurity, nil, nil, false, vmiLaunchSecurity),
		Entry("vmi and preference set", vmiLaunchSecurity, preferenceLaunchSecurity, nil, false, vmiLaunchSecurity),
		Entry("instancetype set", nil, nil, instancetypeLaunchSecurity, false, instancetypeLaunchSecurity),
		Entry("instancetype and preference set", nil, preferenceLaunchSecurity, instancetypeLaunchSecurity, false, instancetypeLaunchSecurity),
		Entry("instancetype and vmi set", vmiLaunchSecurity, nil, instancetypeLaunchSecurity, true, nil),
		Entry("all set", vmiLaunchSecurity, preferenceLaunchSecurity, instancetypeLaunchSecurity, true, nil),
	)
})
