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

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

type MemoryConfigurator struct {
}

func (r MemoryConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Memory == nil ||
		vmi.Spec.Domain.Memory.MaxGuest == nil ||
		(vmi.Spec.Domain.Memory.Guest != nil && vmi.Spec.Domain.Memory.Guest.Equal(*vmi.Spec.Domain.Memory.MaxGuest)) {
		var err error

		domain.Spec.Memory, err = vcpu.QuantityToByte(*vcpu.GetVirtualMemory(vmi))
		if err != nil {
			return err
		}
		return nil
	}

	maxMemory, err := vcpu.QuantityToByte(*vmi.Spec.Domain.Memory.MaxGuest)
	if err != nil {
		return err
	}

	domain.Spec.MaxMemory = &api.MaxMemory{
		Unit:  maxMemory.Unit,
		Value: maxMemory.Value,
	}

	currentMemory, err := vcpu.QuantityToByte(*vmi.Spec.Domain.Memory.Guest)
	if err != nil {
		return err
	}

	domain.Spec.Memory = currentMemory

	return nil
}
