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
	"os"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var defaultEmulatorBinaryExists = func(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("required emulator binary not found: %s", path)
	}
	return nil
}

var emulatorBinaryExists = defaultEmulatorBinaryExists

// SetEmulatorBinaryExistsFunc overrides the emulator binary existence check (for testing).
func SetEmulatorBinaryExistsFunc(f func(string) error) {
	emulatorBinaryExists = f
}

// ResetEmulatorBinaryExistsFunc restores the default emulator binary existence check.
func ResetEmulatorBinaryExistsFunc() {
	emulatorBinaryExists = defaultEmulatorBinaryExists
}

type KvmDomainConfigurator struct {
	allowEmulation          bool
	kvmAvailable            bool
	allowCrossArchEmulation bool
	hostArchitecture        string
}

// NewKvmDomainConfigurator creates a new hypervisor domain configurator
func NewKvmDomainConfigurator(allowEmulation bool, kvmAvailable bool) KvmDomainConfigurator {
	return KvmDomainConfigurator{
		allowEmulation: allowEmulation,
		kvmAvailable:   kvmAvailable,
	}
}

// NewKvmDomainConfiguratorWithCrossArch creates a new hypervisor domain configurator with cross-architecture emulation support
func NewKvmDomainConfiguratorWithCrossArch(allowEmulation, kvmAvailable, allowCrossArchEmulation bool, hostArchitecture string) KvmDomainConfigurator {
	return KvmDomainConfigurator{
		allowEmulation:          allowEmulation,
		kvmAvailable:            kvmAvailable,
		allowCrossArchEmulation: allowCrossArchEmulation,
		hostArchitecture:        hostArchitecture,
	}
}

// Configure configures the domain hypervisor settings based on KVM availability and emulation settings
func (k KvmDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	crossArchEmulation := k.isCrossArchEmulation(vmi)

	if !k.kvmAvailable || crossArchEmulation {
		if !k.allowEmulation {
			return fmt.Errorf("kvm not present or cross-architecture emulation requested, but emulation is not allowed")
		}

		if crossArchEmulation && !k.allowCrossArchEmulation {
			return fmt.Errorf("cross-architecture emulation requires the CrossArchitectureVirtualization feature gate")
		}

		logger := log.DefaultLogger()
		if crossArchEmulation {
			logger.Infof("Cross-architecture emulation: host=%s, guest=%s. Using software emulation.",
				k.hostArchitecture, vmi.Spec.Architecture)
		} else {
			logger.Infof("kvm not present. Using software emulation.")
		}

		domain.Spec.Type = "qemu"

		if crossArchEmulation {
			path := emulatorPath(vmi.Spec.Architecture)
			if path == "" {
				return fmt.Errorf("unsupported guest architecture for cross-architecture emulation: %s", vmi.Spec.Architecture)
			}
			if err := emulatorBinaryExists(path); err != nil {
				return err
			}
			domain.Spec.Devices.Emulator = path
		}
	}

	return nil
}

func (k KvmDomainConfigurator) isCrossArchEmulation(vmi *v1.VirtualMachineInstance) bool {
	if k.hostArchitecture == "" || vmi.Spec.Architecture == "" {
		return false
	}
	return vmi.Spec.Architecture != k.hostArchitecture
}

func emulatorPath(guestArch string) string {
	switch guestArch {
	case "arm64", "aarch64":
		return "/usr/bin/qemu-system-aarch64"
	case "amd64", "x86_64":
		return "/usr/bin/qemu-system-x86_64"
	case "s390x":
		return "/usr/bin/qemu-system-s390x"
	default:
		return ""
	}
}
