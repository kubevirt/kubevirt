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

package converter

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	iommupci "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/iommu-pci"
)

var _ = Describe("PCI Placement IOMMU isolation", func() {
	var (
		originalPciBasePath  string
		originalNodeBasePath string
		fakePciBasePath      string
		fakeNodeBasePath     string
	)

	BeforeEach(func() {
		iommupci.ParseConfigHybridFn = func(_ string) (bool, bool, bool, int, int, error) {
			return true, true, true, 20, 48, nil
		}
		iommupci.CalculatePCIHole64SizeFn = func(_ string) (uint64, error) {
			return 64 * 1024 * 1024 * 1024, nil
		}

		originalPciBasePath = hardware.PciBasePath
		originalNodeBasePath = hardware.NodeBasePath

		var err error
		fakePciBasePath, err = os.MkdirTemp("", "iommu_pci_devices")
		Expect(err).ToNot(HaveOccurred())
		fakeNodeBasePath, err = os.MkdirTemp("", "iommu_numa_nodes")
		Expect(err).ToNot(HaveOccurred())

		for pciAddr, numaNode := range map[string]string{
			"0000:01:00.0": "0",
			"0000:03:00.0": "0",
		} {
			pciDevicePath := filepath.Join(fakePciBasePath, pciAddr)
			Expect(os.MkdirAll(pciDevicePath, 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(pciDevicePath, "numa_node"), []byte(numaNode+"\n"), 0o644)).To(Succeed())
		}
		for numaID, cpuList := range map[string]string{"0": "0-3", "1": "4-7"} {
			numaNodePath := filepath.Join(fakeNodeBasePath, "node"+numaID)
			Expect(os.MkdirAll(numaNodePath, 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(numaNodePath, "cpulist"), []byte(cpuList+"\n"), 0o644)).To(Succeed())
		}

		hardware.PciBasePath = fakePciBasePath
		hardware.NodeBasePath = fakeNodeBasePath

		DeferCleanup(func() {
			iommupci.ParseConfigHybridFn = nil
			iommupci.CalculatePCIHole64SizeFn = nil
			hardware.PciBasePath = originalPciBasePath
			hardware.NodeBasePath = originalNodeBasePath
			os.RemoveAll(fakePciBasePath)
			os.RemoveAll(fakeNodeBasePath)
		})
	})

	Context("per-device expander bus isolation", func() {
		It("should isolate multiple IOMMU devices on the same NUMA node", func() {
			domainSpec := createTestDomainSpecForIOMMU()
			iommu := &iommupci.IommuPCI{
				IommufdEnabled: pointer.P(true),
				SMMUEnabled:    true,
			}

			assigner := newExpanderBusAssigner(domainSpec, iommu)

			err := assigner.PlaceNumaAlignedDevices()
			Expect(err).ToNot(HaveOccurred())

			expanderBusCount := 0
			rootPortCount := 0
			for _, ctrl := range domainSpec.Devices.Controllers {
				if ctrl.Model == api.ControllerModelPCIeExpanderBus {
					expanderBusCount++
				}
				if ctrl.Model == api.ControllerModelPCIeRootPort {
					rootPortCount++
				}
			}

			// Two IOMMU devices on same NUMA node -> 2 expander buses + 2 root ports
			Expect(expanderBusCount).To(Equal(2))
			Expect(rootPortCount).To(Equal(2))

			// Each expander bus should have an SMMUv3 IOMMU device
			Expect(domainSpec.Devices.IOMMU).To(HaveLen(2))
			for _, dev := range domainSpec.Devices.IOMMU {
				Expect(dev.Model).To(Equal("smmuv3"))
				Expect(dev.Driver).NotTo(BeNil())
				Expect(dev.Driver.ATS).To(Equal("on"))
				Expect(dev.Driver.SSIDSize).To(Equal("20"))
				Expect(dev.Driver.OAS).To(Equal("48"))
			}

			// PCIHoleSize should be accumulated
			Expect(iommu.PCIHoleSize).To(Equal(uint64(2 * 64 * 1024 * 1024 * 1024)))
		})

		It("should share expander bus for single IOMMU device on NUMA node", func() {
			domainSpec := createTestDomainSpecForIOMMUSingle()
			iommu := &iommupci.IommuPCI{
				IommufdEnabled: pointer.P(true),
				SMMUEnabled:    true,
			}

			assigner := newExpanderBusAssigner(domainSpec, iommu)

			err := assigner.PlaceNumaAlignedDevices()
			Expect(err).ToNot(HaveOccurred())

			expanderBusCount := 0
			for _, ctrl := range domainSpec.Devices.Controllers {
				if ctrl.Model == api.ControllerModelPCIeExpanderBus {
					expanderBusCount++
				}
			}

			// Single IOMMU device -> shared expander bus (same as non-IOMMU)
			Expect(expanderBusCount).To(Equal(1))
			Expect(domainSpec.Devices.IOMMU).To(HaveLen(1))
		})

		It("should not change behavior when IommuPCI is nil", func() {
			domainSpec := createTestDomainSpecForIOMMU()

			assigner := newExpanderBusAssigner(domainSpec, nil)
			err := assigner.PlaceNumaAlignedDevices()
			Expect(err).ToNot(HaveOccurred())

			// No IOMMU devices should be created
			Expect(domainSpec.Devices.IOMMU).To(BeEmpty())
		})
	})
})

