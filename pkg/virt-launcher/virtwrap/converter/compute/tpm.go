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
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/tpm"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type TPMDomainConfigurator struct{}

func (t TPMDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if tpm.HasDevice(&vmi.Spec) {
		domain.Spec.Devices.TPMs = []api.TPM{
			{
				Model: "tpm-tis",
				Backend: api.TPMBackend{
					Type:    "emulator",
					Version: "2.0",
				},
			},
		}
		if tpm.HasPersistentDevice(&vmi.Spec) {
			domain.Spec.Devices.TPMs[0].Backend.PersistentState = "yes"
			// tpm-crb is not techincally required for persistence, but since there was a desire for both,
			//   we decided to introduce them together. Ultimately, we should use tpm-crb for all cases,
			//   as it is now the generally preferred model
			domain.Spec.Devices.TPMs[0].Model = "tpm-crb"
		}
	}

	return nil
}
