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
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("instancetype.spec.ioThreads", func() {
	const (
		expectedThreadCount     uint32 = 4
		userProvidedThreadCount uint32 = 6
	)

	var (
		applier          = apply.NewVMIApplier()
		field            = k8sfield.NewPath("spec", "template", "spec")
		instancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
			IOThreads: &virtv1.DiskIOThreads{
				SupplementalPoolThreadCount: pointer.P(expectedThreadCount),
			},
		}
	)

	DescribeTable("should apply SupplementalPoolThreadCount when", func(vmi *virtv1.VirtualMachineInstance) {
		Expect(applier.ApplyToVMI(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())
		Expect(vmi.Spec.Domain.IOThreads).ToNot(BeNil())
		Expect(vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount).ToNot(BeNil())
		Expect(vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount).To(HaveValue(Equal(expectedThreadCount)))
	},
		Entry("IOThreads nil within VMI", libvmi.New()),
		Entry("SupplementalPoolThreadCount nil within VMI", libvmi.New(libvmi.WithIOThreads(virtv1.DiskIOThreads{}))),
	)

	It("should not apply SupplementalPoolThreadCount when SupplementalPoolThreadCount provided within VMI", func() {
		vmi := libvmi.New(libvmi.WithIOThreads(virtv1.DiskIOThreads{SupplementalPoolThreadCount: pointer.P(userProvidedThreadCount)}))

		Expect(applier.ApplyToVMI(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)).To(
			ContainElement(conflict.New("spec", "template", "spec", "domain", "ioThreads", "supplementalPoolThreadCount")))
		Expect(vmi.Spec.Domain.IOThreads).ToNot(BeNil())
		Expect(vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount).ToNot(BeNil())
		Expect(vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount).To(HaveValue(Equal(userProvidedThreadCount)))
	})
})
