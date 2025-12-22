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

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type OSDomainConfigurator struct {
	isSMBiosNeeded bool
}

func NewOSDomainConfigurator(isSMBiosNeeded bool) OSDomainConfigurator {
	return OSDomainConfigurator{
		isSMBiosNeeded: isSMBiosNeeded,
	}
}

func (o OSDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if o.isSMBiosNeeded {
		domain.Spec.OS.SMBios = &api.SMBios{
			Mode: "sysinfo",
		}
	}

	if machine := vmi.Spec.Domain.Machine; machine != nil {
		domain.Spec.OS.Type.Machine = machine.Type
	}

	// set bootmenu to give time to access bios
	if vmi.ShouldStartPaused() {
		const bootMenuTimeoutMS = uint(10000)
		domain.Spec.OS.BootMenu = &api.BootMenu{
			Enable:  "yes",
			Timeout: pointer.P(bootMenuTimeoutMS),
		}
	}

	return nil
}
