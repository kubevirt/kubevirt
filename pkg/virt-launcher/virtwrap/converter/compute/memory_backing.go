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

package compute

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

type MemoryBackingDomainConfigurator struct {
}

func NewMemoryBackingDomainConfigurator() MemoryBackingDomainConfigurator {
	return MemoryBackingDomainConfigurator{}
}

func (m MemoryBackingDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	var isMemfdRequired = false
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		domain.Spec.MemoryBacking = &api.MemoryBacking{
			HugePages: &api.HugePages{},
		}
		if val := vmi.Annotations[v1.MemfdMemoryBackend]; val != "false" {
			isMemfdRequired = true
		}
	}
	// virtiofs require shared access
	if util.IsVMIVirtiofsEnabled(vmi) {
		if domain.Spec.MemoryBacking == nil {
			domain.Spec.MemoryBacking = &api.MemoryBacking{}
		}
		domain.Spec.MemoryBacking.Access = &api.MemoryBackingAccess{
			Mode: "shared",
		}
		isMemfdRequired = true
	}

	if isMemfdRequired {
		// Set memfd as memory backend to solve SELinux restrictions
		// See the issue: https://github.com/kubevirt/kubevirt/issues/3781
		domain.Spec.MemoryBacking.Source = &api.MemoryBackingSource{Type: "memfd"}

		// NUMA is required in order to use memfd
		if domain.Spec.CPU.NUMA == nil {
			domain.Spec.CPU.NUMA = &api.NUMA{
				Cells: []api.NUMACell{
					{
						ID:     "0",
						CPUs:   fmt.Sprintf("0-%d", domain.Spec.VCPU.CPUs-1),
						Memory: uint64(vcpu.GetVirtualMemory(vmi).Value() / int64(1024)),
						Unit:   "KiB",
					},
				},
			}
		}
	}
	return nil
}
