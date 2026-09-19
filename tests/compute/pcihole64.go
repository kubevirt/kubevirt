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
	"regexp"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe(SIG("64-Bit PCI hole", decorators.RequiresAMD64, func() {
	It("should not be present when annotation was set to true", func() {
		// SeaBIOS only turns its 64-bit PCI window on once the guest has memory
		// above 4Gi, which on q35 needs more than the 2Gi low memory split.
		vmi := libvmops.RunVMIAndExpectLaunch(
			libvmifact.NewAlpine(
				libvmi.WithAnnotation(v1.DisablePCIHole64, "true"),
				libvmi.WithMemoryRequest("3Gi"),
			), flags.StartupTimeoutSecondsSmall(),
		)
		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		By("checking that the fwcfg entry reached the guest firmware")
		Expect(readSeaBIOSPCI64FWCfg(vmi)).To(Equal("no"))

		iomem := runGuestCommands(vmi, "cat /proc/iomem")

		By("checking that the guest got memory above 4Gi")
		Expect(sizeAbove4Gi(iomem, "System RAM")).ToNot(BeZero())

		By("checking that the guest sees no PCI range above 4Gi at all")
		Expect(sizeAbove4Gi(iomem, "PCI")).To(BeZero())
	})
}))

func readSeaBIOSPCI64FWCfg(vmi *v1.VirtualMachineInstance) string {
	output := runGuestCommands(vmi,
		"modprobe qemu_fw_cfg",
		"d=/sys/firmware/qemu_fw_cfg/by_name/opt/org.seabios",
		`echo "PCI64=$(cat $d/pci64/raw 2>&1)"`,
	)

	matches := regexp.MustCompile(`(?m)^PCI64=(.*?)\s*$`).FindAllStringSubmatch(output, -1)
	ExpectWithOffset(1, matches).ToNot(BeEmpty(), "guest did not report the fw_cfg entry, got: %s", output)

	return matches[len(matches)-1][1]
}

func runGuestCommands(vmi *v1.VirtualMachineInstance, commands ...string) string {
	var batch []expect.Batcher
	for _, command := range commands {
		batch = append(batch,
			&expect.BSnd{S: command + "\n"},
			&expect.BExp{R: ""},
		)
	}

	res, err := console.SafeExpectBatchWithResponse(vmi, batch, 15)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return res[len(res)-1].Output
}

// sizeAbove4Gi sums the /proc/iomem ranges of the given resource that live in the
// 64-bit address space. Lines look like "100000000-13fffffff : System RAM".
func sizeAbove4Gi(iomem, resource string) uint64 {
	ranges := regexp.MustCompile(`(?im)^([0-9a-f]+)-([0-9a-f]+) : `+resource).FindAllStringSubmatch(iomem, -1)

	size := uint64(0)
	for _, r := range ranges {
		start, err := strconv.ParseUint(r[1], 16, 64)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		end, err := strconv.ParseUint(r[2], 16, 64)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		// Ensure that we have got a valid reading from iomem, with insufficient
		// privileges reading from iomem will return only zero ranges.
		ExpectWithOffset(1, end).To(BeNumerically(">", start))
		if start > 0xFFFFFFFF {
			size += end - start + 1
		}
	}

	return size
}
