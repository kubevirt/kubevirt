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

package libvmops

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
)

func RunVMAndExpectLaunchWithRunStrategy(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
	By("Starting the VirtualMachine")
	vm, err := updateVMRunningStatus(virtClient, vm, &runStrategy)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	By("Waiting for VMI to be running")
	EventuallyWithOffset(1, ThisVMIWith(vm.Namespace, vm.Name), 300*time.Second, 1*time.Second).Should(BeRunning())

	By("Waiting for VM to be ready")
	EventuallyWithOffset(1, ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

	vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return vm
}

func updateVMRunningStatus(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, runStrategy *v1.VirtualMachineRunStrategy) (*v1.VirtualMachine, error) {
	var err error
	var updatedVM *v1.VirtualMachine

	EventuallyWithOffset(1, func() error {
		updatedVM, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = nil
		updatedVM.Spec.RunStrategy = runStrategy
		updatedVM, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), updatedVM, metav1.UpdateOptions{})
		return err
	}, 30*time.Second, 1*time.Second).Should(Succeed())

	return updatedVM, err
}

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Running}, watcher.WarningsPolicy{FailOnWarnings: true}, timeout)
}

func RunVMIAndExpectLaunchIgnoreWarnings(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Running}, watcher.WarningsPolicy{FailOnWarnings: false}, timeout)
}

func RunVMIAndExpectScheduling(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running}, watcher.WarningsPolicy{FailOnWarnings: true}, timeout)
}

func RunVMIAndExpectSchedulingWithWarningPolicy(vmi *v1.VirtualMachineInstance, timeout int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	return runVMI(vmi, []v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running}, wp, timeout)
}

func runVMI(vmi *v1.VirtualMachineInstance, phases []v1.VirtualMachineInstancePhase, wp watcher.WarningsPolicy, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	By("Waiting until the VirtualMachineInstance reaches the desired phase")
	return libwait.WaitForVMIPhase(vmi, phases, libwait.WithWarningsPolicy(&wp), libwait.WithTimeout(timeout))
}
