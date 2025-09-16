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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VMIDefaults", func() {

	Context("Disk defaults", func() {

		It("[test_id:4115]Should be applied to VMIs", func() {
			// create VMI with missing disk target
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithMemoryRequest("8192Ki"),
				libvmi.WithContainerDisk("testdisk", "dummy"),
			)

			// create the VMI first
			_, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			newVMI, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// check defaults
			disk := newVMI.Spec.Domain.Devices.Disks[0]
			Expect(disk.Disk).ToNot(BeNil(), "DiskTarget should not be nil")
			Expect(disk.Disk.Bus).ToNot(BeEmpty(), "DiskTarget's bus should not be empty")
		})

	})

	Context("MemBalloon defaults", func() {
		var (
			kvConfiguration v1.KubeVirtConfiguration
			vmi             *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			// create VMI with missing disk target
			vmi = libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithMemoryRequest("128Mi"),
			)

			kv := libkubevirt.GetCurrentKv(kubevirt.Client())
			kvConfiguration = kv.Spec.Configuration
		})

		It("[test_id:4556]Should be present in domain", func() {
			By("Creating a virtual machine")
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for successful start")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Getting domain of vmi")
			domain, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			expected := api.MemBalloon{
				Model: "virtio-non-transitional",
				Stats: &api.Stats{
					Period: 10,
				},
				Address: &api.Address{
					Type:     api.AddressPCI,
					Domain:   "0x0000",
					Bus:      "0x07",
					Slot:     "0x00",
					Function: "0x0",
				},
			}
			if kvConfiguration.VirtualMachineOptions != nil && kvConfiguration.VirtualMachineOptions.DisableFreePageReporting != nil {
				expected.FreePageReporting = "off"
			} else {
				expected.FreePageReporting = "on"
			}
			Expect(domain.Devices.Ballooning).ToNot(BeNil(), "There should be default memballoon device")
			Expect(*domain.Devices.Ballooning).To(Equal(expected), "Default to virtio model and 10 seconds pooling")
		})

		DescribeTable("Should override period in domain if present in virt-config ", Serial, func(period uint32, expected api.MemBalloon) {
			By("Adding period to virt-config")
			kvConfigurationCopy := kvConfiguration.DeepCopy()
			kvConfigurationCopy.MemBalloonStatsPeriod = &period
			config.UpdateKubeVirtConfigValueAndWait(*kvConfigurationCopy)
			if kvConfiguration.VirtualMachineOptions != nil && kvConfiguration.VirtualMachineOptions.DisableFreePageReporting != nil {
				expected.FreePageReporting = "off"
			} else {
				expected.FreePageReporting = "on"
			}

			By("Creating a virtual machine")
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for successful start")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Getting domain of vmi")
			domain, err := libdomain.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Devices.Ballooning).ToNot(BeNil(), "There should be memballoon device")
			Expect(*domain.Devices.Ballooning).To(Equal(expected))
		},
			Entry("[test_id:4557]with period 12", uint32(12), api.MemBalloon{
				Model: "virtio-non-transitional",
				Stats: &api.Stats{
					Period: 12,
				},
				Address: &api.Address{
					Type:     api.AddressPCI,
					Domain:   "0x0000",
					Bus:      "0x07",
					Slot:     "0x00",
					Function: "0x0",
				},
			}),
			Entry("[test_id:4558]with period 0", uint32(0), api.MemBalloon{
				Model: "virtio-non-transitional",
				Address: &api.Address{
					Type:     api.AddressPCI,
					Domain:   "0x0000",
					Bus:      "0x07",
					Slot:     "0x00",
					Function: "0x0",
				},
			}),
		)

		It("[test_id:4559]Should not be present in domain ", func() {
			By("Creating a virtual machine with autoAttachmemballoon set to false")
			vmi.Spec.Domain.Devices.AutoattachMemBalloon = pointer.P(false)
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for successful start")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Getting domain of vmi")
			domain, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			expected := api.MemBalloon{
				Model: "none",
			}
			Expect(domain.Devices.Ballooning).ToNot(BeNil(), "There should be memballoon device")
			Expect(*domain.Devices.Ballooning).To(Equal(expected))
		})

	})

	Context("Input defaults", func() {

		It("[test_id:TODO]Should be applied to a device added by AutoattachInputDevice", func() {
			By("Creating a VirtualMachine with AutoattachInputDevice enabled")
			vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm.Spec.Template.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)
			vm, err := kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Getting VirtualMachineInstance")
			Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name)).WithPolling(1 * time.Second).WithTimeout(60 * time.Second).Should(matcher.Exist())
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vmi.Spec.Domain.Devices.Inputs).ToNot(BeEmpty(), "There should be input devices")
			Expect(vmi.Spec.Domain.Devices.Inputs[0].Name).To(Equal("default-0"))
			Expect(vmi.Spec.Domain.Devices.Inputs[0].Type).To(Equal(v1.InputTypeTablet))
			Expect(vmi.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(v1.InputBusUSB))
		})

	})
}))
