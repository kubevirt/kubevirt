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
				Entry("realtime is configured", "128Mi",
					libvmi.WithDedicatedCPUPlacement(),
					libvmi.WithRealtimeMask(""),
					libvmi.WithNUMAGuestMappingPassthrough(),
					libvmi.WithHugepages("2Mi"),
					libvmi.WithGuestMemory("64Mi"),
				),
				Entry("launchSecurity is configured", "128Mi",
					libvmi.WithSEV(true),
					libvmi.WithGuestMemory("64Mi")),
				Entry("guest mapping passthrough is configured", "128Mi",
					libvmi.WithNUMAGuestMappingPassthrough(),
					libvmi.WithHugepages("2Mi"),
					libvmi.WithGuestMemory("64Mi"),
				),
				Entry("guest memory is not set", "128Mi"),
				Entry("guest memory is greater than maxGuest", "32Mi",
					libvmi.WithGuestMemory("64Mi"),
				),
				Entry("maxGuest is not properly aligned", "333Mi", libvmi.WithGuestMemory("64Mi")),
				Entry("guest memory is not properly aligned", "128Mi", libvmi.WithGuestMemory("123")),
				Entry("guest memory with hugepages is not properly aligned", "16Gi",
					libvmi.WithGuestMemory("2G"),
					libvmi.WithHugepages("1Gi"),
				),
				Entry("architecture is not amd64", "128Mi",
					libvmi.WithArchitecture("risc-v"),
					libvmi.WithGuestMemory("64Mi"),
				),
			)
		})
	})
})
