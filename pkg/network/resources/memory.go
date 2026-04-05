/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
