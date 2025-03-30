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
 * Copyright The Kubevirt Authors
 *
 */

package memory_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

var _ = Describe("LiveUpdate Memory", func() {
	Context("Memory", func() {
		Context("Validation", func() {
			DescribeTable("should reject VM creation if", func(maxGuestStr string, opts ...libvmi.Option) {
				vmiOpts := []libvmi.Option{
					libvmi.WithArchitecture("amd64"),
				}
				vmiOpts = append(vmiOpts, opts...)

				vm := libvmi.NewVirtualMachine(libvmi.New(vmiOpts...))

				maxGuest := resource.MustParse(maxGuestStr)
				err := memory.ValidateLiveUpdateMemory(&vm.Spec.Template.Spec, &maxGuest)
				Expect(err).To(HaveOccurred())
			},
				Entry("realtime is configured", "4Gi",
					libvmi.WithDedicatedCPUPlacement(),
					libvmi.WithRealtimeMask(""),
					libvmi.WithNUMAGuestMappingPassthrough(),
					libvmi.WithHugepages("2Mi"),
					libvmi.WithGuestMemory("1Gi"),
				),
				Entry("launchSecurity is configured", "4Gi",
					libvmi.WithSEV(true),
					libvmi.WithGuestMemory("1Gi")),
				Entry("guest mapping passthrough is configured", "4Gi",
					libvmi.WithNUMAGuestMappingPassthrough(),
					libvmi.WithHugepages("2Mi"),
					libvmi.WithGuestMemory("1Gi"),
				),
				Entry("guest memory is not set", "4Gi"),
				Entry("guest memory is greater than maxGuest", "2Gi",
					libvmi.WithGuestMemory("4Gi"),
				),
				Entry("maxGuest is not properly aligned", "333Mi", libvmi.WithGuestMemory("1Gi")),
				Entry("guest memory is not properly aligned", "4Gi", libvmi.WithGuestMemory("123")),
				Entry("guest memory with hugepages is not properly aligned", "16Gi",
					libvmi.WithGuestMemory("2G"),
					libvmi.WithHugepages("1Gi"),
				),
				Entry("architecture is not amd64", "4Gi",
					libvmi.WithArchitecture("risc-v"),
					libvmi.WithGuestMemory("1Gi"),
				),
				Entry("guest memory is less than 1Gi", "4Gi",
					libvmi.WithGuestMemory("1022Mi"),
				),
			)
		})

		Context("virtio-mem device", func() {
			DescribeTable("should be correctly built", func(opts ...libvmi.Option) {
				currentGuestMemory := resource.MustParse("64Mi")

				vmiOpts := []libvmi.Option{
					libvmi.WithArchitecture("amd64"),
					libvmi.WithGuestMemory("128Mi"),
					libvmi.WithMaxGuest("256Mi"),
				}
				vmiOpts = append(vmiOpts, opts...)

				vmi := libvmi.New(vmiOpts...)

				vmi.Status = v1.VirtualMachineInstanceStatus{
					Memory: &v1.MemoryStatus{
						GuestCurrent:   &currentGuestMemory,
						GuestRequested: &currentGuestMemory,
						GuestAtBoot:    &currentGuestMemory,
					},
				}

				memoryDevice, err := memory.BuildMemoryDevice(vmi)
				Expect(err).ToNot(HaveOccurred())

				size, err := vcpu.QuantityToByte(resource.MustParse("192Mi"))
				Expect(err).ToNot(HaveOccurred())

				requested, err := vcpu.QuantityToByte(resource.MustParse("64Mi"))
				Expect(err).ToNot(HaveOccurred())

				block := api.Memory{Unit: "b", Value: uint64(memory.HotplugBlockAlignmentBytes)}

				hugepages := vmi.Spec.Domain.Memory.Hugepages
				if hugepages != nil {
					var err error
					block, err = vcpu.QuantityToByte(resource.MustParse(hugepages.PageSize))
					Expect(err).ToNot(HaveOccurred())
				}
				Expect(err).ToNot(HaveOccurred())

				Expect(memoryDevice).ToNot(BeNil())
				Expect(*memoryDevice).To(Equal(api.MemoryDevice{
					Model: "virtio-mem",
					Target: &api.MemoryTarget{
						Size:      size,
						Node:      "0",
						Block:     block,
						Requested: requested,
					},
				}))
			},
				Entry("when using a common VM"),
				Entry("when using a VM with 2Mi sized hugepages", libvmi.WithHugepages("2Mi")),
				Entry("when using a VM with 1Gi sized hugepages", libvmi.WithHugepages("1Gi")),
			)
		})
	})
})
