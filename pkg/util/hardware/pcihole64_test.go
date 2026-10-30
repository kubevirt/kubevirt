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
	"math"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PCI 64-bit hole sizing", func() {
	Describe("PCI64BitPrefetchableBARBytes", func() {
		It("sums only 64-bit prefetchable memory BARs", func() {
			devicesPath := GinkgoT().TempDir()
			devicePath := filepath.Join(devicesPath, "0000:81:00.0")
			Expect(os.MkdirAll(devicePath, 0755)).To(Succeed())
			resource := strings.Join([]string{
				"0x0000000000001000 0x0000000000001fff 0x00000101", // 32-bit memory BAR, excluded
				"0x0000000000002000 0x0000000000002fff 0x00000000", // I/O BAR, excluded
				"0x0000000100000000 0x000000013fffffff 0x00000204", // 64-bit non-prefetchable, excluded
				"0x0000000200000000 0x000000020fffffff 0x0000020c", // 64-bit prefetchable, included (256 MiB)
				"0x0000000300000000 0x00000003ffffffff 0x0000020c", // 64-bit prefetchable, included (4 GiB)
			}, "\n") + "\n"
			Expect(os.WriteFile(filepath.Join(devicePath, "resource"), []byte(resource), 0644)).To(Succeed())

			size, err := PCI64BitPrefetchableBARBytes(devicesPath, "0000:81:00.0")

			Expect(err).ToNot(HaveOccurred())
			Expect(size).To(Equal(uint64(0x10000000 + 0x100000000)))
		})

		It("returns an error when the device's resource file is missing", func() {
			devicesPath := GinkgoT().TempDir()

			_, err := PCI64BitPrefetchableBARBytes(devicesPath, "0000:81:00.0")

			Expect(err).To(HaveOccurred())
		})

		It("returns a wrapped error when a BAR address fails to parse", func() {
			devicesPath := GinkgoT().TempDir()
			devicePath := filepath.Join(devicesPath, "0000:81:00.0")
			Expect(os.MkdirAll(devicePath, 0755)).To(Succeed())
			resource := "not-hex 0x0000000000001fff 0x00000101\n"
			Expect(os.WriteFile(filepath.Join(devicePath, "resource"), []byte(resource), 0644)).To(Succeed())

			_, err := PCI64BitPrefetchableBARBytes(devicesPath, "0000:81:00.0")

			Expect(err).To(MatchError(ContainSubstring("failed to parse start address")))
		})

		It("ignores an SR-IOV PF's aggregate VF BAR reservation lines", func() {
			devicesPath := GinkgoT().TempDir()
			devicePath := filepath.Join(devicesPath, "0000:81:00.0")
			Expect(os.MkdirAll(devicePath, 0755)).To(Succeed())
			resource := strings.Join([]string{
				"0x0000000200000000 0x000000020fffffff 0x0000020c", // BAR 0: 64-bit prefetchable, included (256 MiB)
				"0x0000000000000000 0x0000000000000000 0x00000000", // BAR 1
				"0x0000000000000000 0x0000000000000000 0x00000000", // BAR 2
				"0x0000000000000000 0x0000000000000000 0x00000000", // BAR 3
				"0x0000000000000000 0x0000000000000000 0x00000000", // BAR 4
				"0x0000000000000000 0x0000000000000000 0x00000000", // BAR 5
				"0x0000000000000000 0x0000000000000000 0x00000000", // expansion ROM
				// PCI_IOV_RESOURCES: the PF's aggregate VF BAR reservation,
				// far larger than the PF's own BAR and must not be counted.
				"0x0000004000000000 0x00000047ffffffff 0x0000020c", // 32 GiB, excluded
			}, "\n") + "\n"
			Expect(os.WriteFile(filepath.Join(devicePath, "resource"), []byte(resource), 0644)).To(Succeed())

			size, err := PCI64BitPrefetchableBARBytes(devicesPath, "0000:81:00.0")

			Expect(err).ToNot(HaveOccurred())
			Expect(size).To(Equal(uint64(0x10000000)))
		})
	})

	Describe("PCIHole64KiB", func() {
		It("adds the margin and rounds up to the next power of two", func() {
			sizeBytes := uint64(256) << 30 // 256 GiB of BARs
			marginKiB := uint64(1) << 20   // 1 GiB margin

			size, err := PCIHole64KiB(sizeBytes, marginKiB)

			Expect(err).ToNot(HaveOccurred())
			Expect(size).To(Equal(uint64(512) << 20)) // 512 GiB in KiB
		})

		It("returns 0 for a zero size", func() {
			size, err := PCIHole64KiB(0, 1<<20)

			Expect(err).ToNot(HaveOccurred())
			Expect(size).To(Equal(uint64(0)))
		})

		It("returns an error when the margin addition overflows", func() {
			_, err := PCIHole64KiB(math.MaxUint64, math.MaxUint64)

			Expect(err).To(MatchError(ContainSubstring("plus margin")))
		})

		It("returns an error when rounding to a power of two overflows", func() {
			// sizeKiB is tiny (1 KiB) but the margin pushes the sum past the
			// largest representable power of two, overflowing the rounding
			// loop even though the margin addition itself did not overflow.
			_, err := PCIHole64KiB(1024, math.MaxUint64-2)

			Expect(err).To(MatchError(ContainSubstring("rounded to a power of two overflows")))
		})
	})

	Describe("HostCPUAddressInfo", func() {
		var previousPath string

		BeforeEach(func() {
			previousPath = CPUInfoPath
		})

		AfterEach(func() {
			CPUInfoPath = previousPath
		})

		It("parses the physical address width and vendor ID", func() {
			cpuinfo := "processor\t: 0\n" +
				"vendor_id\t: AuthenticAMD\n" +
				"address sizes\t: 48 bits physical, 48 bits virtual\n"
			path := filepath.Join(GinkgoT().TempDir(), "cpuinfo")
			Expect(os.WriteFile(path, []byte(cpuinfo), 0644)).To(Succeed())
			CPUInfoPath = path

			physBits, vendorID, err := HostCPUAddressInfo()

			Expect(err).ToNot(HaveOccurred())
			Expect(physBits).To(Equal(uint64(48)))
			Expect(vendorID).To(Equal("AuthenticAMD"))
		})

		It("returns an error when address sizes is missing", func() {
			path := filepath.Join(GinkgoT().TempDir(), "cpuinfo")
			Expect(os.WriteFile(path, []byte("vendor_id\t: GenuineIntel\n"), 0644)).To(Succeed())
			CPUInfoPath = path

			_, _, err := HostCPUAddressInfo()

			Expect(err).To(HaveOccurred())
		})

		It("returns an error when the file is missing", func() {
			CPUInfoPath = filepath.Join(GinkgoT().TempDir(), "does-not-exist")

			_, _, err := HostCPUAddressInfo()

			Expect(err).To(HaveOccurred())
		})
	})
})
