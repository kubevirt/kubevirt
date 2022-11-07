/*
 * This file is part of the kubevirt project
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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("Network interface hotplug", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error

		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("a running VMI", func() {
		const (
			bridgeName  = "supadupabr"
			networkName = "skynet"
		)

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = setupVMI(virtClient, libvmi.NewAlpineWithTestTooling())
			Expect(
				createBridgeNetworkAttachmentDefinition(virtClient, util.NamespaceTestDefault, networkName, bridgeName),
			).To(Succeed())
			Expect(assertHotpluggedIfaceDoesNotExist(vmi, "eth1")).To(Succeed())
		})

		It("can be hotplugged a network interface", func() {
			const ifaceName = "iface1"

			Expect(
				virtClient.VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					vmi.GetName(),
					addIfaceOptions(networkName, ifaceName),
				),
			).To(Succeed())

			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				var err error

				vmi, err = virtClient.VirtualMachineInstance(vmi.GetNamespace()).Get(vmi.GetName(), &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return filterHotpluggedNetworkInterfaces(vmi)
			}, 30*time.Second).Should(
				WithTransform(
					CleanMACAddressesFromStatus,
					ConsistOf(v1.VirtualMachineInstanceNetworkInterface{
						Name:          hotpluggedNetworkInterfaceName(networkName, ifaceName),
						InterfaceName: "eth1",
						InfoSource:    "domain, guest-agent",
						QueueCount:    1,
						HotplugInterface: &v1.HotplugInterfaceStatus{
							Phase: v1.InterfaceHotplugPhaseReady,
							Type:  v1.Plug,
						},
					})))
			Expect(assertHotpluggedIfaceExists(vmi, "eth1")).To(Succeed())
		})

		It("cannot hotplug multiple network interfaces for a q35 machine type by default", func() {
			const ifaceName = "iface1"

			Expect(
				virtClient.VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					vmi.GetName(),
					addIfaceOptions(networkName, ifaceName),
				),
			).To(Succeed())

			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				var err error

				vmi, err = virtClient.VirtualMachineInstance(vmi.GetNamespace()).Get(vmi.GetName(), &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return filterHotpluggedNetworkInterfaces(vmi)
			}, 30*time.Second).Should(
				WithTransform(
					CleanMACAddressesFromStatus,
					ConsistOf(
						v1.VirtualMachineInstanceNetworkInterface{
							Name:          hotpluggedNetworkInterfaceName(networkName, ifaceName),
							InterfaceName: "eth1",
							InfoSource:    "domain, guest-agent",
							QueueCount:    1,
							HotplugInterface: &v1.HotplugInterfaceStatus{
								Phase: v1.InterfaceHotplugPhaseReady,
								Type:  v1.Plug,
							},
						})))

			By("hotplugging the second interface")
			const secondHotpluggedIfaceName = "iface2"
			Expect(
				virtClient.VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					vmi.GetName(),
					addIfaceOptions(networkName, secondHotpluggedIfaceName),
				),
			).To(Succeed())
			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				var err error

				vmi, err = virtClient.VirtualMachineInstance(vmi.GetNamespace()).Get(vmi.GetName(), &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return filterHotpluggedNetworkInterfaces(vmi)
			}, 30*time.Second).Should(
				WithTransform(
					CleanMACAddressesFromStatus,
					ConsistOf(
						v1.VirtualMachineInstanceNetworkInterface{
							Name:          hotpluggedNetworkInterfaceName(networkName, ifaceName),
							InterfaceName: "eth1",
							InfoSource:    "domain, guest-agent",
							QueueCount:    1,
							HotplugInterface: &v1.HotplugInterfaceStatus{
								Phase: v1.InterfaceHotplugPhaseReady,
								Type:  v1.Plug,
							},
						},
						v1.VirtualMachineInstanceNetworkInterface{
							Name:          hotpluggedNetworkInterfaceName(networkName, secondHotpluggedIfaceName),
							InterfaceName: secondHotpluggedIfaceName,
							HotplugInterface: &v1.HotplugInterfaceStatus{
								Phase:           v1.InterfaceHotplugPhaseFailed,
								Type:            v1.Plug,
								DetailedMessage: noAvailablePCISlotsError(),
							},
						})))
		})
	})
})

func noAvailablePCISlotsError() string {
	return "failed to update domain: server error. command SyncVMI failed: \"LibvirtError(Code=1, Domain=20, Message='internal error: No more available PCI slots')\""
}

func assertHotpluggedIfaceExists(vmi *v1.VirtualMachineInstance, ifaceName string) error {
	return runSafeCommand(vmi, fmt.Sprintf("ip addr show %s\n", ifaceName))
}

func assertHotpluggedIfaceDoesNotExist(vmi *v1.VirtualMachineInstance, ifaceName string) error {
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: fmt.Sprintf("ip addr show %s | wc -l\n", ifaceName)},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
}

func addIfaceOptions(networkName, ifaceName string) *v1.AddInterfaceOptions {
	return &v1.AddInterfaceOptions{
		NetworkName:   networkName,
		InterfaceName: ifaceName,
	}
}

func createBridgeNetworkAttachmentDefinition(virtClient kubecli.KubevirtClient, namespace, networkName string, bridgeName string) error {
	return createNetworkAttachmentDefinition(
		virtClient,
		networkName,
		namespace,
		fmt.Sprintf(linuxBridgeNAD, networkName, namespace, bridgeCNIType, bridgeName),
	)
}

func filterHotpluggedNetworkInterfaces(vmi *v1.VirtualMachineInstance) []v1.VirtualMachineInstanceNetworkInterface {
	var hotpluggedIfaces []v1.VirtualMachineInstanceNetworkInterface
	for i, iface := range vmi.Status.Interfaces {
		if iface.HotplugInterface != nil {
			hotpluggedIfaces = append(hotpluggedIfaces, vmi.Status.Interfaces[i])
		}
	}
	return hotpluggedIfaces
}

func CleanMACAddressesFromStatus(status []v1.VirtualMachineInstanceNetworkInterface) []v1.VirtualMachineInstanceNetworkInterface {
	for i := range status {
		status[i].MAC = ""
	}
	return status
}

func hotpluggedNetworkInterfaceName(networkName, ifaceName string) string {
	return fmt.Sprintf("%s_%s", networkName, ifaceName)
}
