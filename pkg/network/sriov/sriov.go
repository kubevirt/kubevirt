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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package sriov

import (
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const (
	AliasPrefix        = "sriov-"
	NetworkPCIMapAnnot = "kubevirt.io/network-pci-map"
	MountPath          = "/etc/podinfo"
	VolumeName         = "network-pci-map-annotation"
	VolumePath         = "network-pci-map"
)

func CreateNetworkPCIAnnotationValue(networks []v1.Network, interfaces []v1.Interface, networkStatusAnnotationValue string) string {
	networkPCIMap, err := mapNetworkNameToPCIAddress(networks, interfaces, networkStatusAnnotationValue)
	if err != nil {
		log.Log.Warningf("failed to create network-pci-map: %v", err)
		networkPCIMap = map[string]string{}
	}

	networkPCIMapBytes, err := json.Marshal(networkPCIMap)
	if err != nil {
		log.Log.Warningf("failed to marshal network-pci-map: %v", err)
		return ""
	}

	return string(networkPCIMapBytes)
}

func mapNetworkNameToPCIAddress(networks []v1.Network, interfaces []v1.Interface,
	networkStatusAnnotationValue string) (map[string]string, error) {
	multusInterfaceNameToNetworkStatusMap, err := mapMultusInterfaceNameToNetworkStatus(networkStatusAnnotationValue)
	if err != nil {
		return nil, err
	}
	networkNameScheme := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(networks, multusInterfaceNameToNetworkStatusMap)

	networkPCIMap := map[string]string{}
	for _, sriovIface := range vmispec.FilterSRIOVInterfaces(interfaces) {
		multusInterfaceName := networkNameScheme[sriovIface.Name]
		networkStatusEntry, exist := multusInterfaceNameToNetworkStatusMap[multusInterfaceName]
		if !exist {
			continue // The interface is not plugged yet
		}
		if networkStatusEntry.DeviceInfo == nil || networkStatusEntry.DeviceInfo.Pci == nil {
			return nil, fmt.Errorf("failed to find device-info/pci-address in network-status annotation for SR-IOV interface %q", sriovIface.Name)
		}

		pciAddress := networkStatusEntry.DeviceInfo.Pci.PciAddress
		if pciAddress == "" {
			return nil, fmt.Errorf("failed to associate pci-address to SR-IOV interface %q", sriovIface.Name)
		}
		networkPCIMap[sriovIface.Name] = pciAddress
	}
	return networkPCIMap, nil
}

func mapMultusInterfaceNameToNetworkStatus(networkStatusAnnotationValue string) (map[string]networkv1.NetworkStatus, error) {
	if networkStatusAnnotationValue == "" {
		return nil, fmt.Errorf("network-status annotation is not present")
	}
	var networkStatusList []networkv1.NetworkStatus
	if err := json.Unmarshal([]byte(networkStatusAnnotationValue), &networkStatusList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network-status annotation: %v", err)
	}

	multusInterfaceNameToNetworkStatusMap := map[string]networkv1.NetworkStatus{}
	for _, networkStatus := range networkStatusList {
		multusInterfaceNameToNetworkStatusMap[networkStatus.Interface] = networkStatus
	}

	return multusInterfaceNameToNetworkStatusMap, nil
}
