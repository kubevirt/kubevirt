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
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Instancetype.Spec.Annotations", func() {
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
