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

	interfaceStatuses = append(interfaceStatuses, calculatePodIfaceStatuses(vmi, networkStatuses)...)

	secondaryIfaceStatuses, err := calculateSecondaryIfaceStatuses(vmi, networkStatuses)
	if err != nil {
		return err
	}

	interfaceStatuses = append(interfaceStatuses, secondaryIfaceStatuses...)

	// Preserve interfaces discovered by the virt-handler which are not specified in the VMI.Spec.
	interfaceStatuses = append(interfaceStatuses, filterUnspecifiedSpecIfaces(vmi.Status.Interfaces, vmi.Spec.Networks)...)

	vmi.Status.Interfaces = interfaceStatuses
	return nil
}

// calculatePodIfaceStatuses returns interface statuses for all pod networks and multus default networks.
// This supports multiple pod networks (not just one primary).
func calculatePodIfaceStatuses(
	vmi *v1.VirtualMachineInstance,
	networkStatuses []networkv1.NetworkStatus,
) []v1.VirtualMachineInstanceNetworkInterface {
	var interfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	// Get primary pod interface name for the first pod/default network
	primaryPodIfaceName := multus.LookupPodPrimaryIfaceName(networkStatuses)
	if primaryPodIfaceName == "" {
		primaryPodIfaceName = namescheme.PrimaryPodInterfaceName
	}

	isFirstDefaultNetwork := true
	for _, network := range vmi.Spec.Networks {
		// Process pod networks and multus default networks
		isPodNetwork := network.Pod != nil
		isMultusDefault := network.Multus != nil && network.Multus.Default
		if !isPodNetwork && !isMultusDefault {
			continue
		}

		var podIfaceName string
		if isFirstDefaultNetwork {
			podIfaceName = primaryPodIfaceName
			isFirstDefaultNetwork = false
		} else {
			// For secondary pod networks, use the network name as pod interface name
			podIfaceName = network.Name
		}

		ifaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, network.Name)
		if ifaceStatus == nil {
			interfaceStatuses = append(interfaceStatuses, v1.VirtualMachineInstanceNetworkInterface{
				Name:             network.Name,
				PodInterfaceName: podIfaceName,
			})
		} else {
			ifaceStatusCopy := *ifaceStatus
			ifaceStatusCopy.PodInterfaceName = podIfaceName
			interfaceStatuses = append(interfaceStatuses, ifaceStatusCopy)
		}
	}

	return interfaceStatuses
}

func calculateSecondaryIfaceStatuses(
	vmi *v1.VirtualMachineInstance,
	networkStatuses []networkv1.NetworkStatus,
) ([]v1.VirtualMachineInstanceNetworkInterface, error) {
	var interfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networkStatusesByPodIfaceName := multus.NetworkStatusesByPodIfaceName(networkStatuses)
	podIfaceNamesByNetworkName := namescheme.CreateFromNetworkStatuses(vmi.Spec.Networks, networkStatuses)
	for _, network := range vmispec.FilterMultusNonDefaultNetworks(vmi.Spec.Networks) {
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
) []v1.VirtualMachineInstanceNetworkInterface {
	var unspecifiedIfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networksByName := vmispec.IndexNetworkSpecByName(networks)

	for _, ifaceStatus := range ifaceStatuses {
		if _, exist := networksByName[ifaceStatus.Name]; !exist {
			unspecifiedIfaceStatuses = append(unspecifiedIfaceStatuses, ifaceStatus)
		}
	}

	return unspecifiedIfaceStatuses
}
