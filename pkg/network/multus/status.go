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

const (
	// logVerbosityLevel defines the verbosity level for detailed network status logging
	logVerbosityLevel = 4
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
// It first checks for migration network IPs in pod network status annotations,
// falling back to standard pod network IPs if the migration network is not configured.
// This matches the logic used by virt-operator when updating synchronizationAddresses.
func GetMigrationNetworkIPs(pod *k8scorev1.Pod, migrationInterface string) []string {
	// Check for migration network IPs in pod annotations
	networkStatuses := NetworkStatusesFromPod(pod)
	for _, networkStatus := range networkStatuses {
		if networkStatus.Interface == migrationInterface {
			if len(networkStatus.IPs) > 0 {
				log.Log.Object(pod).V(logVerbosityLevel).Infof("found migration network IP addresses %v", networkStatus.IPs)
				return networkStatus.IPs
			}
			break
		}
	}

	log.Log.Object(pod).V(logVerbosityLevel).Infof("didn't find migration network IP in annotations, using pod IPs")

	// Fall back to pod network IPs if migration network is not configured
	if len(pod.Status.PodIPs) > 0 {
		ips := make([]string, len(pod.Status.PodIPs))
		for i, podIP := range pod.Status.PodIPs {
			ips[i] = podIP.IP
		}
		return ips
	}

	// Last resort: use single pod IP
	if pod.Status.PodIP != "" {
		return []string{pod.Status.PodIP}
	}

	return nil
}
