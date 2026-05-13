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

	v1 "kubevirt.io/api/core/v1"
)

func ToPodResourceClaims(resourceClaims []v1.VirtualMachineInstanceResourceClaim) []k8sv1.PodResourceClaim {
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
