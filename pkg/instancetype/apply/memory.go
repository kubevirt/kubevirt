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
package apply

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

func applyMemory(
	baseConflict *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if vmiSpec.Domain.Memory != nil {
		return conflict.Conflicts{baseConflict.NewChild("domain", "memory")}
	}

	if _, hasMemoryRequests := vmiSpec.Domain.Resources.Requests[k8sv1.ResourceMemory]; hasMemoryRequests {
		return conflict.Conflicts{baseConflict.NewChild("domain", "resources", "requests", string(k8sv1.ResourceMemory))}
	}

	if _, hasMemoryLimits := vmiSpec.Domain.Resources.Limits[k8sv1.ResourceMemory]; hasMemoryLimits {
		return conflict.Conflicts{baseConflict.NewChild("domain", "resources", "limits", string(k8sv1.ResourceMemory))}
	}

	instancetypeMemory := instancetypeSpec.Memory.Guest.DeepCopy()
	vmiSpec.Domain.Memory = &virtv1.Memory{
		Guest: &instancetypeMemory,
	}

	// If memory overcommit has been requested, set the memory requests to be
	// lower than the guest memory by the requested percent.
	const totalPercentage = 100
	if instancetypeOverCommit := instancetypeSpec.Memory.OvercommitPercent; instancetypeOverCommit > 0 {
		if vmiSpec.Domain.Resources.Requests == nil {
			vmiSpec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}

		podRequestedMemory := int64(float32(instancetypeMemory.Value()) * (1 - float32(instancetypeOverCommit)/totalPercentage))

		vmiSpec.Domain.Resources.Requests[k8sv1.ResourceMemory] = *resource.NewQuantity(podRequestedMemory, instancetypeMemory.Format)
	}

	if instancetypeSpec.Memory.Hugepages != nil {
		vmiSpec.Domain.Memory.Hugepages = instancetypeSpec.Memory.Hugepages.DeepCopy()
	}

	if instancetypeSpec.Memory.MaxGuest != nil {
		m := instancetypeSpec.Memory.MaxGuest.DeepCopy()
		vmiSpec.Domain.Memory.MaxGuest = &m
	}

	return nil
}
