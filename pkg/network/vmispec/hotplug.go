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

import (
	"strings"

	v1 "kubevirt.io/api/core/v1"
)

func NetworksToHotplugWhosePodIfacesAreReady(vmi *v1.VirtualMachineInstance) []v1.Network {
	var networksToHotplug []v1.Network
	interfacesToHoplug := IndexInterfaceStatusByName(
		vmi.Status.Interfaces,
		func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool {
			return strings.Contains(ifaceStatus.InfoSource, InfoSourceMultusStatus) &&
				!strings.Contains(ifaceStatus.InfoSource, InfoSourceDomain)
		},
	)

	for _, network := range vmi.Spec.Networks {
		if _, isIfacePluggedIntoPod := interfacesToHoplug[network.Name]; isIfacePluggedIntoPod {
			networksToHotplug = append(networksToHotplug, network)
		}
	}

	return networksToHotplug
}