func createTestDomainSpecForIOMMU() *api.DomainSpec {
	return &api.DomainSpec{
		CPU: api.CPU{
			NUMA: &api.NUMA{
				Cells: []api.NUMACell{
					{ID: "0", CPUs: "0-3"},
					{ID: "1", CPUs: "4-7"},
				},
			},
		},
		CPUTune: &api.CPUTune{
			VCPUPin: []api.CPUTuneVCPUPin{
				{VCPU: 0, CPUSet: "0"},
				{VCPU: 1, CPUSet: "1"},
				{VCPU: 2, CPUSet: "4"},
				{VCPU: 3, CPUSet: "5"},
			},
		},
		Devices: api.Devices{
			HostDevices: []api.HostDevice{
				{
					Type:    api.HostDevicePCI,
					Managed: "no",
					Source: api.HostDeviceSource{
						Address: &api.Address{
							Domain:   "0x0000",
							Bus:      "0x01",
							Slot:     "0x00",
							Function: "0x0",
						},
					},
					ACPI: &api.ACPIHostDev{NodeSet: "tofill"},
				},
				{
					Type:    api.HostDevicePCI,
					Managed: "no",
					Source: api.HostDeviceSource{
						Address: &api.Address{
							Domain:   "0x0000",
							Bus:      "0x03",
							Slot:     "0x00",
							Function: "0x0",
						},
					},
					ACPI: &api.ACPIHostDev{NodeSet: "tofill"},
				},
			},
		},
	}
}

func createTestDomainSpecForIOMMUSingle() *api.DomainSpec {
	return &api.DomainSpec{
		CPU: api.CPU{
			NUMA: &api.NUMA{
				Cells: []api.NUMACell{
					{ID: "0", CPUs: "0-3"},
				},
			},
		},
		CPUTune: &api.CPUTune{
			VCPUPin: []api.CPUTuneVCPUPin{
				{VCPU: 0, CPUSet: "0"},
				{VCPU: 1, CPUSet: "1"},
			},
		},
		Devices: api.Devices{
			HostDevices: []api.HostDevice{
				{
					Type:    api.HostDevicePCI,
					Managed: "no",
					Source: api.HostDeviceSource{
						Address: &api.Address{
							Domain:   "0x0000",
							Bus:      "0x01",
							Slot:     "0x00",
							Function: "0x0",
						},
					},
					ACPI: &api.ACPIHostDev{NodeSet: "tofill"},
				},
			},
		},
	}
}
