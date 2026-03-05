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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/hypervisor/kvm"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Hypervisor Domain Configurator", func() {
	var (
		domain api.Domain
	)
	const (
		emulationAllowed = true
	)

	BeforeEach(func() {
		domain = api.Domain{}
	})

	It("Should not modify domain type when emulation is not allowed", func() {
		configurator := kvm.NewKvmDomainConfigurator(!emulationAllowed)
		Expect(configurator.Configure(libvmi.New(), &domain)).To(Succeed())
		Expect(domain.Spec.Type).To(Equal(""))
		Expect(domain).To(Equal(api.Domain{}))

	})

	It("Should modify domain type when emulation is allowed", func() {
		configurator := kvm.NewKvmDomainConfigurator(emulationAllowed)
		Expect(configurator.Configure(libvmi.New(), &domain)).To(Succeed())
		Expect(domain.Spec.Type).To(Equal("qemu"))
	})
})
