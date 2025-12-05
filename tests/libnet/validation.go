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

package libnet

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

// ValidateVMIandPodIPMatch Checks that the vmi pod and vmi scheme have matching Ip/Ips fields for primary interface
func ValidateVMIandPodIPMatch(vmi *v1.VirtualMachineInstance, vmiPod *k8sv1.Pod) error {
	if vmi.Status.Interfaces[0].IP != vmiPod.Status.PodIP {
		return fmt.Errorf("VMI Status.Interfaces[0].IP %s does not equal pod's Status.PodIP %s",
			vmi.Status.Interfaces[0].IP, vmiPod.Status.PodIP)
	}

	if len(vmi.Status.Interfaces[0].IPs) != len(vmiPod.Status.PodIPs) {
		return fmt.Errorf("VMI Status.Interfaces[0].IPs %s len does not equal pod's Status.PodIPs %s len",
			vmi.Status.Interfaces[0].IPs, vmiPod.Status.PodIPs)
	}

	if len(vmi.Status.Interfaces[0].IPs) == 0 {
		return fmt.Errorf("VMI Status.Interfaces[0].IPs len is zero")
	}

	for i, ip := range vmiPod.Status.PodIPs {
		if vmi.Status.Interfaces[0].IPs[i] != ip.IP {
			return fmt.Errorf("VMI Status.Interfaces[0].IPs %s do not equal pod's Status.PodIPs %s",
				vmi.Status.Interfaces[0].IPs, vmiPod.Status.PodIPs)
		}
	}

	return nil
}

// AssertAllPodIPsReportedOnVMI checks that each Pod IP is present in the VMI reported IPs
func AssertAllPodIPsReportedOnVMI(vmi *v1.VirtualMachineInstance, vmiPod *k8sv1.Pod) error {
	if vmi.Status.Interfaces[0].IP != vmiPod.Status.PodIP {
		return fmt.Errorf("VMI Status.Interfaces[0].IP %s does not equal pod's Status.PodIP %s",
			vmi.Status.Interfaces[0].IP, vmiPod.Status.PodIP)
	}

	// Ensure each Pod IP is present in the VMI reported IP set.
	vmiIPSet := ipSliceToSet(vmi.Status.Interfaces[0].IPs)
	for _, podIP := range vmiPod.Status.PodIPs {
		if _, exists := vmiIPSet[podIP.IP]; !exists {
			return fmt.Errorf("pod IP %q to be present in VMI IPs %v",
				podIP.IP, vmi.Status.Interfaces[0].IPs)
		}
	}

	return nil
}

// Convert the VMI reported IPs list into a set for efficient lookups.
func ipSliceToSet(ips []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		set[ip] = struct{}{}
	}
	return set
}
