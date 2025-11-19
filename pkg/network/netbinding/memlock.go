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

package netbinding

import (
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

func NetBindingHasMemoryLockRequirements(binding *v1.InterfaceBindingPlugin) bool {
	if binding == nil {
		return false
	}

	limits := binding.MemoryLockLimits
	if limits == nil {
		return false
	}

	return limits.LockGuestMemory || limits.Offset != nil && !limits.Offset.IsZero()
}

func ApplyNetBindingMemlockRequirements(
	guestMemory *resource.Quantity,
	vmi *v1.VirtualMachineInstance,
	registeredPlugins map[string]v1.InterfaceBindingPlugin,
) *resource.Quantity {
	ratio, offset := gatherNetBindingMemLockRequirements(vmi, registeredPlugins)

	if ratio == 0 && offset.IsZero() {
		return guestMemory
	} else if ratio == 0 {
		ratio = 1
	}

	res := resource.NewScaledQuantity(guestMemory.ScaledValue(resource.Kilo)*ratio, resource.Kilo)
	res.Add(*offset)

	return res
}

func gatherNetBindingMemLockRequirements(
	vmi *v1.VirtualMachineInstance,
	registeredPlugins map[string]v1.InterfaceBindingPlugin,
) (int64, *resource.Quantity) {
	resOffset := resource.NewScaledQuantity(0, resource.Kilo)
	var resRatio int64 = 0

	for bindingName, count := range netBindingCountMap(vmi, registeredPlugins) {
		limits := registeredPlugins[bindingName].MemoryLockLimits
		if limits == nil {
			continue
		}

		if limits.Offset != nil {
			resOffset.Add(*limits.Offset)
		}
		if limits.LockGuestMemory {
			resRatio += count
		}
	}

	return resRatio, resOffset
}

func netBindingCountMap(
	vmi *v1.VirtualMachineInstance,
	registeredPlugins map[string]v1.InterfaceBindingPlugin,
) map[string]int64 {
	counter := map[string]int64{}
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Binding == nil {
			continue
		}
		bindingName := iface.Binding.Name
		if _, exists := registeredPlugins[bindingName]; !exists {
			continue
		}

		counter[bindingName]++
	}
	return counter
}
