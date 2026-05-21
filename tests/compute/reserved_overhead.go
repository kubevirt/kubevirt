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
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libhypervisor"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Reserved Overhead", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	predictOverheadMemory := func(vmi *v1.VirtualMachineInstance) resource.Quantity {
		hypervisorName := libhypervisor.GetHypervisorDeviceName(virtClient)
		launcherResources := hypervisor.NewLauncherHypervisorResources(hypervisorName)
		return launcherResources.GetMemoryOverhead(vmi, runtime.GOARCH, nil)
	}

	calculateExpectedPodMemory := func(vmi *v1.VirtualMachineInstance) resource.Quantity {
		expectedMemory := predictOverheadMemory(vmi)
		vmiMemory := vmi.Spec.Domain.Resources.Requests.Memory()
		expectedMemory.Add(*vmiMemory)
		return expectedMemory
	}

	calculateExpectedMemlockLimit := func(vmi *v1.VirtualMachineInstance) int64 {
		vmiMem := vmi.Spec.Domain.Memory
		if vmiMem == nil ||
			vmiMem.ReservedOverhead == nil ||
			vmiMem.ReservedOverhead.MemLock == nil ||
			*vmiMem.ReservedOverhead.MemLock != v1.MemLockRequired {
			// Defaults 8192k memlock limit
			return pointer.P(resource.MustParse("8Mi")).Value()
		}

		vmiMemory := vmi.Spec.Domain.Resources.Requests.Memory()
		expectedLimit := predictOverheadMemory(vmi)
		expectedLimit.Add(*resource.NewScaledQuantity(vmiMemory.ScaledValue(resource.Kilo), resource.Kilo))
		return expectedLimit.Value()
	}

	findVirtqemudPid := func(pod *k8sv1.Pod) (int, error) {
		output, err := exec.ExecuteCommandOnPod(pod, "compute", []string{"pgrep", "virtqemud"})
		if err != nil {
			return -1, err
		}

		pidStr := strings.TrimSpace(output)
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return -1, fmt.Errorf("failed to parse pid %q: %v", pidStr, err)
		}

		return pid, nil
	}

	getProcessMemlockLimit := func(pod *k8sv1.Pod) (int64, error) {
		pid, err := findVirtqemudPid(pod)
		if err != nil {
			return 0, err
		}
		limitsPath := fmt.Sprintf("/proc/%d/limits", pid)
		output, err := exec.ExecuteCommandOnPod(pod, "compute", []string{"cat", limitsPath})
		if err != nil {
			return 0, err
		}

		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "Max locked memory") {
				fields := strings.Fields(line)
				if len(fields) < 4 {
					return 0, fmt.Errorf("unexpected format for Max locked memory line: %q", line)
				}
				limitStr := fields[3]
				if limitStr == "unlimited" {
					return -1, nil
				}
				limit, err := strconv.ParseInt(limitStr, 10, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse memlock limit %q: %v", limitStr, err)
				}
				return limit, nil
			}
		}

		return 0, fmt.Errorf("Max locked memory not found in limits output")
	}

	verifyMemoryRequestsAndMemlockLimits := func(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) {
		By("Verifying compute container memory requests include addedOverhead")
		computeContainer := libpod.LookupComputeContainer(pod)
		actualMemory := computeContainer.Resources.Requests[k8sv1.ResourceMemory]

		expectedMemory := calculateExpectedPodMemory(vmi)
		Expect(actualMemory.Value()).To(Equal(expectedMemory.Value()),
			"Expected pod memory to be %s, got %s",
			expectedMemory.String(), actualMemory.String())

		By("Verifying virtqemud process memlock limits")
		expectedLimit := calculateExpectedMemlockLimit(vmi)
		Eventually(func(g Gomega) int64 {
			limit, err := getProcessMemlockLimit(pod)
			g.Expect(err).ToNot(HaveOccurred())
			return limit
		}, 30*time.Second, 1*time.Second).Should(Equal(expectedLimit))
	}

	DescribeTable("should configure a new virt-launcher pod properly",
		func(addedOverhead string, memlock v1.MemLockRequirement) {
			vmiOpts := []libvmi.Option{
				libvmi.WithMemoryRequest("128Mi"),
			}
			if addedOverhead != "" {
				vmiOpts = append(vmiOpts, libvmi.WithAddedOverhead(addedOverhead))
			}
			if memlock != "" {
				vmiOpts = append(vmiOpts, libvmi.WithMemLock(memlock))
			}

			vmi := libvmifact.NewAlpine(
				vmiOpts...,
			)

			By("Starting the VMI")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

			By("Getting the virt-launcher pod")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			verifyMemoryRequestsAndMemlockLimits(vmi, pod)
		},
		Entry("with addedOverhead configured", "256Mi", v1.MemLockRequirement("")),
		Entry("with memLock configured", "", v1.MemLockRequirement(v1.MemLockRequired)),
		Entry("with both addedOverhead and memLock configured", "64Mi", v1.MemLockRequirement(v1.MemLockRequired)),
		Entry("without addedOverhead or memLock configured", "", v1.MemLockRequirement("")),
		Entry("with memLock explicitly not required", "", v1.MemLockRequirement(v1.MemLockNotRequired)),
	)

	It("should configure the target virt-launcher pod during migration", func() {
		vmiOpts := []libvmi.Option{
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithAddedOverhead("128Mi"),
			libvmi.WithMemLock(v1.MemLockRequired),
			libnet.WithMasqueradeNetworking(),
		}

		vmi := libvmifact.NewAlpine(
			vmiOpts...,
		)

		By("Starting the VMI")
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

		migration := libmigration.New(vmi.Name, testsuite.GetTestNamespace(vmi))
		migration = libmigration.RunMigration(virtClient, migration)

		By("Getting migration target pod")
		var targetPod *k8sv1.Pod
		Eventually(func() error {
			var err error
			targetPod, err = libpod.GetTargetPodForMigration(migration)
			return err
		}, 60, 1).Should(Succeed())

		verifyMemoryRequestsAndMemlockLimits(vmi, targetPod)

		By("Let the migration finish successfully")
		libmigration.ExpectMigrationToSucceed(virtClient, migration, libmigration.MigrationWaitTime)
	})

}))
