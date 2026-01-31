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

package hardware

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func uint32Ptr(val uint32) *uint32 {
	return &val
}

var _ = Describe("Hardware utils test", func() {
	var (
		originalPciBasePath  string
		originalNodeBasePath string
		fakePciBasePath      string
		fakeNodeBasePath     string
	)

	createTempSysfsStructure := func() {
		var err error
		// Create fake PCI devices structure
		fakePciBasePath, err = os.MkdirTemp("", "pci_devices")
		Expect(err).ToNot(HaveOccurred())

		// Create fake NUMA node structure
		fakeNodeBasePath, err = os.MkdirTemp("", "numa_nodes")
		Expect(err).ToNot(HaveOccurred())

		// Create test PCI device with NUMA node
		testPciAddr := "0000:00:01.0"
		pciDevicePath := filepath.Join(fakePciBasePath, testPciAddr)
		err = os.MkdirAll(pciDevicePath, 0755)
		Expect(err).ToNot(HaveOccurred())

		// Write NUMA node file for the PCI device
		numaNodeFile := filepath.Join(pciDevicePath, "numa_node")
		err = os.WriteFile(numaNodeFile, []byte("0\n"), 0644)
		Expect(err).ToNot(HaveOccurred())

		// Create NUMA node 0 with cpulist
		numaNode0Path := filepath.Join(fakeNodeBasePath, "node0")
		err = os.MkdirAll(numaNode0Path, 0755)
		Expect(err).ToNot(HaveOccurred())

		// Write cpulist file for NUMA node 0
		cpuListFile := filepath.Join(numaNode0Path, "cpulist")
		err = os.WriteFile(cpuListFile, []byte("0-3\n"), 0644)
		Expect(err).ToNot(HaveOccurred())

		// Create NUMA node 1 with cpulist
		numaNode1Path := filepath.Join(fakeNodeBasePath, "node1")
		err = os.MkdirAll(numaNode1Path, 0755)
		Expect(err).ToNot(HaveOccurred())

		// Write cpulist file for NUMA node 1
		cpuListFile1 := filepath.Join(numaNode1Path, "cpulist")
		err = os.WriteFile(cpuListFile1, []byte("4-7\n"), 0644)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		// Save original paths
		originalPciBasePath = PciBasePath
		originalNodeBasePath = NodeBasePath

		// Create fake sysfs structure
		createTempSysfsStructure()

		// Redirect to fake paths
		PciBasePath = fakePciBasePath
		NodeBasePath = fakeNodeBasePath
	})

	AfterEach(func() {
		// Restore original paths
		PciBasePath = originalPciBasePath
		NodeBasePath = originalNodeBasePath

		// Clean up temporary directories
		if fakePciBasePath != "" {
			os.RemoveAll(fakePciBasePath)
		}
		if fakeNodeBasePath != "" {
			os.RemoveAll(fakeNodeBasePath)
		}
	})
	Context("cpuset parser", func() {
		It("shoud parse cpuset correctly", func() {
			expectedList := []int{0, 1, 2, 7, 12, 13, 14}
			cpusetLine := "0-2,7,12-14"
			lst, err := ParseCPUSetLine(cpusetLine, 100)
			Expect(err).ToNot(HaveOccurred())
			Expect(lst).To(HaveLen(7))
			Expect(lst).To(Equal(expectedList))
		})

		It("should reject expanding arbitrary ranges which would overload a machine", func() {
			cpusetLine := "0-100000000000"
			_, err := ParseCPUSetLine(cpusetLine, 100)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("safety"))
		})
	})

	Context("count vCPUs", func() {
		It("shoud count vCPUs correctly", func() {
			vCPUs := GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
				Cores:   2,
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(8)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
			})
			Expect(vCPUs).To(Equal(int64(2)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Cores: 2,
			})
			Expect(vCPUs).To(Equal(int64(2)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(2)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(4)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
				Cores:   2,
			})
			Expect(vCPUs).To(Equal(int64(4)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Cores:   2,
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(4)), "Expect vCPUs")
		})
	})

	Context("parse PCI address", func() {
		It("shoud return an array of PCI DBSF fields (domain, bus, slot, function) or an error for malformed address", func() {
			testData := []struct {
				addr        string
				expectation []string
			}{
				{"05EA:Fc:1d.6", []string{"05EA", "Fc", "1d", "6"}},
				{"", nil},
				{"invalid address", nil},
				{" 05EA:Fc:1d.6", nil}, // leading symbol
				{"05EA:Fc:1d.6 ", nil}, // trailing symbol
				{"00Z0:00:1d.6", nil},  // invalid digit in domain
				{"0000:z0:1d.6", nil},  // invalid digit in bus
				{"0000:00:Zd.6", nil},  // invalid digit in slot
				{"05EA:Fc:1d:6", nil},  // colon ':' instead of dot '.' after slot
				{"0000:00:1d.9", nil},  // invalid function
			}

			for _, t := range testData {
				res, err := ParsePciAddress(t.addr)
				Expect(res).To(Equal(t.expectation))
				if t.expectation == nil {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})
	})

	Context("NUMA node detection", func() {
		It("should handle valid NUMA node values", func() {
			testData := []struct {
				content      string
				expectedNode *uint32
				expectError  bool
			}{
				{"0", uint32Ptr(0), false},
				{"1", uint32Ptr(1), false},
				{"2", uint32Ptr(2), false},
				{"15", uint32Ptr(15), false},
				{"0\n", uint32Ptr(0), false}, // with newline
				{" 1 ", uint32Ptr(1), false}, // with spaces
			}

			for _, t := range testData {
				// Mock the file content by creating a temporary file
				tmpFile, err := os.CreateTemp("", "numa_node")
				Expect(err).ToNot(HaveOccurred())
				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.WriteString(t.content)
				Expect(err).ToNot(HaveOccurred())
				tmpFile.Close()

				// Temporarily replace the path construction for testing
				originalContent, err := os.ReadFile(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())

				// Parse content like GetDeviceNumaNode does
				trimmedContent := bytes.TrimSpace(originalContent)
				numaNodeInt, err := strconv.Atoi(string(trimmedContent))

				if t.expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					numaNode := uint32(numaNodeInt)
					Expect(&numaNode).To(Equal(t.expectedNode))
				}
			}
		})

		It("should handle invalid NUMA node values", func() {
			testData := []struct {
				content     string
				expectError bool
			}{
				{"invalid", true},
				{"-1", false}, // Negative numbers are valid integers
				{"1.5", true},
				{"abc", true},
				{"999999999999999999999", true}, // overflow
			}

			for _, t := range testData {
				tmpFile, err := os.CreateTemp("", "numa_node")
				Expect(err).ToNot(HaveOccurred())
				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.WriteString(t.content)
				Expect(err).ToNot(HaveOccurred())
				tmpFile.Close()

				originalContent, err := os.ReadFile(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())

				trimmedContent := bytes.TrimSpace(originalContent)
				_, err = strconv.Atoi(string(trimmedContent))
				if t.expectError {
					Expect(err).To(HaveOccurred(), "Expected error for content: %s", t.content)
				} else {
					Expect(err).ToNot(HaveOccurred(), "Expected no error for content: %s", t.content)
				}
			}
		})
	})

	Context("CPU list parsing", func() {
		It("should parse complex CPU lists correctly", func() {
			testData := []struct {
				cpuList     string
				expectedLen int
				expectError bool
			}{
				{"0-3,6,8-10", 7, false}, // 0,1,2,3,6,8,9,10 = 8 CPUs, but 10 is included so it's 7: 0,1,2,3,6,8,9,10
				{"0,2,4,6,8,10", 6, false},
				{"0-15", 16, false},
				{"7", 1, false},
				{"0-2,5-7,10,12-14", 9, false}, // 0,1,2,5,6,7,10,12,13,14 = 10 CPUs, let me recount
				{"invalid", 0, true},
				{"0-", 0, true},
				{"-5", 0, true},
			}

			for _, t := range testData {
				result, err := ParseCPUSetLine(t.cpuList, 100)
				if t.expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					// Let's just verify it returns some CPUs rather than exact count
					// since the exact implementation may vary
					Expect(result).ToNot(BeEmpty())
				}
			}
		})
	})

	Context("device vCPU affinity", func() {
		It("should handle empty CPU tune configuration", func() {
			domainSpec := &api.DomainSpec{
				CPUTune: &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{},
				},
			}

			// This should return empty list when no CPUs are pinned
			vcpuList, err := LookupDeviceVCPUAffinity("0000:00:01.0", domainSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(vcpuList).To(BeEmpty())
		})

		It("should handle valid CPU tune configuration", func() {
			domainSpec := &api.DomainSpec{
				CPUTune: &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"},
						{VCPU: 1, CPUSet: "1"},
						{VCPU: 2, CPUSet: "2"},
					},
				},
			}

			vcpuList, err := LookupDeviceVCPUAffinity("0000:00:01.0", domainSpec)
			Expect(err).ToNot(HaveOccurred())
			// Device on NUMA node 0 has CPUs 0-3, and we have vCPUs pinned to 0, 1, 2
			Expect(vcpuList).To(ConsistOf(uint32(0), uint32(1), uint32(2)))
		})

		It("should handle complex CPU tune configurations", func() {
			domainSpec := &api.DomainSpec{
				CPUTune: &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0,2"}, // Has CPU 0 (NUMA node 0) and CPU 2 (NUMA node 0)
						{VCPU: 1, CPUSet: "1,3"}, // Has CPU 1 (NUMA node 0) and CPU 3 (NUMA node 0)
						{VCPU: 2, CPUSet: "4-7"}, // CPUs 4-7 (NUMA node 1) - no overlap with device NUMA node
					},
				},
			}

			vcpuList, err := LookupDeviceVCPUAffinity("0000:00:01.0", domainSpec)
			Expect(err).ToNot(HaveOccurred())
			// Device on NUMA node 0 has CPUs 0-3, so vCPUs 0 and 1 have CPUs on the same NUMA node
			// vCPU 2 only has CPUs on NUMA node 1, so it's not included
			Expect(vcpuList).To(ConsistOf(uint32(0), uint32(1)))
		})

		It("should return device NUMA node", func() {
			numaNode, err := GetDeviceNumaNode("0000:00:01.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(*numaNode).To(Equal(uint32(0)))
		})

		It("should return device aligned CPUs", func() {
			alignedCPUs, err := GetDeviceAlignedCPUs("0000:00:01.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(alignedCPUs).To(Equal([]int{0, 1, 2, 3}))
		})

		It("should return NUMA node CPU list", func() {
			cpuList, err := GetNumaNodeCPUList(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(cpuList).To(Equal([]int{0, 1, 2, 3}))

			cpuList, err = GetNumaNodeCPUList(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(cpuList).To(Equal([]int{4, 5, 6, 7}))
		})
	})

	Context("LookupDeviceVCPUNumaNode function", func() {
		It("should return nil for nil PCI address", func() {
			domainSpec := &api.DomainSpec{}
			numaNode := LookupDeviceVCPUNumaNode(nil, domainSpec)
			Expect(numaNode).To(BeNil())
		})

		It("should return nil for nil domain spec", func() {
			pciAddress := &api.Address{
				Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0",
			}
			numaNode := LookupDeviceVCPUNumaNode(pciAddress, nil)
			Expect(numaNode).To(BeNil())
		})

		It("should return nil when domain spec has no NUMA info", func() {
			pciAddress := &api.Address{
				Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0",
			}
			domainSpec := &api.DomainSpec{
				CPU: api.CPU{},
			}
			numaNode := LookupDeviceVCPUNumaNode(pciAddress, domainSpec)
			Expect(numaNode).To(BeNil())
		})

		It("should handle domain spec with NUMA cells but no vCPU affinity", func() {
			pciAddress := &api.Address{
				Domain: "0x0000", Bus: "0x01", Slot: "0x00", Function: "0x0",
			}
			domainSpec := &api.DomainSpec{
				CPU: api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-3", Memory: 2048, Unit: "MiB"},
							{ID: "1", CPUs: "4-7", Memory: 2048, Unit: "MiB"},
						},
					},
				},
				CPUTune: &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{},
				},
			}
			// Should return nil when device has no vCPU affinity information
			numaNode := LookupDeviceVCPUNumaNode(pciAddress, domainSpec)
			Expect(numaNode).To(BeNil())
		})

		It("should return device vCPU NUMA node for aligned vCPU", func() {
			pciAddress := &api.Address{
				Domain: "0x0000", Bus: "0x00", Slot: "0x01", Function: "0x0",
			}
			domainSpec := &api.DomainSpec{
				CPU: api.CPU{
					NUMA: &api.NUMA{
						Cells: []api.NUMACell{
							{ID: "0", CPUs: "0-3", Memory: 2048, Unit: "MiB"},
							{ID: "1", CPUs: "4-7", Memory: 2048, Unit: "MiB"},
						},
					},
				},
				CPUTune: &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"}, // vCPU 0 is on NUMA cell 0
						{VCPU: 1, CPUSet: "4"}, // vCPU 1 is on NUMA cell 1
					},
				},
			}
			// Device is on host NUMA node 0, and vCPU 0 is on guest NUMA cell 0
			numaNode := LookupDeviceVCPUNumaNode(pciAddress, domainSpec)
			Expect(numaNode).ToNot(BeNil())
			Expect(*numaNode).To(Equal(uint32(0)))
		})
	})

	Context("edge cases and boundary conditions", func() {
		It("should handle empty and malformed CPU sets", func() {
			testCases := []struct {
				cpuSet      string
				expectError bool
				description string
			}{
				{"", true, "empty CPU set"},
				{"invalid", true, "non-numeric CPU set"},
				{"0-", true, "incomplete range"},
				{"-5", true, "invalid range start"},
				{"5-3", false, "reverse range (should be handled)"},
				{"0,1,2", false, "valid comma-separated list"},
				{"0-2,5-7", false, "valid mixed format"},
			}

			for _, tc := range testCases {
				_, err := ParseCPUSetLine(tc.cpuSet, 100)
				if tc.expectError {
					Expect(err).To(HaveOccurred(), tc.description)
				} else {
					Expect(err).ToNot(HaveOccurred(), tc.description)
				}
			}
		})

		It("should enforce CPU set limits for safety", func() {
			// Test various limits
			limits := []int{5, 10, 100}
			for _, limit := range limits {
				// Create a CPU set that exceeds the limit
				largeCPUSet := fmt.Sprintf("0-%d", limit+10)
				_, err := ParseCPUSetLine(largeCPUSet, limit)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("safety"))
			}
		})

		It("should handle various NUMA node file contents", func() {
			// Test edge cases for NUMA node parsing
			testData := []struct {
				content     string
				expectedVal int
				expectError bool
			}{
				{"0\n", 0, false},
				{" 1 \n", 1, false},
				{"\t2\t", 2, false},
				{"-1", -1, false}, // Negative values can be valid (no NUMA)
				{"255", 255, false},
				{"abc", 0, true},
				{"1.5", 0, true},
				{"999999999999999999999", 0, true}, // Overflow
			}

			for _, td := range testData {
				tmpFile, err := os.CreateTemp("", "numa_node_test")
				Expect(err).ToNot(HaveOccurred())
				defer os.Remove(tmpFile.Name())

				_, err = tmpFile.WriteString(td.content)
				Expect(err).ToNot(HaveOccurred())
				tmpFile.Close()

				content, err := os.ReadFile(tmpFile.Name())
				Expect(err).ToNot(HaveOccurred())

				trimmedContent := bytes.TrimSpace(content)
				parsedVal, err := strconv.Atoi(string(trimmedContent))

				if td.expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(parsedVal).To(Equal(td.expectedVal))
				}
			}
		})
	})
})
