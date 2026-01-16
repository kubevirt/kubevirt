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

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type IOMMUDomainConfigurator struct {
	architecture string
}

func NewIOMMUDomainConfigurator(architecture string) IOMMUDomainConfigurator {
	return IOMMUDomainConfigurator{architecture: architecture}
}

func (i IOMMUDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	vmiIOMMU := vmi.Spec.Domain.Devices.IOMMU
	if vmiIOMMU == nil {
		return nil
	}

	// Validate model based on architecture
	model := vmiIOMMU.Model
	if model == "" {
		model = "smmuv3" // default for arm64
	}

	if model == "smmuv3" && i.architecture != "arm64" {
		return fmt.Errorf("IOMMU model smmuv3 is only supported on arm64 architecture")
	}

	iommu := &api.IOMMU{
		Model: model,
	}

	// Configure driver options if specified
	if vmiIOMMU.Driver != nil {
		iommu.Driver = &api.IOMMUDriver{
			PCIBus: vmiIOMMU.Driver.PCIBus,
		}
	}

	domain.Spec.Devices.IOMMU = iommu
	return nil
}
