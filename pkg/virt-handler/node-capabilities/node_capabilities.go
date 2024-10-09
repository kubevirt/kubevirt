/*
Copyright 2024 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodecapabilities

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"libvirt.org/go/libvirtxml"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	CapabilitiesVolumePath    = "/var/lib/kubevirt-node-labeller/"
	HostCapabilitiesFilename  = "capabilities.xml"
	DomCapabiliitesFilename   = "virsh_domcapabilities.xml"
	SupportedFeaturesFilename = "supported_features.xml"

	RequirePolicy = "require"
	KVMPath       = "/dev/kvm"
	VmxFeature    = "vmx"
)

type SupportedCPU struct {
	Vendor           string
	Model            string
	RequiredFeatures []string
	UsableModels     []string
}

type SupportedSEV struct {
	Supported   bool
	SupportedES bool
}

func HostCapabilities(hostCapabilities string) (*libvirtxml.Caps, error) {
	var capabilities libvirtxml.Caps
	if err := capabilities.Unmarshal(hostCapabilities); err != nil {
		return nil, err
	}
	return &capabilities, nil
}

func DomCapabilities(hostDomCapabilities string) (*libvirtxml.DomainCaps, error) {
	var capabilities libvirtxml.DomainCaps
	if err := capabilities.Unmarshal(hostDomCapabilities); err != nil {
		return nil, err
	}
	return &capabilities, nil
}

func SupportedHostCPUs(cpuModes []libvirtxml.DomainCapsCPUMode, arch string) (*SupportedCPU, error) {
	var supportedCPU SupportedCPU
	for _, mode := range cpuModes {
		if mode.Name == v1.CPUModeHostModel {
			if virtconfig.IsARM64(arch) {
				log.Log.Warning("host-model cpu mode is not supported for ARM architecture")
				continue
			}

			supportedCPU.Vendor = mode.Vendor
			// On s390x the xml does not include a CPU Vendor, however there is only one company selling them anyway.
			if virtconfig.IsS390X(arch) && mode.Vendor == "" {
				supportedCPU.Vendor = "IBM"
			}

			if len(mode.Models) < 1 {
				return nil, fmt.Errorf("host model mode is expected to contain a model")
			}
			if len(mode.Models) > 1 {
				log.Log.Warning("host model mode is expected to contain only one model")
			}

			supportedCPU.Model = mode.Models[0].Name
			for _, feature := range mode.Features {
				if feature.Policy == "require" {
					supportedCPU.RequiredFeatures = append(supportedCPU.RequiredFeatures, feature.Name)
				}
			}
		}

		for _, model := range mode.Models {
			if model.Usable == "no" || model.Usable == "" {
				continue
			}
			supportedCPU.UsableModels = append(supportedCPU.UsableModels, model.Name)
		}
	}
	return &supportedCPU, nil
}

func SupportedHostSEV(sev *libvirtxml.DomainCapsFeatureSEV) *SupportedSEV {
	supported := sev.Supported == "yes"
	return &SupportedSEV{
		Supported:   supported,
		SupportedES: supported && sev.MaxESGuests > 0,
	}
}

func SupportedFeatures(hostCPU string, arch string) ([]string, error) {
	var cpu libvirtxml.DomainCPU
	if err := cpu.Unmarshal(hostCPU); err != nil {
		return nil, err
	}
	supportedFeatures := make([]string, 0)
	for _, feature := range cpu.Features {
		// On s390x, the policy is not set
		if feature.Policy == "require" || (virtconfig.IsS390X(arch) && feature.Policy == "") {
			supportedFeatures = append(supportedFeatures, feature.Name)
		}
	}
	return supportedFeatures, nil

}
