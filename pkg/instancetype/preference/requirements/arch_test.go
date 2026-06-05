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

	v1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/requirements"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Architecture requirements", func() {
	requirementsChecker := requirements.New()

	DescribeTable("should pass when architecture requirement is met",
		func(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) {
			conflicts, err := requirementsChecker.Check(nil, preferenceSpec, vmiSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(conflicts).ToNot(HaveOccurred())
		},
		Entry("when preference requires amd64 and vmi uses amd64",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Architecture: pointer.P("amd64"),
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "amd64",
			},
		),
		Entry("when preference requires arm64 and vmi uses arm64",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Architecture: pointer.P("arm64"),
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "arm64",
			},
		),
		Entry("when preference requires s390x and vmi uses s390x",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Architecture: pointer.P("s390x"),
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "s390x",
			},
		),
		Entry("when preference has no architecture requirement",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "amd64",
			},
		),
		Entry("when preference has no requirements at all",
			&v1beta1.VirtualMachinePreferenceSpec{},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "amd64",
			},
		),
	)

	DescribeTable("should be rejected when architecture requirement is not met",
		func(
			preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
			vmiSpec *v1.VirtualMachineInstanceSpec,
			expectedConflict conflict.Conflicts,
			errSubString string,
		) {
			conflicts, err := requirementsChecker.Check(nil, preferenceSpec, vmiSpec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errSubString))
			Expect(conflicts).To(Equal(expectedConflict))
		},
		Entry("when preference requires amd64 but vmi uses arm64",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Architecture: pointer.P("amd64"),
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "arm64",
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "architecture")},
			fmt.Sprintf("preference requires architecture %s but %s is being requested", "amd64", "arm64"),
		),
		Entry("when preference requires arm64 but vmi uses amd64",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Architecture: pointer.P("arm64"),
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "amd64",
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "architecture")},
			fmt.Sprintf("preference requires architecture %s but %s is being requested", "arm64", "amd64"),
		),
		Entry("when preference requires s390x but vmi uses amd64",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Architecture: pointer.P("s390x"),
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "amd64",
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "architecture")},
			fmt.Sprintf("preference requires architecture %s but %s is being requested", "s390x", "amd64"),
		),
		Entry("when preference requires amd64 but vmi uses empty architecture",
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					Architecture: pointer.P("amd64"),
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Architecture: "",
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "architecture")},
			fmt.Sprintf("preference requires architecture %s but %s is being requested", "amd64", ""),
		),
	)
})
