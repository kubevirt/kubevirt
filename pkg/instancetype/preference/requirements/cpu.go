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
 * Copyright 2024 Red Hat, Inc.
 *
 */
package requirements

import (
	"fmt"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	preferenceApply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

const (
	InsufficientInstanceTypeCPUResourcesErrorFmt = "insufficient CPU resources of %d vCPU provided by instance type, preference requires " +
		"%d vCPU"
	InsufficientVMCPUResourcesErrorFmt = "insufficient CPU resources of %d vCPU provided by VirtualMachine, preference " +
		"requires %d vCPU provided as %s"
	NoVMCPUResourcesDefinedErrorFmt = "no CPU resources provided by VirtualMachine, preference requires %d vCPU"
)

func (h *Handler) checkCPU() (apply.Conflicts, error) {
	if h.instancetypeSpec != nil {
		if h.instancetypeSpec.CPU.Guest < h.preferenceSpec.Requirements.CPU.Guest {
			return apply.Conflicts{k8sfield.NewPath("spec", "instancetype")},
				fmt.Errorf(
					InsufficientInstanceTypeCPUResourcesErrorFmt, h.instancetypeSpec.CPU.Guest, h.preferenceSpec.Requirements.CPU.Guest)
		}
		return nil, nil
	}

	cpuField := k8sfield.NewPath("spec", "template", "spec", "domain", "cpu")
	if h.vmiSpec.Domain.CPU == nil {
		return apply.Conflicts{cpuField}, fmt.Errorf(NoVMCPUResourcesDefinedErrorFmt, h.preferenceSpec.Requirements.CPU.Guest)
	}

	switch preferenceApply.GetPreferredTopology(h.preferenceSpec) {
	case v1beta1.DeprecatedPreferThreads, v1beta1.Threads:
		if h.vmiSpec.Domain.CPU.Threads < h.preferenceSpec.Requirements.CPU.Guest {
			return apply.Conflicts{cpuField.Child("threads")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, h.vmiSpec.Domain.CPU.Threads, h.preferenceSpec.Requirements.CPU.Guest, "threads")
		}
	case v1beta1.DeprecatedPreferCores, v1beta1.Cores:
		if h.vmiSpec.Domain.CPU.Cores < h.preferenceSpec.Requirements.CPU.Guest {
			return apply.Conflicts{cpuField.Child("cores")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, h.vmiSpec.Domain.CPU.Cores, h.preferenceSpec.Requirements.CPU.Guest, "cores")
		}
	case v1beta1.DeprecatedPreferSockets, v1beta1.Sockets:
		if h.vmiSpec.Domain.CPU.Sockets < h.preferenceSpec.Requirements.CPU.Guest {
			return apply.Conflicts{cpuField.Child("sockets")},
				fmt.Errorf(
					InsufficientVMCPUResourcesErrorFmt, h.vmiSpec.Domain.CPU.Sockets, h.preferenceSpec.Requirements.CPU.Guest, "sockets")
		}
	case v1beta1.DeprecatedPreferSpread, v1beta1.Spread:
		return h.checkSpread()
	case v1beta1.DeprecatedPreferAny, v1beta1.Any:
		cpuResources := h.vmiSpec.Domain.CPU.Cores * h.vmiSpec.Domain.CPU.Sockets * h.vmiSpec.Domain.CPU.Threads
		if cpuResources < h.preferenceSpec.Requirements.CPU.Guest {
			return apply.Conflicts{cpuField.Child("cores"), cpuField.Child("sockets"), cpuField.Child("threads")},
				fmt.Errorf(InsufficientVMCPUResourcesErrorFmt,
					cpuResources, h.preferenceSpec.Requirements.CPU.Guest, "cores, sockets and threads")
		}
	}

	return nil, nil
}

func (h *Handler) checkSpread() (apply.Conflicts, error) {
	var (
		vCPUs     uint32
		conflicts apply.Conflicts
	)
	cpuField := k8sfield.NewPath("spec", "template", "spec", "domain", "cpu")
	_, across := preferenceApply.GetSpreadOptions(h.preferenceSpec)
	switch across {
	case v1beta1.SpreadAcrossSocketsCores:
		vCPUs = h.vmiSpec.Domain.CPU.Sockets * h.vmiSpec.Domain.CPU.Cores
		conflicts = apply.Conflicts{cpuField.Child("sockets"), cpuField.Child("cores")}
	case v1beta1.SpreadAcrossCoresThreads:
		vCPUs = h.vmiSpec.Domain.CPU.Cores * h.vmiSpec.Domain.CPU.Threads
		conflicts = apply.Conflicts{cpuField.Child("cores"), cpuField.Child("threads")}
	case v1beta1.SpreadAcrossSocketsCoresThreads:
		vCPUs = h.vmiSpec.Domain.CPU.Sockets * h.vmiSpec.Domain.CPU.Cores * h.vmiSpec.Domain.CPU.Threads
		conflicts = apply.Conflicts{cpuField.Child("sockets"), cpuField.Child("cores"), cpuField.Child("threads")}
	}
	if vCPUs < h.preferenceSpec.Requirements.CPU.Guest {
		return conflicts, fmt.Errorf(InsufficientVMCPUResourcesErrorFmt, vCPUs, h.preferenceSpec.Requirements.CPU.Guest, across)
	}
	return nil, nil
}
