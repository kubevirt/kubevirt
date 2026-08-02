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
 */

package cpudedicated

import (
	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convxml "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/libvirtxml"
)

func ConvertCPUDedicatedFields(domain *api.Domain, domcfg *libvirtxml.Domain) error {
	if domcfg.CPU == nil {
		domcfg.CPU = &libvirtxml.DomainCPU{}
	}
	domcfg.CPU.Topology = convxml.ConvertKubeVirtCPUTopologyToDomainCPUTopology(domain.Spec.CPU.Topology)
	domcfg.VCPU = convxml.ConvertKubeVirtVCPUToDomainVCPU(domain.Spec.VCPU)
	domcfg.CPUTune = convxml.ConvertKubeVirtCPUTuneToDomainCPUTune(domain.Spec.CPUTune)
	domcfg.NUMATune = convxml.ConvertKubeVirtNUMATuneToDomainNUMATune(domain.Spec.NUMATune)
	domcfg.Features = convxml.ConvertKubeVirtFeaturesToDomainFeatureList(domain.Spec.Features)

	return nil
}
