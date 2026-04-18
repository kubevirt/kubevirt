/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
