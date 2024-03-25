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

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

const (
	MountPath = "/etc/podinfo"
)

func CreateNetworkDeviceInfoAnnotationValue(networks []v1.Network, interfaces []v1.Interface, networkStatusAnnotationValue string) string {
	networkDeviceInfoMap, err := mapNetworkNameToDeviceInfo(networks, interfaces, networkStatusAnnotationValue)
	if err != nil {
		log.Log.Warningf("failed to create network-device-info-map: %v", err)
		networkDeviceInfoMap = map[string]*networkv1.DeviceInfo{}
	}

	filteredMap := filterNonEmpyEntries(networkDeviceInfoMap)
	if len(filteredMap) == 0 {
		return ""
	}
	networkDeviceInfoMapBytes, err := json.Marshal(filteredMap)
	if err != nil {
		log.Log.Warningf("failed to marshal network-device-info-map: %v", err)
		return ""
	}

	return string(networkDeviceInfoMapBytes)
}

func mapNetworkNameToDeviceInfo(networks []v1.Network, interfaces []v1.Interface,
	networkStatusAnnotationValue string) (map[string]*networkv1.DeviceInfo, error) {
	multusInterfaceNameToNetworkStatusMap, err := mapMultusInterfaceNameToNetworkStatus(networkStatusAnnotationValue)
	if err != nil {
		return nil, err
	}
	networkNameScheme := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(networks, multusInterfaceNameToNetworkStatusMap)

	networkDeviceInfoMap := map[string]*networkv1.DeviceInfo{}
	for _, iface := range interfaces {
		multusInterfaceName := networkNameScheme[iface.Name]
		networkStatusEntry, exist := multusInterfaceNameToNetworkStatusMap[multusInterfaceName]
		if !exist {
			continue // The interface is not plugged yet
		}

		networkDeviceInfoMap[iface.Name] = networkStatusEntry.DeviceInfo
	}
	return networkDeviceInfoMap, nil
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

func filterNonEmpyEntries(networkDeviceInfoMap map[string]*networkv1.DeviceInfo) map[string]*networkv1.DeviceInfo {
	filteredMap := map[string]*networkv1.DeviceInfo{}
	for networkName, deviceInfo := range networkDeviceInfoMap {
		if deviceInfo != nil {
			filteredMap[networkName] = deviceInfo
		}
	}
	return filteredMap
}
