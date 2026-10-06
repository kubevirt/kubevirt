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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VMI Memory Overhead Reporting", decorators.RequiresTwoSchedulableNodes, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("memory overhead after CPU hotplug and migration", func(migrationShouldFail bool) {
		const maxSockets uint32 = 2

		By("Creating a VM with 1 socket, 1 core and maxSockets=2")
		vmiOpts := []libvmi.Option{
			libnet.WithMasqueradeNetworking(),
			libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
			libvmi.WithCPUCount(1, 1, 1),
			libvmi.WithMaxSockets(maxSockets),
		}
		if migrationShouldFail {
			vmiOpts = append(vmiOpts, libvmi.WithAnnotation(v1.FuncTestForceLauncherMigrationFailureAnnotation, ""))
		}
		vmi := libvmifact.NewGuestless(vmiOpts...)
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

		vm, err := virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVMIWith(vm.Namespace, vm.Name), 360*time.Second, 1*time.Second).Should(BeRunning())
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying memory overhead is reported in VMI status after startup")
		Expect(vmi.Status.Memory.MemoryOverhead).ToNot(BeNil(),
			"status.memory.memoryOverhead should be populated")
		overheadBeforeHotplug := vmi.Status.Memory.MemoryOverhead.DeepCopy()

		By("Hotplugging a CPU socket (1 -> 2 sockets)")
		patchData, err := patch.New(
			patch.WithTest("/spec/template/spec/domain/cpu/sockets", 1),
			patch.WithReplace("/spec/template/spec/domain/cpu/sockets", 2),
		).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())
		_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for hot change CPU condition to appear")
		Eventually(ThisVMI(vmi), 1*time.Minute, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceVCPUChange))

		By("Triggering a live migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		if migrationShouldFail {
			migrationUID := libmigration.RunMigrationAndExpectFailure(migration, flags.MigrationTimeout())
			vmi = libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)

			By("Verifying memory overhead did not change after failed migration")
			Expect(vmi.Status.Memory).ToNot(BeNil())
			Expect(vmi.Status.Memory.MemoryOverhead).ToNot(BeNil(),
				"status.memory.memoryOverhead should be set after migration")
			Expect(vmi.Status.Memory.MemoryOverhead.Value()).To(Equal(overheadBeforeHotplug.Value()),
				"memory overhead should not change after a failed migration")
		} else {
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying memory overhead increased after successful migration")
			Expect(vmi.Status.Memory).ToNot(BeNil())
			Expect(vmi.Status.Memory.MemoryOverhead).ToNot(BeNil(),
				"status.memory.memoryOverhead should be set after migration")
			Expect(vmi.Status.Memory.MemoryOverhead.Value()).To(BeNumerically(">", overheadBeforeHotplug.Value()),
				"memory overhead should increase after successful migration with CPU hotplug")
		}
	},
		Entry("should increase after successful migration", false),
		Entry("should remain unchanged after failed migration", true),
	)
}))
