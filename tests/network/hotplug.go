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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("nic-hotplug", func() {
	Context("a running VMI", func() {
		const (
			bridgeName  = "supadupabr"
			ifaceName   = "iface1"
			networkName = "skynet"
			vmIfaceName = "eth1"
		)

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = tests.RunVMIAndExpectLaunch(libvmi.NewAlpineWithTestTooling(libvmi.WithMasqueradeNetworking()...), 60)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			Expect(
				createBridgeNetworkAttachmentDefinition(util.NamespaceTestDefault, networkName, bridgeName),
			).To(Succeed())
			err := libnet.InterfaceExists(vmi, vmIfaceName)
			Expect(err).To(HaveOccurred())

			Expect(
				kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					context.Background(),
					vmi.GetName(),
					addIfaceOptions(networkName, ifaceName),
				),
			).To(Succeed())

			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				return cleanMACAddressesFromStatus(vmiCurrentInterfaces(vmi.GetNamespace(), vmi.GetName()))
			}, 30*time.Second).Should(
				ConsistOf(interfaceStatusFromInterfaceNames(ifaceName)))
		})

		It("can be hotplugged a network interface", func() {
			Expect(libnet.InterfaceExists(vmi, vmIfaceName)).To(Succeed())
		})

		It("cannot hotplug multiple network interfaces for a q35 machine type by default", func() {
			By("hotplugging the second interface")
			const secondHotpluggedIfaceName = "iface2"
			Expect(
				kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					context.Background(),
					vmi.GetName(),
					addIfaceOptions(networkName, secondHotpluggedIfaceName),
				),
			).To(Succeed())
			Eventually(func() []corev1.Event {
				events, err := kubevirt.Client().CoreV1().Events(vmi.GetNamespace()).List(context.Background(), metav1.ListOptions{})
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				return events.Items
			}, 30*time.Second).Should(
				WithTransform(
					filterVMISyncErrorEvents,
					ContainElement(noPCISlotsAvailableError())))
		})
	})
})

func vmiCurrentInterfaces(vmiNamespace, vmiName string) []v1.VirtualMachineInstanceNetworkInterface {
	vmi, err := kubevirt.Client().VirtualMachineInstance(vmiNamespace).Get(context.Background(), vmiName, &metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return secondaryInterfaces(vmi)
}

func addIfaceOptions(networkName, ifaceName string) *v1.AddInterfaceOptions {
	return &v1.AddInterfaceOptions{
		NetworkName:   networkName,
		InterfaceName: ifaceName,
	}
}

func createBridgeNetworkAttachmentDefinition(namespace, networkName string, bridgeName string) error {
	return createNetworkAttachmentDefinition(
		kubevirt.Client(),
		networkName,
		namespace,
		fmt.Sprintf(linuxBridgeNAD, networkName, namespace, bridgeCNIType, bridgeName),
	)
}

func secondaryInterfaces(vmi *v1.VirtualMachineInstance) []v1.VirtualMachineInstanceNetworkInterface {
	indexedSecondaryNetworks := indexVMsSecondaryNetworks(vmi)

	var nonDefaultInterfaces []v1.VirtualMachineInstanceNetworkInterface
	for _, iface := range vmi.Status.Interfaces {
		if _, isNonDefaultPodNetwork := indexedSecondaryNetworks[iface.Name]; isNonDefaultPodNetwork {
			nonDefaultInterfaces = append(nonDefaultInterfaces, iface)
		}
	}
	return nonDefaultInterfaces
}

func indexVMsSecondaryNetworks(vmi *v1.VirtualMachineInstance) map[string]v1.Network {
	indexedSecondaryNetworks := map[string]v1.Network{}
	for _, network := range vmi.Spec.Networks {
		if network.Multus != nil && !network.Multus.Default {
			indexedSecondaryNetworks[network.Name] = network
		}
	}
	return indexedSecondaryNetworks
}

func cleanMACAddressesFromStatus(status []v1.VirtualMachineInstanceNetworkInterface) []v1.VirtualMachineInstanceNetworkInterface {
	for i := range status {
		status[i].MAC = ""
	}
	return status
}

func interfaceStatusFromInterfaceNames(ifaceNames ...string) []v1.VirtualMachineInstanceNetworkInterface {
	const initialIfacesInVMI = 1
	var ifaceStatus []v1.VirtualMachineInstanceNetworkInterface
	for i, ifaceName := range ifaceNames {
		ifaceStatus = append(ifaceStatus, v1.VirtualMachineInstanceNetworkInterface{
			Name:          ifaceName,
			InterfaceName: fmt.Sprintf("eth%d", i+initialIfacesInVMI),
			InfoSource:    "domain, guest-agent",
			QueueCount:    1,
			PodConfigDone: true,
		})
	}
	return ifaceStatus
}

func filterVMISyncErrorEvents(events []corev1.Event) []string {
	const desiredEvent = "SyncFailed"
	return filterEvents(events, func(event corev1.Event) bool {
		return event.Reason == desiredEvent
	})
}

func filterEvents(events []corev1.Event, p func(event corev1.Event) bool) []string {
	var eventMsgs []string
	for _, event := range events {
		if p(event) {
			eventMsgs = append(eventMsgs, event.Message)
		}
	}
	return eventMsgs
}

func noPCISlotsAvailableError() string {
	return "server error. command SyncVMI failed: \"LibvirtError(Code=1, Domain=20, Message='internal error: No more available PCI slots')\""
}
