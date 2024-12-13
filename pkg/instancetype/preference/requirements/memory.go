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

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

const (
	InsufficientInstanceTypeMemoryResourcesErrorFmt = "insufficient Memory resources of %s provided by instance type, preference requires %s"
	InsufficientVMMemoryResourcesErrorFmt           = "insufficient Memory resources of %s provided by VirtualMachine, preference requires %s"
)

func (h *Handler) checkMemory() (conflict.Conflicts, error) {
	if h.instancetypeSpec != nil && h.instancetypeSpec.Memory.Guest.Cmp(h.preferenceSpec.Requirements.Memory.Guest) < 0 {
		instancetypeMemory := h.instancetypeSpec.Memory.Guest.String()
		preferenceMemory := h.preferenceSpec.Requirements.Memory.Guest.String()
		return conflict.Conflicts{conflict.New("spec", "instancetype")},
			fmt.Errorf(InsufficientInstanceTypeMemoryResourcesErrorFmt, instancetypeMemory, preferenceMemory)
	}

	vmiMemory := h.vmiSpec.Domain.Memory
	if h.instancetypeSpec == nil && vmiMemory != nil && vmiMemory.Guest.Cmp(h.preferenceSpec.Requirements.Memory.Guest) < 0 {
		return conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "memory")},
			fmt.Errorf(InsufficientVMMemoryResourcesErrorFmt, vmiMemory.Guest.String(), h.preferenceSpec.Requirements.Memory.Guest.String())
	}
	return nil, nil
}
