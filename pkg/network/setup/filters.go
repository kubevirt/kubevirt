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

package network

import (
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func FilterNetsForVMStartup(vmi *v1.VirtualMachineInstance) []v1.Network {
	nonAbsentIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})

	return vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, nonAbsentIfaces)
}

func FilterNetsForLiveUpdate(vmi *v1.VirtualMachineInstance) []v1.Network {
	netsToHotplug := networksToHotplugWhosePodIfacesAreReady(vmi)
	nonAbsentIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	netsToHotplug = vmispec.FilterNetworksByInterfaces(netsToHotplug, nonAbsentIfaces)

	netsToHotunplug := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, filterIfacesToUnplug(vmi))

	return append(netsToHotplug, netsToHotunplug...)
}

func FilterNetsForMigrationTarget(vmi *v1.VirtualMachineInstance) []v1.Network {
	return vmi.Spec.Networks
}

func networksToHotplugWhosePodIfacesAreReady(vmi *v1.VirtualMachineInstance) []v1.Network {
	var networksToHotplug []v1.Network
	interfacesToHoplug := vmispec.IndexInterfaceStatusByName(
		vmi.Status.Interfaces,
		func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool {
			return strings.Contains(ifaceStatus.InfoSource, vmispec.InfoSourceMultusStatus) &&
				!strings.Contains(ifaceStatus.InfoSource, vmispec.InfoSourceDomain)
		},
	)

	for _, network := range vmi.Spec.Networks {
		if _, isIfacePluggedIntoPod := interfacesToHoplug[network.Name]; isIfacePluggedIntoPod {
			networksToHotplug = append(networksToHotplug, network)
		}
	}

	return networksToHotplug
}

func filterIfacesToUnplug(vmi *v1.VirtualMachineInstance) []v1.Interface {
	ifaceStatusesNotInDomain := vmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces, func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool {
		return !vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceDomain)
	})

	return vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		_, isIfaceDetached := ifaceStatusesNotInDomain[iface.Name]
		return iface.State == v1.InterfaceStateAbsent && isIfaceDetached
	})
}
