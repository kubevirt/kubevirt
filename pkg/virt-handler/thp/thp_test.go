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

package thp_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-handler/thp"
)

func smapsEntry(header string, thpEligible int, sizeKiB, anonHugeKiB, shmemPmdKiB int) string {
	return fmt.Sprintf("%s\nSize:           %d kB\nAnonHugePages:         %d kB\nShmemPmdMapped: %d kB\nTHPeligible:           %d\n",
		header, sizeKiB, anonHugeKiB, shmemPmdKiB, thpEligible)
}

var _ = Describe("THP smaps region selection", func() {
	It("selects THPeligible regions larger than 16Mi and skips vga/acpi", func() {
		smaps := "" +
			smapsEntry("7f9af8c00000-7f9af9c00000 rw-p 00000000 00:00 0", 1, 16384, 0, 0) +
			smapsEntry("7f9b0c200000-7f9b0c400000 rw-p 00000000 00:00 0", 1, 2048, 0, 0) +
			smapsEntry("7f9b27e00000-7f9f27e00000 rw-p 00000000 00:00 0", 1, 16777216, 0, 0)

		regions, err := thp.ParseTHPEligibleRegions(smaps)
		Expect(err).NotTo(HaveOccurred())
		Expect(regions).To(HaveLen(1))
		Expect(regions[0].Start).To(Equal(uintptr(0x7f9b27e00000)))
		Expect(regions[0].Size).To(Equal(uint64(16 * 1024 * 1024 * 1024)))
	})

	It("records AnonHugePages and ShmemPmdMapped coverage", func() {
		smaps := smapsEntry("7fe91be00000-7fed1be00000 rw-s 00000000 00:01 80254 /memfd:memory-backend-memfd (deleted)",
			1, 16777216, 0, 16777216)
		regions, err := thp.ParseTHPEligibleRegions(smaps)
		Expect(err).NotTo(HaveOccurred())
		Expect(regions).To(HaveLen(1))
		Expect(regions[0].ShmemPmdMapped).To(Equal(uint64(16 * 1024 * 1024 * 1024)))
		Expect(regions[0].Coverage()).To(BeNumerically("~", 1.0, 0.0001))
		Expect(regions[0].FullyBacked()).To(BeTrue())
	})

	It("treats coverage at the guaranteed threshold as fully backed", func() {
		smaps := smapsEntry("7fe91be00000-7fed1be00000 rw-s 00000000 00:01 80254 /memfd:memory-backend-memfd (deleted)",
			1, 16777216, 0, 15938356)
		regions, err := thp.ParseTHPEligibleRegions(smaps)
		Expect(err).NotTo(HaveOccurred())
		r := regions[0]
		Expect(r.Coverage()).To(BeNumerically(">=", thp.GuaranteedCoverageThreshold))
		Expect(r.FullyBacked()).To(BeTrue())
	})

	It("skips misaligned memfd (THPeligible=0)", func() {
		smaps := smapsEntry("7f3cbffff000-7f40bffff000 rw-s 00000000 00:01 73431 /memfd:memory-backend-memfd (deleted)",
			0, 16777216, 0, 0)
		regions, err := thp.ParseTHPEligibleRegions(smaps)
		Expect(err).NotTo(HaveOccurred())
		Expect(regions).To(BeEmpty())
	})

	It("returns every large THPeligible region for NUMA guests", func() {
		smaps := "" +
			smapsEntry("7f0000000000-7f0200000000 rw-p 00000000 00:00 0", 1, 8388608, 8388608, 0) +
			smapsEntry("7f0200000000-7f0400000000 rw-p 00000000 00:00 0", 1, 8388608, 4194304, 0)
		regions, err := thp.ParseTHPEligibleRegions(smaps)
		Expect(err).NotTo(HaveOccurred())
		Expect(regions).To(HaveLen(2))
		Expect(regions[0].Coverage()).To(BeNumerically("~", 1.0, 0.0001))
		Expect(regions[0].FullyBacked()).To(BeTrue())
		Expect(regions[1].Coverage()).To(BeNumerically("~", 0.5, 0.0001))
		Expect(regions[1].FullyBacked()).To(BeFalse())
	})
})

var _ = Describe("THP coverage helpers", func() {
	It("formats CollapseFailedError with the guaranteed threshold", func() {
		err := &thp.CollapseFailedError{Coverage: 0.5}
		Expect(err.Error()).To(ContainSubstring("50.0%"))
		Expect(err.Error()).To(ContainSubstring("95%"))
	})
})
