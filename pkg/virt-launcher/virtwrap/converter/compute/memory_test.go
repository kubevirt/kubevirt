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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Memory Domain Configurator", func() {
	DescribeTable("should calculate memory in bytes", func(quantity string, bytes int) {
		vmi := libvmi.New(
			libvmi.WithMemoryRequest(quantity),
		)

		var domain api.Domain
		configurator := compute.MemoryConfigurator{}
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Memory: api.Memory{
					Value: uint64(bytes),
					Unit:  "b",
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("specifying memory 64M", "64M", 64*1000*1000),
		Entry("specifying memory 64Mi", "64Mi", 64*1024*1024),
		Entry("specifying memory 3G", "3G", 3*1000*1000*1000),
		Entry("specifying memory 3Gi", "3Gi", 3*1024*1024*1024),
		Entry("specifying memory 45Gi", "45Gi", 45*1024*1024*1024),
		Entry("specifying memory 2780Gi", "2780Gi", 2780*1024*1024*1024),
		Entry("specifying memory 451231 bytes", "451231", 451231),
		Entry("specifying float memory", "2222222200m", 2222222),
	)
	It("should fail when memory size is negative", func() {
		By("specyfing negative memory size -45Gi")
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("-45Gi"),
		)
		var domain api.Domain
		configurator := compute.MemoryConfigurator{}
		err := configurator.Configure(vmi, &domain)
		Expect(err).To(HaveOccurred())
		// Since conversion failed, domain should remain empty
		Expect(domain).To(Equal(api.Domain{}))
	})

	Context("configure multiple memory fields", func() {
		var guestMemory resource.Quantity = resource.MustParse("32Mi")
		var maxGuestMemory resource.Quantity = resource.MustParse("128Mi")
		var guestMemoryOption libvmi.Option = libvmi.WithGuestMemory(guestMemory.String())

		DescribeTable("maxGuest and guest memory settings",
			func(vmi *v1.VirtualMachineInstance, expectedGuestMemory *resource.Quantity, expectedMaxMemory *resource.Quantity) {
				domain := &api.Domain{}
				configurator := compute.MemoryConfigurator{}
				Expect(configurator.Configure(vmi, domain)).To(Succeed())

				var expectedMaxMemorySpec *api.MaxMemory = nil
				if expectedMaxMemory != nil {
					expectedMaxMemorySpec = &api.MaxMemory{
						Unit:  "b",
						Value: uint64(expectedMaxMemory.Value()),
					}
				}

				expectedDomain := &api.Domain{
					Spec: api.DomainSpec{
						Memory: api.Memory{
							Value: uint64(expectedGuestMemory.Value()),
							Unit:  "b",
						},
						MaxMemory: expectedMaxMemorySpec,
					},
				}

				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("maxGuest is missing", libvmi.New(guestMemoryOption), &guestMemory, nil),
			Entry("maxGuest equal to guest memory", libvmi.New(guestMemoryOption, libvmi.WithMaxGuest(guestMemory.String())), &guestMemory, nil),
			Entry("maxGuest greater than guest memory", libvmi.New(guestMemoryOption, libvmi.WithMaxGuest(maxGuestMemory.String())), &guestMemory, &maxGuestMemory),
		)

		DescribeTable("guest memory and resource requests/limits settings",
			func(expectedMemoryBytes int64, opts ...libvmi.Option) {
				var domain api.Domain
				vmi := libvmi.New(opts...)

				configurator := compute.MemoryConfigurator{}
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())

				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Memory: api.Memory{
							Unit:  "b",
							Value: uint64(expectedMemoryBytes),
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
			},
			Entry("provided by domain spec directly (guest memory takes precedence over limits)",
				int64(512*1024*1024),
				libvmi.WithGuestMemory("512Mi"),
			),
			Entry("provided by resources limits (no guest memory, no request)",
				int64(256*1024*1024),
				libvmi.WithMemoryLimit("256Mi"),
			),
			Entry("provided by resources requests (request takes precedence over limit when both set)",
				int64(64*1024*1024),
				libvmi.WithMemoryRequest("64Mi"),
				libvmi.WithMemoryLimit("256Gi"),
			),
			Entry("provided by resources requests only",
				int64(128*1024*1024),
				libvmi.WithMemoryRequest("128Mi"),
			),
			Entry("provided by guest memory and resources requests (guest memory takes precedence)",
				int64(128974848),
				libvmi.WithGuestMemory("123Mi"),
				libvmi.WithMemoryRequest("100Mi"),
			),
		)
	})
})
