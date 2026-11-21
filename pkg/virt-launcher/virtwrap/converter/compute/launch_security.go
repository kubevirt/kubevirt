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
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
	"kubevirt.io/kubevirt/pkg/vmitrait"
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
		launchSecurity, err := amd64LaunchSecurity(vmi)
		if err != nil {
			return err
		}
		domain.Spec.LaunchSecurity = launchSecurity
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

func amd64LaunchSecurity(vmi *v1.VirtualMachineInstance) (*api.LaunchSecurity, error) {
	launchSec := vmi.Spec.Domain.LaunchSecurity

	// Admission already guarantees the types are mutually exclusive, but re-check
	if types := util.LaunchSecurityTypes(launchSec); len(types) > 1 {
		return nil, fmt.Errorf("only one launchSecurity type may be set, got %v", types)
	}

	switch {
	case launchSec.SEV != nil:
		sevPolicyBits := launchsecurity.SEVPolicyToBits(launchSec.SEV.Policy)
		return &api.LaunchSecurity{
			Type:    "sev",
			Policy:  "0x" + strconv.FormatUint(uint64(sevPolicyBits), 16),
			DHCert:  launchSec.SEV.DHCert,
			Session: launchSec.SEV.Session,
		}, nil
	case launchSec.SNP != nil:
		// The AMD firmware rejects a direct kernel boot unless the kernel, initrd
		// and cmdline hashes are measured into the launch digest.
		kernelHashes := ""
		if vmitrait.HasDirectKernelBoot(vmi) {
			kernelHashes = "yes"
		}

		// Use Default Policy
		snpPolicyBits := launchsecurity.SEVSNPPolicyToBits(launchSec.SNP)
		return &api.LaunchSecurity{
			Type:         "sev-snp",
			Policy:       "0x" + strconv.FormatUint(uint64(snpPolicyBits), 16),
			KernelHashes: kernelHashes,
		}, nil
	case launchSec.TDX != nil:
		return &api.LaunchSecurity{
			Type:                   "tdx",
			QuoteGenerationService: &api.QGS{Path: vmi.Annotations[v1.QGSSocketPathAnnotation]},
		}, nil
	}

	// Fail closed: never return a domain without the requested launch security
	return nil, fmt.Errorf("no launchSecurity type set")
}
