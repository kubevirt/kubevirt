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

package virthandler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libvirt.org/go/libvirtxml"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

var _ = Describe("Parsing VMI Options", func() {
	Context("virt-handler VM processes VMI options during update", func() {
		const memoryUnit = "KiB"
		DescribeTable("should convert libvirtxml.Caps to cmdv1.Topology when Caps is nil or empty", func(caps *libvirtxml.Caps) {
			actualTopology := capabilitiesToTopology(caps)

			expectedTopology := &cmdv1.Topology{}

			Expect(actualTopology).To(Equal(expectedTopology))
		},
			Entry("libvirtxml.Caps is nil", nil),
			Entry("libvirtxml.Caps is empty", &libvirtxml.Caps{
				Host: libvirtxml.CapsHost{
					NUMA: &libvirtxml.CapsHostNUMATopology{
						Cells: &libvirtxml.CapsHostNUMACells{},
					},
				},
			}),
		)
		It("should convert libvirtxml.Caps to cmdv1.Topology with single NUMA node", func() {
			caps := &libvirtxml.Caps{Host: libvirtxml.CapsHost{NUMA: &libvirtxml.CapsHostNUMATopology{}}}
			caps.Host.NUMA.Cells = &libvirtxml.CapsHostNUMACells{
				Cells: []libvirtxml.CapsHostNUMACell{
					{
						Memory: &libvirtxml.CapsHostNUMAMemory{Unit: memoryUnit, Size: 16256896},
						PageInfo: []libvirtxml.CapsHostNUMAPageInfo{
							{Unit: memoryUnit, Size: 4, Count: 4064224},
							{Unit: memoryUnit, Size: 2048, Count: 0},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: &libvirtxml.CapsHostNUMADistances{Siblings: []libvirtxml.CapsHostNUMASibling{{ID: 0, Value: 10}}},
						CPUS: &libvirtxml.CapsHostNUMACPUs{CPUs: []libvirtxml.CapsHostNUMACPU{
							{ID: 0, Siblings: "0,4"},
							{ID: 1, Siblings: "1,5"},
							{ID: 2, Siblings: "2,6"},
							{ID: 3, Siblings: "3,7"},
							{ID: 4, Siblings: "0,4"},
							{ID: 5, Siblings: "1,5"},
							{ID: 6, Siblings: "2,6"},
							{ID: 7, Siblings: "3,7"},
						}},
					},
				},
			}

			actualTopology := capabilitiesToTopology(caps)

			expectedTopology := &cmdv1.Topology{
				NumaCells: []*cmdv1.Cell{
					{
						Memory: &cmdv1.Memory{Unit: memoryUnit, Amount: 16256896},
						Pages: []*cmdv1.Pages{
							{Unit: memoryUnit, Size: 4, Count: 4064224},
							{Unit: memoryUnit, Size: 2048, Count: 0},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: []*cmdv1.Sibling{{Id: 0, Value: 10}},
						Cpus: []*cmdv1.CPU{
							{Id: 0, Siblings: []uint32{0, 4}},
							{Id: 1, Siblings: []uint32{1, 5}},
							{Id: 2, Siblings: []uint32{2, 6}},
							{Id: 3, Siblings: []uint32{3, 7}},
							{Id: 4, Siblings: []uint32{0, 4}},
							{Id: 5, Siblings: []uint32{1, 5}},
							{Id: 6, Siblings: []uint32{2, 6}},
							{Id: 7, Siblings: []uint32{3, 7}},
						},
					},
				},
			}

			Expect(actualTopology).To(Equal(expectedTopology))
		})

		It("should convert libvirtxml.Caps to cmdv1.Topology with multiple NUMA nodes, CPUs and pages", func() {
			caps := &libvirtxml.Caps{Host: libvirtxml.CapsHost{NUMA: &libvirtxml.CapsHostNUMATopology{}}}
			caps.Host.NUMA.Cells = &libvirtxml.CapsHostNUMACells{
				Cells: []libvirtxml.CapsHostNUMACell{
					{
						Memory: &libvirtxml.CapsHostNUMAMemory{Unit: memoryUnit, Size: 1289144},
						PageInfo: []libvirtxml.CapsHostNUMAPageInfo{
							{Unit: memoryUnit, Size: 4, Count: 314094},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: &libvirtxml.CapsHostNUMADistances{Siblings: []libvirtxml.CapsHostNUMASibling{
							{ID: 0, Value: 10},
							{ID: 1, Value: 10},
							{ID: 2, Value: 10},
							{ID: 3, Value: 10},
						}},
						CPUS: &libvirtxml.CapsHostNUMACPUs{CPUs: []libvirtxml.CapsHostNUMACPU{
							{ID: 0, Siblings: "0"},
							{ID: 1, Siblings: "1"},
							{ID: 2, Siblings: "2"},
							{ID: 3, Siblings: "3"},
							{ID: 4, Siblings: "4"},
							{ID: 5, Siblings: "5"},
						}},
					},
					{
						Memory: &libvirtxml.CapsHostNUMAMemory{Unit: memoryUnit, Size: 1223960},
						PageInfo: []libvirtxml.CapsHostNUMAPageInfo{
							{Unit: memoryUnit, Size: 4, Count: 297798},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: &libvirtxml.CapsHostNUMADistances{Siblings: []libvirtxml.CapsHostNUMASibling{
							{ID: 0, Value: 10},
							{ID: 1, Value: 10},
							{ID: 2, Value: 10},
							{ID: 3, Value: 10},
						}},
						CPUS: &libvirtxml.CapsHostNUMACPUs{CPUs: []libvirtxml.CapsHostNUMACPU{
							{ID: 0, Siblings: "0"},
							{ID: 1, Siblings: "1"},
							{ID: 2, Siblings: "2"},
							{ID: 3, Siblings: "3"},
							{ID: 4, Siblings: "4"},
							{ID: 5, Siblings: "5"},
						}},
					},
					{
						Memory: &libvirtxml.CapsHostNUMAMemory{Unit: memoryUnit, Size: 1251752},
						PageInfo: []libvirtxml.CapsHostNUMAPageInfo{
							{Unit: memoryUnit, Size: 4, Count: 304746},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: &libvirtxml.CapsHostNUMADistances{Siblings: []libvirtxml.CapsHostNUMASibling{
							{ID: 0, Value: 10},
							{ID: 1, Value: 10},
							{ID: 2, Value: 10},
							{ID: 3, Value: 10},
						}},
						CPUS: &libvirtxml.CapsHostNUMACPUs{CPUs: []libvirtxml.CapsHostNUMACPU{
							{ID: 0, Siblings: "0"},
							{ID: 1, Siblings: "1"},
							{ID: 2, Siblings: "2"},
							{ID: 3, Siblings: "3"},
							{ID: 4, Siblings: "4"},
							{ID: 5, Siblings: "5"},
						}},
					},
					{
						Memory: &libvirtxml.CapsHostNUMAMemory{Unit: memoryUnit, Size: 1289404},
						PageInfo: []libvirtxml.CapsHostNUMAPageInfo{
							{Unit: memoryUnit, Size: 4, Count: 314159},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: &libvirtxml.CapsHostNUMADistances{Siblings: []libvirtxml.CapsHostNUMASibling{
							{ID: 0, Value: 10},
							{ID: 1, Value: 10},
							{ID: 2, Value: 10},
							{ID: 3, Value: 10},
						}},
						CPUS: &libvirtxml.CapsHostNUMACPUs{CPUs: []libvirtxml.CapsHostNUMACPU{
							{ID: 0, Siblings: "0"},
							{ID: 1, Siblings: "1"},
							{ID: 2, Siblings: "2"},
							{ID: 3, Siblings: "3"},
							{ID: 4, Siblings: "4"},
							{ID: 5, Siblings: "5"},
						}},
					},
				},
			}

			actualTopology := capabilitiesToTopology(caps)

			expectedTopology := &cmdv1.Topology{
				NumaCells: []*cmdv1.Cell{
					{
						Memory: &cmdv1.Memory{Unit: memoryUnit, Amount: 1289144},
						Pages: []*cmdv1.Pages{
							{Unit: memoryUnit, Size: 4, Count: 314094},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: []*cmdv1.Sibling{
							{Id: 0, Value: 10},
							{Id: 1, Value: 10},
							{Id: 2, Value: 10},
							{Id: 3, Value: 10},
						},
						Cpus: []*cmdv1.CPU{
							{Id: 0, Siblings: []uint32{0}},
							{Id: 1, Siblings: []uint32{1}},
							{Id: 2, Siblings: []uint32{2}},
							{Id: 3, Siblings: []uint32{3}},
							{Id: 4, Siblings: []uint32{4}},
							{Id: 5, Siblings: []uint32{5}},
						},
					},
					{
						Memory: &cmdv1.Memory{Unit: memoryUnit, Amount: 1223960},
						Pages: []*cmdv1.Pages{
							{Unit: memoryUnit, Size: 4, Count: 297798},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: []*cmdv1.Sibling{
							{Id: 0, Value: 10},
							{Id: 1, Value: 10},
							{Id: 2, Value: 10},
							{Id: 3, Value: 10},
						},
						Cpus: []*cmdv1.CPU{
							{Id: 0, Siblings: []uint32{0}},
							{Id: 1, Siblings: []uint32{1}},
							{Id: 2, Siblings: []uint32{2}},
							{Id: 3, Siblings: []uint32{3}},
							{Id: 4, Siblings: []uint32{4}},
							{Id: 5, Siblings: []uint32{5}},
						},
					},
					{
						Memory: &cmdv1.Memory{Unit: memoryUnit, Amount: 1251752},
						Pages: []*cmdv1.Pages{
							{Unit: memoryUnit, Size: 4, Count: 304746},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: []*cmdv1.Sibling{
							{Id: 0, Value: 10},
							{Id: 1, Value: 10},
							{Id: 2, Value: 10},
							{Id: 3, Value: 10},
						},
						Cpus: []*cmdv1.CPU{
							{Id: 0, Siblings: []uint32{0}},
							{Id: 1, Siblings: []uint32{1}},
							{Id: 2, Siblings: []uint32{2}},
							{Id: 3, Siblings: []uint32{3}},
							{Id: 4, Siblings: []uint32{4}},
							{Id: 5, Siblings: []uint32{5}},
						},
					},
					{
						Memory: &cmdv1.Memory{Unit: memoryUnit, Amount: 1289404},
						Pages: []*cmdv1.Pages{
							{Unit: memoryUnit, Size: 4, Count: 314159},
							{Unit: memoryUnit, Size: 2048, Count: 16},
							{Unit: memoryUnit, Size: 1048576, Count: 0},
						},
						Distances: []*cmdv1.Sibling{
							{Id: 0, Value: 10},
							{Id: 1, Value: 10},
							{Id: 2, Value: 10},
							{Id: 3, Value: 10},
						},
						Cpus: []*cmdv1.CPU{
							{Id: 0, Siblings: []uint32{0}},
							{Id: 1, Siblings: []uint32{1}},
							{Id: 2, Siblings: []uint32{2}},
							{Id: 3, Siblings: []uint32{3}},
							{Id: 4, Siblings: []uint32{4}},
							{Id: 5, Siblings: []uint32{5}},
						},
					},
				},
			}

			Expect(actualTopology).To(Equal(expectedTopology))
		})
		It("should convert libvirtxml.Caps to cmdv1.Topology when certain fields of Caps are not initialized", func() {
			caps := &libvirtxml.Caps{Host: libvirtxml.CapsHost{NUMA: &libvirtxml.CapsHostNUMATopology{}}}
			caps.Host.NUMA.Cells = &libvirtxml.CapsHostNUMACells{
				Cells: []libvirtxml.CapsHostNUMACell{
					{
						Memory: &libvirtxml.CapsHostNUMAMemory{Unit: memoryUnit, Size: 16256896},
						PageInfo: []libvirtxml.CapsHostNUMAPageInfo{
							{Size: 4, Count: 4064224},
							{Unit: memoryUnit, Count: 0},
							{Unit: memoryUnit, Size: 1048576},
						},
						Distances: &libvirtxml.CapsHostNUMADistances{Siblings: []libvirtxml.CapsHostNUMASibling{{ID: 0, Value: 10}}},
						CPUS: &libvirtxml.CapsHostNUMACPUs{CPUs: []libvirtxml.CapsHostNUMACPU{
							{ID: 0},
							{Siblings: "1,5"},
							{ID: 2, Siblings: "2,6"},
							{ID: 3, Siblings: "3,7"},
							{ID: 4, Siblings: "0,4"},
							{ID: 5, Siblings: "1,5"},
							{ID: 6, Siblings: "2,6"},
							{ID: 7, Siblings: "3,7"},
						}},
					},
				},
			}

			actualTopology := capabilitiesToTopology(caps)

			expectedTopology := &cmdv1.Topology{
				NumaCells: []*cmdv1.Cell{
					{
						Memory: &cmdv1.Memory{Unit: memoryUnit, Amount: 16256896},
						Pages: []*cmdv1.Pages{
							{Size: 4, Count: 4064224},
							{Unit: memoryUnit, Count: 0},
							{Unit: memoryUnit, Size: 1048576},
						},
						Distances: []*cmdv1.Sibling{{Id: 0, Value: 10}},
						Cpus: []*cmdv1.CPU{
							{Id: 0},
							{Siblings: []uint32{1, 5}},
							{Id: 2, Siblings: []uint32{2, 6}},
							{Id: 3, Siblings: []uint32{3, 7}},
							{Id: 4, Siblings: []uint32{0, 4}},
							{Id: 5, Siblings: []uint32{1, 5}},
							{Id: 6, Siblings: []uint32{2, 6}},
							{Id: 7, Siblings: []uint32{3, 7}},
						},
					},
				},
			}

			Expect(actualTopology).To(Equal(expectedTopology))
		})
	})
	It("should handle and avoid panic on host NUMA cells with nil values", func() {
		caps := &libvirtxml.Caps{
			Host: libvirtxml.CapsHost{
				NUMA: &libvirtxml.CapsHostNUMATopology{
					Cells: &libvirtxml.CapsHostNUMACells{
						Cells: []libvirtxml.CapsHostNUMACell{
							{
								Memory:    nil,
								Distances: nil,
								CPUS:      nil,
							},
						},
					},
				},
			},
		}

		var actualTopology *cmdv1.Topology
		Expect(func() {
			actualTopology = capabilitiesToTopology(caps)
		}).ShouldNot(Panic())

		expectedTopology := &cmdv1.Topology{
			NumaCells: []*cmdv1.Cell{
				{
					Id:        0,
					Memory:    nil,
					Pages:     nil,
					Distances: nil,
					Cpus:      nil,
				},
			},
		}

		Expect(actualTopology).To(Equal(expectedTopology))
	})
})
