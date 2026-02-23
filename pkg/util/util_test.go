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

package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"

	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("GPU VMI Predicates", func() {
	It("should detect VMI with GPU device plugins", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							{
								Name:       "gpu1",
								DeviceName: "nvidia.com/gpu",
							},
						},
					},
				},
			},
		}
		Expect(IsGPUVMI(vmi)).To(BeTrue())
	})

	It("should not detect GPU VMI when no GPUs are specified", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{},
				},
			},
		}
		Expect(IsGPUVMI(vmi)).To(BeFalse())
	})
})

var _ = Describe("Host Device VMI Predicates", func() {
	It("should detect VMI with host device plugins", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name:       "hostdev1",
								DeviceName: "vendor.com/device",
							},
						},
					},
				},
			},
		}
		Expect(IsHostDevVMI(vmi)).To(BeTrue())
	})

	It("should detect VMI with host device DRA", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							{
								Name: "hostdev1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   pointer.P("hostdev-claim"),
									RequestName: pointer.P("hostdev-request"),
								},
							},
						},
					},
				},
			},
		}
		Expect(IsHostDevVMI(vmi)).To(BeTrue())
	})
})

var _ = DescribeTable("memory overhead reservation requirements",
	func(vmi *v1.VirtualMachineInstance, expected bool) {
		res := RequiresMemoryOverheadReservation(vmi)
		Expect(res).To(Equal(expected))
	},
	Entry(
		"Domain Memory reference is nil",
		&v1.VirtualMachineInstance{},
		false,
	),
	Entry(
		"ReservedOverhead reference is nil",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{},
				},
			},
		},
		false,
	),
	Entry(
		"ReservedOverhead reference is empty",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						ReservedOverhead: &v1.ReservedOverhead{},
					},
				},
			},
		},
		false,
	),
	Entry(
		"AddedOverhead has a value",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						ReservedOverhead: &v1.ReservedOverhead{
							AddedOverhead: pointer.P(resource.MustParse("1Gi")),
						},
					},
				},
			},
		},
		true,
	),
)

var _ = DescribeTable("memory lock limit requirements",
	func(vmi *v1.VirtualMachineInstance, expected bool) {
		res := RequiresLockingMemory(vmi)
		Expect(res).To(Equal(expected))
	},
	Entry(
		"Domain Memory reference is nil",
		&v1.VirtualMachineInstance{},
		false,
	),
	Entry(
		"ReservedOverhead reference is nil",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{},
				},
			},
		},
		false,
	),
	Entry(
		"ReservedOverhead reference is empty",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						ReservedOverhead: &v1.ReservedOverhead{},
					},
				},
			},
		},
		false,
	),
	Entry(
		"MemLock is whatever other than Required",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						ReservedOverhead: &v1.ReservedOverhead{
							MemLock: pointer.P(v1.MemLockRequirement("notExpectedValue")),
						},
					},
				},
			},
		},
		false,
	),
	Entry(
		"MemLock is NotRequired",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						ReservedOverhead: &v1.ReservedOverhead{
							MemLock: pointer.P(v1.MemLockRequirement(v1.MemLockNotRequired)),
						},
					},
				},
			},
		},
		false,
	),
	Entry(
		"MemLock is Required",
		&v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Memory: &v1.Memory{
						ReservedOverhead: &v1.ReservedOverhead{
							MemLock: pointer.P(v1.MemLockRequirement(v1.MemLockRequired)),
						},
					},
				},
			},
		},
		true,
	),
)
