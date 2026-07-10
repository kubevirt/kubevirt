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
	v1 "kubevirt.io/api/core/v1"

	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MemoryBackingConfigurator struct {
	isMemfdSupported bool
}

func NewMemoryBackingConfigurator(isMemfdSupported bool) MemoryBackingConfigurator {
	return MemoryBackingConfigurator{isMemfdSupported: isMemfdSupported}
}

func (c MemoryBackingConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	hasHugepages := vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil
	needsSharedAccess := util.IsVMIVirtiofsEnabled(vmi) || netvmispec.HasPasstBinding(vmi)

	if !hasHugepages && !needsSharedAccess {
		return nil
	}

	mb := &api.MemoryBacking{}
	if hasHugepages {
		mb.HugePages = &api.HugePages{}
	}
	if needsSharedAccess {
		mb.Access = &api.MemoryBackingAccess{Mode: "shared"}
	}
	if c.isMemfdSupported && isMemfdRequired(vmi) {
		mb.Source = &api.MemoryBackingSource{Type: "memfd"}
	}
	domain.Spec.MemoryBacking = mb

	return nil
}

func isMemfdRequired(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		if vmi.Annotations[v1.MemfdMemoryBackend] != "false" {
			return true
		}
	}
	return util.IsVMIVirtiofsEnabled(vmi) || netvmispec.HasPasstBinding(vmi)
}
