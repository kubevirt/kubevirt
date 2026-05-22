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

package iommu

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func createMockNUMANode(nodeID int, distances string, memTotalKB int, cpulist string) {
	nodeDir := filepath.Join(SysfsNodeBasePath, fmt.Sprintf("node%d", nodeID))
	Expect(os.MkdirAll(nodeDir, 0755)).To(Succeed())

	Expect(os.WriteFile(filepath.Join(nodeDir, "distance"), []byte(distances+"\n"), 0644)).To(Succeed())

	meminfo := fmt.Sprintf("Node %d MemTotal:       %d kB\nNode %d MemFree:        0 kB\n", nodeID, memTotalKB, nodeID)
	Expect(os.WriteFile(filepath.Join(nodeDir, "meminfo"), []byte(meminfo), 0644)).To(Succeed())

	Expect(os.WriteFile(filepath.Join(nodeDir, "cpulist"), []byte(cpulist+"\n"), 0644)).To(Succeed())
}

func createMockPCIDevice(bdf string, numaNode int) {
	devDir := filepath.Join(SysfsPCIBasePath, bdf)
	Expect(os.MkdirAll(devDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(devDir, "numa_node"), []byte(fmt.Sprintf("%d\n", numaNode)), 0644)).To(Succeed())
}

// setupGrace2Socket2GPU creates a mock sysfs layout resembling a real
// 2-superchip GB200 with 2 GPUs, both on socket 0. On real hardware,
// GPUs report the CPU socket NUMA node (0), not a GI node.
//
// Real GB200 layout from VOYAGER-707:
//
//	Node 0:    Grace CPU #1 (CPUs 0-71, ~490 GB LPDDR5X)
//	Node 1:    Grace CPU #2 (CPUs 72-143, ~490 GB LPDDR5X)
//	Nodes 2-9:  GI/MIG nodes for GPU 0 (memory-less, CPU-less)
//	Nodes 10-17: GI/MIG nodes for GPU 1 (memory-less, CPU-less)
//
// GPU PCI devices report numa_node=0 (CPU socket), NOT GI node IDs.
func setupGrace2Socket2GPU() {
	Expect(os.WriteFile(filepath.Join(SysfsNodeBasePath, "online"), []byte("0-17\n"), 0644)).To(Succeed())

	//                        0   1   2   3   4   5   6   7   8   9  10  11  12  13  14  15  16  17
	node0Dist := "10 40 80 80 80 80 80 80 80 80 120 120 120 120 120 120 120 120"
	node1Dist := "40 10 120 120 120 120 120 120 120 120 80 80 80 80 80 80 80 80"

	createMockNUMANode(0, node0Dist, 468713472, "0-59")
	createMockNUMANode(1, node1Dist, 468713472, "60-119")

	giDistTemplate0 := []string{
		"80 120 10 11 11 11 11 11 11 11 40 40 40 40 40 40 40 40",
		"80 120 11 10 11 11 11 11 11 11 40 40 40 40 40 40 40 40",
		"80 120 11 11 10 11 11 11 11 11 40 40 40 40 40 40 40 40",
		"80 120 11 11 11 10 11 11 11 11 40 40 40 40 40 40 40 40",
		"80 120 11 11 11 11 10 11 11 11 40 40 40 40 40 40 40 40",
		"80 120 11 11 11 11 11 10 11 11 40 40 40 40 40 40 40 40",
		"80 120 11 11 11 11 11 11 10 11 40 40 40 40 40 40 40 40",
		"80 120 11 11 11 11 11 11 11 10 40 40 40 40 40 40 40 40",
	}
	for i, dist := range giDistTemplate0 {
		createMockNUMANode(2+i, dist, 0, "")
	}

	giDistTemplate1 := []string{
		"120 80 40 40 40 40 40 40 40 40 10 11 11 11 11 11 11 11",
		"120 80 40 40 40 40 40 40 40 40 11 10 11 11 11 11 11 11",
		"120 80 40 40 40 40 40 40 40 40 11 11 10 11 11 11 11 11",
		"120 80 40 40 40 40 40 40 40 40 11 11 11 10 11 11 11 11",
		"120 80 40 40 40 40 40 40 40 40 11 11 11 11 10 11 11 11",
		"120 80 40 40 40 40 40 40 40 40 11 11 11 11 11 10 11 11",
		"120 80 40 40 40 40 40 40 40 40 11 11 11 11 11 11 10 11",
		"120 80 40 40 40 40 40 40 40 40 11 11 11 11 11 11 11 10",
	}
	for i, dist := range giDistTemplate1 {
		createMockNUMANode(10+i, dist, 0, "")
	}

	createMockPCIDevice("0008:01:00.0", 0)
	createMockPCIDevice("0009:01:00.0", 0)
}

