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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"

	v1 "kubevirt.io/api/core/v1"
)

func StopVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	return StopVirtualMachineWithTimeout(vm, 300*time.Second)
}

func StopVirtualMachineWithTimeout(vm *v1.VirtualMachine, timeout time.Duration) *v1.VirtualMachine {
	virtClient := kubevirt.Client()

	if err := virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{}); err != nil {
		if errVMNotRunning(err) {
			return vm
		}
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	EventuallyWithOffset(1, ThisVMIWith(vm.Namespace, vm.Name)).WithTimeout(timeout).WithPolling(time.Second).Should(BeGone(), "The vmi did not disappear")

	By("Waiting for VM to stop")
	EventuallyWithOffset(1, ThisVM(vm), 300*time.Second, 1*time.Second).Should(Not(BeReady()))

	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return updatedVM
}

func StartVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	virtClient := kubevirt.Client()

	ExpectWithOffset(1, virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})).To(Succeed())
	EventuallyWithOffset(1, ThisVMIWith(vm.Namespace, vm.Name)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(Exist())

	By("Waiting for VM to be ready")
	EventuallyWithOffset(1, ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return updatedVM
}

func errVMNotRunning(err error) bool {
	return strings.Contains(err.Error(), "VM is not running")
}
