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

package kvm_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/kvm"
)

var _ = Describe("Hypervisor Domain Configurator", func() {
	var (
		vmi    *v1.VirtualMachineInstance
		domain api.Domain
	)
	const (
		kvmEnabled       = true
		emulationAllowed = true
	)

	BeforeEach(func() {
		vmi = libvmi.New()
		domain = api.Domain{}
	})

	Context("When KVM is available", func() {
		It("Should not modify domain type", func() {
			configurator := kvm.NewKvmDomainConfigurator(!emulationAllowed, kvmEnabled)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal(""))
			Expect(domain).To(Equal(api.Domain{}))

		})

		It("Should not modify domain type even when emulation is allowed", func() {
			configurator := kvm.NewKvmDomainConfigurator(emulationAllowed, kvmEnabled)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal(""))
			Expect(domain).To(Equal(api.Domain{}))
		})
	})

	Context("When KVM is not available", func() {
		It("Should return error when emulation is not allowed", func() {
			configurator := kvm.NewKvmDomainConfigurator(!emulationAllowed, !kvmEnabled)
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("kvm not present and emulation is not allowed"))
		})

		It("Should set domain type to qemu when emulation is allowed", func() {
			configurator := kvm.NewKvmDomainConfigurator(emulationAllowed, !kvmEnabled)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal("qemu"))
		})
	})

	Context("Cross-architecture emulation", func() {
		const (
			crossArchAllowed = true
		)

		BeforeEach(func() {
			// Skip binary existence check in unit tests
			kvm.SetEmulatorBinaryExistsFunc(func(path string) error {
				return nil
			})
		})

		AfterEach(func() {
			kvm.ResetEmulatorBinaryExistsFunc()
		})

		It("Should set domain type to qemu and emulator path for cross-arch", func() {
			vmi.Spec.Architecture = "arm64"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, crossArchAllowed, "amd64",
			)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal("qemu"))
			Expect(domain.Spec.Devices.Emulator).To(Equal("/usr/bin/qemu-system-aarch64"))
		})

		It("Should return error when cross-arch feature gate is disabled", func() {
			vmi.Spec.Architecture = "arm64"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, !crossArchAllowed, "amd64",
			)
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MultiArchitectureSoftwareEmulation"))
		})

		It("Should succeed for cross-arch even when useEmulation is disabled", func() {
			vmi.Spec.Architecture = "arm64"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				!emulationAllowed, kvmEnabled, crossArchAllowed, "amd64",
			)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal("qemu"))
			Expect(domain.Spec.Devices.Emulator).To(Equal("/usr/bin/qemu-system-aarch64"))
		})

		It("Should not trigger cross-arch when guest matches host", func() {
			vmi.Spec.Architecture = "amd64"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, crossArchAllowed, "amd64",
			)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal(""))
		})

		It("Should not trigger cross-arch when guest architecture is empty", func() {
			vmi.Spec.Architecture = ""
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, crossArchAllowed, "amd64",
			)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal(""))
		})

		It("Should set correct emulator path for amd64 guest on arm64 host", func() {
			vmi.Spec.Architecture = "amd64"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, crossArchAllowed, "arm64",
			)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal("qemu"))
			Expect(domain.Spec.Devices.Emulator).To(Equal("/usr/bin/qemu-system-x86_64"))
		})

		It("Should set correct emulator path for s390x guest", func() {
			vmi.Spec.Architecture = "s390x"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, crossArchAllowed, "amd64",
			)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal("qemu"))
			Expect(domain.Spec.Devices.Emulator).To(Equal("/usr/bin/qemu-system-s390x"))
		})

		It("Should return error when emulator binary is missing", func() {
			kvm.SetEmulatorBinaryExistsFunc(func(path string) error {
				return fmt.Errorf("required emulator binary not found: %s", path)
			})

			vmi.Spec.Architecture = "arm64"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, crossArchAllowed, "amd64",
			)
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("required emulator binary not found"))
			Expect(err.Error()).To(ContainSubstring("qemu-system-aarch64"))
		})

		It("Should return error for unsupported guest architecture", func() {
			vmi.Spec.Architecture = "riscv64"
			configurator := kvm.NewKvmDomainConfiguratorWithCrossArch(
				emulationAllowed, kvmEnabled, crossArchAllowed, "amd64",
			)
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported guest architecture"))
		})
	})
})