func buildGrace2GPUDomain() *api.DomainSpec {
	domain := &api.DomainSpec{
		NUMATune: &api.NUMATune{
			MemNodes: []api.MemNode{
				{CellID: 0, Mode: "strict", NodeSet: "0"},
				{CellID: 1, Mode: "strict", NodeSet: "1"},
			},
		},
	}
	domain.CPU.NUMA = &api.NUMA{
		Cells: []api.NUMACell{
			{ID: "0", CPUs: "0-59", Memory: pointer.P(uint64(468713472)), Unit: "KiB"},
			{ID: "1", CPUs: "60-119", Memory: pointer.P(uint64(468713472)), Unit: "KiB"},
		},
	}
	for i := 2; i <= 17; i++ {
		domain.CPU.NUMA.Cells = append(domain.CPU.NUMA.Cells, api.NUMACell{
			ID:     fmt.Sprintf("%d", i),
			Memory: pointer.P(uint64(0)),
			Unit:   "KiB",
		})
	}
	domain.Devices.HostDevices = []api.HostDevice{
		{
			Source: api.HostDeviceSource{
				Address: &api.Address{Domain: "0x0008", Bus: "0x01", Slot: "0x00", Function: "0x0"},
			},
			ACPI: &api.ACPIHostDev{NodeSet: "2-9"},
		},
		{
			Source: api.HostDeviceSource{
				Address: &api.Address{Domain: "0x0009", Bus: "0x01", Slot: "0x00", Function: "0x0"},
			},
			ACPI: &api.ACPIHostDev{NodeSet: "10-17"},
		},
	}
	return domain
}

func findSibling(siblings []api.NUMACellSibling, targetID string) *api.NUMACellSibling {
	for i := range siblings {
		if siblings[i].ID == targetID {
			return &siblings[i]
		}
	}
	return nil
}

