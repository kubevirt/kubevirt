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

// GetMigrationNetworkIPs extracts IP addresses from a pod for migration communication.
// It checks for migration network IPs in pod network status annotations, returns empty slice if not found.
func GetMigrationNetworkIPs(pod *k8scorev1.Pod, migrationInterface string) []string {
	// Check for migration network IPs in pod annotations
	networkStatuses := NetworkStatusesFromPod(pod)
	for _, networkStatus := range networkStatuses {
		if networkStatus.Interface == migrationInterface {
			if len(networkStatus.IPs) > 0 {
				log.Log.Object(pod).V(4).Infof("found migration network IP addresses %v", networkStatus.IPs) //nolint:mnd
				return networkStatus.IPs
			}
			break
		}
	}
	return nil
}
