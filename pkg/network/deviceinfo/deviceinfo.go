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

package deviceinfo

import (
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

func MapNetworkNameToDeviceInfo(networks []v1.Network,
	interfaces []v1.Interface,
	networkStatuses []networkv1.NetworkStatus,
) map[string]*networkv1.DeviceInfo {
	multusInterfaceNameToNetworkStatus := multus.NetworkStatusesByPodIfaceName(networkStatuses)
	podIfaceNamesByNetworkName := namescheme.CreateFromNetworkStatuses(networks, networkStatuses)

	networkDeviceInfo := map[string]*networkv1.DeviceInfo{}
	for _, iface := range interfaces {
		multusInterfaceName := podIfaceNamesByNetworkName[iface.Name]
		networkStatusEntry, exist := multusInterfaceNameToNetworkStatus[multusInterfaceName]
		if exist && networkStatusEntry.DeviceInfo != nil {
			networkDeviceInfo[iface.Name] = networkStatusEntry.DeviceInfo
			// Note: there is no need to return an error in case the interface doesn't exist,
			// it may mean the interface is not plugged yet
		}
	}
	return networkDeviceInfo
}
