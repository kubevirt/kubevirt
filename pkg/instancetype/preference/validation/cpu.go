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
package validation

import (
	"fmt"
	"slices"

	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

func IsPreferredTopologySupported(topology v1beta1.PreferredCPUTopology) bool {
	supportedTopologies := []v1beta1.PreferredCPUTopology{
		v1beta1.DeprecatedPreferSockets,
		v1beta1.DeprecatedPreferCores,
		v1beta1.DeprecatedPreferThreads,
		v1beta1.DeprecatedPreferSpread,
		v1beta1.DeprecatedPreferAny,
		v1beta1.Sockets,
		v1beta1.Cores,
		v1beta1.Threads,
		v1beta1.Spread,
		v1beta1.Any,
	}
	return slices.Contains(supportedTopologies, topology)
}

const (
	instancetypeCPUGuestPath       = "instancetype.spec.cpu.guest"
	spreadAcrossSocketsCoresErrFmt = "%d vCPUs provided by the instance type are not divisible by the " +
		"Spec.PreferSpreadSocketToCoreRatio or Spec.CPU.PreferSpreadOptions.Ratio of %d provided by the preference"
	spreadAcrossCoresThreadsErrFmt        = "%d vCPUs provided by the instance type are not divisible by the number of threads per core %d"
	spreadAcrossSocketsCoresThreadsErrFmt = "%d vCPUs provided by the instance type are not divisible by the number of threads per core " +
		"%d and Spec.PreferSpreadSocketToCoreRatio or Spec.CPU.PreferSpreadOptions.Ratio of %d"
)

func CheckSpreadCPUTopology(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
) *conflict.Conflict {
	topology := apply.GetPreferredTopology(preferenceSpec)
	if instancetypeSpec == nil || (topology != v1beta1.Spread && topology != v1beta1.DeprecatedPreferSpread) {
		return nil
	}

	ratio, across := apply.GetSpreadOptions(preferenceSpec)
	switch across {
	case v1beta1.SpreadAcrossSocketsCores:
		if (instancetypeSpec.CPU.Guest % ratio) > 0 {
			return conflict.NewWithMessage(
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, instancetypeSpec.CPU.Guest, ratio),
				instancetypeCPUGuestPath,
			)
		}
	case v1beta1.SpreadAcrossCoresThreads:
		if (instancetypeSpec.CPU.Guest % ratio) > 0 {
			return conflict.NewWithMessage(
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, instancetypeSpec.CPU.Guest, ratio),
				instancetypeCPUGuestPath,
			)
		}
	case v1beta1.SpreadAcrossSocketsCoresThreads:
		const threadsPerCore = 2
		if (instancetypeSpec.CPU.Guest%threadsPerCore) > 0 || ((instancetypeSpec.CPU.Guest/threadsPerCore)%ratio) > 0 {
			return conflict.NewWithMessage(
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, instancetypeSpec.CPU.Guest, threadsPerCore, ratio),
				instancetypeCPUGuestPath,
			)
		}
	}
	return nil
}
