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

package kvm

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type KvmDomainConfigurator struct {
	allowEmulation bool
	kvmAvailable   bool
}

// NewKvmDomainConfigurator creates a new hypervisor domain configurator
func NewKvmDomainConfigurator(allowEmulation bool, kvmAvailable bool) KvmDomainConfigurator {
	return KvmDomainConfigurator{
		allowEmulation: allowEmulation,
		kvmAvailable:   kvmAvailable,
	}
}

// Configure configures the domain hypervisor settings based on KVM availability and emulation settings
func (k KvmDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if !k.kvmAvailable {
		if k.allowEmulation {
			logger := log.DefaultLogger()
			logger.Infof("kvm not present. Using software emulation.")
			domain.Spec.Type = "qemu"
		} else {
			return fmt.Errorf("kvm not present")
		}
	}

	return nil
}
