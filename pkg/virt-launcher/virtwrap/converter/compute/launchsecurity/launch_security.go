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

package launchsecurity

import (
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
)

type Configurator interface {
	ConfigureLaunchSecurity(vmi *v1.VirtualMachineInstance, domain *api.Domain)
}

func NewConfigurator(architecture string) Configurator {
	switch architecture {
	case "amd64":
		return amd64LaunchSecurityConfigurator{}
	case "arm64":
		return arm64LaunchSecurityConfigurator{}
	case "s390x":
		return s390xLaunchSecurityConfigurator{}
	default:
		return nil
	}
}

type amd64LaunchSecurityConfigurator struct{}

func (amd64LaunchSecurityConfigurator) ConfigureLaunchSecurity(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	domain.Spec.LaunchSecurity = amd64LaunchSecurity(vmi)
}

type arm64LaunchSecurityConfigurator struct{}

func (arm64LaunchSecurityConfigurator) ConfigureLaunchSecurity(_ *v1.VirtualMachineInstance, domain *api.Domain) {
	domain.Spec.LaunchSecurity = nil
}

type s390xLaunchSecurityConfigurator struct{}

func (s390xLaunchSecurityConfigurator) ConfigureLaunchSecurity(_ *v1.VirtualMachineInstance, domain *api.Domain) {
	// We would want to set launchsecurity with type "s390-pv" here, but this does not work in privileged pod.
	// Instead, virt-launcher will set iommu=on for all devices manually, which is the same action as what libvirt
	// would do when the launchsecurity type is set.
	domain.Spec.LaunchSecurity = nil
}

func amd64LaunchSecurity(vmi *v1.VirtualMachineInstance) *api.LaunchSecurity {
	launchSec := vmi.Spec.Domain.LaunchSecurity
	if launchSec.SEV == nil && launchSec.SNP != nil {
		snpPolicyBits := launchsecurity.SEVSNPPolicyToBits(launchSec.SNP)
		domain := &api.LaunchSecurity{
			Type: "sev-snp",
		}
		// Use Default Policy
		domain.Policy = "0x" + strconv.FormatUint(uint64(snpPolicyBits), 16)
		return domain
	} else if launchSec.SEV != nil {
		sevPolicyBits := launchsecurity.SEVPolicyToBits(launchSec.SEV.Policy)
		domain := &api.LaunchSecurity{
			Type:   "sev",
			Policy: "0x" + strconv.FormatUint(uint64(sevPolicyBits), 16),
		}
		if launchSec.SEV.DHCert != "" {
			domain.DHCert = launchSec.SEV.DHCert
		}
		if launchSec.SEV.Session != "" {
			domain.Session = launchSec.SEV.Session
		}
		return domain
	} else if launchSec.TDX != nil {
		qgsSocketPath := vmi.Annotations[v1.QGSSocketPathAnnotation]
		return &api.LaunchSecurity{
			Type:                   "tdx",
			QuoteGenerationService: &api.QGS{Path: qgsSocketPath},
		}
	}

	return nil
}
