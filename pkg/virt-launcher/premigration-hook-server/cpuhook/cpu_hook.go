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

package cpuhook

import (
	"encoding/xml"
	"fmt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cpudedicated"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// CPUDedicatedHook handles CPU pinning adjustments for dedicated CPU migrations
func CPUDedicatedHook(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if !vmi.IsCPUDedicated() {
		return nil
	}
	// If the VMI has dedicated CPUs, we need to replace the old CPUs that were
	// assigned in the source node with the new CPUs assigned in the target node
	xmlstr, err := domain.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal domain to XML: %w", err)
	}

	var apiDomainSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &apiDomainSpec)
	if err != nil {
		return fmt.Errorf("failed to unmarshal XML to api.DomainSpec: %w", err)
	}

	processedDomain, err := cpudedicated.GenerateDomainForTargetCPUSetAndTopology(vmi, &apiDomainSpec)
	if err != nil {
		return fmt.Errorf("failed to generate domain for target CPU set and topology: %w", err)
	}

	if err = cpudedicated.ConvertCPUDedicatedFields(processedDomain, domain); err != nil {
		return fmt.Errorf("failed to convert CPU dedicated fields: %w", err)
	}

	log.Log.Object(vmi).Info("cpuDedicatedHook: CPU dedicated processing completed")
	return nil
}
