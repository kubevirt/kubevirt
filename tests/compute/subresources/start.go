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

package subresources

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(compute.SIG("Start subresource", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("[test_id:1529]should start a stopped VM only once", func(runStrategy, expectedRunStrategy v1.VirtualMachineRunStrategy) {
		By("Creating a VM")
		vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(runStrategy))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Starting the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for VM to be ready")
		Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())

		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Spec.RunStrategy).ToNot(BeNil())
		Expect(*vm.Spec.RunStrategy).To(Equal(expectedRunStrategy))

		By("Ensuring stateChangeRequests list is cleared")
		Expect(vm.Status.StateChangeRequests).To(BeEmpty())

		By("Ensuring a second invocation should fail")
		err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
		Expect(err).To(MatchError(ContainSubstring("VM is already running")))
	},
		Entry("[test_id:2036]with RunStrategyManual", v1.RunStrategyManual, v1.RunStrategyManual),
		Entry("[test_id:2037]with RunStrategyHalted", v1.RunStrategyHalted, v1.RunStrategyAlways),
	)

	It("[test_id:6311]should start in paused state using RunStrategyManual", func() {
		By("Creating a VM with RunStrategyManual")
		vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(v1.RunStrategyManual))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Starting the VM in paused state")
		err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{Paused: true})
		Expect(err).ToNot(HaveOccurred())

		By("Getting the status of the VM")
		Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeCreated())

		By("Getting running VirtualMachineInstance with paused condition")
		Eventually(func() *v1.VirtualMachineInstance {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(*vmi.Spec.StartStrategy).To(Equal(v1.StartStrategyPaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
			return vmi
		}, 240*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))
	})
}))
