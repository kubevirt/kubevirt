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

package vmispec

import v1 "kubevirt.io/api/core/v1"

func NetworksToHotplug(networks []v1.Network, interfaceStatus []v1.VirtualMachineInstanceNetworkInterface) []v1.Network {
	var networksToHotplug []v1.Network
	indexedIfacesFromStatus := indexedInterfacesFromStatus(interfaceStatus)
	for _, iface := range networks {
		if _, wasFound := indexedIfacesFromStatus[iface.Name]; !wasFound {
			networksToHotplug = append(networksToHotplug, iface)
		}
	}
	return networksToHotplug
}

func indexedInterfacesFromStatus(interfaces []v1.VirtualMachineInstanceNetworkInterface) map[string]v1.VirtualMachineInstanceNetworkInterface {
	indexedInterfaceStatus := map[string]v1.VirtualMachineInstanceNetworkInterface{}
	for _, iface := range interfaces {
		indexedInterfaceStatus[iface.Name] = iface
	}
	return indexedInterfaceStatus
}
