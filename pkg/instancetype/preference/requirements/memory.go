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

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
)

const (
	InsufficientInstanceTypeMemoryResourcesErrorFmt = "insufficient Memory resources of %s provided by instance type, preference requires %s"
	InsufficientVMMemoryResourcesErrorFmt           = "insufficient Memory resources of %s provided by VirtualMachine, preference requires %s"
)

func checkMemory(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflict.Conflicts, error) {
	if instancetypeSpec != nil && instancetypeSpec.Memory.Guest.Cmp(preferenceSpec.Requirements.Memory.Guest) < 0 {
		instancetypeMemory := instancetypeSpec.Memory.Guest.String()
		preferenceMemory := preferenceSpec.Requirements.Memory.Guest.String()
		return conflict.Conflicts{conflict.New("spec", "instancetype")},
			fmt.Errorf(InsufficientInstanceTypeMemoryResourcesErrorFmt, instancetypeMemory, preferenceMemory)
	}

	vmiMemory := vmiSpec.Domain.Memory
	if instancetypeSpec == nil && vmiMemory != nil && vmiMemory.Guest.Cmp(preferenceSpec.Requirements.Memory.Guest) < 0 {
		return conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "memory")},
			fmt.Errorf(InsufficientVMMemoryResourcesErrorFmt, vmiMemory.Guest.String(), preferenceSpec.Requirements.Memory.Guest.String())
	}
	return nil, nil
}
