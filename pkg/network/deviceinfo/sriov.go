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

package deviceinfo

import (
	"encoding/json"
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const (
	SRIOVAliasPrefix        = "sriov-"
	NetworkPCIMapAnnot      = "kubevirt.io/network-pci-map"
	NetwrokPCIMapVolumeName = "network-pci-map-annotation"
	NetworkPCIMapVolumePath = "network-pci-map"
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
	networkNameToDeviceInfo, err := mapNetworkNameToDeviceInfo(
		networks,
		networkStatusAnnotationValue,
		vmispec.FilterSRIOVInterfaces(interfaces),
	)
	if err != nil {
		return nil, err
	}

	networkPCIMap := map[string]string{}
	for netName, deviceInfo := range networkNameToDeviceInfo {
		if deviceInfo == nil || deviceInfo.Pci == nil {
			return nil, fmt.Errorf("failed to find device-info/pci-address in network-status annotation for SR-IOV interface %q", netName)
		}

		pciAddress := deviceInfo.Pci.PciAddress
		if pciAddress == "" {
			return nil, fmt.Errorf("failed to associate pci-address to SR-IOV interface %q", netName)
		}
		networkPCIMap[netName] = pciAddress
	}
	return networkPCIMap, nil
}
