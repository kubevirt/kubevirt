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

package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]Pod Disruption Budget (PDB)", decorators.SigCompute, func() {

	It("should ensure the Pod Disruption Budget (PDB) is deleted when the VM is stopped via guest OS", func() {
		By("Creating test VM")
		vm := libvmi.NewVirtualMachine(
			libvmifact.NewCirros(
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
			),
			libvmi.WithRunStrategy(v1.RunStrategyOnce),
		)
		vm.Namespace = testsuite.GetTestNamespace(vm)

		vm, err := kubevirt.Client().VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVMIWith(vm.Namespace, vm.Name), 10*time.Second, 1*time.Second).Should(Exist())

		By("Verifying PDB existence")
		Eventually(AllPDBs(vm.Namespace), 60*time.Second, 1*time.Second).Should(Not(BeEmpty()), "The Pod Disruption Budget should be created")

		vmi, err := kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

		By("Issuing a poweroff command from inside VM")
		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "sudo poweroff\n"},
			&expect.BExp{R: console.PromptExpression},
		}, 10)).To(Succeed())

		By("Ensuring the VirtualMachineInstance enters Succeeded phase")
		Eventually(ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(HaveSucceeded())

		By("Ensuring the PDB is deleted after the VM is powered off")
		Eventually(AllPDBs(vm.Namespace), 60*time.Second, 1*time.Second).Should(BeEmpty(), "The Pod Disruption Budget should be deleted")
	})
})
