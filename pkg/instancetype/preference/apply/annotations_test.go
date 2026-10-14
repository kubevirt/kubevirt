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

var _ = Describe("Preference.Annotations", func() {
	var (
		vmi            *virtv1.VirtualMachineInstance
		preferenceSpec *v1beta1.VirtualMachinePreferenceSpec

		vmiApplier = apply.NewVMIApplier()
		field      = k8sfield.NewPath("spec", "template", "spec")
	)

	BeforeEach(func() {
		vmi = libvmi.New()
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			Annotations: map[string]string{
				"annotation-1": "1",
				"annotation-2": "2",
				"annotation-3": "3",
			},
		}
	})

	It("should apply to VMI", func() {
		Expect(vmiApplier.ApplyToVMI(field, nil, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Annotations).To(Equal(preferenceSpec.Annotations))
	})

	It("should fill missing annotations while preserving existing values, including empty strings", func() {
		vmi.Annotations = map[string]string{
			"annotation-1":    "user-value",
			"annotation-2":    "",
			"user-annotation": "preserved",
		}

		Expect(vmiApplier.ApplyToVMI(field, nil, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Annotations).To(Equal(map[string]string{
			"annotation-1":    "user-value",
			"annotation-2":    "",
			"annotation-3":    "3",
			"user-annotation": "preserved",
		}))
	})

	It("should preserve annotations applied by the instancetype", func() {
		instancetypeSpec := &v1beta1.VirtualMachineInstancetypeSpec{
			Annotations: map[string]string{
				"annotation-1": "instancetype-value",
			},
		}

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Annotations).To(Equal(map[string]string{
			"annotation-1": "instancetype-value",
			"annotation-2": "2",
			"annotation-3": "3",
		}))
	})
})
