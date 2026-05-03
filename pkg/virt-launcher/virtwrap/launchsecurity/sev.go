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
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	// Guest policy bits as defined in AMD SEV API specification
	SEVPolicyNoDebug        uint = 1 << 0
	SEVPolicyEncryptedState uint = 1 << 2

	// SEV-SNP guest policy as defined in AMD SEV-SNP API specification
	// Bits 0-7: ABI Minor version
	// Bits 8-15: ABI Major version
	// Bit 16: SMT allowed (0=disallowed, 1=allowed)
	//         NOTE: RHEL 9.7 kernel (5.14.0-611.36.1.el9_7) unconditionally requires this bit to be 1
	// Bit 17: Reserved must be set to 1 (RSVD_MBO)
	// Bit 18: Migration Agent allowed (0=disallowed, 1=allowed)
	// Bit 19: Debug allowed (0=disallowed, 1=allowed)
	// Bit 20: Single Socket (0=multi-socket, 1=single socket)
	//         NOTE: RHEL 9.7 kernel explicitly rejects this bit if set
	// Bit 21: Compute Express link (CXL) allowed (0=disallowed, 1=allowed)
	// Bit 22: AES-256 required (0=not required, 1=required)
	// Bit 23: RAPL enablement (0=RAPL does not need to be disabled, 1=RAPL must be disabled)
	// Bit 24: Ciphertest enablement (0=cipher test not required, 1=cipher test required)
	// Bit 25: Page sqp enablement (1=disables guest access to SNP_PAGE_MOVE, SNP_SWAP_OUT, and SNP_SWAP_IN instructions)
	SNPPolicySmt      uint = 1 << 16
	SNPPolicyReserved uint = 1 << 17
	SNPPolicyDefault       = "0x30000" // Default policy with SMT allowed (bit 16) and reserved MBO (bit 17)
	// Both bits 16 and 17 are required by RHEL 9.7 kernel validation (see arch/x86/kvm/svm/sev.c)
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
	//snpPolicyBits := SEVSNPPolicyToBits(snpConfig)
	domain := &api.LaunchSecurity{
		Type: "sev-snp",
	}
	if snpConfig != nil && snpConfig.Policy != "" {
		domain.LaunchSecuritySEVCommon.Policy = snpConfig.Policy
	} else {
		// Use Default Policy
		domain.LaunchSecuritySEVCommon.Policy = SNPPolicyDefault
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
