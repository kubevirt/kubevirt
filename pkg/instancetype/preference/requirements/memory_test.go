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

package requirements_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/requirements"
)

var _ = Describe("Preferences - Requirement - Memory", func() {
	requirementsChecker := requirements.New()

	DescribeTable("should pass when sufficient resources are provided",
		func(instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
			vmiSpec *v1.VirtualMachineInstanceSpec,
		) {
			conflict, err := requirementsChecker.Check(instancetypeSpec, preferenceSpec, vmiSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(conflict).ToNot(HaveOccurred())
		},
		Entry("by an instance type for Memory",
			&v1beta1.VirtualMachineInstancetypeSpec{
				Memory: v1beta1.MemoryInstancetype{
					Guest: resource.MustParse("1Gi"),
				},
			},
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("1Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{},
		),
		Entry("by a VM for Memory",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("1Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						Guest: resource.NewQuantity(1024*1024*1024, resource.BinarySI),
					},
				},
			},
		),
	)

	DescribeTable("should be rejected when insufficient resources are provided",
		func(instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
			vmiSpec *v1.VirtualMachineInstanceSpec, expectedConflict conflict.Conflicts, errSubString string,
		) {
			conflicts, err := requirementsChecker.Check(instancetypeSpec, preferenceSpec, vmiSpec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errSubString))
			Expect(conflicts).To(Equal(expectedConflict))
		},
		Entry("by an instance type for Memory",
			&v1beta1.VirtualMachineInstancetypeSpec{
				Memory: v1beta1.MemoryInstancetype{
					Guest: resource.MustParse("1Gi"),
				},
			},
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("2Gi"),
					},
				},
			},
			nil,
			conflict.Conflicts{conflict.New("spec", "instancetype")},
			fmt.Sprintf(requirements.InsufficientInstanceTypeMemoryResourcesErrorFmt, "1Gi", "2Gi"),
		),
		Entry("by a VM for Memory",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Memory: &v1beta1.MemoryPreferenceRequirement{
						Guest: resource.MustParse("2Gi"),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						Guest: resource.NewQuantity(1024*1024*1024, resource.BinarySI),
					},
				},
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "memory")},
			fmt.Sprintf(requirements.InsufficientVMMemoryResourcesErrorFmt, "1Gi", "2Gi"),
		),
	)
})
