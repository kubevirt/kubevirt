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
 * Copyright 2021
 *
 */

package launchsecurity

import (
	"fmt"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	// Guest policy bits as defined in AMD SEV API specification
	SEVPolicyNoDebug        uint = 1 << 0
	SEVPolicyEncryptedState uint = 1 << 2

	// SEV-SNP guest policy as defined in AMD SEV-SNP API specification
	SNPPolicySmt      uint = 1 << 16
	SNPPolicyReserved uint = 1 << 17
)

func SEVPolicyToBits(policy *v1.SEVPolicy) uint {
	// NoDebug is always true for SEV and SEV-ES
	bits := uint(SEVPolicyNoDebug)

	if policy != nil {
		if policy.EncryptedState != nil && *policy.EncryptedState {
			bits = bits | SEVPolicyEncryptedState
		}
	}

	return bits
}

func SEVSNPPolicyToBits(snpConfig *v1.SEVSNP) uint {
	if snpConfig != nil && snpConfig.Policy == "" {
		// Default policy: SMT allowed (bit 16) and reserved MBO (bit 17)
		// Required by AMD SEV-SNP specification and enforced by RHEL 9.7 kernel validation
		return SNPPolicySmt | SNPPolicyReserved
	}
	return 0
}

func ConfigureSEVSNPLaunchSecurity(snpConfig *v1.SEVSNP) *api.LaunchSecurity {
	domain := &api.LaunchSecurity{Type: "sev-snp"}

	if snpConfig != nil && snpConfig.Policy != "" {
		// Normalize the policy to hex string so the value handed to
		// libvirt matches what the webhook validated, which accepts
		// dec and hex values.
		if dec, err := strconv.ParseUint(snpConfig.Policy, 0, 64); err == nil {
			domain.LaunchSecuritySEVCommon.Policy = fmt.Sprintf("0x%x", dec)
		} else {
			domain.LaunchSecuritySEVCommon.Policy = snpConfig.Policy
		}
	} else {
		// Use Default Policy
		domain.LaunchSecuritySEVCommon.Policy = fmt.Sprintf("0x%x", SEVSNPPolicyToBits(snpConfig))
	}

	// All remaining fields live on snpConfig; nothing more to copy when it's nil
	if snpConfig == nil {
		return domain
	}

	if snpConfig.VCEK != "" {
		domain.LaunchSecuritySNP.VCEK = snpConfig.VCEK
	}
	if snpConfig.AuthorKey != nil && *snpConfig.AuthorKey {
		domain.LaunchSecuritySNP.AuthorKey = snpConfig.AuthorKey
	}
	if snpConfig.KernelHashes != nil {
		// libvirt's takes KernelHashes as "yes"/"no"
		if *snpConfig.KernelHashes {
			domain.LaunchSecuritySEVCommon.KernelHashes = "yes"
		} else {
			domain.LaunchSecuritySEVCommon.KernelHashes = "no"
		}
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
	if snpConfig.HostData != "" {
		domain.LaunchSecuritySNP.HostData = snpConfig.HostData
	}
	if snpConfig.GuestVisibleWorkarounds != "" {
		domain.LaunchSecuritySNP.GuestVisibleWorkarounds = snpConfig.GuestVisibleWorkarounds
	}
	return domain
}

func ConfigureSEVLaunchSecurity(sevConfig *v1.SEV) *api.LaunchSecurity {
	sevPolicyBits := SEVPolicyToBits(sevConfig.Policy)
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
