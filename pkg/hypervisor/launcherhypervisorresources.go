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

package hypervisor

import (
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor/kvm"
)

// Interface to abstract the hypervisor-specific resources available to the virt-launcher,
// such as devices and memory overhead.
type LauncherHypervisorResources interface {
	GetHypervisorDevice() string
	// TODO: Remove GetMemoryOverhead from this interface once VmiMemoryOverheadReport feature gate is GA
	// and we are sure that all VMIs include the MemoryOverhead status field. At that point,
	// memory overhead should only be calculated in virt-controller and stored in VMI status.
	GetMemoryOverhead(vmi *v1.VirtualMachineInstance, arch string, additionalOverheadRatio *string) resource.Quantity
}

func NewLauncherHypervisorResources(hypervisor string) LauncherHypervisorResources {
	switch hypervisor {
	// Other hypervisors can be added here
	default:
		return kvm.NewKvmHypervisorBackend()
	}
}
