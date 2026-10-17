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

package thp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/hypervisor/common"
)

var _ = Describe("ProcessMadviseCollapse stub", func() {
	It("dispatches through ProcessMadviseCollapseFunc", func() {
		orig := common.ProcessMadviseCollapseFunc
		DeferCleanup(func() { common.ProcessMadviseCollapseFunc = orig })

		var gotPID int
		var gotAddr uintptr
		var gotLen uint64
		common.ProcessMadviseCollapseFunc = func(pid int, addr uintptr, length uint64) (int, error) {
			gotPID, gotAddr, gotLen = pid, addr, length
			return int(length), nil
		}

		n, err := common.ProcessMadviseCollapse(42, 0x1000, 0x2000)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(0x2000))
		Expect(gotPID).To(Equal(42))
		Expect(gotAddr).To(Equal(uintptr(0x1000)))
		Expect(gotLen).To(Equal(uint64(0x2000)))
	})
})
