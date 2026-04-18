/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package netbinding

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hooks"
)

func NetBindingPluginSidecarList(vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
	var pluginSidecars hooks.HookSidecarList

	netbindingPluginSidecars, err := netBindingPluginSidecar(vmi, config)
	if err != nil {
		return nil, err
	}
	pluginSidecars = append(pluginSidecars, netbindingPluginSidecars...)

	return pluginSidecars, nil
}

func netBindingPluginSidecar(vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
	var pluginSidecars hooks.HookSidecarList
	bindingByName := map[string]v1.InterfaceBindingPlugin{}
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Binding != nil {
			var exist bool
			var pluginInfo v1.InterfaceBindingPlugin
			if config.NetworkConfiguration != nil && config.NetworkConfiguration.Binding != nil {
				pluginInfo, exist = config.NetworkConfiguration.Binding[iface.Binding.Name]
				bindingByName[iface.Binding.Name] = pluginInfo
			}

			if !exist {
				return nil, fmt.Errorf("couldn't find configuration for network binding: %s", iface.Binding.Name)
			}
		}
	}

	for _, pluginInfo := range bindingByName {
		if pluginInfo.SidecarImage != "" {
			pluginSidecars = append(pluginSidecars, hooks.HookSidecar{
				Image:           pluginInfo.SidecarImage,
				ImagePullPolicy: config.ImagePullPolicy,
				DownwardAPI:     pluginInfo.DownwardAPI,
			})
		}
	}

	return pluginSidecars, nil
}
