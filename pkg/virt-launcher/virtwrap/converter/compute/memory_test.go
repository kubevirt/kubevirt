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

package compute_test

import (
	"encoding/xml"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

var _ = Describe("Memory Domain Configurator", func() {
	DescribeTable("should calculate memory in bytes", func(quantity string, bytes int) {
		m64, _ := resource.ParseQuantity(quantity)
		memory, err := vcpu.QuantityToByte(m64)
		Expect(memory.Value).To(BeNumerically("==", bytes))
		Expect(memory.Unit).To(Equal("b"))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("specifying memory 64M", "64M", 64*1000*1000),
		Entry("specifying memory 64Mi", "64Mi", 64*1024*1024),
		Entry("specifying memory 3G", "3G", 3*1000*1000*1000),
		Entry("specifying memory 3Gi", "3Gi", 3*1024*1024*1024),
		Entry("specifying memory 45Gi", "45Gi", 45*1024*1024*1024),
		Entry("specifying memory 2780Gi", "2780Gi", 2780*1024*1024*1024),
		Entry("specifying memory 451231 bytes", "451231", 451231),
	)
	It("should calculate memory in bytes", func() {
		By("specyfing negative memory size -45Gi")
		m45gi, _ := resource.ParseQuantity("-45Gi")
		_, err := vcpu.QuantityToByte(m45gi)
		Expect(err).To(HaveOccurred())
	})

	Context("with v1.VirtualMachineInstance", func() {
		It("should handle float memory", func() {
			vmi := libvmi.New(
				libvmi.WithMemoryRequest("2222222200m"),
			)
			var domain api.Domain

			configurator := compute.MemoryConfigurator{}
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			xmlBytes, err := xml.MarshalIndent(domain.Spec, "", "  ")
			Expect(err).ToNot(HaveOccurred())

			xml := string(xmlBytes)

			Expect(strings.Contains(xml, `<memory unit="b">2222222</memory>`)).To(BeTrue(), xml)
		})

		It("should use guest memory instead of requested memory if present", func() {
			vmi := libvmi.New(
				libvmi.WithGuestMemory("123Mi"),
			)
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			var domain api.Domain

			configurator := compute.MemoryConfigurator{}
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			Expect(domain.Spec.Memory.Value).To(Equal(uint64(128974848)))
			Expect(domain.Spec.Memory.Unit).To(Equal("b"))
		})

		Context("memory", func() {
			var vmi *v1.VirtualMachineInstance
			var domain *api.Domain
			var guestMemory resource.Quantity
			var maxGuestMemory resource.Quantity

			BeforeEach(func() {
				guestMemory = resource.MustParse("32Mi")
				maxGuestMemory = resource.MustParse("128Mi")

				vmi = &v1.VirtualMachineInstance{
					ObjectMeta: k8smeta.ObjectMeta{
						Name:      "testvmi",
						Namespace: "mynamespace",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Memory: &v1.Memory{
								Guest:    &guestMemory,
								MaxGuest: &maxGuestMemory,
							},
						},
					},
					Status: v1.VirtualMachineInstanceStatus{
						Memory: &v1.MemoryStatus{
							GuestAtBoot:  &guestMemory,
							GuestCurrent: &guestMemory,
						},
					},
				}

				domain = &api.Domain{
					Spec: api.DomainSpec{
						VCPU: &api.VCPU{
							CPUs: 2,
						},
					},
				}

				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			})

			It("should not setup hotplug when maxGuest is missing", func() {
				vmi.Spec.Domain.Memory.MaxGuest = nil
				configurator := compute.MemoryConfigurator{}
				Expect(configurator.Configure(vmi, domain)).To(Succeed())
				Expect(domain.Spec.MaxMemory).To(BeNil())
			})

			It("should not setup hotplug when maxGuest equals guest memory", func() {
				vmi.Spec.Domain.Memory.MaxGuest = &guestMemory
				configurator := compute.MemoryConfigurator{}
				Expect(configurator.Configure(vmi, domain)).To(Succeed())
				Expect(domain.Spec.MaxMemory).To(BeNil())
			})

			It("should setup hotplug when maxGuest is set", func() {
				configurator := compute.MemoryConfigurator{}
				Expect(configurator.Configure(vmi, domain)).To(Succeed())

				Expect(domain.Spec.MaxMemory).ToNot(BeNil())
				Expect(domain.Spec.MaxMemory.Unit).To(Equal("b"))
				Expect(domain.Spec.MaxMemory.Value).To(Equal(uint64(maxGuestMemory.Value())))

				Expect(domain.Spec.Memory).ToNot(BeNil())
				Expect(domain.Spec.Memory.Unit).To(Equal("b"))
				Expect(domain.Spec.Memory.Value).To(Equal(uint64(guestMemory.Value())))
			})

			DescribeTable("should correctly convert memory configuration from VMI spec to domain",
				func(expectedMemoryMiB int64, opts ...libvmi.Option) {

					vmi := libvmi.New(opts...)

					v1.SetObjectDefaults_VirtualMachineInstance(vmi)

					configurator := compute.MemoryConfigurator{}
					Expect(configurator.Configure(vmi, domain)).To(Succeed())
					apiDomainSpec := domain.Spec

					expectedBytes := expectedMemoryMiB * 1024 * 1024
					Expect(apiDomainSpec.Memory.Value).To(Equal(uint64(expectedBytes)),
						"Memory value should be %d bytes (%d MiB)", expectedBytes, expectedMemoryMiB)
					Expect(apiDomainSpec.Memory.Unit).To(Equal("b"))
				},
				Entry("provided by domain spec directly (guest memory takes precedence over limits)",
					int64(512),
					libvmi.WithGuestMemory("512Mi"),
				),
				Entry("provided by resources limits (no guest memory, no request)",
					int64(256),
					libvmi.WithMemoryLimit("256Mi"),
				),
				Entry("provided by resources requests (request takes precedence over limit when both set)",
					int64(64),
					libvmi.WithMemoryRequest("64Mi"),
					libvmi.WithMemoryLimit("256Gi"),
				),
				Entry("provided by resources requests only",
					int64(128),
					libvmi.WithMemoryRequest("128Mi"),
				),
			)
		})
	})
})
