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

	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("LiveUpdate Memory", func() {
	Context("Memory", func() {
		Context("Validation", func() {
			maxGuest := resource.MustParse("128Mi")

			DescribeTable("should reject VM creation if", func(vmSetup func(*v1.VirtualMachine)) {

				vm := &v1.VirtualMachine{
					Spec: v1.VirtualMachineSpec{
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: v1.VirtualMachineInstanceSpec{
								Architecture: "amd64",
								Domain: v1.DomainSpec{
									Memory: &v1.Memory{
										Guest: pointer.P(resource.MustParse("64Mi")),
									},
								},
							},
						},
					},
				}

				vmSetup(vm)

				err := memory.ValidateLiveUpdateMemory(&vm.Spec.Template.Spec, &maxGuest)
				Expect(err).To(HaveOccurred())
			},
				Entry("realtime is configured", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						DedicatedCPUPlacement: true,
						Realtime:              &v1.Realtime{},
						NUMA: &v1.NUMA{
							GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
						},
					}
					vm.Spec.Template.Spec.Domain.Memory.Hugepages = &v1.Hugepages{
						PageSize: "2Mi",
					}
				}),
				Entry("launchSecurity is configured", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{}
				}),
				Entry("guest mapping passthrough is configured", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
						DedicatedCPUPlacement: true,
						NUMA: &v1.NUMA{
							GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
						},
					}
					vm.Spec.Template.Spec.Domain.Memory.Hugepages = &v1.Hugepages{
						PageSize: "2Mi",
					}
				}),
				Entry("guest memory is not set", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.Memory.Guest = nil
				}),
				Entry("guest memory is greater than maxGuest", func(vm *v1.VirtualMachine) {
					moreThanMax := maxGuest.DeepCopy()
					moreThanMax.Add(resource.MustParse("16Mi"))

					vm.Spec.Template.Spec.Domain.Memory.Guest = &moreThanMax
				}),
				Entry("maxGuest is not properly aligned", func(vm *v1.VirtualMachine) {
					maxGuest = resource.MustParse("333Mi")
				}),
				Entry("guest memory is not properly aligned", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.Memory.Guest = pointer.P(resource.MustParse("123"))
				}),
				Entry("guest memory with hugepages is not properly aligned", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Domain.Memory.Guest = pointer.P(resource.MustParse("2G"))
					vm.Spec.Template.Spec.Domain.Memory.MaxGuest = pointer.P(resource.MustParse("16Gi"))
					vm.Spec.Template.Spec.Domain.Memory.Hugepages = &v1.Hugepages{PageSize: "1Gi"}
				}),
				Entry("architecture is not amd64", func(vm *v1.VirtualMachine) {
					vm.Spec.Template.Spec.Architecture = "risc-v"
				}),
			)
		})
	})
})
