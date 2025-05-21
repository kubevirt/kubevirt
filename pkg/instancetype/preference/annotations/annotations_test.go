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

package annotations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/annotations"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Preferences - annotations", func() {
	Context("set annotations", func() {
		const preferenceName = "preference-name"

		var (
			vm   *v1.VirtualMachine
			meta *metav1.ObjectMeta
		)

		BeforeEach(func() {
			vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithPreference(preferenceName))

			meta = &metav1.ObjectMeta{}
		})

		It("should add preference name annotation", func() {
			vm.Spec.Preference.Kind = apiinstancetype.SingularPreferenceResourceName

			annotations.Set(vm, meta)

			Expect(meta.Annotations).To(HaveKeyWithValue(v1.PreferenceAnnotation, preferenceName))
			Expect(meta.Annotations).ToNot(HaveKey(v1.ClusterPreferenceAnnotation))
		})

		It("should add cluster preference name annotation", func() {
			vm.Spec.Preference.Kind = apiinstancetype.ClusterSingularPreferenceResourceName

			annotations.Set(vm, meta)

			Expect(meta.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
			Expect(meta.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, preferenceName))
		})

		It("should add cluster name annotation, if preference.kind is empty", func() {
			vm.Spec.Preference.Kind = ""

			annotations.Set(vm, meta)

			Expect(meta.Annotations).ToNot(HaveKey(v1.PreferenceAnnotation))
			Expect(meta.Annotations).To(HaveKeyWithValue(v1.ClusterPreferenceAnnotation, preferenceName))
		})
	})

	Context("apply to vmi", func() {
		var (
			vmi            *v1.VirtualMachineInstance
			preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec

			vmiApplier = apply.NewVMIApplier()
			field      = k8sfield.NewPath("spec", "template", "spec")
		)

		BeforeEach(func() {
			vmi = libvmi.New()

			preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
				Annotations: map[string]string{
					"annotation-1": "1",
					"annotation-2": "2",
				},
			}
		})

		It("should apply to VMI", func() {
			Expect(vmiApplier.ApplyToVMI(field, nil, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Annotations).To(Equal(preferenceSpec.Annotations))
		})

		It("should not overwrite already existing values", func() {
			vmiAnnotations := map[string]string{
				"annotation-1": "dont-overwrite",
				"annotation-2": "dont-overwrite",
				"annotation-3": "dont-overwrite",
			}
			vmi.Annotations = vmiAnnotations

			Expect(vmiApplier.ApplyToVMI(field, nil, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
			Expect(vmi.Annotations).To(HaveLen(3))
			Expect(vmi.Annotations).To(Equal(vmiAnnotations))
		})
	})
})
