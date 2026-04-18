/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