var _ = Describe("NUMA Distances", func() {
	BeforeEach(func() {
		tmpDir := GinkgoT().TempDir()

		origNodeBase := SysfsNodeBasePath
		origPCIBase := SysfsPCIBasePath
		SysfsNodeBasePath = filepath.Join(tmpDir, "sys/devices/system/node")
		SysfsPCIBasePath = filepath.Join(tmpDir, "sys/bus/pci/devices")

		Expect(os.MkdirAll(SysfsNodeBasePath, 0755)).To(Succeed())
		Expect(os.MkdirAll(SysfsPCIBasePath, 0755)).To(Succeed())

		DeferCleanup(func() {
			SysfsNodeBasePath = origNodeBase
			SysfsPCIBasePath = origPCIBase
		})
	})

	Describe("readHostNUMADistances", func() {
		It("should parse sysfs distance files correctly", func() {
			Expect(os.WriteFile(filepath.Join(SysfsNodeBasePath, "online"), []byte("0-1\n"), 0644)).To(Succeed())
			createMockNUMANode(0, "10 40", 468713472, "0-59")
			createMockNUMANode(1, "40 10", 468713472, "60-119")

			distances, err := readHostNUMADistances([]int{0, 1})
			Expect(err).NotTo(HaveOccurred())

			Expect(distances[0][0]).To(Equal(uint64(10)))
			Expect(distances[0][1]).To(Equal(uint64(40)))
			Expect(distances[1][0]).To(Equal(uint64(40)))
			Expect(distances[1][1]).To(Equal(uint64(10)))
		})
	})

	Describe("discoverAllGINodes", func() {
		It("should find all memory-less CPU-less nodes", func() {
			Expect(os.WriteFile(filepath.Join(SysfsNodeBasePath, "online"), []byte("0-17\n"), 0644)).To(Succeed())
			createMockNUMANode(0, "10 40", 468713472, "0-59")
			createMockNUMANode(1, "40 10", 468713472, "60-119")
			for i := 2; i <= 17; i++ {
				createMockNUMANode(i, "10", 0, "")
			}

			giNodes, err := discoverAllGINodes()
			Expect(err).NotTo(HaveOccurred())
			Expect(giNodes).To(HaveLen(16))
			Expect(giNodes[0]).To(Equal(2))
			Expect(giNodes[15]).To(Equal(17))
		})
	})

	Describe("buildGuestToHostMapping", func() {
		It("should map CPU cells from NUMATune and GI cells from sysfs", func() {
			setupGrace2Socket2GPU()

			domain := buildGrace2GPUDomain()
			mapping, err := buildGuestToHostMapping(domain)
			Expect(err).NotTo(HaveOccurred())

			// CPU cells mapped via NUMATune
			Expect(mapping[0]).To(Equal(0))
			Expect(mapping[1]).To(Equal(1))
			// GPU 0: first GI cell maps to CPU socket (0), rest to host GI nodes 2-8
			Expect(mapping[2]).To(Equal(0), "GPU 0 first GI cell → CPU socket")
			for i := 0; i < 7; i++ {
				Expect(mapping[3+i]).To(Equal(2+i), "GPU 0 GI cell %d → host node %d", 3+i, 2+i)
			}
			// GPU 1: first GI cell maps to CPU socket (0), rest to host GI nodes 10-16
			Expect(mapping[10]).To(Equal(0), "GPU 1 first GI cell → CPU socket")
			for i := 0; i < 7; i++ {
				Expect(mapping[11+i]).To(Equal(10+i), "GPU 1 GI cell %d → host node %d", 11+i, 10+i)
			}
		})
	})

	Describe("applyNUMADistances", func() {
		Context("with a Grace 2-socket 2-GPU topology", func() {
			BeforeEach(func() {
				setupGrace2Socket2GPU()
			})

			It("should set correct distances on CPU cell 0", func() {
				domain := buildGrace2GPUDomain()
				applyNUMADistances(domain)

				cell0 := domain.CPU.NUMA.Cells[0]
				Expect(cell0.Distances).NotTo(BeNil())
				Expect(findSibling(cell0.Distances.Siblings, "0").Value).To(Equal(uint64(10))) // self
				Expect(findSibling(cell0.Distances.Siblings, "1").Value).To(Equal(uint64(40))) // cross-socket
				// Guest cell 2 (GPU 0 first GI) maps to host node 0 (CPU socket);
				// bumped from 10→11 because libvirt reserves 10 for self-distance only
				Expect(findSibling(cell0.Distances.Siblings, "2").Value).To(Equal(uint64(11)))
				// Guest cell 3 (GPU 0 second GI) maps to host GI node 2
				Expect(findSibling(cell0.Distances.Siblings, "3").Value).To(Equal(uint64(80))) // CPU→GI (local)
			})

			It("should set correct distances on CPU cell 1", func() {
				domain := buildGrace2GPUDomain()
				applyNUMADistances(domain)

				cell1 := domain.CPU.NUMA.Cells[1]
				Expect(cell1.Distances).NotTo(BeNil())
				Expect(findSibling(cell1.Distances.Siblings, "0").Value).To(Equal(uint64(40))) // cross-socket
				Expect(findSibling(cell1.Distances.Siblings, "1").Value).To(Equal(uint64(10))) // self
			})

			It("should set correct distances on GI cell 3 (GPU 0, host GI node 2)", func() {
				domain := buildGrace2GPUDomain()
				applyNUMADistances(domain)

				cell3 := domain.CPU.NUMA.Cells[3]
				Expect(cell3.Distances).NotTo(BeNil())
				Expect(findSibling(cell3.Distances.Siblings, "0").Value).To(Equal(uint64(80)))  // local CPU
				Expect(findSibling(cell3.Distances.Siblings, "1").Value).To(Equal(uint64(120))) // remote CPU
				Expect(findSibling(cell3.Distances.Siblings, "3").Value).To(Equal(uint64(10)))  // self (host node 2)
				Expect(findSibling(cell3.Distances.Siblings, "4").Value).To(Equal(uint64(11)))  // same GPU group
			})

			It("should set distances on mapped cells", func() {
				domain := buildGrace2GPUDomain()
				applyNUMADistances(domain)

				// CPU cells and mapped GI cells should have distances
				Expect(domain.CPU.NUMA.Cells[0].Distances).NotTo(BeNil(), "CPU cell 0")
				Expect(domain.CPU.NUMA.Cells[1].Distances).NotTo(BeNil(), "CPU cell 1")
				Expect(domain.CPU.NUMA.Cells[2].Distances).NotTo(BeNil(), "GPU 0 first GI cell")
				Expect(domain.CPU.NUMA.Cells[3].Distances).NotTo(BeNil(), "GPU 0 GI cell (host node 2)")
			})
		})

		Context("with a single GPU sharing the CPU NUMA node", func() {
			BeforeEach(func() {
				Expect(os.WriteFile(filepath.Join(SysfsNodeBasePath, "online"), []byte("0-8\n"), 0644)).To(Succeed())

				//                 0   1   2   3   4   5   6   7   8
				node0Dist := "10 80 80 80 80 80 80 80 80"
				createMockNUMANode(0, node0Dist, 468713472, "0-1")

				giDists := []string{
					"80 10 11 11 11 11 11 11 11",
					"80 11 10 11 11 11 11 11 11",
					"80 11 11 10 11 11 11 11 11",
					"80 11 11 11 10 11 11 11 11",
					"80 11 11 11 11 10 11 11 11",
					"80 11 11 11 11 11 10 11 11",
					"80 11 11 11 11 11 11 10 11",
					"80 11 11 11 11 11 11 11 10",
				}
				for i, dist := range giDists {
					createMockNUMANode(1+i, dist, 0, "")
				}

				createMockPCIDevice("0009:01:00.0", 0)
			})

			It("should bump distance from 10 to 11 when GPU first GI cell maps to same host node as CPU", func() {
				domain := &api.DomainSpec{
					NUMATune: &api.NUMATune{
						MemNodes: []api.MemNode{
							{CellID: 0, Mode: "strict", NodeSet: "0"},
						},
					},
				}
				domain.CPU.NUMA = &api.NUMA{
					Cells: []api.NUMACell{
						{ID: "0", CPUs: "0,1", Memory: pointer.P(uint64(8589934592)), Unit: "b"},
					},
				}
				for i := 1; i <= 8; i++ {
					domain.CPU.NUMA.Cells = append(domain.CPU.NUMA.Cells, api.NUMACell{
						ID:     fmt.Sprintf("%d", i),
						Memory: pointer.P(uint64(0)),
						Unit:   "KiB",
					})
				}
				domain.Devices.HostDevices = []api.HostDevice{
					{
						Source: api.HostDeviceSource{
							Address: &api.Address{Domain: "0x0009", Bus: "0x01", Slot: "0x00", Function: "0x0"},
						},
						ACPI: &api.ACPIHostDev{NodeSet: "1-8"},
					},
				}

				applyNUMADistances(domain)

				cell0 := domain.CPU.NUMA.Cells[0]
				Expect(cell0.Distances).NotTo(BeNil())
				Expect(findSibling(cell0.Distances.Siblings, "0").Value).To(Equal(uint64(10)))
				Expect(findSibling(cell0.Distances.Siblings, "1").Value).To(Equal(uint64(11)))
				Expect(findSibling(cell0.Distances.Siblings, "2").Value).To(Equal(uint64(80)))

				cell1 := domain.CPU.NUMA.Cells[1]
				Expect(cell1.Distances).NotTo(BeNil())
				Expect(findSibling(cell1.Distances.Siblings, "0").Value).To(Equal(uint64(11)))
				Expect(findSibling(cell1.Distances.Siblings, "1").Value).To(Equal(uint64(10)))
			})
		})

		Context("graceful degradation", func() {
			It("should be a no-op when there are no NUMA cells", func() {
				domain := &api.DomainSpec{}
				applyNUMADistances(domain)

				domain.CPU.NUMA = &api.NUMA{Cells: []api.NUMACell{}}
				applyNUMADistances(domain)
			})

			It("should still apply CPU-to-CPU distances when there are no GI devices", func() {
				Expect(os.WriteFile(filepath.Join(SysfsNodeBasePath, "online"), []byte("0-1\n"), 0644)).To(Succeed())
				createMockNUMANode(0, "10 40", 468713472, "0-59")
				createMockNUMANode(1, "40 10", 468713472, "60-119")

				domain := &api.DomainSpec{
					NUMATune: &api.NUMATune{
						MemNodes: []api.MemNode{
							{CellID: 0, Mode: "strict", NodeSet: "0"},
							{CellID: 1, Mode: "strict", NodeSet: "1"},
						},
					},
				}
				domain.CPU.NUMA = &api.NUMA{
					Cells: []api.NUMACell{
						{ID: "0", CPUs: "0-59", Memory: pointer.P(uint64(468713472)), Unit: "KiB"},
						{ID: "1", CPUs: "60-119", Memory: pointer.P(uint64(468713472)), Unit: "KiB"},
					},
				}

				applyNUMADistances(domain)

				Expect(domain.CPU.NUMA.Cells[0].Distances).NotTo(BeNil())
				Expect(findSibling(domain.CPU.NUMA.Cells[0].Distances.Siblings, "0").Value).To(Equal(uint64(10)))
				Expect(findSibling(domain.CPU.NUMA.Cells[0].Distances.Siblings, "1").Value).To(Equal(uint64(40)))
			})
		})
	})
})
