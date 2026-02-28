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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
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
			configurator := NewKvmDomainConfigurator(!emulationAllowed, kvmEnabled)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal(""))
			Expect(domain).To(Equal(api.Domain{}))

		})

		It("Should not modify domain type even when emulation is allowed", func() {
			configurator := NewKvmDomainConfigurator(emulationAllowed, kvmEnabled)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal(""))
			Expect(domain).To(Equal(api.Domain{}))
		})
	})

	Context("When KVM is not available", func() {
		It("Should return error when emulation is not allowed", func() {
			configurator := NewKvmDomainConfigurator(!emulationAllowed, !kvmEnabled)
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("kvm not present"))
		})

		It("Should set domain type to qemu when emulation is allowed", func() {
			configurator := NewKvmDomainConfigurator(emulationAllowed, !kvmEnabled)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Type).To(Equal("qemu"))
		})
	})
})
