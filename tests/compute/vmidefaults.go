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

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VMIDefaults", func() {

	Context("Disk defaults", decorators.WgS390x, func() {

		It("[test_id:4115]Should be applied to VMIs", func() {
			// create VMI with missing disk target
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithMemoryRequest("8192Ki"),
				libvmi.WithContainerDisk("testdisk", "dummy"),
			)

			_, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newVMI, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			disk := newVMI.Spec.Domain.Devices.Disks[0]
			Expect(disk.Disk).ToNot(BeNil(), "DiskTarget should not be nil")
			Expect(disk.Disk.Bus).ToNot(BeEmpty(), "DiskTarget's bus should not be empty")
		})

	})

	Context("Runtime defaults", decorators.WgS390x, func() {

		It("[test_id:4556]should apply input, balloon, and mergeable memory defaults", func() {
			By("Creating a VirtualMachine with default guestless template")
			vm := libvmi.NewVirtualMachine(
				libvmifact.NewGuestless(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithMemoryRequest(guestMemoryRequest),
				),
				libvmi.WithRunStrategy(v1.RunStrategyAlways),
			)
			vm.Spec.Template.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)

			vm, err := kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VirtualMachineInstance")
			Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name)).WithPolling(1 * time.Second).WithTimeout(60 * time.Second).Should(matcher.BeRunning())

			vmi, err := kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking input device defaults on the VirtualMachineInstance spec")
			Expect(vmi.Spec.Domain.Devices.Inputs).ToNot(BeEmpty(), "There should be input devices")
			Expect(vmi.Spec.Domain.Devices.Inputs[0].Name).To(Equal("default-0"))
			Expect(vmi.Spec.Domain.Devices.Inputs[0].Type).To(Equal(v1.InputTypeTablet))
			Expect(vmi.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(v1.InputBusUSB))

			By("Checking that the memory balloon reports stats")
			expectBalloonStats(vmi)

			By("Checking that guest RAM is mergeable by default")
			expectGuestRAMMergeable(vmi, true)
		})

	})

}))
