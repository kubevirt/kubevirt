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
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
)

type LaunchSecurityDomainConfigurator struct {
	architecture string
}

func NewLaunchSecurityDomainConfigurator(architecture string) LaunchSecurityDomainConfigurator {
	return LaunchSecurityDomainConfigurator{
		architecture: architecture,
	}
}

func (l LaunchSecurityDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if !util.UseLaunchSecurity(vmi) {
		return nil
	}

	switch l.architecture {
	case "amd64":
		domain.Spec.LaunchSecurity = Amd64LaunchSecurity(vmi)
	case "arm64":
		domain.Spec.LaunchSecurity = nil
	case "s390x":
		// We would want to set launchsecurity with type "s390-pv" here, but this does not work in privileged pod.
		// Instead, virt-launcher will set iommu=on for all devices manually, which is the same action as what libvirt
		// would do when the launchsecurity type is set.
		domain.Spec.LaunchSecurity = nil
	}

	return nil
}

func ConfigureSEVSNPLaunchSecurity(snpConfig *v1.SEVSNP) *api.LaunchSecurity {
	snpPolicyBits := launchsecurity.SEVSNPPolicyToBits(snpConfig)
	domain := &api.LaunchSecurity{
		Type: "sev-snp",
	}
	if snpConfig.Policy != "" {
		domain.LaunchSecuritySEVCommon.Policy = snpConfig.Policy
	} else {
		// Use Default Policy
		domain.LaunchSecuritySEVCommon.Policy = "0x" + strconv.FormatUint(uint64(snpPolicyBits), 16)
	}
	if snpConfig.VCEK != "" {
		domain.LaunchSecuritySNP.VCEK = snpConfig.VCEK
	}
	if snpConfig.AuthorKey != "" {
		domain.LaunchSecuritySNP.AuthorKey = snpConfig.AuthorKey
	}
	if snpConfig.KernelHashes != "" {
		domain.LaunchSecuritySEVCommon.KernelHashes = snpConfig.KernelHashes
	}
	if snpConfig.Cbitpos != "" {
		domain.LaunchSecuritySEVCommon.Cbitpos = snpConfig.Cbitpos
	}
	if snpConfig.IdBlock != "" {
		domain.LaunchSecuritySNP.IdBlock = snpConfig.IdBlock
	}
	if snpConfig.IdAuth != "" {
		domain.LaunchSecuritySNP.IdAuth = snpConfig.IdAuth
	}
	if snpConfig.ReducedPhysBits != "" {
		domain.LaunchSecuritySEVCommon.ReducedPhysBits = snpConfig.ReducedPhysBits
	}
	return domain
}

func ConfigureSEVLaunchSecurity(sevConfig *v1.SEV) *api.LaunchSecurity {
	sevPolicyBits := launchsecurity.SEVPolicyToBits(sevConfig.Policy)
	domain := &api.LaunchSecurity{
		Type: "sev",
		LaunchSecuritySEVCommon: api.LaunchSecuritySEVCommon{
			Policy: "0x" + strconv.FormatUint(uint64(sevPolicyBits), 16),
		},
	}
	if sevConfig.DHCert != "" {
		domain.LaunchSecuritySEV.DHCert = sevConfig.DHCert
	}
	if sevConfig.Session != "" {
		domain.LaunchSecuritySEV.Session = sevConfig.Session
	}
	return domain
}

func Amd64LaunchSecurity(vmi *v1.VirtualMachineInstance) *api.LaunchSecurity {
	launchSec := vmi.Spec.Domain.LaunchSecurity
	if launchSec.SNP != nil {
		return ConfigureSEVSNPLaunchSecurity(launchSec.SNP)
	} else if launchSec.SEV != nil {
		return ConfigureSEVLaunchSecurity(launchSec.SEV)
	} else if launchSec.TDX != nil {
		return &api.LaunchSecurity{
			Type: "tdx",
		}
	}

	return nil
}
