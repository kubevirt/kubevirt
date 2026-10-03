//go:build linux

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

package thp

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
)

func smapsEntry(header string, thpEligible int, sizeKiB, anonHugeKiB, shmemPmdKiB int) string {
	return fmt.Sprintf("%s\nSize:           %d kB\nAnonHugePages:         %d kB\nShmemPmdMapped: %d kB\nTHPeligible:           %d\n",
		header, sizeKiB, anonHugeKiB, shmemPmdKiB, thpEligible)
}

var _ = Describe("Collapse orchestration", func() {
	const pid = 4242

	var (
		origSmaps  func(int) (string, error)
		origMadvise func(int, uintptr, uint64) (int, error)
		smapsCalls int
		preSmaps   string
		postSmaps  string
		madviseCalls int
	)

	BeforeEach(func() {
		origSmaps = readProcessSmapsFunc
		origMadvise = common.ProcessMadviseCollapseFunc
		smapsCalls = 0
		madviseCalls = 0
		preSmaps = smapsEntry("7f0000000000-7f0100000000 rw-p 00000000 00:00 0", 1, 4194304, 0, 0) // 4Gi, 0%
		postSmaps = smapsEntry("7f0000000000-7f0100000000 rw-p 00000000 00:00 0", 1, 4194304, 4194304, 0)
		readProcessSmapsFunc = func(gotPID int) (string, error) {
			Expect(gotPID).To(Equal(pid))
			smapsCalls++
			if smapsCalls == 1 {
				return preSmaps, nil
			}
			return postSmaps, nil
		}
		common.ProcessMadviseCollapseFunc = func(gotPID int, addr uintptr, length uint64) (int, error) {
			Expect(gotPID).To(Equal(pid))
			madviseCalls++
			return int(length), nil
		}
	})

	AfterEach(func() {
		readProcessSmapsFunc = origSmaps
		common.ProcessMadviseCollapseFunc = origMadvise
	})

	It("hard-fails when no THPeligible regions exist", func() {
		preSmaps = smapsEntry("7f3cbffff000-7f40bffff000 rw-s 00000000 00:01 1 /memfd:x (deleted)", 0, 16777216, 0, 0)
		_, err := Collapse(pid)
		Expect(err).To(BeAssignableToTypeOf(&CollapseFailedError{}))
		Expect(madviseCalls).To(BeZero())
		Expect(smapsCalls).To(Equal(1))
	})

	It("skips FullyBacked regions and does not call madvise", func() {
		preSmaps = smapsEntry("7f0000000000-7f0100000000 rw-p 00000000 00:00 0", 1, 4194304, 4194304, 0)
		result, err := Collapse(pid)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RegionsCollapsed).To(BeZero())
		Expect(result.Coverage).To(BeNumerically("~", 1.0, 0.0001))
		Expect(madviseCalls).To(BeZero())
		Expect(smapsCalls).To(Equal(1))
	})

	It("collapses incomplete regions and re-reads coverage", func() {
		result, err := Collapse(pid)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RegionsCollapsed).To(Equal(1))
		Expect(result.Coverage).To(BeNumerically("~", 1.0, 0.0001))
		Expect(madviseCalls).To(Equal(1))
		Expect(smapsCalls).To(Equal(2))
	})

	It("retries remaining regions across passes then succeeds", func() {
		common.ProcessMadviseCollapseFunc = func(gotPID int, addr uintptr, length uint64) (int, error) {
			madviseCalls++
			if madviseCalls < 3 {
				return 0, fmt.Errorf("transient")
			}
			return int(length), nil
		}
		result, err := Collapse(pid)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RegionsCollapsed).To(Equal(1))
		Expect(madviseCalls).To(Equal(3))
		Expect(smapsCalls).To(Equal(2))
	})

	It("does not requeue after exhausting passes; re-measures coverage", func() {
		postSmaps = preSmaps // still 0% after soft failures
		common.ProcessMadviseCollapseFunc = func(gotPID int, addr uintptr, length uint64) (int, error) {
			madviseCalls++
			return 0, fmt.Errorf("permanent")
		}
		result, err := Collapse(pid)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RegionsCollapsed).To(BeZero())
		Expect(result.Coverage).To(BeNumerically("~", 0.0, 0.0001))
		Expect(madviseCalls).To(Equal(collapseAttempts))
		Expect(smapsCalls).To(Equal(2))
	})

	It("treats partial process_madvise byte count as a failed pass", func() {
		common.ProcessMadviseCollapseFunc = func(gotPID int, addr uintptr, length uint64) (int, error) {
			madviseCalls++
			if madviseCalls < collapseAttempts {
				return int(length / 2), nil
			}
			return int(length), nil
		}
		result, err := Collapse(pid)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RegionsCollapsed).To(Equal(1))
		Expect(madviseCalls).To(Equal(collapseAttempts))
	})

	It("hard-fails when re-read finds no THPeligible regions", func() {
		postSmaps = smapsEntry("7f3cbffff000-7f40bffff000 rw-s 00000000 00:01 1 /memfd:x (deleted)", 0, 16777216, 0, 0)
		_, err := Collapse(pid)
		Expect(err).To(BeAssignableToTypeOf(&CollapseFailedError{}))
		Expect(smapsCalls).To(Equal(2))
	})
})
