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

var _ = Describe("Preferences - Requirement - CPU", func() {
	requirementsChecker := requirements.New()

	DescribeTable("should pass when sufficient resources are provided",
		func(instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
			vmiSpec *v1.VirtualMachineInstanceSpec,
		) {
			conflict, err := requirementsChecker.Check(instancetypeSpec, preferenceSpec, vmiSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(conflict).ToNot(HaveOccurred())
		},
		Entry("by an instance type for vCPUs",
			&v1beta1.VirtualMachineInstancetypeSpec{
				CPU: v1beta1.CPUInstancetype{
					Guest: uint32(2),
				},
			},
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{},
		),
		Entry("by a VM for vCPUs using PreferSockets (default)",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Sockets),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Sockets: uint32(2),
					},
				},
			},
		),
		Entry("by a VM for vCPUs using PreferCores",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Cores),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores: uint32(2),
					},
				},
			},
		),
		Entry("by a VM for vCPUs using PreferThreads",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Threads),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Threads: uint32(2),
					},
				},
			},
		),
		Entry("by a VM for vCPUs using PreferSpread by default across SocketsCores",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Spread),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(6),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores:   uint32(2),
						Sockets: uint32(3),
					},
				},
			},
		),
		Entry("by a VM for vCPUs using PreferSpread across CoresThreads",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Spread),
					SpreadOptions: &v1beta1.SpreadOptions{
						Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
					},
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(6),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Threads: uint32(2),
						Cores:   uint32(3),
					},
				},
			},
		),
		Entry("by a VM for vCPUs using PreferSpread across SocketsCoresThreads",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Spread),
					SpreadOptions: &v1beta1.SpreadOptions{
						Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
					},
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(8),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Threads: uint32(2),
						Cores:   uint32(2),
						Sockets: uint32(2),
					},
				},
			},
		),
		Entry("by a VM for vCPUs using PreferAny",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Any),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(4),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores:   uint32(2),
						Sockets: uint32(2),
						Threads: uint32(1),
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
			Expect(conflicts).To(Equal(expectedConflict))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errSubString))
		},
		Entry("by an instance type for vCPUs",
			&v1beta1.VirtualMachineInstancetypeSpec{
				CPU: v1beta1.CPUInstancetype{
					Guest: uint32(1),
				},
			},
			&v1beta1.VirtualMachinePreferenceSpec{
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			nil,
			conflict.Conflicts{conflict.New("spec", "instancetype")},
			fmt.Sprintf(requirements.InsufficientInstanceTypeCPUResourcesErrorFmt, uint32(1), uint32(2)),
		),
		Entry("by a VM for vCPUs using PreferSockets (default)",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Sockets),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Sockets: uint32(1),
					},
				},
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "cpu", "sockets")},
			fmt.Sprintf(requirements.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(2), "sockets"),
		),
		Entry("by a VM for vCPUs using PreferCores",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Cores),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores: uint32(1),
					},
				},
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "cpu", "cores")},
			fmt.Sprintf(requirements.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(2), "cores"),
		),
		Entry("by a VM for vCPUs using PreferThreads",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Threads),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(2),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Threads: uint32(1),
					},
				},
			},
			conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "cpu", "threads")},
			fmt.Sprintf(requirements.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(2), "threads"),
		),
		Entry("by a VM for vCPUs using PreferSpread by default across SocketsCores",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Spread),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(4),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores:   uint32(1),
						Sockets: uint32(1),
					},
				},
			},
			conflict.Conflicts{
				conflict.New("spec", "template", "spec", "domain", "cpu", "sockets"),
				conflict.New("spec", "template", "spec", "domain", "cpu", "cores"),
			},
			fmt.Sprintf(requirements.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(4), v1beta1.SpreadAcrossSocketsCores),
		),
		Entry("by a VM for vCPUs using PreferSpread across CoresThreads",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					SpreadOptions: &v1beta1.SpreadOptions{
						Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
					},
					PreferredCPUTopology: pointer.P(v1beta1.Spread),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(4),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Cores:   uint32(1),
						Threads: uint32(1),
					},
				},
			},
			conflict.Conflicts{
				conflict.New("spec", "template", "spec", "domain", "cpu", "cores"),
				conflict.New("spec", "template", "spec", "domain", "cpu", "threads"),
			},
			fmt.Sprintf(requirements.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(4), v1beta1.SpreadAcrossCoresThreads),
		),
		Entry("by a VM for vCPUs using PreferSpread across SocketsCoresThreads",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					SpreadOptions: &v1beta1.SpreadOptions{
						Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
					},
					PreferredCPUTopology: pointer.P(v1beta1.Spread),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(4),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Sockets: uint32(1),
						Cores:   uint32(1),
						Threads: uint32(1),
					},
				},
			},
			conflict.Conflicts{
				conflict.New("spec", "template", "spec", "domain", "cpu", "sockets"),
				conflict.New("spec", "template", "spec", "domain", "cpu", "cores"),
				conflict.New("spec", "template", "spec", "domain", "cpu", "threads"),
			},
			fmt.Sprintf(requirements.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(4), v1beta1.SpreadAcrossSocketsCoresThreads),
		),
		Entry("by a VM for vCPUs using PreferAny",
			nil,
			&v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(v1beta1.Any),
				},
				Requirements: &v1beta1.PreferenceRequirements{
					CPU: &v1beta1.CPUPreferenceRequirement{
						Guest: uint32(4),
					},
				},
			},
			&v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Sockets: uint32(2),
						Cores:   uint32(1),
						Threads: uint32(1),
					},
				},
			},
			conflict.Conflicts{
				conflict.New("spec", "template", "spec", "domain", "cpu", "cores"),
				conflict.New("spec", "template", "spec", "domain", "cpu", "sockets"),
				conflict.New("spec", "template", "spec", "domain", "cpu", "threads"),
			},
			fmt.Sprintf(requirements.InsufficientVMCPUResourcesErrorFmt, uint32(2), uint32(4), "cores, sockets and threads"),
		),
	)
})
