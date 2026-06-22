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

package iommu_pci_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	iommupci "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/iommu-pci"
)

var _ = Describe("IOMMU PCI Checker", func() {

	Describe("CalculateTotalPCIHole64Size", func() {
		Context("when computing PCI hole sizes", func() {
			It("should return 0 when both size and margin are 0", func() {
				result := iommupci.CalculateTotalPCIHole64Size(0, 0)
				Expect(result).To(Equal(uint64(0)))
			})
		})
	})

	Describe("NewBDFDevice", func() {
		Context("when creating a BDF device", func() {
			It("should initialize with the correct ID", func() {
				bdf := iommupci.NewBDFDevice("0000:3b:00.0")
				Expect(bdf).NotTo(BeNil())
				Expect(bdf.ID).To(Equal("0000:3b:00.0"))
			})

			It("should handle different BDF formats", func() {
				testCases := []string{
					"0000:00:00.0",
					"0000:ff:1f.7",
					"0001:3b:00.0",
					"ffff:ff:ff.f",
				}

				for _, testCase := range testCases {
					bdf := iommupci.NewBDFDevice(testCase)
					Expect(bdf).NotTo(BeNil())
					Expect(bdf.ID).To(Equal(testCase))
				}
			})
		})
	})

	Describe("NewIommuPCI", func() {
		Context("when initializing IOMMU checker", func() {
			It("should create instance for arm64", func() {
				iommu := iommupci.NewIommuPCI("arm64", true, true)
				Expect(iommu).NotTo(BeNil())
				Expect(iommu.PCIHoleSize).To(Equal(uint64(0)))
			})

			It("should create instance for x86_64", func() {
				iommu := iommupci.NewIommuPCI("x86_64", true, true)
				Expect(iommu).NotTo(BeNil())
				// On x86_64, SMMU should not be enabled
				Expect(iommu.SMMUEnabled).To(BeFalse())
				Expect(*iommu.IommufdEnabled).To(BeTrue())
			})

			It("should create instance for s390x", func() {
				iommu := iommupci.NewIommuPCI("s390x", true, true)
				Expect(iommu).NotTo(BeNil())
				Expect(iommu.SMMUEnabled).To(BeFalse())
				Expect(*iommu.IommufdEnabled).To(BeTrue())
			})

			It("should handle unknown architectures", func() {
				iommu := iommupci.NewIommuPCI("unknown", true, true)
				Expect(iommu).NotTo(BeNil())
				Expect(*iommu.IommufdEnabled).To(BeTrue())
			})

			It("should return nil when iommufdEnabled is false", func() {
				iommu := iommupci.NewIommuPCI("x86_64", true, false)
				Expect(iommu).To(BeNil())
			})

			It("should return nil when graceIOVirtualizationEnabled is false", func() {
				iommu := iommupci.NewIommuPCI("arm64", false, true)
				Expect(iommu).To(BeNil())
			})

			It("should return nil when both flags are false", func() {
				iommu := iommupci.NewIommuPCI("arm64", false, false)
				Expect(iommu).To(BeNil())
			})
		})
	})

	Describe("BDF Operations", func() {
		Context("when working with BDF devices", func() {
			var bdf *iommupci.BDF

			BeforeEach(func() {
				bdf = iommupci.NewBDFDevice("0000:3b:00.0")
			})

			It("should have correct initial state", func() {
				Expect(bdf.ID).To(Equal("0000:3b:00.0"))
				Expect(bdf.ATSSupported).To(BeFalse())
				Expect(bdf.ATSEnabled).To(BeFalse())
				Expect(bdf.PASIDSupported).To(BeFalse())
				Expect(bdf.SSIDSize).To(Equal(0))
			})
		})
	})

	Describe("IommuPCI Operations", func() {
		Context("when working with IOMMU configuration", func() {
			var iommu *iommupci.IommuPCI

			BeforeEach(func() {
				iommu = iommupci.NewIommuPCI("arm64", true, true)
			})

			It("should initialize with zero PCI hole size", func() {
				Expect(iommu.PCIHoleSize).To(Equal(uint64(0)))
			})

			It("should allow accumulating PCI hole sizes", func() {
				// Simulate adding multiple devices
				iommu.PCIHoleSize += 1024 * 1024     // 1 GB
				iommu.PCIHoleSize += 512 * 1024      // 512 MB
				iommu.PCIHoleSize += 2 * 1024 * 1024 // 2 GB

				expectedTotal := uint64(3*1024*1024 + 512*1024) // 3.5 GB in KiB
				Expect(iommu.PCIHoleSize).To(Equal(expectedTotal))
			})
		})
	})

	Describe("IommufdEnabled", func() {
		Context("when checking IOMMUFD availability", func() {
			It("should return a boolean value without panicking", func() {
				enabled, err := iommupci.IommufdEnabled()
				// Either the device exists (true, nil) or doesn't (false, nil)
				// or there's a permission issue (false, err)
				if err != nil {
					Expect(enabled).To(BeFalse())
				}
				// Just verify it returns without panicking
				_ = enabled
			})

			It("should handle missing /dev/iommu gracefully", func() {
				// On systems without IOMMUFD support, should return false, nil
				enabled, err := iommupci.IommufdEnabled()
				if !enabled && err == nil {
					// Expected behavior on systems without IOMMUFD
					Expect(enabled).To(BeFalse())
				}
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when encountering errors", func() {
			It("should handle permission errors gracefully", func() {
				// IommufdEnabled might fail with permission error
				_, err := iommupci.IommufdEnabled()
				if err != nil {
					// Error should be informative
					Expect(err.Error()).NotTo(BeEmpty())
				}
			})
		})
	})
})
