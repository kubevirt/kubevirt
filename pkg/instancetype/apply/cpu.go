//nolint:gocyclo
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
package apply

import (
	k8sv1 "k8s.io/api/core/v1"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1 "kubevirt.io/api/instancetype/v1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	preferenceApply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

func applyCPU(
	baseConflict *conflict.Conflict,
	instancetypeSpec *instancetypev1.VirtualMachineInstancetypeSpec,
	preferenceSpec *instancetypev1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if vmiSpec.Domain.CPU == nil {
		vmiSpec.Domain.CPU = &virtv1.CPU{}
	}

	// If we have any conflicts return as there's no need to apply the topology below
	if conflicts := validateCPU(baseConflict, instancetypeSpec, vmiSpec); len(conflicts) > 0 {
		return conflicts
	}

	if instancetypeSpec.CPU.Model != nil {
		vmiSpec.Domain.CPU.Model = *instancetypeSpec.CPU.Model
	}

	if instancetypeSpec.CPU.DedicatedCPUPlacement != nil {
		vmiSpec.Domain.CPU.DedicatedCPUPlacement = *instancetypeSpec.CPU.DedicatedCPUPlacement
	}

	if instancetypeSpec.CPU.IsolateEmulatorThread != nil {
		vmiSpec.Domain.CPU.IsolateEmulatorThread = *instancetypeSpec.CPU.IsolateEmulatorThread
	}

	if instancetypeSpec.CPU.NUMA != nil {
		vmiSpec.Domain.CPU.NUMA = instancetypeSpec.CPU.NUMA.DeepCopy()
	}

	if instancetypeSpec.CPU.Realtime != nil {
		vmiSpec.Domain.CPU.Realtime = instancetypeSpec.CPU.Realtime.DeepCopy()
	}

	if instancetypeSpec.CPU.MaxSockets != nil {
		vmiSpec.Domain.CPU.MaxSockets = *instancetypeSpec.CPU.MaxSockets
	}

	applyGuestCPUTopology(instancetypeSpec.CPU.Guest, preferenceSpec, vmiSpec)

	return nil
}

func applyGuestCPUTopology(
	vCPUs uint32,
	preferenceSpec *instancetypev1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) {
	// Apply the default topology here to avoid duplication below
	vmiSpec.Domain.CPU.Cores = 1
	vmiSpec.Domain.CPU.Sockets = 1
	vmiSpec.Domain.CPU.Threads = 1

	if vCPUs == 1 {
		return
	}

	switch preferenceApply.GetPreferredTopology(preferenceSpec) {
	case instancetypev1.DeprecatedPreferCores, instancetypev1.Cores:
		vmiSpec.Domain.CPU.Cores = vCPUs
	case instancetypev1.DeprecatedPreferSockets, instancetypev1.DeprecatedPreferAny, instancetypev1.Sockets, instancetypev1.Any:
		vmiSpec.Domain.CPU.Sockets = vCPUs
	case instancetypev1.DeprecatedPreferThreads, instancetypev1.Threads:
		vmiSpec.Domain.CPU.Threads = vCPUs
	case instancetypev1.DeprecatedPreferSpread, instancetypev1.Spread:
		ratio, across := preferenceApply.GetSpreadOptions(preferenceSpec)
		switch across {
		case instancetypev1.SpreadAcrossSocketsCores:
			vmiSpec.Domain.CPU.Cores = ratio
			vmiSpec.Domain.CPU.Sockets = vCPUs / ratio
		case instancetypev1.SpreadAcrossCoresThreads:
			vmiSpec.Domain.CPU.Threads = ratio
			vmiSpec.Domain.CPU.Cores = vCPUs / ratio
		case instancetypev1.SpreadAcrossSocketsCoresThreads:
			const threadsPerCore = 2
			vmiSpec.Domain.CPU.Threads = threadsPerCore
			vmiSpec.Domain.CPU.Cores = ratio
			vmiSpec.Domain.CPU.Sockets = vCPUs / threadsPerCore / ratio
		}
	}
}

func validateCPU(
	baseConflict *conflict.Conflict,
	instancetypeSpec *instancetypev1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflicts conflict.Conflicts) {
	if _, hasCPURequests := vmiSpec.Domain.Resources.Requests[k8sv1.ResourceCPU]; hasCPURequests {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "resources", "requests", string(k8sv1.ResourceCPU)))
	}

	if _, hasCPULimits := vmiSpec.Domain.Resources.Limits[k8sv1.ResourceCPU]; hasCPULimits {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "resources", "limits", string(k8sv1.ResourceCPU)))
	}

	if vmiSpec.Domain.CPU.Sockets != 0 {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "sockets"))
	}

	if vmiSpec.Domain.CPU.Cores != 0 {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "cores"))
	}

	if vmiSpec.Domain.CPU.Threads != 0 {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "threads"))
	}

	if vmiSpec.Domain.CPU.Model != "" && instancetypeSpec.CPU.Model != nil {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "model"))
	}

	if vmiSpec.Domain.CPU.DedicatedCPUPlacement && instancetypeSpec.CPU.DedicatedCPUPlacement != nil {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "dedicatedCPUPlacement"))
	}

	if vmiSpec.Domain.CPU.IsolateEmulatorThread && instancetypeSpec.CPU.IsolateEmulatorThread != nil {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "isolateEmulatorThread"))
	}

	if vmiSpec.Domain.CPU.NUMA != nil && instancetypeSpec.CPU.NUMA != nil {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "numa"))
	}

	if vmiSpec.Domain.CPU.Realtime != nil && instancetypeSpec.CPU.Realtime != nil {
		conflicts = append(conflicts, baseConflict.NewChild("domain", "cpu", "realtime"))
	}

	return conflicts
}
