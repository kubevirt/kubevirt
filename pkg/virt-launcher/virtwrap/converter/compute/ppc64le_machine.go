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

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// PPC64LEMachineDomainConfigurator is only registered for ppc64le guests. It disables the pseries large decrementer capability.
//
// QEMU probes /proc/device-tree/cpus/ to decide whether the large decrementer is
// available. Container runtimes mask /sys/firmware (which /proc/device-tree points to)
// for unprivileged containers, so the probe fails with "No large decrementer support".
// Turning the capability off removes the need for a privileged compute container.
// libvirt has no domain XML element for this capability, hence the QEMU argument.
type PPC64LEMachineDomainConfigurator struct{}

func (PPC64LEMachineDomainConfigurator) Configure(_ *v1.VirtualMachineInstance, domain *api.Domain) error {
	initializeQEMUCmdAndQEMUArg(domain)
	domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg,
		api.Arg{Value: "-machine"},
		api.Arg{Value: "cap-large-decr=off"},
	)

	return nil
}
