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

package hypervisor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Hypervisor", func() {
	Context("NewHypervisor factory", func() {
		It("should return KVMHypervisor for 'kvm' input", func() {
			hypervisor := NewHypervisor("kvm")
			Expect(hypervisor).To(BeAssignableToTypeOf(&KVMHypervisor{}))
		})

		It("should return HyperVLayeredHypervisor for 'hyperv-layered' input", func() {
			hypervisor := NewHypervisor("hyperv-layered")
			Expect(hypervisor).To(BeAssignableToTypeOf(&HyperVLayeredHypervisor{}))
		})

		It("should return KVMHypervisor as default for unknown input", func() {
			hypervisor := NewHypervisor("unknown")
			Expect(hypervisor).To(BeAssignableToTypeOf(&KVMHypervisor{}))
		})

		It("should return KVMHypervisor as default for empty string", func() {
			hypervisor := NewHypervisor("")
			Expect(hypervisor).To(BeAssignableToTypeOf(&KVMHypervisor{}))
		})

		It("should handle case sensitivity correctly", func() {
			hypervisor := NewHypervisor("KVM")
			Expect(hypervisor).To(BeAssignableToTypeOf(&KVMHypervisor{}))

			hypervisor = NewHypervisor("HyperV-Layered")
			Expect(hypervisor).To(BeAssignableToTypeOf(&KVMHypervisor{}))
		})
	})

	Context("KVMHypervisor", func() {
		var kvmHypervisor *KVMHypervisor
		var vmi *v1.VirtualMachineInstance
		var domain *api.Domain

		BeforeEach(func() {
			kvmHypervisor = &KVMHypervisor{}
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "test-namespace",
				},
			}
			domain = &api.Domain{
				Spec: api.DomainSpec{
					Type: "original-type",
				},
			}
		})

		It("should not modify the domain", func() {
			originalType := domain.Spec.Type
			kvmHypervisor.AdjustDomain(vmi, domain)
			Expect(domain.Spec.Type).To(Equal(originalType))
		})

		It("should handle nil VMI gracefully", func() {
			originalType := domain.Spec.Type
			Expect(func() {
				kvmHypervisor.AdjustDomain(nil, domain)
			}).NotTo(Panic())
			Expect(domain.Spec.Type).To(Equal(originalType))
		})

		It("should handle nil domain gracefully", func() {
			Expect(func() {
				kvmHypervisor.AdjustDomain(vmi, nil)
			}).NotTo(Panic())
		})

		It("should handle both nil VMI and domain gracefully", func() {
			Expect(func() {
				kvmHypervisor.AdjustDomain(nil, nil)
			}).NotTo(Panic())
		})
	})

	Context("HyperVLayeredHypervisor", func() {
		var hypervHypervisor *HyperVLayeredHypervisor
		var vmi *v1.VirtualMachineInstance
		var domain *api.Domain

		BeforeEach(func() {
			hypervHypervisor = &HyperVLayeredHypervisor{}
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "test-namespace",
				},
			}
			domain = &api.Domain{
				Spec: api.DomainSpec{
					Type: "original-type",
				},
			}
		})

		It("should set domain type to 'hyperv'", func() {
			hypervHypervisor.AdjustDomain(vmi, domain)
			Expect(domain.Spec.Type).To(Equal("hyperv"))
		})

		It("should override existing domain type", func() {
			domain.Spec.Type = "kvm"
			hypervHypervisor.AdjustDomain(vmi, domain)
			Expect(domain.Spec.Type).To(Equal("hyperv"))
		})

		It("should set domain type even when original type is empty", func() {
			domain.Spec.Type = ""
			hypervHypervisor.AdjustDomain(vmi, domain)
			Expect(domain.Spec.Type).To(Equal("hyperv"))
		})

		It("should handle nil VMI gracefully", func() {
			Expect(func() {
				hypervHypervisor.AdjustDomain(nil, domain)
			}).NotTo(Panic())
			Expect(domain.Spec.Type).To(Equal("hyperv"))
		})

		It("should handle nil domain gracefully", func() {
			Expect(func() {
				hypervHypervisor.AdjustDomain(vmi, nil)
			}).NotTo(Panic())
		})

		It("should handle both nil VMI and domain gracefully", func() {
			Expect(func() {
				hypervHypervisor.AdjustDomain(nil, nil)
			}).NotTo(Panic())
		})
	})

	Context("Integration tests", func() {
		var vmi *v1.VirtualMachineInstance
		var domain *api.Domain

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "test-namespace",
				},
			}
			domain = &api.Domain{
				Spec: api.DomainSpec{
					Type: "original-type",
				},
			}
		})

		It("should work end-to-end for KVM hypervisor", func() {
			hypervisor := NewHypervisor(v1.KvmHypervisorName)
			originalType := domain.Spec.Type
			hypervisor.AdjustDomain(vmi, domain)
			Expect(domain.Spec.Type).To(Equal(originalType))
		})

		It("should work end-to-end for HyperV Layered hypervisor", func() {
			hypervisor := NewHypervisor(v1.HyperVLayeredHypervisorName)
			hypervisor.AdjustDomain(vmi, domain)
			Expect(domain.Spec.Type).To(Equal("hyperv"))
		})

		It("should demonstrate the difference between hypervisors", func() {
			kvmDomain := &api.Domain{Spec: api.DomainSpec{Type: "test"}}
			hypervDomain := &api.Domain{Spec: api.DomainSpec{Type: "test"}}

			kvmHypervisor := NewHypervisor(v1.KvmHypervisorName)
			hypervHypervisor := NewHypervisor(v1.HyperVLayeredHypervisorName)

			kvmHypervisor.AdjustDomain(vmi, kvmDomain)
			hypervHypervisor.AdjustDomain(vmi, hypervDomain)

			Expect(kvmDomain.Spec.Type).To(Equal("test"))      // unchanged
			Expect(hypervDomain.Spec.Type).To(Equal("hyperv")) // changed to hyperv
		})
	})
})
