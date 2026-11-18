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

type clusterConfigProvider interface {
	NetworkDevicesWithDRAGateEnabled() bool
}

type VMIStatusUpdater struct {
	clusterConfigProvider clusterConfigProvider
}

func NewVMIStatusUpdater(clusterConfigProvider clusterConfigProvider) VMIStatusUpdater {
	return VMIStatusUpdater{clusterConfigProvider: clusterConfigProvider}
}

func (v VMIStatusUpdater) Update(vmi *v1.VirtualMachineInstance, pod *k8scorev1.Pod) error {
	var interfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	networkStatuses := multus.NetworkStatusesFromPod(pod)

	interfaceStatuses = append(interfaceStatuses, calculatePrimaryIfaceStatus(vmi, networkStatuses)...)

	networkDRAEnabled := v.clusterConfigProvider.NetworkDevicesWithDRAGateEnabled()
	secondaryIfaceStatuses, err := calculateSecondaryIfaceStatuses(vmi, networkStatuses, networkDRAEnabled)
	if err != nil {
		return err
	}

	interfaceStatuses = append(interfaceStatuses, secondaryIfaceStatuses...)

	// Preserve DRA resource-claim interface statuses regardless of the gate;
	// the pod-status info source is only added while the gate is enabled.
	interfaceStatuses = append(interfaceStatuses, calculateDRAIfaceStatuses(vmi, pod, networkDRAEnabled)...)

	// Preserve interfaces discovered by the virt-handler which are not specified in the VMI.Spec.
	interfaceStatuses = append(interfaceStatuses, filterUnspecifiedSpecIfaces(vmi.Status.Interfaces, vmi.Spec.Networks)...)

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
	networkDRAEnabled bool,
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
			infoSource := vmispec.InfoSourceMultusStatus
			if networkDRAEnabled {
				infoSource = vmispec.AddInfoSource(infoSource, vmispec.InfoSourcePodStatus)
			}
			interfaceStatuses = append(interfaceStatuses, v1.VirtualMachineInstanceNetworkInterface{
				Name:             network.Name,
				InfoSource:       infoSource,
				PodInterfaceName: podIfaceName,
			})
		case exists && vmiIfaceStatus != nil:
			updatedIfaceStatus := *vmiIfaceStatus
			updatedIfaceStatus.InfoSource = vmispec.AddInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
			if networkDRAEnabled {
				updatedIfaceStatus.InfoSource = vmispec.AddInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourcePodStatus)
			} else {
				// Clean a pod-status left over from when the gate was enabled.
				updatedIfaceStatus.InfoSource = vmispec.RemoveInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourcePodStatus)
			}
			updatedIfaceStatus.PodInterfaceName = podIfaceName
			interfaceStatuses = append(interfaceStatuses, updatedIfaceStatus)
		case !exists && vmiIfaceStatus != nil:
			updatedIfaceStatus := *vmiIfaceStatus
			updatedIfaceStatus.InfoSource = vmispec.RemoveInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
			// Remove pod-status unconditionally so it is cleaned even after the
			// gate is disabled.
			updatedIfaceStatus.InfoSource = vmispec.RemoveInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourcePodStatus)
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

// calculateDRAIfaceStatuses preserves existing DRA interface statuses and adds
// the pod-status info source once the resource claim is present on the pod.
//
// Unlike the Multus path, pod-status is only added, never removed: DRA network
// devices are not hot-pluggable, so a claim present on the pod stays for the
// VMI's lifetime and there is no runtime transition that requires clearing it.
func calculateDRAIfaceStatuses(
	vmi *v1.VirtualMachineInstance,
	pod *k8scorev1.Pod,
	networkDRAEnabled bool,
) []v1.VirtualMachineInstanceNetworkInterface {
	var interfaceStatuses []v1.VirtualMachineInstanceNetworkInterface

	claimsOnPod := resourceClaimsOnPod(pod)
	for _, network := range vmi.Spec.Networks {
		if !vmispec.IsDRANetwork(network) {
			continue
		}

		ifaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, network.Name)
		claimOnPod := networkDRAEnabled && claimsOnPod[network.ResourceClaim.ClaimName]

		switch {
		case ifaceStatus != nil:
			updatedIfaceStatus := *ifaceStatus
			if claimOnPod {
				updatedIfaceStatus.InfoSource = vmispec.AddInfoSource(updatedIfaceStatus.InfoSource, vmispec.InfoSourcePodStatus)
			}
			interfaceStatuses = append(interfaceStatuses, updatedIfaceStatus)
		case claimOnPod:
			interfaceStatuses = append(interfaceStatuses, v1.VirtualMachineInstanceNetworkInterface{
				Name:       network.Name,
				InfoSource: vmispec.InfoSourcePodStatus,
			})
		}
	}

	return interfaceStatuses
}

// resourceClaimsOnPod returns the set of pod resource-claim names whose
// ResourceClaim object has been generated for the pod. A nil ResourceClaimName
// means no claim was created, so the entry is ignored.
func resourceClaimsOnPod(pod *k8scorev1.Pod) map[string]bool {
	claims := map[string]bool{}
	for _, claimStatus := range pod.Status.ResourceClaimStatuses {
		if claimStatus.ResourceClaimName != nil {
			claims[claimStatus.Name] = true
		}
	}
	return claims
}
