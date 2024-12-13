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
 * Copyright The KubeVirt Authors
 *
 */
package requirements

import (
	"fmt"

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

func (h *Handler) checkCPU() (conflict.Conflicts, error) {
	if h.instancetypeSpec != nil {
		if h.instancetypeSpec.CPU.Guest < h.preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{conflict.New("spec", "instancetype")},
				fmt.Errorf(
					InsufficientInstanceTypeCPUResourcesErrorFmt, h.instancetypeSpec.CPU.Guest, h.preferenceSpec.Requirements.CPU.Guest)
		}
		return nil, nil
	}

	if h.vmiSpec.Domain.CPU == nil {
		return conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "cpu")},
			fmt.Errorf(NoVMCPUResourcesDefinedErrorFmt, h.preferenceSpec.Requirements.CPU.Guest)
	}

	baseConflict := conflict.New("spec", "template", "spec", "domain", "cpu")
	switch preferenceApply.GetPreferredTopology(h.preferenceSpec) {
	case v1beta1.DeprecatedPreferThreads, v1beta1.Threads:
		if h.vmiSpec.Domain.CPU.Threads < h.preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{baseConflict.NewChild("threads")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, h.vmiSpec.Domain.CPU.Threads, h.preferenceSpec.Requirements.CPU.Guest, "threads")
		}
	case v1beta1.DeprecatedPreferCores, v1beta1.Cores:
		if h.vmiSpec.Domain.CPU.Cores < h.preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{baseConflict.NewChild("cores")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, h.vmiSpec.Domain.CPU.Cores, h.preferenceSpec.Requirements.CPU.Guest, "cores")
		}
	case v1beta1.DeprecatedPreferSockets, v1beta1.Sockets:
		if h.vmiSpec.Domain.CPU.Sockets < h.preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{baseConflict.NewChild("sockets")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, h.vmiSpec.Domain.CPU.Sockets, h.preferenceSpec.Requirements.CPU.Guest, "sockets")
		}
	case v1beta1.DeprecatedPreferSpread, v1beta1.Spread:
		return h.checkSpread()
	case v1beta1.DeprecatedPreferAny, v1beta1.Any:
		cpuResources := h.vmiSpec.Domain.CPU.Cores * h.vmiSpec.Domain.CPU.Sockets * h.vmiSpec.Domain.CPU.Threads
		if cpuResources < h.preferenceSpec.Requirements.CPU.Guest {
			return conflict.Conflicts{
					baseConflict.NewChild("cores"),
					baseConflict.NewChild("sockets"),
					baseConflict.NewChild("threads"),
				},
				fmt.Errorf(InsufficientVMCPUResourcesErrorFmt,
					cpuResources, h.preferenceSpec.Requirements.CPU.Guest, "cores, sockets and threads")
		}
	}

	return nil, nil
}

func (h *Handler) checkSpread() (conflict.Conflicts, error) {
	var (
		vCPUs     uint32
		conflicts conflict.Conflicts
	)
	baseConflict := conflict.New("spec", "template", "spec", "domain", "cpu")
	_, across := preferenceApply.GetSpreadOptions(h.preferenceSpec)
	switch across {
	case v1beta1.SpreadAcrossSocketsCores:
		vCPUs = h.vmiSpec.Domain.CPU.Sockets * h.vmiSpec.Domain.CPU.Cores
		conflicts = conflict.Conflicts{
			baseConflict.NewChild("sockets"),
			baseConflict.NewChild("cores"),
		}
	case v1beta1.SpreadAcrossCoresThreads:
		vCPUs = h.vmiSpec.Domain.CPU.Cores * h.vmiSpec.Domain.CPU.Threads
		conflicts = conflict.Conflicts{
			baseConflict.NewChild("cores"),
			baseConflict.NewChild("threads"),
		}
	case v1beta1.SpreadAcrossSocketsCoresThreads:
		vCPUs = h.vmiSpec.Domain.CPU.Sockets * h.vmiSpec.Domain.CPU.Cores * h.vmiSpec.Domain.CPU.Threads
		conflicts = conflict.Conflicts{
			baseConflict.NewChild("sockets"),
			baseConflict.NewChild("cores"),
			baseConflict.NewChild("threads"),
		}
	}
	if vCPUs < h.preferenceSpec.Requirements.CPU.Guest {
		return conflicts, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, vCPUs, h.preferenceSpec.Requirements.CPU.Guest, across)
	}
	return nil, nil
}
