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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Hardware utils test", func() {
	const (
		testPCIAddress = "0000:00:01.0"
	)

	var (
		originalPciBasePath  string
		originalNodeBasePath string

		fakePciBasePath  string
		fakeNodeBasePath string
	)

	createTempSysfsStructure := func() {
		var err error
		fakePciBasePath, err = os.MkdirTemp("", "pci_devices")
		Expect(err).ToNot(HaveOccurred())

		fakeNodeBasePath, err = os.MkdirTemp("", "numa_nodes")
		Expect(err).ToNot(HaveOccurred())

		// Create test PCI device with NUMA node
		pciDevicePath := filepath.Join(fakePciBasePath, testPCIAddress)
		err = os.MkdirAll(pciDevicePath, 0o755)
		Expect(err).ToNot(HaveOccurred())

		numaNodeFile := filepath.Join(pciDevicePath, "numa_node")
		err = os.WriteFile(numaNodeFile, []byte("0\n"), 0o644)
		Expect(err).ToNot(HaveOccurred())

		// Create NUMA node 0 with cpulist
		numaNode0Path := filepath.Join(fakeNodeBasePath, "node0")
		err = os.MkdirAll(numaNode0Path, 0o755)
		Expect(err).ToNot(HaveOccurred())

		// Write cpulist file for NUMA node 0
		cpuListFile := filepath.Join(numaNode0Path, "cpulist")
		err = os.WriteFile(cpuListFile, []byte("0-3\n"), 0o644)
		Expect(err).ToNot(HaveOccurred())

		// Create NUMA node 1 with cpulist
		numaNode1Path := filepath.Join(fakeNodeBasePath, "node1")
		err = os.MkdirAll(numaNode1Path, 0o755)
		Expect(err).ToNot(HaveOccurred())

		// Write cpulist file for NUMA node 1
		cpuListFile1 := filepath.Join(numaNode1Path, "cpulist")
		err = os.WriteFile(cpuListFile1, []byte("4-7\n"), 0o644)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		originalPciBasePath = PciBasePath
		originalNodeBasePath = NodeBasePath

		createTempSysfsStructure()

		// Redirect to fake paths
		PciBasePath = fakePciBasePath
		NodeBasePath = fakeNodeBasePath
	})

	AfterEach(func() {
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

	Context("get device numa node", func() {
		It("should return device NUMA node", func() {
			numaNode, err := GetDeviceNumaNode(testPCIAddress)
			Expect(err).ToNot(HaveOccurred())
			Expect(*numaNode).To(Equal(uint32(0)))
		})
	})

	Context("get device aligned CPUs", func() {
		It("should return device aligned CPUs", func() {
			alignedCPUs, err := GetDeviceAlignedCPUs(testPCIAddress)
			Expect(err).ToNot(HaveOccurred())
			Expect(alignedCPUs).To(Equal([]int{0, 1, 2, 3}))
		})
	})

	Context("get NUMA node CPU list", func() {
		It("should return CPU list for NUMA node", func() {
			cpuList, err := GetNumaNodeCPUList(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(cpuList).To(Equal([]int{0, 1, 2, 3}))

			cpuList, err = GetNumaNodeCPUList(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(cpuList).To(Equal([]int{4, 5, 6, 7}))
		})
	})

	Context("device vCPU affinity", func() {
		It("should handle empty CPU tune configuration", func() {
			domainSpec := &api.DomainSpec{
				CPUTune: &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{},
				},
			}

			vcpuList, err := LookupDeviceVCPUAffinity(testPCIAddress, domainSpec)
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

			vcpuList, err := LookupDeviceVCPUAffinity(testPCIAddress, domainSpec)
			Expect(err).ToNot(HaveOccurred())
			// Device on NUMA node 0 has CPUs 0-3, and we have vCPUs pinned to 0, 1, 2
			Expect(vcpuList).To(ConsistOf(uint32(0), uint32(1), uint32(2)))
		})

		It("should handle complex CPU tune configurations", func() {
			domainSpec := &api.DomainSpec{
				CPUTune: &api.CPUTune{
					VCPUPin: []api.CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "0"}, // CPU 0 (NUMA node 0)
						{VCPU: 1, CPUSet: "3"}, // CPU 3 (NUMA node 0)
						{VCPU: 2, CPUSet: "4"}, // CPU 4 (NUMA node 1) - no overlap with device NUMA node
					},
				},
			}

			vcpuList, err := LookupDeviceVCPUAffinity(testPCIAddress, domainSpec)
			Expect(err).ToNot(HaveOccurred())
			// Device on NUMA node 0 has CPUs 0-3, so vCPUs 0 and 1 have CPUs on the same NUMA node
			// vCPU 2 only has CPUs on NUMA node 1, so it's not included
			Expect(vcpuList).To(ConsistOf(uint32(0), uint32(1)))
		})
	})

	Context("devices NUMA affinity", func() {
		It("should return an empty result for no PCI addresses", func() {
			domainSpec := &api.DomainSpec{}
			devicesNumaNodes := LookupDevicesNumaNodes([]string{}, domainSpec)
			Expect(devicesNumaNodes).To(BeEmpty())
		})

		It("should return an empty result for nil domain spec", func() {
			devicesNumaNodes := LookupDevicesNumaNodes([]string{testPCIAddress}, nil)
			Expect(devicesNumaNodes).To(BeEmpty())
		})

		It("should return an empty result when domain spec has no NUMA info", func() {
			domainSpec := &api.DomainSpec{
				CPU: api.CPU{},
			}
			devicesNumaNodes := LookupDevicesNumaNodes([]string{testPCIAddress}, domainSpec)
			Expect(devicesNumaNodes).To(BeEmpty())
		})

		It("should handle domain spec with NUMA cells but no vCPU affinity", func() {
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

			devicesNumaNodes := LookupDevicesNumaNodes([]string{testPCIAddress}, domainSpec)
			Expect(devicesNumaNodes).To(BeEmpty())
		})

		It("should return devices vCPU NUMA nodes for their aligned vCPUs", func() {
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
			devicesNumaNodes := LookupDevicesNumaNodes([]string{testPCIAddress}, domainSpec)
			Expect(devicesNumaNodes).ToNot(BeEmpty())
			Expect(devicesNumaNodes).To(HaveKey(testPCIAddress))
			Expect(devicesNumaNodes[testPCIAddress]).To(Equal(uint32(0)))
		})
	})
})
