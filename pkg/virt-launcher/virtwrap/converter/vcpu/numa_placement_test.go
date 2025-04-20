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
 */

package vcpu

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("NumaPlacement", func() {

	var givenSpec *api.DomainSpec
	var givenVMI *v1.VirtualMachineInstance
	var givenTopology *cmdv1.Topology
	var expectedSpec *api.DomainSpec
	var MiBInBytes_2 = strconv.Itoa(2 * 1024 * 1024)
	var MiBInBytes_22 uint64 = 22 * 1024 * 1024
	var MiBInBytes_20 uint64 = 20 * 1024 * 1024
	var MiBInBytes_32 uint64 = 32 * 1024 * 1024

	BeforeEach(func() {
		var err error
		givenSpec = &api.DomainSpec{
			CPUTune: &api.CPUTune{
				VCPUPin: []api.CPUTuneVCPUPin{
					{VCPU: 0, CPUSet: "10"},
					{VCPU: 1, CPUSet: "20"},
					{VCPU: 3, CPUSet: "30"},
				},
				IOThreadPin: nil,
				EmulatorPin: nil,
			},
		}
		givenSpec.Memory, err = QuantityToByte(resource.MustParse("64Mi"))
		Expect(err).ToNot(HaveOccurred())
		givenTopology = &cmdv1.Topology{
			NumaCells: []*cmdv1.Cell{
				{
					Id: 0,
					Cpus: []*cmdv1.CPU{
						{Id: 10},
						{Id: 20},
					},
				},
				{
					Id: 4,
					Cpus: []*cmdv1.CPU{
						{Id: 30},
						{Id: 50},
					},
				},
			},
		}
		expectedSpec = &api.DomainSpec{
			CPU: api.CPU{NUMA: &api.NUMA{Cells: []api.NUMACell{
				{ID: "0", CPUs: "0,1", Memory: MiBInBytes_32, Unit: "b"},
				{ID: "1", CPUs: "3", Memory: MiBInBytes_32, Unit: "b"},
			}}},
			CPUTune: &api.CPUTune{VCPUPin: []api.CPUTuneVCPUPin{
				{VCPU: 0, CPUSet: "10"},
				{VCPU: 1, CPUSet: "20"},
				{VCPU: 3, CPUSet: "30"},
			}},
			NUMATune: &api.NUMATune{
				Memory: api.NumaTuneMemory{Mode: "strict", NodeSet: "0,4"},
				MemNodes: []api.MemNode{
					{CellID: 0, Mode: "strict", NodeSet: "0"},
					{CellID: 1, Mode: "strict", NodeSet: "4"},
				}},
		}
		givenVMI = &v1.VirtualMachineInstance{}
		memory := resource.MustParse("64Mi")
		givenVMI.Spec.Domain.Memory = &v1.Memory{Guest: &memory}
	})

	It("should not map the numa topology without hugepages requested", func() {
		Expect(numaMapping(givenVMI, givenSpec, givenTopology)).ToNot(Succeed())
	})

	DescribeTable("it should do nothing", func(givenTopology *cmdv1.Topology) {
		expectedSpec := givenSpec.DeepCopy()
		Expect(numaMapping(givenVMI, givenSpec, givenTopology)).To(Succeed())
		Expect(givenSpec.CPUTune).To(Equal(expectedSpec.CPUTune))
		Expect(givenSpec.NUMATune).To(Equal(expectedSpec.NUMATune))
		Expect(givenSpec.CPU).To(Equal(expectedSpec.CPU))
	},
		Entry("if no topology is provided", nil),
		Entry("if no numa cells are reported", &cmdv1.Topology{NumaCells: nil}),
	)

	It("should detect invalid cpu pinning", func() {
		givenSpec.CPUTune.VCPUPin = append(givenSpec.CPUTune.VCPUPin, api.CPUTuneVCPUPin{
			VCPU:   4,
			CPUSet: "40",
		})
		Expect(numaMapping(givenVMI, givenSpec, givenTopology)).ToNot(Succeed())
	})

	Context("with hugepages", func() {
		var expectedMemoryBacking *api.MemoryBacking
		BeforeEach(func() {
			givenVMI.Spec.Domain.Memory.Hugepages = &v1.Hugepages{
				PageSize: "2Mi",
			}

			givenSpec.MemoryBacking = &api.MemoryBacking{
				HugePages: &api.HugePages{},
			}
			expectedMemoryBacking = &api.MemoryBacking{
				HugePages: &api.HugePages{HugePage: []api.HugePage{
					{Size: MiBInBytes_2, Unit: "b", NodeSet: "0"},
					{Size: MiBInBytes_2, Unit: "b", NodeSet: "1"},
				}},
				Allocation: &api.MemoryAllocation{Mode: api.MemoryAllocationModeImmediate},
			}
		})
		It("should detect hugepages and map them equally to nodes", func() {
			Expect(numaMapping(givenVMI, givenSpec, givenTopology)).To(Succeed())
			Expect(givenSpec.CPUTune).To(Equal(expectedSpec.CPUTune))
			Expect(givenSpec.NUMATune).To(Equal(expectedSpec.NUMATune))
			Expect(givenSpec.CPU).To(Equal(expectedSpec.CPU))
			Expect(givenSpec.MemoryBacking).To(Equal(expectedMemoryBacking))
		})

		It("should detect if not enough memory is requested", func() {
			var err error
			memory := resource.MustParse("2Mi")
			givenSpec.Memory, err = QuantityToByte(memory)
			Expect(err).ToNot(HaveOccurred())
			givenVMI.Spec.Domain.Memory.Guest = &memory
			Expect(numaMapping(givenVMI, givenSpec, givenTopology)).ToNot(Succeed())
		})

		It("should detect not divisable hugepages and shuffle the memory", func() {
			var err error
			givenSpec.Memory, err = QuantityToByte(resource.MustParse("66Mi"))
			Expect(err).ToNot(HaveOccurred())
			givenSpec.CPUTune.VCPUPin = append(givenSpec.CPUTune.VCPUPin, api.CPUTuneVCPUPin{
				VCPU: 4, CPUSet: "40",
			})
			givenTopology.NumaCells = append(givenTopology.NumaCells, &cmdv1.Cell{
				Id: 5,
				Cpus: []*cmdv1.CPU{
					{Id: 40},
				},
			})

			expectedSpec.CPUTune.VCPUPin = append(expectedSpec.CPUTune.VCPUPin, api.CPUTuneVCPUPin{
				VCPU: 4, CPUSet: "40",
			})
			expectedSpec.NUMATune.Memory = api.NumaTuneMemory{
				Mode: "strict", NodeSet: "0,4,5",
			}
			expectedSpec.NUMATune.MemNodes = append(expectedSpec.NUMATune.MemNodes, api.MemNode{
				CellID: 2, Mode: "strict", NodeSet: "5",
			})
			expectedMemoryBacking := &api.MemoryBacking{
				HugePages: &api.HugePages{HugePage: []api.HugePage{
					{Size: MiBInBytes_2, Unit: "b", NodeSet: "0"},
					{Size: MiBInBytes_2, Unit: "b", NodeSet: "1"},
					{Size: MiBInBytes_2, Unit: "b", NodeSet: "2"},
				}},
				Allocation: &api.MemoryAllocation{Mode: api.MemoryAllocationModeImmediate},
			}
			expectedSpec.CPU.NUMA.Cells = []api.NUMACell{
				{ID: "0", CPUs: "0,1", Memory: MiBInBytes_22, Unit: "b"},
				{ID: "1", CPUs: "3", Memory: MiBInBytes_22, Unit: "b"},
				{ID: "2", CPUs: "4", Memory: MiBInBytes_20, Unit: "b"},
			}

			Expect(numaMapping(givenVMI, givenSpec, givenTopology)).To(Succeed())
			Expect(givenSpec.CPUTune).To(Equal(expectedSpec.CPUTune))
			Expect(givenSpec.NUMATune).To(Equal(expectedSpec.NUMATune))
			Expect(givenSpec.CPU).To(Equal(expectedSpec.CPU))
			Expect(givenSpec.MemoryBacking).To(Equal(expectedMemoryBacking))
		})
		It("should process no shared pages when tuned for real time", func() {
			givenVMI.Spec.Domain.CPU = &v1.CPU{Realtime: &v1.Realtime{}}
			Expect(numaMapping(givenVMI, givenSpec, givenTopology)).To(Succeed())

			Expect(givenSpec.MemoryBacking.NoSharePages).To(Equal(&api.NoSharePages{}))
		})
	})
})
