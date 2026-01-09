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

var _ = Describe(compute.SIG("Stop subresource", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("should stop a running VM", func(runStrategy, expectedRunStrategy v1.VirtualMachineRunStrategy) {
		By("Creating a VM")
		vm := libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(runStrategy))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		if runStrategy == v1.RunStrategyManual {
			By("Starting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		By("Waiting for VM to be ready")
		Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())

		By("Stopping the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring the VirtualMachineInstance is removed")
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(matcher.Exist())
		By("Ensuring stateChangeRequests list gets cleared")
		Eventually(matcher.ThisVM(vm), 30*time.Second, 1*time.Second).Should(Not(matcher.HaveStateChangeRequests()))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Spec.RunStrategy).ToNot(BeNil())
		Expect(*vm.Spec.RunStrategy).To(Equal(expectedRunStrategy))
	},
		Entry("[test_id:3163]with RunStrategyAlways", v1.RunStrategyAlways, v1.RunStrategyHalted),
		Entry("[test_id:2186]with RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure, v1.RunStrategyRerunOnFailure),
		Entry("[test_id:2189]with RunStrategyManual", v1.RunStrategyManual, v1.RunStrategyManual),
	)

	It("should be able to stop a VM during crashloop backoff when when 'runStrategy: Always' is set", func() {
		By("Creating VirtualMachine")
		vm := libvmi.NewVirtualMachine(libvmifact.NewAlpine(
			libvmi.WithAnnotation(v1.FuncTestLauncherFailFastAnnotation, ""),
		), libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("waiting for crash loop state")
		Eventually(matcher.ThisVM(vm), 60*time.Second, 5*time.Second).Should(matcher.BeInCrashLoop())

		By("Stopping the VM while in a crash loop")
		err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting on crash loop status to be removed.")
		Eventually(matcher.ThisVM(vm), 120*time.Second, 5*time.Second).Should(matcher.NotBeInCrashLoop())
	})
}))
