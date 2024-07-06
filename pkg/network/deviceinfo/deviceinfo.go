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
 * Copyright the KubeVirt Authors.
 *
 */

package deviceinfo

import (
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

func MapNetworkNameToDeviceInfo(networks []v1.Network,
	networkStatusAnnotationValue string,
	interfaces []v1.Interface,
) (map[string]*networkv1.DeviceInfo, error) {
	multusInterfaceNameToNetworkStatus, err := mapMultusInterfaceNameToNetworkStatus(networkStatusAnnotationValue)
	if err != nil {
		return nil, err
	}
	networkNameScheme := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(networks, multusInterfaceNameToNetworkStatus)

	networkDeviceInfo := map[string]*networkv1.DeviceInfo{}
	for _, iface := range interfaces {
		multusInterfaceName := networkNameScheme[iface.Name]
		networkStatusEntry, exist := multusInterfaceNameToNetworkStatus[multusInterfaceName]
		if exist && networkStatusEntry.DeviceInfo != nil {
			networkDeviceInfo[iface.Name] = networkStatusEntry.DeviceInfo
			// Note: there is no need to return an error in case the interface doesn't exist,
			// it may mean the interface is not plugged yet
		}
	}
	return networkDeviceInfo, nil
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
