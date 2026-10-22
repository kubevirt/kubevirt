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
	})

	Describe("PCIHole64KiB", func() {
		It("adds the margin and rounds up to the next power of two", func() {
			sizeBytes := uint64(256) << 30 // 256 GiB of BARs
			marginKiB := uint64(1) << 20   // 1 GiB margin

			Expect(PCIHole64KiB(sizeBytes, marginKiB)).To(Equal(uint64(512) << 20)) // 512 GiB in KiB
		})

		It("returns 0 for a zero size", func() {
			Expect(PCIHole64KiB(0, 1<<20)).To(Equal(uint64(0)))
		})

		It("returns 0 on overflow", func() {
			Expect(PCIHole64KiB(math.MaxUint64, math.MaxUint64)).To(Equal(uint64(0)))
		})
	})
})
