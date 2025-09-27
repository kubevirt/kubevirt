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
 * Copyright The KubeVirt Authors
 *
 */

package compute

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe(SIG("64-Bit PCI hole", func() {
	const pciHole64MaxSize = 2 * 1024 * 1024 * 1024 // 2Gi

	It("should not exceed maximum size when annotation was set to true", func() {
		vmi := libvmops.RunVMIAndExpectLaunch(
			libvmifact.NewAlpine(libvmi.WithAnnotation(v1.DisablePCIHole64, "true")), libvmops.StartupTimeoutSecondsSmall,
		)
		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
			&expect.BSnd{S: "cat /proc/iomem\n"},
			&expect.BExp{R: ""},
		}, 15)
		Expect(err).ToNot(HaveOccurred())

		// Even if disabled, the 64-bit PCI hole can still be up to 2Gi in size under certain circumstances.
		Expect(calculatePCIHole64Size(res[0].Output)).To(BeNumerically("<=", pciHole64MaxSize))
	})
}))

func calculatePCIHole64Size(iomem string) uint64 {
	// Match lines like:
	// 7f800000-efffffff : PCI Bus 0000:00
	// f0000000-f7ffffff : PCI ECAM 0000 [bus 00-7f]
	// 4000000000-7fffffffff : PCI Bus 0000:00
	re := regexp.MustCompile(`(?i)^([0-9a-fA-F]+)-([0-9a-fA-F]+) : PCI`)

	size := uint64(0)
	scanner := bufio.NewScanner(strings.NewReader(iomem))
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) != 3 {
			continue
		}

		start, err := strconv.ParseUint(matches[1], 16, 64)
		Expect(err).ToNot(HaveOccurred())
		end, err := strconv.ParseUint(matches[2], 16, 64)
		Expect(err).ToNot(HaveOccurred())

		// Ensure that we have got a valid reading from iomem,
		// with insufficient privileges reading from iomem will return only zero ranges.
		Expect(end).To(BeNumerically(">", start))
		// If the address range is in the 64-Bit address space add it to size
		if start > 0xFFFFFFFF {
			size += end - start + 1
		}
	}

	return size
}
