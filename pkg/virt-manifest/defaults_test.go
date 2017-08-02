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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virt_manifest

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Defaults", func() {
	Context("Default Domains", func() {
		var dom *v1.DomainSpec
		BeforeEach(func() {
			dom = new(v1.DomainSpec)
		})

		It("Shouldn't add graphics devices", func() {
			AddMinimalDomainSpec(dom)
			Expect(len(dom.Devices.Graphics)).To(Equal(0))
		})

		It("Should ensure spice devices have proper defaults", func() {
			dom.Devices.Graphics = []v1.Graphics{v1.Graphics{Type: "spice"}}
			// Ensure there's no listen address info
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal(""))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal(""))

			AddMinimalDomainSpec(dom)
			// Ensure the listen address has been added
			Expect(dom.Devices.Graphics[0].Type).To(Equal("spice"))
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal("address"))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal("0.0.0.0"))
		})

		It("Should ensure spice devices have proper defaults", func() {
			dom.Devices.Graphics = []v1.Graphics{v1.Graphics{Type: "spice",
				Listen: v1.Listen{Type: "address"}}}
			// Ensure there's no listen address info
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal("address"))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal(""))

			AddMinimalDomainSpec(dom)
			// Ensure the listen address has been added
			Expect(dom.Devices.Graphics[0].Type).To(Equal("spice"))
			Expect(dom.Devices.Graphics[0].Listen.Type).To(Equal("address"))
			Expect(dom.Devices.Graphics[0].Listen.Address).To(Equal("0.0.0.0"))
		})
	})

	Context("Default VMs", func() {
		var vm *v1.VM
		BeforeEach(func() {
			vm = tests.NewRandomVM()
		})

		It("should assign a random name to domain spec", func() {
			vm.Spec.Domain = nil
			AddMinimalVMSpec(vm)
			Expect(vm.GetObjectMeta().GetName()).ToNot(BeNil())
		})

		It("should create missing domain", func() {
			vm.Spec.Domain = nil
			AddMinimalVMSpec(vm)

			Expect(vm.Spec.Domain).ToNot(BeNil())
		})

		It("should not modify an existing domain", func() {
			vm.Spec.Domain = &v1.DomainSpec{Type: "invalid"}

			AddMinimalVMSpec(vm)
			Expect(vm.Spec.Domain.Type).To(Equal("invalid"))
		})
	})
})

func TestDefaults(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Defaults")
}
