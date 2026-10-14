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

	"k8s.io/apimachinery/pkg/api/resource"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("ApplyToTemplate", func() {
	var (
		template         *virtv1.VirtualMachineInstanceTemplateSpec
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		applier = apply.NewVMIApplier()
		field   = k8sfield.NewPath("spec", "template", "spec")
	)

	BeforeEach(func() {
		template = &virtv1.VirtualMachineInstanceTemplateSpec{}
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			CPU:         v1beta1.CPUInstancetype{Guest: 2},
			Memory:      v1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
			Annotations: map[string]string{"shared": "instancetype"},
		}
		preferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
			CPU:                &v1beta1.CPUPreferences{PreferredCPUTopology: pointer.P(v1beta1.Cores)},
			PreferredSubdomain: pointer.P("preferred-subdomain"),
			Annotations:        map[string]string{"shared": "preference", "preferred": "value"},
		}
	})

	It("should apply the instancetype before preferences to the original template", func() {
		Expect(applier.ApplyToTemplate(field, instancetypeSpec, preferenceSpec, template)).To(Succeed())
		Expect(template.Spec.Domain.CPU.Cores).To(Equal(uint32(2)))
		Expect(template.Spec.Subdomain).To(Equal(*preferenceSpec.PreferredSubdomain))
		Expect(template.ObjectMeta.Annotations).To(Equal(map[string]string{"shared": "instancetype", "preferred": "value"}))
	})

	It("should apply preferences without an instancetype", func() {
		Expect(applier.ApplyToTemplate(field, nil, preferenceSpec, template)).To(Succeed())
		Expect(template.Spec.Subdomain).To(Equal(*preferenceSpec.PreferredSubdomain))
		Expect(template.ObjectMeta.Annotations).To(Equal(preferenceSpec.Annotations))
	})

	It("should return instancetype conflicts before applying preferences", func() {
		template.Spec.Domain.CPU = &virtv1.CPU{Cores: 1}
		conflicts := applier.ApplyToTemplate(field, instancetypeSpec, preferenceSpec, template)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.cpu.cores"))
		Expect(template.Spec.Subdomain).To(BeEmpty())
		Expect(template.ObjectMeta.Annotations).To(Equal(instancetypeSpec.Annotations))
	})

	It("should return early for nil specs without dereferencing the target", func() {
		Expect(applier.ApplyToTemplate(field, nil, nil, nil)).To(Succeed())
		Expect(apply.NewVMIApplier().ApplyToVMI(field, nil, nil, nil)).To(Succeed())
	})
})
