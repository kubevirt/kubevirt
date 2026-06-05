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

var _ = Describe(compute.SIG("Restart subresource", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("should restart a running VM", func(runStrategy v1.VirtualMachineRunStrategy) {
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

		By("Getting VMI's UUID")
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Restarting the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring the VirtualMachineInstance is restarted")
		Eventually(matcher.ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(matcher.BeRestarted(vmi.UID))

		vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Spec.RunStrategy).ToNot(BeNil())
		Expect(*vm.Spec.RunStrategy).To(Equal(runStrategy))

		By("Ensuring stateChangeRequests list gets cleared")
		// StateChangeRequest might still exist until the new VMI is created
		// But it must eventually be cleared
		Eventually(matcher.ThisVM(vm), 240*time.Second, 1*time.Second).Should(Not(matcher.HaveStateChangeRequests()))
	},
		Entry("[test_id:3164]with RunStrategyAlways", v1.RunStrategyAlways),
		Entry("[test_id:2187]with RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure),
		Entry("[test_id:2035]with RunStrategyManual", v1.RunStrategyManual),
	)
}))
