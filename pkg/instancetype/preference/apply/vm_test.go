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

package apply

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VM preference applier", func() {
	var preferenceSpec *v1beta1.VirtualMachinePreferenceSpec

	BeforeEach(func() {
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			PreferredSubdomain: pointer.P("preferred-subdomain"),
			Annotations:        map[string]string{"preferred-annotation": "value"},
		}
	})

	It("should return early for nil preferences without dereferencing the VM", func() {
		NewVM().Apply(nil, nil)
	})

	It("should apply preferences to the template while preserving VM annotations", func() {
		vm := &virtv1.VirtualMachine{
			Spec: virtv1.VirtualMachineSpec{Template: &virtv1.VirtualMachineInstanceTemplateSpec{}},
		}
		vm.Annotations = map[string]string{"vm-annotation": "preserved"}
		NewVM().Apply(preferenceSpec, vm)

		Expect(vm.Spec.Template.Spec.Subdomain).To(Equal(*preferenceSpec.PreferredSubdomain))
		Expect(vm.Spec.Template.ObjectMeta.Annotations).To(Equal(preferenceSpec.Annotations))
		Expect(vm.Annotations).To(Equal(map[string]string{"vm-annotation": "preserved"}))
	})

	It("should return early for nil preferences without dereferencing the template", func() {
		NewVM().ApplyToTemplate(nil, nil)
	})

	It("should apply preferences to the template spec and metadata", func() {
		template := &virtv1.VirtualMachineInstanceTemplateSpec{}
		NewVM().ApplyToTemplate(preferenceSpec, template)

		Expect(template.Spec.Subdomain).To(Equal(*preferenceSpec.PreferredSubdomain))
		Expect(template.ObjectMeta.Annotations).To(Equal(preferenceSpec.Annotations))
	})
})
