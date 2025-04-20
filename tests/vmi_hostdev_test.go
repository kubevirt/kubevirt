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
 */

package tests_test

import (
	"context"
	"fmt"
	"strings"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	failedDeleteVMI = "Failed to delete VMI"
)

var _ = Describe("[sig-compute]HostDevices", Serial, decorators.SigCompute, func() {
	var (
		virtClient kubecli.KubevirtClient
		config     v1.KubeVirtConfiguration
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kv := libkubevirt.GetCurrentKv(virtClient)
		config = kv.Spec.Configuration
	})

	AfterEach(func() {
		kv := libkubevirt.GetCurrentKv(virtClient)
		// Reinitialized the DeveloperConfiguration to avoid to influence the next test
		config = kv.Spec.Configuration
		config.DeveloperConfiguration = &v1.DeveloperConfiguration{}
		config.PermittedHostDevices = &v1.PermittedHostDevices{}
		kvconfig.UpdateKubeVirtConfigValueAndWait(config)
	})

	Context("with ephemeral disk", func() {
		DescribeTable("with emulated PCI devices", func(deviceIDs []string) {
			deviceName := "example.org/soundcard"

			By("Adding the emulated sound card to the permitted host devices")
			config.DeveloperConfiguration = &v1.DeveloperConfiguration{
				FeatureGates: []string{featuregate.HostDevicesGate},
				DiskVerification: &v1.DiskVerification{
					MemoryLimit: resource.NewScaledQuantity(2, resource.Giga),
				},
			}
			config.PermittedHostDevices = &v1.PermittedHostDevices{}
			var hostDevs []v1.HostDevice
			for i, id := range deviceIDs {
				config.PermittedHostDevices.PciHostDevices = append(config.PermittedHostDevices.PciHostDevices, v1.PciHostDevice{
					PCIVendorSelector: id,
					ResourceName:      deviceName,
				})
				hostDevs = append(hostDevs, v1.HostDevice{
					Name:       fmt.Sprintf("sound%d", i),
					DeviceName: deviceName,
				})
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			By("Creating a Fedora VMI with the sound card as a host device")
			randomVMI := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			randomVMI.Spec.Domain.Devices.HostDevices = hostDevs
			vmi, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), randomVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Making sure the sound card is present inside the VMI")
			for _, id := range deviceIDs {
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "grep -c " + strings.Replace(id, ":", "", 1) + " /proc/bus/pci/devices\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 15)).To(Succeed(), "Device not found")
			}
			// Make sure to delete the VMI before ending the test otherwise a device could still be taken
			err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
		},
			Entry("Should successfully passthrough an emulated PCI device", []string{"8086:2668"}),
			Entry("Should successfully passthrough 2 emulated PCI devices", []string{"8086:2668", "8086:2415"}),
		)
	})
})
