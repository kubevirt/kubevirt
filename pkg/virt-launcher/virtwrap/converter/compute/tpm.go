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

type TPMDomainConfigurator struct {
	// stateBlockDevice, when non-empty, is the in-pod path of a raw block device
	// (e.g. /dev/vm-state-tpm) that backs the persistent vTPM state file directly via
	// swtpm's file:// backend. Set only when the second Block-mode backend-storage PVC
	// (persistent-tpm-state) is attached. When empty, libvirt manages the swtpm state
	// directory itself (the Filesystem-mode default).
	stateBlockDevice string
}

func NewTPMDomainConfigurator(stateBlockDevice string) TPMDomainConfigurator {
	return TPMDomainConfigurator{stateBlockDevice: stateBlockDevice}
}

func (t TPMDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if !tpm.HasDevice(&vmi.Spec) {
		return nil
	}

	newTPMDevice := api.TPM{
		Model: "tpm-tis",
		Backend: api.TPMBackend{
			Type:    "emulator",
			Version: "2.0",
		},
	}

	if tpm.HasPersistentDevice(&vmi.Spec) {
		newTPMDevice.Backend.PersistentState = "yes"

		// tpm-crb is not technically required for persistence, but since there was a desire for both,
		//   we decided to introduce them together. Ultimately, we should use tpm-crb for all cases,
		//   as it is now the generally preferred model
		newTPMDevice.Model = "tpm-crb"

		if t.stateBlockDevice != "" {
			// Block-mode persistent TPM: back the swtpm state with a raw block device
			// directly (no filesystem), symmetric to the EFI NVRAM pflash. swtpm opens
			// the device via its file:// backend; libvirt exposes it as:
			//   <backend type='emulator' version='2.0' persistent_state='yes'>
			//     <source type='file' path='/dev/vm-state-tpm'/>
			//   </backend>
			// State migration is orchestrated by libvirt/swtpm via persistent_state + the
			// swtpm migration stream, exactly as for the Filesystem-backed persistent TPM.
			newTPMDevice.Backend.Source = &api.TPMSource{
				Type: "file",
				Path: t.stateBlockDevice,
			}
		}
	}

	domain.Spec.Devices.TPMs = []api.TPM{newTPMDevice}
	return nil
}
