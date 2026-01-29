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

package downwardapi

import (
	"cmp"
	"encoding/json"
	"slices"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"kubevirt.io/client-go/log"
)

const (
	NetworkInfoAnnot      = "kubevirt.io/network-info"
	MountPath             = "/etc/podinfo"
	NetworkInfoVolumeName = "network-info-annotation"
	NetworkInfoVolumePath = "network-info"
)

func CreateNetworkInfoAnnotationValue(networkStatusesByNetworkName map[string]networkv1.NetworkStatus) string {
	networkInfo := generateNetworkInfo(networkStatusesByNetworkName)
	networkInfoBytes, err := json.Marshal(networkInfo)
	if err != nil {
		log.Log.Warningf("failed to marshal network-info: %v", err)
		return ""
	}

	return string(networkInfoBytes)
}

func generateNetworkInfo(networkStatusesByNetworkName map[string]networkv1.NetworkStatus) NetworkInfo {
	downwardAPIInterfaces := make([]Interface, 0, len(networkStatusesByNetworkName))

	for networkName, networkStatus := range networkStatusesByNetworkName {
		downwardAPIInterfaces = append(
			downwardAPIInterfaces,
			Interface{
				Network:    networkName,
				DeviceInfo: networkStatus.DeviceInfo,
				Mac:        networkStatus.Mac,
			},
		)
	}

	// Sort by network name to get deterministic order
	slices.SortFunc(downwardAPIInterfaces, func(iface1, iface2 Interface) int {
		return cmp.Compare(iface1.Network, iface2.Network)
	})

	return NetworkInfo{Interfaces: downwardAPIInterfaces}
}
