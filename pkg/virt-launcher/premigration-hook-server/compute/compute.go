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

package compute

import (
	"encoding/json"
	"encoding/xml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"libvirt.org/go/libvirtxml"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	convxml "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/libvirtxml"
)

func GetComputeHooks() []types.HookFunc {
	return []types.HookFunc{
		NewCPUDedicatedHook(util.GetPodCPUSet),
	}
}

// NewCPUDedicatedHook creates a CPU dedicated hook with the provided cpuSetGetter function
func NewCPUDedicatedHook(cpuSetGetter func() ([]int, error)) types.HookFunc {
	return func(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) {
		cpuDedicatedHook(vmi, domain, cpuSetGetter)
	}
}

func cpuDedicatedHook(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain, cpuSetGetter func() ([]int, error)) {
	if !vmi.IsCPUDedicated() {
		return
	}
	// If the VMI has dedicated CPUs, we need to replace the old CPUs that were
	// assigned in the source node with the new CPUs assigned in the target node
	xmlstr, err := domain.Marshal()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to marshal domain to XML")
		return
	}

	// First, unmarshal the XML into api.Domain to populate most fields automatically
	var apiDomainSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &apiDomainSpec)
	if err != nil {
		log.Log.Errorf("Failed to unmarshal XML to api.Domain: %v", err.Error())
		return
	}
	cpuSet, err := cpuSetGetter()
	if err != nil {
		log.Log.Errorf("Failed to get cpuSet: %v", err.Error())
		return
	}
	log.Log.Infof("wallak shuuu %v", cpuSet)
	processedDomain, err := generateDomainForTargetCPUSetAndTopology(vmi, &apiDomainSpec, cpuSet)
	if err != nil {
		log.Log.Errorf("Failed to generate domain for target CPU set and topology: %v", err.Error())
		return
	}

	if err = convertCPUDedicatedFields(processedDomain, domain); err != nil {
		log.Log.Errorf("Failed to convert CPU dedicated fields: %v", err.Error())
		return
	}

	log.Log.Object(vmi).Info("CPUDedicatedHook: CPU dedicated processing completed")
}

func generateDomainForTargetCPUSetAndTopology(vmi *v1.VirtualMachineInstance, domSpec *api.DomainSpec, cpuSet []int) (*api.Domain, error) {
	var targetTopology cmdv1.Topology
	err := json.Unmarshal([]byte(vmi.Status.MigrationState.TargetNodeTopology), &targetTopology)
	if err != nil {
		return nil, err
	}

	useIOThreads := false
	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			useIOThreads = true
			break
		}
	}

	domain := api.NewMinimalDomain(vmi.Name)
	domain.Spec = *domSpec
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)

	// update cpu count to maximum hot plugable CPUs
	vmiCPU := vmi.Spec.Domain.CPU
	if vmiCPU != nil && vmiCPU.MaxSockets != 0 {
		cpuTopology.Sockets = vmiCPU.MaxSockets
		cpuCount = vcpu.CalculateRequestedVCPUs(cpuTopology)
	}
	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
	err = vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, &targetTopology, cpuSet, useIOThreads)
	if err != nil {
		return nil, err
	}

	return domain, err
}

func convertCPUDedicatedFields(domain *api.Domain, domcfg *libvirtxml.Domain) error {
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
