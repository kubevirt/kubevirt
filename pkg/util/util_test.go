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
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"

	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("CountVFIODevices", func() {
	DescribeTable("should count VFIO devices correctly",
		func(devices v1.Devices, expected int) {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: devices,
					},
				},
			}
			Expect(CountVFIODevices(vmi)).To(Equal(expected))
		},
		Entry("no devices", v1.Devices{}, 0),
		Entry("single GPU", v1.Devices{
			GPUs: []v1.GPU{{Name: "gpu1"}},
		}, 1),
		Entry("multiple GPUs", v1.Devices{
			GPUs: []v1.GPU{{Name: "gpu1"}, {Name: "gpu2"}},
		}, 2),
		Entry("single HostDevice", v1.Devices{
			HostDevices: []v1.HostDevice{{Name: "dev1"}},
		}, 1),
		Entry("multiple HostDevices", v1.Devices{
			HostDevices: []v1.HostDevice{{Name: "dev1"}, {Name: "dev2"}},
		}, 2),
		Entry("single SRIOV", v1.Devices{
			Interfaces: []v1.Interface{
				{Name: "sriov1", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}},
			},
		}, 1),
		Entry("non-SRIOV interfaces are not counted", v1.Devices{
			Interfaces: []v1.Interface{
				{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}},
			},
		}, 0),
		Entry("mixed devices", v1.Devices{
			GPUs:        []v1.GPU{{Name: "gpu1"}, {Name: "gpu2"}},
			HostDevices: []v1.HostDevice{{Name: "dev1"}},
			Interfaces: []v1.Interface{
				{Name: "sriov1", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}},
				{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}},
			},
		}, 4),
	)
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

var _ = Describe("Misc Capacity", func() {
	var (
		originalMiscCapacityPath string
		tempDir                  string
	)

	BeforeEach(func() {
		originalMiscCapacityPath = miscCapacityPath
		tempDir, err := os.MkdirTemp("", "cgroup")
		Expect(err).ToNot(HaveOccurred())
		miscCapacityPath = path.Join(tempDir, "misc.capacity")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
		miscCapacityPath = originalMiscCapacityPath
	})

	Context("when reading secure guest capacity from misc.capacity", func() {
		It("should successfully parse TDX capacity", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte("tdx 15\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["tdx"]).To(Equal(15))
		})

		It("should successfully parse SEV-SNP capacity", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte("sev 410\nsev_es 99\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(2))
			Expect(caps["sev"]).To(Equal(410))
			Expect(caps["sev_es"]).To(Equal(99))
		})

		It("should successfully handle empty file", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte(""), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(BeEmpty())
		})

		It("should return error when file does not exist", func() {
			miscCapacityPath = "/nonexisted_path/misc.capacity"
			caps, err := GetMiscCapacity()
			Expect(err).To(HaveOccurred())
			Expect(caps).To(BeNil())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("should skip malformed lines and parse the rest", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte("tdx abc\nsev_es 99\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["sev_es"]).To(Equal(99))
		})
	})
})
