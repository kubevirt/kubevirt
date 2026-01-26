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

package resources

import (
	k8scorev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

type MemoryCalculator struct{}

func (mc MemoryCalculator) Calculate(
	vmi *v1.VirtualMachineInstance,
	registeredPlugins map[string]v1.InterfaceBindingPlugin,
) resource.Quantity {
	totalMemory := resource.Quantity{}

	if vmispec.HasPasstBinding(vmi) {
		totalMemory.Add(getPasstMemoryOverhead())
	}

	totalMemory.Add(sumPluginsMemoryRequests(
		filterUniquePlugins(vmi.Spec.Domain.Devices.Interfaces, registeredPlugins),
	))

	return totalMemory
}

func filterUniquePlugins(interfaces []v1.Interface, registeredPlugins map[string]v1.InterfaceBindingPlugin) []v1.InterfaceBindingPlugin {
	var uniquePlugins []v1.InterfaceBindingPlugin

	uniquePluginsSet := map[string]struct{}{}

	for _, iface := range interfaces {
		if iface.Binding == nil {
			continue
		}

		pluginName := iface.Binding.Name
		if _, seen := uniquePluginsSet[pluginName]; seen {
			continue
		}

		plugin, exists := registeredPlugins[pluginName]
		if !exists {
			continue
		}

		uniquePluginsSet[pluginName] = struct{}{}
		uniquePlugins = append(uniquePlugins, plugin)
	}

	return uniquePlugins
}

func sumPluginsMemoryRequests(uniquePlugins []v1.InterfaceBindingPlugin) resource.Quantity {
	result := resource.Quantity{}

	for _, plugin := range uniquePlugins {
		if plugin.ComputeResourceOverhead == nil {
			continue
		}

		requests := plugin.ComputeResourceOverhead.Requests
		if requests == nil {
			continue
		}

		result.Add(requests[k8scorev1.ResourceMemory])
	}

	return result
}

func getPasstMemoryOverhead() resource.Quantity {
	const passtComputeMemoryOverheadWhenAllPortsAreForwarded = "250Mi"
	return resource.MustParse(passtComputeMemoryOverheadWhenAllPortsAreForwarded)
}
