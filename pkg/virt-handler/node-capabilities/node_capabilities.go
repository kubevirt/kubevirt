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
 * Copyright the KubeVirt Authors.
 *
 */

package nodecapabilities

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"libvirt.org/go/libvirtxml"
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

func SupportedHostSEV(sev *libvirtxml.DomainCapsFeatureSEV) *SupportedSEV {
	supported := sev.Supported == "yes"
	return &SupportedSEV{
		Supported:   supported,
		SupportedES: supported && sev.MaxESGuests > 0,
	}
}

func SupportedHostCPUs(cpuModes []libvirtxml.DomainCapsCPUMode, arch archCapabilities) (*SupportedCPU, error) {
	var supportedCPU SupportedCPU
	for _, mode := range cpuModes {
		if mode.Name == v1.CPUModeHostModel {
			if !arch.supportsHostModel() {
				log.Log.Warningf("host-model cpu mode is not supported for %s architecture", arch.arch())
				continue
			}

			supportedCPU.Vendor = mode.Vendor
			if vendor := arch.defaultVendor(); vendor != "" {
				supportedCPU.Vendor = vendor
			}

			if len(mode.Models) < 1 {
				return nil, fmt.Errorf("host model mode is expected to contain a model")
			}
			if len(mode.Models) > 1 {
				log.Log.Warning("host model mode is expected to contain only one model")
			}

			supportedCPU.Model = mode.Models[0].Name
			supportedCPU.Fallback = mode.Models[0].Fallback

			for _, feature := range mode.Features {
				if feature.Policy == "require" {
					supportedCPU.RequiredFeatures = append(supportedCPU.RequiredFeatures, feature.Name)
				}
			}
		}

		if arch.supportsNamedModels() {
			for _, model := range mode.Models {
				if model.Usable == "no" || model.Usable == "" {
					continue
				}
				supportedCPU.UsableModels = append(supportedCPU.UsableModels, model.Name)
			}
		}
	}
	return &supportedCPU, nil
}

func SupportedFeatures(supportedFeaturesXML string, arch archCapabilities) ([]string, error) {
	var supportedFeatures []string
	if !arch.hasHostSupportedFeatures() {
		return supportedFeatures, nil
	}

	var cpu libvirtxml.DomainCPU
	if err := cpu.Unmarshal(string(supportedFeaturesXML)); err != nil {
		return nil, err
	}

	for _, feature := range cpu.Features {
		// On s390x, the policy is not set
		if arch.requirePolicy(feature.Policy) {
			supportedFeatures = append(supportedFeatures, feature.Name)
		}
	}
	return supportedFeatures, nil

}
