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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package services

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hooks"
)

func NetBindingPluginSidecarList(vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
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
				return nil, fmt.Errorf("couldn't find configuration for network bindining: %s", iface.Binding.Name)
			}
		}
	}
	for _, pluginInfo := range bindingByName {
		if pluginInfo.SidecarImage != "" {
			pluginSidecars = append(pluginSidecars, hooks.HookSidecar{Image: pluginInfo.SidecarImage})
		}
	}
	return pluginSidecars, nil
}

const SlirpNetworkBindingPluginName = "slirp"

func ReadNetBindingPluginConfiguration(kvConfig *v1.KubeVirtConfiguration, pluginName string) *v1.InterfaceBindingPlugin {
	if kvConfig != nil && kvConfig.NetworkConfiguration != nil && kvConfig.NetworkConfiguration.Binding != nil {
		if plugin, exist := kvConfig.NetworkConfiguration.Binding[pluginName]; exist {
			return &plugin
		}
	}

	// TODO: in case no Slirp network binding plugin is registered (i.e.: specified in in Kubevirt config) use the
	// following default image to prevent newly created Slirp VMs from handing, and reduce friction for users who didnt
	// registered and image yet. This workaround should be removed by the next Kubevirt release v1.2.0.
	const defaulSlirpPluginImage = "quay.io/kubevirt/network-slirp-binding:20230830_638c60fc8"
	if pluginName == SlirpNetworkBindingPluginName {
		log.Log.Warningf("no Slirp network binding plugin image is set in Kubevirt config, using %q sidecar image for Slirp network binding configuration", defaulSlirpPluginImage)
		return &v1.InterfaceBindingPlugin{SidecarImage: defaulSlirpPluginImage}
	}

	return nil
}
