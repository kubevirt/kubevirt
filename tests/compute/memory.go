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

package compute

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	guestMemoryRequest       = "128Mi"
	minGuestRAMRegionSizeKiB = 32 * 1024
)

var _ = Describe(SIG("VMI memory", decorators.WgS390x, func() {
	It("should opt out of mergeable guest RAM when annotated", func() {
		vmi := libvmifact.NewGuestless(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithMemoryRequest(guestMemoryRequest),
			libvmi.WithAnnotation(v1.MergeableMemory, "false"),
		)

		vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitForSuccessfulVMIStart(vmi)

		expectGuestRAMMergeable(vmi, false)
	})
}))

func expectBalloonStats(vmi *v1.VirtualMachineInstance) {
	output := libpod.RunCommandOnVmiPod(vmi, []string{"virsh", "dommemstat", "1"})
	Expect(output).To(MatchRegexp(`(?m)^actual\s+\d+`))
}

func expectGuestRAMMergeable(vmi *v1.VirtualMachineInstance, mergeable bool) {
	smaps, err := readQemuSmaps(vmi)
	Expect(err).NotTo(HaveOccurred())

	regions := largeSmapsRegions(smaps, minGuestRAMRegionSizeKiB)
	Expect(regions).NotTo(BeEmpty(), "expected guest RAM mappings in smaps")

	mergeableRegions := 0
	for _, region := range regions {
		if smapsRegionMergeable(region.vmFlags) {
			mergeableRegions++
		}
	}

	if mergeable {
		Expect(mergeableRegions).To(BeNumerically(">", 0), "expected mergeable guest RAM mappings")
		return
	}
	Expect(mergeableRegions).To(Equal(0), "expected no mergeable guest RAM mappings")
}

// readQemuSmaps reads QEMU smaps via virt-handler.
// kubectl exec into compute cannot read another UID's smaps.
func readQemuSmaps(vmi *v1.VirtualMachineInstance) (string, error) {
	Expect(vmi.Status.NodeName).NotTo(BeEmpty())

	guest := fmt.Sprintf("%s_%s", vmi.Namespace, vmi.Name)
	script := fmt.Sprintf(
		`pid=$(pgrep -f 'guest=%s[, ]' | head -1); test -n "$pid"; cat /proc/"$pid"/smaps`,
		guest,
	)
	return libnode.ExecuteCommandInVirtHandlerPod(vmi.Status.NodeName, []string{"/bin/bash", "-c", script})
}

type smapsRegion struct {
	sizeKiB int64
	vmFlags string
}

func largeSmapsRegions(content string, minSizeKiB int64) []smapsRegion {
	var regions []smapsRegion
	var current smapsRegion

	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.HasPrefix(line, "Size:"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				current.sizeKiB, _ = strconv.ParseInt(fields[1], 10, 64)
			}
		case strings.HasPrefix(line, "VmFlags:"):
			current.vmFlags = strings.TrimSpace(strings.TrimPrefix(line, "VmFlags:"))
			if current.sizeKiB >= minSizeKiB {
				regions = append(regions, current)
			}
			current = smapsRegion{}
		}
	}

	return regions
}

func smapsRegionMergeable(vmFlags string) bool {
	for _, flag := range strings.Fields(vmFlags) {
		if flag == "mg" { // MADV_MERGEABLE
			return true
		}
	}
	return false
}
