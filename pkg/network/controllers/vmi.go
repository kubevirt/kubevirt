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

package controllers

import (
	"fmt"

	k8scorev1 "k8s.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func UpdateVMIStatus(vmi *v1.VirtualMachineInstance, pod *k8scorev1.Pod) error {
	var interfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networkStatuses := multus.NetworkStatusesFromPod(pod)

	interfaceStatuses = append(interfaceStatuses, calculatePrimaryIfaceStatus(vmi, networkStatuses)...)

	secondaryIfaceStatuses, err := calculateSecondaryIfaceStatuses(vmi, networkStatuses)
	if err != nil {
		return err
	}

	interfaceStatuses = append(interfaceStatuses, secondaryIfaceStatuses...)

	// Preserve interfaces discovered by the virt-handler which are not specified in the VMI.Spec.
	interfaceStatuses = append(interfaceStatuses,
		filterUnspecifiedSpecIfaces(vmi.Status.Interfaces, vmi.Spec.Networks, vmi.Spec.Domain.Devices.Interfaces)...)

	vmi.Status.Interfaces = interfaceStatuses
	return nil
}

func calculatePrimaryIfaceStatus(
	vmi *v1.VirtualMachineInstance,
	networkStatuses []networkv1.NetworkStatus,
) []v1.VirtualMachineInstanceNetworkInterface {
	primaryNetworkSpec := vmispec.LookUpDefaultNetwork(vmi.Spec.Networks)
	if primaryNetworkSpec == nil {
		return nil
	}

	primaryPodIfaceName := multus.LookupPodPrimaryIfaceName(networkStatuses)
	if primaryPodIfaceName == "" {
		primaryPodIfaceName = namescheme.PrimaryPodInterfaceName
	}

	primaryIfaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, primaryNetworkSpec.Name)
	if primaryIfaceStatus == nil {
		return []v1.VirtualMachineInstanceNetworkInterface{
			{Name: primaryNetworkSpec.Name, PodInterfaceName: primaryPodIfaceName},
		}
	}

	primaryIfaceStatusCopy := *primaryIfaceStatus
	primaryIfaceStatusCopy.PodInterfaceName = primaryPodIfaceName
	return []v1.VirtualMachineInstanceNetworkInterface{primaryIfaceStatusCopy}
}

func calculateSecondaryIfaceStatuses(
	vmi *v1.VirtualMachineInstance,
	networkStatuses []networkv1.NetworkStatus,
) ([]v1.VirtualMachineInstanceNetworkInterface, error) {
	var interfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networkStatusesByPodIfaceName := multus.NetworkStatusesByPodIfaceName(networkStatuses)
	podIfaceNamesByNetworkName := namescheme.CreateFromNetworkStatuses(vmi.Spec.Networks, networkStatuses)

	// Filter out networks that correspond to absent interfaces
	// Only networks corresponding to non-absent interfaces are processed.
	// Hot unplugged interfaces (InterfaceStateAbsent) are excluded from status calculation.
	// This is to ensure that VMI status correctly reflects only the currently active interfaces
	nonAbsentInterfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	networksToProcess := vmispec.FilterNetworksByInterfaces(vmispec.FilterMultusNonDefaultNetworks(vmi.Spec.Networks), nonAbsentInterfaces)

	for _, network := range networksToProcess {
		vmiIfaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, network.Name)
		podIfaceName, wasFound := podIfaceNamesByNetworkName[network.Name]
		if !wasFound {
			return nil, fmt.Errorf("could not find the pod interface name for network [%s]", network.Name)
		}

		_, exists := networkStatusesByPodIfaceName[podIfaceName]
		switch {
		case exists && vmiIfaceStatus == nil:
			interfaceStatuses = append(interfaceStatuses, v1.VirtualMachineInstanceNetworkInterface{
				Name:             network.Name,
				InfoSource:       vmispec.InfoSourceMultusStatus,
				PodInterfaceName: podIfaceName,
			})
		case exists && vmiIfaceStatus != nil:
			updatedIfaceStatus := *vmiIfaceStatus
			updatedIfaceStatus.InfoSource = vmispec.AddInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
			updatedIfaceStatus.PodInterfaceName = podIfaceName
			interfaceStatuses = append(interfaceStatuses, updatedIfaceStatus)
		case !exists && vmiIfaceStatus != nil:
			updatedIfaceStatus := *vmiIfaceStatus
			updatedIfaceStatus.InfoSource = vmispec.RemoveInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
			updatedIfaceStatus.PodInterfaceName = podIfaceName
			interfaceStatuses = append(interfaceStatuses, updatedIfaceStatus)
		}
	}

	return interfaceStatuses, nil
}

func filterUnspecifiedSpecIfaces(
	ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface,
	networks []v1.Network,
	interfaces []v1.Interface,
) []v1.VirtualMachineInstanceNetworkInterface {
	var unspecifiedIfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networksByName := vmispec.IndexNetworkSpecByName(networks)
	// Create a map of interface names to interface state
	interfaceStateByName := make(map[string]v1.InterfaceState)
	for _, iface := range interfaces {
		interfaceStateByName[iface.Name] = iface.State
	}

	for _, ifaceStatus := range ifaceStatuses {
		// Skip interface statuses that are not specified in networks
		if _, exist := networksByName[ifaceStatus.Name]; !exist {
			// Only preserve if the corresponding interface is not absent
			if state, hasInterface := interfaceStateByName[ifaceStatus.Name]; !hasInterface || state != v1.InterfaceStateAbsent {
				unspecifiedIfaceStatuses = append(unspecifiedIfaceStatuses, ifaceStatus)
			}
		}
	}

	return unspecifiedIfaceStatuses
}
