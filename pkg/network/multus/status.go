/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package multus

import (
	"encoding/json"

	k8scorev1 "k8s.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"kubevirt.io/client-go/log"
)

// NetworkStatusesByPodIfaceName creates a map of NetworkStatus objects by pod interface name.
func NetworkStatusesByPodIfaceName(networkStatuses []networkv1.NetworkStatus) map[string]networkv1.NetworkStatus {
	statusesByPodIfaceName := map[string]networkv1.NetworkStatus{}

	for _, ns := range networkStatuses {
		statusesByPodIfaceName[ns.Interface] = ns
	}

	return statusesByPodIfaceName
}

func NetworkStatusesFromPod(pod *k8scorev1.Pod) []networkv1.NetworkStatus {
	var networkStatuses []networkv1.NetworkStatus

	if rawNetworkStatus := pod.Annotations[networkv1.NetworkStatusAnnot]; rawNetworkStatus != "" {
		if err := json.Unmarshal([]byte(rawNetworkStatus), &networkStatuses); err != nil {
			log.Log.Errorf("failed to unmarshall pod network status: %v", err)
		}
	}

	return networkStatuses
}

func LookupPodPrimaryIfaceName(networkStatuses []networkv1.NetworkStatus) string {
	for _, ns := range networkStatuses {
		if ns.Default && ns.Interface != "" {
			return ns.Interface
		}
	}

	return ""
}
