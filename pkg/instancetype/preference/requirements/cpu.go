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
 *
 */
package requirements

import (
	"fmt"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	preferenceApply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

const (
	InsufficientInstanceTypeCPUResourcesErrorFmt = "insufficient CPU resources of %d vCPU provided by instance type, preference requires " +
		"%d vCPU"
	InsufficientVMCPUResourcesErrorFmt = "insufficient CPU resources of %d vCPU provided by VirtualMachine, preference " +
		"requires %d vCPU provided as %s"
	NoVMCPUResourcesDefinedErrorFmt = "no CPU resources provided by VirtualMachine, preference requires %d vCPU"
)

func checkCPU(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflict.Conflicts, error) {
	if instancetypeSpec != nil {
		if instancetypeSpec.CPU.Guest < preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{conflict.New("spec", "instancetype")},
				fmt.Errorf(
					InsufficientInstanceTypeCPUResourcesErrorFmt, instancetypeSpec.CPU.Guest, preferenceSpec.Requirements.CPU.Guest)
		}
		return nil, nil
	}

	if vmiSpec.Domain.CPU == nil {
		return conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "cpu")},
			fmt.Errorf(NoVMCPUResourcesDefinedErrorFmt, preferenceSpec.Requirements.CPU.Guest)
	}

	baseConflict := conflict.New("spec", "template", "spec", "domain", "cpu")
	switch preferenceApply.GetPreferredTopology(preferenceSpec) {
	case v1beta1.DeprecatedPreferThreads, v1beta1.Threads:
		if vmiSpec.Domain.CPU.Threads < preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{baseConflict.NewChild("threads")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, vmiSpec.Domain.CPU.Threads, preferenceSpec.Requirements.CPU.Guest, "threads")
		}
	case v1beta1.DeprecatedPreferCores, v1beta1.Cores:
		if vmiSpec.Domain.CPU.Cores < preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{baseConflict.NewChild("cores")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, vmiSpec.Domain.CPU.Cores, preferenceSpec.Requirements.CPU.Guest, "cores")
		}
	case v1beta1.DeprecatedPreferSockets, v1beta1.Sockets:
		if vmiSpec.Domain.CPU.Sockets < preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{baseConflict.NewChild("sockets")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, vmiSpec.Domain.CPU.Sockets, preferenceSpec.Requirements.CPU.Guest, "sockets")
		}
	case v1beta1.DeprecatedPreferSpread, v1beta1.Spread:
		return checkSpread(preferenceSpec, vmiSpec)
	case v1beta1.DeprecatedPreferAny, v1beta1.Any:
		cpuResources := vmiSpec.Domain.CPU.Cores * vmiSpec.Domain.CPU.Sockets * vmiSpec.Domain.CPU.Threads
		if cpuResources < preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{
					baseConflict.NewChild("cores"),
					baseConflict.NewChild("sockets"),
					baseConflict.NewChild("threads"),
				},
				fmt.Errorf(InsufficientVMCPUResourcesErrorFmt,
					cpuResources, preferenceSpec.Requirements.CPU.Guest, "cores, sockets and threads")
		}
	}

	return nil, nil
}

func checkSpread(
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflict.Conflicts, error) {
	var (
		vCPUs     uint32
		conflicts conflict.Conflicts
	)
	baseConflict := conflict.New("spec", "template", "spec", "domain", "cpu")
	_, across := preferenceApply.GetSpreadOptions(preferenceSpec)
	switch across {
	case v1beta1.SpreadAcrossSocketsCores:
		vCPUs = vmiSpec.Domain.CPU.Sockets * vmiSpec.Domain.CPU.Cores
		conflicts = conflict.Conflicts{
			baseConflict.NewChild("sockets"),
			baseConflict.NewChild("cores"),
		}
	case v1beta1.SpreadAcrossCoresThreads:
		vCPUs = vmiSpec.Domain.CPU.Cores * vmiSpec.Domain.CPU.Threads
		conflicts = conflict.Conflicts{
			baseConflict.NewChild("cores"),
			baseConflict.NewChild("threads"),
		}
	case v1beta1.SpreadAcrossSocketsCoresThreads:
		vCPUs = vmiSpec.Domain.CPU.Sockets * vmiSpec.Domain.CPU.Cores * vmiSpec.Domain.CPU.Threads
		conflicts = conflict.Conflicts{
			baseConflict.NewChild("sockets"),
			baseConflict.NewChild("cores"),
			baseConflict.NewChild("threads"),
		}
	}
	if vCPUs < preferenceSpec.Requirements.CPU.Guest {
		return conflicts, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, vCPUs, preferenceSpec.Requirements.CPU.Guest, across)
	}
	return nil, nil
}
