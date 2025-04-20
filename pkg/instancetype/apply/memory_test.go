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

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("instancetype.Spec.Memory", func() {
	var (
		vmi              *virtv1.VirtualMachineInstance
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec
		preferenceSpec   *v1beta1.VirtualMachinePreferenceSpec

		vmiApplier = apply.NewVMIApplier()
		field      = k8sfield.NewPath("spec", "template", "spec")
		maxGuest   = resource.MustParse("2G")
	)

	BeforeEach(func() {
		vmi = libvmi.New()

		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			Memory: v1beta1.MemoryInstancetype{
				Guest: resource.MustParse("512M"),
				Hugepages: &virtv1.Hugepages{
					PageSize: "1Gi",
				},
				MaxGuest: &maxGuest,
			},
		}
	})

	It("should apply to VMI", func() {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		Expect(vmi.Spec.Domain.Memory.Guest).To(HaveValue(Equal(instancetypeSpec.Memory.Guest)))
		Expect(vmi.Spec.Domain.Memory.Hugepages).To(HaveValue(Equal(*instancetypeSpec.Memory.Hugepages)))
		Expect(vmi.Spec.Domain.Memory.MaxGuest.Equal(*instancetypeSpec.Memory.MaxGuest)).To(BeTrue())
	})

	It("should apply memory overcommit correctly to VMI", func() {
		instancetypeSpec.Memory.Hugepages = nil
		instancetypeSpec.Memory.OvercommitPercent = 15

		expectedOverhead := int64(float32(instancetypeSpec.Memory.Guest.Value()) * (1 - float32(instancetypeSpec.Memory.OvercommitPercent)/100))
		Expect(expectedOverhead).ToNot(Equal(instancetypeSpec.Memory.Guest.Value()))

		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
		Expect(memRequest.Value()).To(Equal(expectedOverhead))
	})

	It("should detect memory conflict", func() {
		vmiMemGuest := resource.MustParse("512M")
		vmi.Spec.Domain.Memory = &virtv1.Memory{
			Guest: &vmiMemGuest,
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.memory"))
	})

	It("should return a conflict if vmi.Spec.Domain.Resources.Requests[k8svirtv1.ResourceMemory] already defined", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			Memory: v1beta1.MemoryInstancetype{
				Guest: resource.MustParse("512M"),
			},
		}

		vmi.Spec.Domain.Resources = virtv1.ResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("128Mi"),
			},
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.requests.memory"))
	})

	It("should return a conflict if vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] already defined", func() {
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			Memory: v1beta1.MemoryInstancetype{
				Guest: resource.MustParse("512M"),
			},
		}

		vmi.Spec.Domain.Resources = virtv1.ResourceRequirements{
			Limits: k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("128Mi"),
			},
		}

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.limits.memory"))
	})
})
