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

package netbinding

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	k8scorev1 "k8s.io/api/core/v1"
	k8srecord "k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hooks"
)

func NetBindingPluginSidecarList(vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration, recorder k8srecord.EventRecorder) (hooks.HookSidecarList, error) {
	var pluginSidecars hooks.HookSidecarList

	if slirpSidecar := slirpNetBindingPluginSidecar(vmi, config, recorder); slirpSidecar != nil {
		pluginSidecars = append(pluginSidecars, *slirpSidecar)
	}

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
				return nil, fmt.Errorf("couldn't find configuration for network bindining: %s", iface.Binding.Name)
			}
		}
	}

	for _, pluginInfo := range bindingByName {
		if pluginInfo.SidecarImage != "" {
			pluginSidecars = append(pluginSidecars, hooks.HookSidecar{
				Image:           pluginInfo.SidecarImage,
				ImagePullPolicy: config.ImagePullPolicy,
			})
		}
	}

	return pluginSidecars, nil
}

const (
	SlirpNetworkBindingPluginName = "slirp"
	DefaultSlirpPluginImage       = "quay.io/kubevirt/network-slirp-binding:20230830_638c60fc8"

	// UnregisteredNetworkBindingPluginReason is added to event when a requested network binding plugin's image is not
	// registered, i.e.: not specified in Kubevirt config network configuration.
	UnregisteredNetworkBindingPluginReason = "UnregisteredNetworkBindingPlugin"
)

func slirpNetBindingPluginSidecar(vmi *v1.VirtualMachineInstance, kvConfig *v1.KubeVirtConfiguration, recorder k8srecord.EventRecorder) *hooks.HookSidecar {
	slirpIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(i v1.Interface) bool {
		return i.Slirp != nil
	})
	if len(slirpIfaces) == 0 {
		return nil
	}

	var slirpSidecarImage string
	if plugin := ReadNetBindingPluginConfiguration(kvConfig, SlirpNetworkBindingPluginName); plugin == nil {
		// In case no Slirp network binding plugin is registered (i.e.: specified in in Kubevirt config) use default image
		// to prevent newly created Slirp VMs from hanging, and reduce friction for users who didn't register an image yet.
		// TODO: remove this workaround by next Kubevirt release v1.2.0.
		msg := fmt.Sprintf("no Slirp network binding plugin image is set in Kubevirt config, "+
			"using '%s' sidecar image for Slirp network binding configuration", DefaultSlirpPluginImage)
		recorder.Event(vmi, k8scorev1.EventTypeWarning, UnregisteredNetworkBindingPluginReason, msg)
		slirpSidecarImage = DefaultSlirpPluginImage
	} else {
		slirpSidecarImage = plugin.SidecarImage
	}

	return &hooks.HookSidecar{
		Image:           slirpSidecarImage,
		ImagePullPolicy: kvConfig.ImagePullPolicy,
	}
}

func ReadNetBindingPluginConfiguration(kvConfig *v1.KubeVirtConfiguration, pluginName string) *v1.InterfaceBindingPlugin {
	if kvConfig != nil && kvConfig.NetworkConfiguration != nil && kvConfig.NetworkConfiguration.Binding != nil {
		if plugin, exist := kvConfig.NetworkConfiguration.Binding[pluginName]; exist {
			return &plugin
		}
	}

	return nil
}
