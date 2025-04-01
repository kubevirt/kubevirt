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
	netsToHotplug := filterNetsToHotplug(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, vmi.Status.Interfaces)
	netsToHotunplug := filterNetsToHotunplug(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, vmi.Status.Interfaces)

	return append(netsToHotplug, netsToHotunplug...)
}

func FilterNetsForMigrationTarget(vmi *v1.VirtualMachineInstance) []v1.Network {
	return vmi.Spec.Networks
}

func filterNetsToHotplug(
	ifaces []v1.Interface,
	nets []v1.Network,
	ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface,
) []v1.Network {
	nonAbsentIfaces := vmispec.FilterInterfacesSpec(ifaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})

	nonAbsentNets := vmispec.FilterNetworksByInterfaces(nets, nonAbsentIfaces)

	ifaceStatusesInPodAndNotInDomain := vmispec.IndexInterfaceStatusByName(
		ifaceStatuses,
		func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool {
			return vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceMultusStatus) &&
				!vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceDomain)
		},
	)

	var networksToHotplug []v1.Network

	for _, network := range nonAbsentNets {
		if _, isIfaceInPodAndNotInDomain := ifaceStatusesInPodAndNotInDomain[network.Name]; isIfaceInPodAndNotInDomain {
			networksToHotplug = append(networksToHotplug, network)
		}
	}

	return networksToHotplug
}

func filterNetsToHotunplug(
	ifaces []v1.Interface,
	nets []v1.Network,
	ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface,
) []v1.Network {
	ifaceStatusesNotInDomain := vmispec.IndexInterfaceStatusByName(ifaceStatuses, func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool {
		return !vmispec.ContainsInfoSource(ifaceStatus.InfoSource, vmispec.InfoSourceDomain)
	})

	ifacesToUnplug := vmispec.FilterInterfacesSpec(ifaces, func(iface v1.Interface) bool {
		_, isIfaceDetached := ifaceStatusesNotInDomain[iface.Name]
		return iface.State == v1.InterfaceStateAbsent && isIfaceDetached
	})

	return vmispec.FilterNetworksByInterfaces(nets, ifacesToUnplug)
}
