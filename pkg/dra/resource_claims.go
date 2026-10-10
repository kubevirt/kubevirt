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

package dra

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"
)

func toPodResourceClaims(resourceClaims []v1.VirtualMachineInstanceResourceClaim) []k8sv1.PodResourceClaim {
	if len(resourceClaims) == 0 {
		return nil
	}

	podResourceClaims := make([]k8sv1.PodResourceClaim, len(resourceClaims))
	for i, resourceClaim := range resourceClaims {
		podResourceClaims[i] = k8sv1.PodResourceClaim{
			Name:                      resourceClaim.Name,
			ResourceClaimName:         resourceClaim.ResourceClaimName,
			ResourceClaimTemplateName: resourceClaim.ResourceClaimTemplateName,
		}
	}
	return podResourceClaims
}

func CPUsFromDRA(vmi *v1.VirtualMachineInstance, cpuDRAEnabled bool) bool {
	return cpuDRAEnabled && ShouldSynthesizeCPUResourceClaim(vmi)
}

func ShouldSynthesizeCPUResourceClaim(vmi *v1.VirtualMachineInstance) bool {
	if !vmi.IsCPUDedicated() {
		return false
	}
	cpu := vmi.Spec.Domain.CPU
	if cpu != nil && cpu.NUMA != nil && cpu.NUMA.GuestMappingPassthrough != nil {
		return false
	}
	return true
}

// PodResourceClaimsForVMI returns pod.spec.resourceClaims for the VMI, declaring the synthesized
// CPU claim alongside them when the caller decided this pod takes its CPUs from DRA.
func PodResourceClaimsForVMI(vmi *v1.VirtualMachineInstance, cpusFromDRA bool) []k8sv1.PodResourceClaim {
	claims := toPodResourceClaims(vmi.Spec.ResourceClaims)
	if !cpusFromDRA {
		return claims
	}

	if podResourceClaimNamed(claims, CPUClaimRefName) {
		return claims
	}

	return append(claims, k8sv1.PodResourceClaim{
		Name:              CPUClaimRefName,
		ResourceClaimName: ptr.To(CPUResourceClaimName(vmi.Name)),
	})
}

// PodCPUsFromDRA reports whether pod draws its CPUs from DRA instead of from kubelet's CPU
// manager, by looking for the synthesized CPU claim reference in its spec. It describes the pod as
// it was created, so it keeps answering for a running pod however the CPUsWithDRA feature gate
// moves afterwards.
func PodCPUsFromDRA(pod *k8sv1.Pod) bool {
	return pod != nil && podResourceClaimNamed(pod.Spec.ResourceClaims, CPUClaimRefName)
}

func podResourceClaimNamed(claims []k8sv1.PodResourceClaim, name string) bool {
	for _, claim := range claims {
		if claim.Name == name {
			return true
		}
	}
	return false
}
