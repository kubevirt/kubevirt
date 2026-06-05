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

package virtiofs

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

// Default resource values for virtiofs containers
var (
	DefaultCPURequest    = resource.MustParse("10m")
	DefaultCPULimit      = resource.MustParse("100m")
	DefaultMemoryRequest = resource.MustParse("1M")
	DefaultMemoryLimit   = resource.MustParse("80M")
)

// SupportContainerResourceConfig provides access to support container resource configuration
type SupportContainerResourceConfig interface {
	GetSupportContainerRequest(typeName v1.SupportContainerType, resourceName k8sv1.ResourceName) *resource.Quantity
	GetSupportContainerLimit(typeName v1.SupportContainerType, resourceName k8sv1.ResourceName) *resource.Quantity
}

// ResourcesForVirtioFSContainer returns the resource requirements for a virtiofs container.
// The dedicatedCPUs and guaranteedQOS parameters control whether CPU and memory requests
// should be set equal to limits for QOS guarantees.
func ResourcesForVirtioFSContainer(dedicatedCPUs, guaranteedQOS bool, config SupportContainerResourceConfig) k8sv1.ResourceRequirements {
	resources := k8sv1.ResourceRequirements{
		Requests: k8sv1.ResourceList{},
		Limits:   k8sv1.ResourceList{},
	}

	// CPU request
	resources.Requests[k8sv1.ResourceCPU] = DefaultCPURequest
	if config != nil {
		if reqCpu := config.GetSupportContainerRequest(v1.VirtioFS, k8sv1.ResourceCPU); reqCpu != nil {
			resources.Requests[k8sv1.ResourceCPU] = *reqCpu
		}
	}

	// Memory limit
	resources.Limits[k8sv1.ResourceMemory] = DefaultMemoryLimit
	if config != nil {
		if limMem := config.GetSupportContainerLimit(v1.VirtioFS, k8sv1.ResourceMemory); limMem != nil {
			resources.Limits[k8sv1.ResourceMemory] = *limMem
		}
	}

	// CPU limit
	resources.Limits[k8sv1.ResourceCPU] = DefaultCPULimit
	if config != nil {
		if limCpu := config.GetSupportContainerLimit(v1.VirtioFS, k8sv1.ResourceCPU); limCpu != nil {
			resources.Limits[k8sv1.ResourceCPU] = *limCpu
		}
	}

	// For dedicated CPUs or guaranteed QOS, set CPU request equal to limit
	if dedicatedCPUs || guaranteedQOS {
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
	}

	// Memory request
	if guaranteedQOS {
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	} else {
		resources.Requests[k8sv1.ResourceMemory] = DefaultMemoryRequest
		if config != nil {
			if reqMem := config.GetSupportContainerRequest(v1.VirtioFS, k8sv1.ResourceMemory); reqMem != nil {
				resources.Requests[k8sv1.ResourceMemory] = *reqMem
			}
		}
	}

	return resources
}
