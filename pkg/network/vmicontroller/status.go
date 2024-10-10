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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package vmicontroller

import (
	"fmt"

	k8scorev1 "k8s.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func UpdateStatus(vmi *v1.VirtualMachineInstance, pod *k8scorev1.Pod) error {
	var updatedInterfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networkStatuses := multus.NetworkStatusesFromPod(pod)

	if primaryInterfaceStatus := calculatePrimaryInterfaceStatus(vmi, networkStatuses); primaryInterfaceStatus != nil {
		updatedInterfaceStatuses = append(updatedInterfaceStatuses, *primaryInterfaceStatus)
	}

	secondaryInterfacesStatus, err := calculateSecondaryInterfacesStatus(vmi, networkStatuses)
	if err != nil {
		return err
	}

	updatedInterfaceStatuses = append(updatedInterfaceStatuses, secondaryInterfacesStatus...)

	vmi.Status.Interfaces = updatedInterfaceStatuses

	return nil
}

func calculatePrimaryInterfaceStatus(
	vmi *v1.VirtualMachineInstance,
	networkStatuses []networkv1.NetworkStatus,
) *v1.VirtualMachineInstanceNetworkInterface {
	vmiPrimaryNetworkSpec := vmispec.LookUpDefaultNetwork(vmi.Spec.Networks)
	if vmiPrimaryNetworkSpec == nil {
		return nil
	}

	primaryPodIfaceName := multus.LookupPodPrimaryIfaceName(networkStatuses)
	if primaryPodIfaceName == "" {
		primaryPodIfaceName = namescheme.PrimaryPodInterfaceName
	}

	vmiPrimaryIfaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, vmiPrimaryNetworkSpec.Name)
	if vmiPrimaryIfaceStatus == nil {
		return &v1.VirtualMachineInstanceNetworkInterface{
			Name:             vmiPrimaryNetworkSpec.Name,
			PodInterfaceName: primaryPodIfaceName,
		}
	}

	vmiPrimaryIfaceStatus.PodInterfaceName = primaryPodIfaceName
	return vmiPrimaryIfaceStatus
}

func calculateSecondaryInterfacesStatus(
	vmi *v1.VirtualMachineInstance,
	networkStatuses []networkv1.NetworkStatus,
) ([]v1.VirtualMachineInstanceNetworkInterface, error) {
	var secondaryIfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networkStatusesByPodIfaceName := multus.NonDefaultNetworkStatusIndexedByPodIfaceName(networkStatuses)
	podIfaceNamesByNetworkName := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(
		vmi.Spec.Networks,
		networkStatusesByPodIfaceName,
	)

	for _, network := range vmispec.FilterMultusNonDefaultNetworks(vmi.Spec.Networks) {
		podIfaceName, wasFound := podIfaceNamesByNetworkName[network.Name]
		if !wasFound {
			return nil, fmt.Errorf("could not find the pod interface name for network [%s]", network.Name)
		}
		_, multusStatusExists := networkStatusesByPodIfaceName[podIfaceName]

		vmiIfaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, network.Name)
		vmiStatusExists := vmiIfaceStatus != nil

		switch {
		case !vmiStatusExists && !multusStatusExists:
			secondaryIfaceStatuses = append(secondaryIfaceStatuses, v1.VirtualMachineInstanceNetworkInterface{
				Name:             network.Name,
				PodInterfaceName: podIfaceName,
			})
		case !vmiStatusExists && multusStatusExists:
			secondaryIfaceStatuses = append(secondaryIfaceStatuses, v1.VirtualMachineInstanceNetworkInterface{
				Name:             network.Name,
				PodInterfaceName: podIfaceName,
				InfoSource:       vmispec.InfoSourceMultusStatus,
			})
		case vmiStatusExists && multusStatusExists:
			updatedIfaceStatus := *vmiIfaceStatus
			updatedIfaceStatus.InfoSource = vmispec.AddInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
			updatedIfaceStatus.PodInterfaceName = podIfaceName
			secondaryIfaceStatuses = append(secondaryIfaceStatuses, updatedIfaceStatus)
		case vmiStatusExists && !multusStatusExists:
			updatedIfaceStatus := *vmiIfaceStatus
			updatedIfaceStatus.InfoSource = vmispec.RemoveInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
			updatedIfaceStatus.PodInterfaceName = podIfaceName
			secondaryIfaceStatuses = append(secondaryIfaceStatuses, updatedIfaceStatus)
		}
	}

	return secondaryIfaceStatuses, nil
}
