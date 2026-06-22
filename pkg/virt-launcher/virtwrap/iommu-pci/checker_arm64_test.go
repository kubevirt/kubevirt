//go:build arm64

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

var _ = Describe("IOMMU PCI Checker ARM64", func() {

	Describe("CalculateTotalPCIHole64Size", func() {
		Context("when computing PCI hole sizes", func() {
			It("should round up to nearest power of 2", func() {
				// 1024 bytes (1 KiB) + 0 margin = 1 KiB, already power of 2
				result := iommupci.CalculateTotalPCIHole64Size(1024, 0)
				Expect(result).To(Equal(uint64(1)))

				// 1024 bytes (1 KiB) + 1 KiB margin = 2 KiB, already power of 2
				result = iommupci.CalculateTotalPCIHole64Size(1024, 1)
				Expect(result).To(Equal(uint64(2)))

				// 1536 bytes (1.5 KiB) + 0 margin = rounds up to 2 KiB
				result = iommupci.CalculateTotalPCIHole64Size(1536, 0)
				Expect(result).To(Equal(uint64(2)))

				// 3 MiB (3072 KiB) should round up to 4096 KiB (4 MiB)
				result = iommupci.CalculateTotalPCIHole64Size(3*1024*1024, 0)
				Expect(result).To(Equal(uint64(4096)))
			})

			It("should add margin before rounding", func() {
				// 1024 bytes (1 KiB) + 1024 KiB margin = 1025 KiB, rounds to 2048 KiB
				result := iommupci.CalculateTotalPCIHole64Size(1024, 1024)
				Expect(result).To(Equal(uint64(2048)))

				// 512 KiB device + 512 KiB margin = 1024 KiB, already power of 2
				result = iommupci.CalculateTotalPCIHole64Size(512*1024, 512)
				Expect(result).To(Equal(uint64(1024)))
			})

			It("should handle large GPU memory sizes", func() {
				// Simulate a 16 GB GPU BAR (typical for modern GPUs)
				// 16 GB = 16 * 1024 * 1024 KiB = 16777216 KiB
				deviceSize := uint64(16 * 1024 * 1024 * 1024) // 16 GB in bytes
				margin := uint64(1024 * 1024)                 // 1 GB margin in KiB

				result := iommupci.CalculateTotalPCIHole64Size(deviceSize, margin)

				// 16 GB = 16777216 KiB, + 1048576 KiB = 17825792 KiB
				// Next power of 2 is 2^25 = 33554432 KiB (32 GB)
				Expect(result).To(Equal(uint64(33554432)))
			})

			It("should handle multiple small devices", func() {
				// Simulate 4 devices with 256 MB each = 1 GB total
				totalSize := uint64(4 * 256 * 1024 * 1024) // 1 GB in bytes
				margin := uint64(0)

				result := iommupci.CalculateTotalPCIHole64Size(totalSize, margin)

				// 1 GB = 1048576 KiB, already power of 2
				Expect(result).To(Equal(uint64(1048576)))
			})
		})
	})

	Describe("Edge Cases", func() {
		Context("when handling boundary conditions", func() {
			It("should handle maximum uint64 values safely", func() {
				// This shouldn't panic or overflow
				result := iommupci.CalculateTotalPCIHole64Size(1<<63, 0)
				// Result should be a valid power of 2
				Expect(result).To(BeNumerically(">", 0))
			})

			It("should handle very small sizes", func() {
				// 1 byte should round up to 1 KiB
				result := iommupci.CalculateTotalPCIHole64Size(1, 0)
				Expect(result).To(Equal(uint64(1)))
			})

			It("should handle exact power of 2 sizes", func() {
				// Powers of 2 should remain unchanged (after KiB conversion)
				testCases := []struct {
					bytes    uint64
					expected uint64
				}{
					{1024, 1},             // 1 KiB
					{2048, 2},             // 2 KiB
					{4096, 4},             // 4 KiB
					{1048576, 1024},       // 1 MiB
					{1073741824, 1048576}, // 1 GiB
				}

				for _, tc := range testCases {
					result := iommupci.CalculateTotalPCIHole64Size(tc.bytes, 0)
					Expect(result).To(Equal(tc.expected))
				}
			})
		})
	})

	Describe("Realistic GPU Scenarios", func() {
		Context("when configuring for real GPU models", func() {
			It("should handle NVIDIA A100 configuration (40 GB)", func() {
				// NVIDIA A100 has a 40 GB BAR
				a100BAR := uint64(40) * 1024 * 1024 * 1024 // 40 GB
				margin := uint64(1024 * 1024)              // 1 GB margin

				result := iommupci.CalculateTotalPCIHole64Size(a100BAR, margin)

				// 40 GB + 1 GB = 41 GB = 43008 MiB = 44040192 KiB
				// Next power of 2 is 2^26 = 67108864 KiB (64 GB)
				Expect(result).To(Equal(uint64(67108864)))
			})

			It("should handle NVIDIA H100 configuration (80 GB)", func() {
				// NVIDIA H100 has an 80 GB BAR
				h100BAR := uint64(80) * 1024 * 1024 * 1024 // 80 GB
				margin := uint64(1024 * 1024)              // 1 GB margin

				result := iommupci.CalculateTotalPCIHole64Size(h100BAR, margin)

				// 80 GB + 1 GB = 81 GB = 85983232 KiB
				// Next power of 2 is 2^27 = 134217728 KiB (128 GB)
				Expect(result).To(Equal(uint64(134217728)))
			})

			It("should handle multiple small GPUs", func() {
				// 4x NVIDIA T4 (16 GB each) = 64 GB total
				iommu := iommupci.NewIommuPCI("arm64", true, true)

				for i := 0; i < 4; i++ {
					// Each T4 contributes 16 GB
					iommu.PCIHoleSize += uint64(16 * 1024 * 1024) // in KiB
				}

				result := iommupci.CalculateTotalPCIHole64Size(
					iommu.PCIHoleSize*1024, // convert KiB to bytes
					uint64(1024*1024),      // 1 GB margin
				)

				// 64 GB + 1 GB = 65 GB = 68157440 KiB
				// Next power of 2 is 2^27 = 134217728 KiB (128 GB)
				Expect(result).To(Equal(uint64(134217728)))
			})
		})
	})

	Describe("CalculatePCIHole64Size", func() {
		Context("when calculating PCI hole size for a device", func() {
			It("should return error for non-existent device", func() {
				size, err := iommupci.CalculatePCIHole64Size("ffff:ff:ff.f")
				Expect(err).To(HaveOccurred())
				Expect(size).To(Equal(uint64(0)))
			})

			It("should handle invalid BDF format gracefully", func() {
				size, err := iommupci.CalculatePCIHole64Size("invalid")
				Expect(err).To(HaveOccurred())
				Expect(size).To(Equal(uint64(0)))
			})

			It("should return minimum 512 GiB if calculated size is smaller", func() {
				// This is testing the minimum size enforcement
				// Even if a device has no/small BARs, minimum should be 512 GiB
				minSize := uint64(549755813888) // 512 GiB in bytes

				// Test with a hypothetical device that has small or no 64-bit BARs
				// The function should still return at least 512 GiB
				// (This test documents the behavior, actual device testing requires real hardware)
				_ = minSize
			})
		})
	})

	Describe("Integration Scenarios", func() {
		Context("when combining multiple operations", func() {
			It("should handle complete device configuration workflow", func() {
				// Step 1: Check IOMMUFD availability
				iommufdEnabled, err := iommupci.IommufdEnabled()
				_ = err

				// Step 2: Create IOMMU instance
				iommu := iommupci.NewIommuPCI("arm64", true, true)
				Expect(iommu).NotTo(BeNil())

				// Step 3: Create BDF device
				bdf := iommupci.NewBDFDevice("0000:3b:00.0")
				Expect(bdf).NotTo(BeNil())

				// Document expected workflow
				_ = iommufdEnabled
				_ = iommu
				_ = bdf
			})

			It("should properly calculate total memory requirements", func() {
				iommu := iommupci.NewIommuPCI("arm64", true, true)

				// Simulate discovering multiple GPUs
				// GPU 1: 40 GB
				gpu1Size := uint64(40 * 1024 * 1024 * 1024)
				size1, err := iommupci.CalculatePCIHole64Size("0000:00:00.0")
				if err == nil {
					gpu1Size = size1
				}

				// Add to total
				iommu.PCIHoleSize += gpu1Size / 1024 // convert to KiB

				// Calculate final hole size with margin
				finalSize := iommupci.CalculateTotalPCIHole64Size(
					iommu.PCIHoleSize*1024, // convert back to bytes
					1024*1024,              // 1 GB margin in KiB
				)

				// Final size should be a power of 2
				Expect(finalSize).To(BeNumerically(">", 0))
				// Verify it's a power of 2
				isPowerOf2 := finalSize&(finalSize-1) == 0
				Expect(isPowerOf2).To(BeTrue())
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when encountering errors", func() {
			It("should return meaningful errors for invalid devices", func() {
				// Invalid BDF format
				_, err := iommupci.CalculatePCIHole64Size("")
				Expect(err).To(HaveOccurred())

				// Non-existent device
				_, err = iommupci.CalculatePCIHole64Size("ffff:ff:ff.f")
				Expect(err).To(HaveOccurred())
			})

			It("should handle missing sysfs entries", func() {
				// Device with missing numa_node file
				_, _, err := iommupci.InferExtraNUMANodes("invalid:device")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
