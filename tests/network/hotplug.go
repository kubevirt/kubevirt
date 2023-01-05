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
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("nic-hotplug", func() {
	const (
		bridgeName  = "supadupabr"
		ifaceName   = "iface1"
		networkName = "skynet"
		vmIfaceName = "eth1"
	)

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error

		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("a running VMI with the default number of PCI slots", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("creating VM")
			vmi = setupVMI(virtClient, libvmi.NewAlpineWithTestTooling(libvmi.WithMasqueradeNetworking()...))
			Expect(
				createBridgeWithIPAMNetworkAttachmentDefinition(virtClient, util.NamespaceTestDefault, networkName, bridgeName),
			).To(Succeed())
			Expect(assertHotpluggedIfaceDoesNotExist(vmi, vmIfaceName)).To(Succeed())

			By("sent request to hotplug iface")
			Expect(
				virtClient.VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					vmi.GetName(),
					addIfaceOptions(networkName, ifaceName),
				),
			).To(Succeed())

			By("wait for interface to attach to the domain")
			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				return vmiCurrentInterfaces(virtClient, vmi.GetNamespace(), vmi.GetName())
			}, 30*time.Second).Should(
				WithTransform(
					CleanMACAddressesFromStatus,
					ConsistOf(interfaceStatusFromInterfaceNames(ifaceName))))
		})

		It("can be hotplugged a network interface", func() {
			Expect(assertHotpluggedIfaceExists(vmi, vmIfaceName)).To(Succeed())
		})

		Context("unplug", func() {
			BeforeEach(func() {
				pluggedIfaceVmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("sent request to hot-unplug iface")
				Expect(
					virtClient.VirtualMachineInstance(pluggedIfaceVmi.GetNamespace()).RemoveInterface(
						pluggedIfaceVmi.GetName(),
						removeIfaceOptions(networkName, ifaceName),
					),
				).To(Succeed())

				By("get updated vmi")
				var updatedVMI *v1.VirtualMachineInstance
				Eventually(func() error {
					var getVmiErr error
					updatedVMI, getVmiErr = virtClient.VirtualMachineInstance(pluggedIfaceVmi.Namespace).Get(pluggedIfaceVmi.Name, &metav1.GetOptions{})
					return getVmiErr
				}, 30*time.Second).ShouldNot(HaveOccurred())

				By("wait for iface to detach from the domain")
				Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
					vmi, err := virtClient.VirtualMachineInstance(updatedVMI.Namespace).Get(updatedVMI.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					var currentStatusSecondaryIfaces []v1.VirtualMachineInstanceNetworkInterface
					for _, iface := range vmi.Status.Interfaces {
						if iface.Name == defaultPodNetworkName {
							continue
						}
						currentStatusSecondaryIfaces = append(currentStatusSecondaryIfaces, iface)
					}
					fmt.Printf("vmi current status secondary ifaces: %+v\n", currentStatusSecondaryIfaces)
					return currentStatusSecondaryIfaces
				}, 30*time.Second).Should(BeEmpty())

				By("VMI state:")
				Eventually(func() error {
					var getVmiErr error
					updatedVMI, getVmiErr = virtClient.VirtualMachineInstance(pluggedIfaceVmi.Namespace).Get(pluggedIfaceVmi.Name, &metav1.GetOptions{})
					return getVmiErr
				}, 30*time.Second).ShouldNot(HaveOccurred())
				// raw, _ := json.MarshalIndent(updatedVMI, "", " ")
				// fmt.Println(string(raw))
			})

			FIt("should hot unplug a network interface", func() {
				err := assertHotpluggedIfaceExists(vmi, vmIfaceName)
				if err == nil {
					fmt.Println("test failed, iface apper in status but still attached to domain...")
					// time.Sleep(time.Hour * 1)
				}
				Expect(err).ToNot(Succeed())
				Expect(true).To(BeFalse(), "DEBUG: INJECTED ERROR")
			})
		})

		It("cannot hotplug multiple network interfaces for a q35 machine type by default", func() {
			By("hotplugging the second interface")
			const secondHotpluggedIfaceName = "iface2"
			Expect(
				virtClient.VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					vmi.GetName(),
					addIfaceOptions(networkName, secondHotpluggedIfaceName),
				),
			).To(Succeed())
			Eventually(func() []corev1.Event {
				events, err := virtClient.CoreV1().Events(vmi.GetNamespace()).List(context.TODO(), metav1.ListOptions{})
				ExpectWithOffset(1, err).NotTo(HaveOccurred())
				return events.Items
			}, 30*time.Second).Should(
				WithTransform(
					filterVMISyncErrorEvents,
					ContainElement(noPCISlotsAvailableError())))
		})

		It("can migrate a VMI with hotplugged interfaces", func() {
			checks.SkipIfMigrationIsNotPossible()

			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)
			tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)
		})
	})

	Context("a running VMI with a user specified number of PCI slots", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = setupVMI(virtClient, libvmi.NewAlpineWithTestTooling(withPCISlots(10)))
			Expect(
				createBridgeNetworkAttachmentDefinition(virtClient, util.NamespaceTestDefault, networkName, bridgeName),
			).To(Succeed())
			Expect(assertHotpluggedIfaceDoesNotExist(vmi, "eth1")).To(Succeed())

			Expect(
				virtClient.VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					vmi.GetName(),
					addIfaceOptions(networkName, ifaceName),
				),
			).To(Succeed())
			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				return vmiCurrentInterfaces(virtClient, vmi.GetNamespace(), vmi.GetName())
			}, 30*time.Second).Should(
				WithTransform(
					CleanMACAddressesFromStatus,
					ConsistOf(interfaceStatusFromInterfaceNames(ifaceName))))
		})

		It("can be hotplugged a network interface", func() {
			By("hotplugging the second interface")
			const secondHotpluggedIfaceName = "iface2"
			Expect(
				virtClient.VirtualMachineInstance(vmi.GetNamespace()).AddInterface(
					vmi.GetName(),
					addIfaceOptions(networkName, secondHotpluggedIfaceName),
				),
			).To(Succeed())
			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				return vmiCurrentInterfaces(virtClient, vmi.GetNamespace(), vmi.GetName())
			}, 30*time.Second).Should(
				WithTransform(
					CleanMACAddressesFromStatus,
					ConsistOf(interfaceStatusFromInterfaceNames(ifaceName, secondHotpluggedIfaceName))))
		})
	})

	Context("a running VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			Expect(
				createBridgeNetworkAttachmentDefinition(virtClient, util.NamespaceTestDefault, networkName, bridgeName),
			).To(Succeed())
			vm = tests.NewRandomVirtualMachine(libvmi.NewAlpineWithTestTooling(), true)

			var err error
			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).NotTo(HaveOccurred())
			var vmi *v1.VirtualMachineInstance
			Eventually(func() error {
				var err error
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vm.GetName(), &metav1.GetOptions{})
				return err
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
			tests.WaitUntilVMIReady(vmi, console.LoginToAlpine)
		})

		It("can be hotplugged a network interface", func() {
			const ifaceName = "iface1"

			Expect(
				virtClient.VirtualMachine(vm.GetNamespace()).AddInterface(
					vm.GetName(),
					addIfaceOptions(networkName, ifaceName),
				),
			).To(Succeed())

			vmi, err := virtClient.VirtualMachineInstance(vm.GetNamespace()).Get(vm.GetName(), &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() []v1.VirtualMachineInstanceNetworkInterface {
				return vmiCurrentInterfaces(virtClient, vmi.GetNamespace(), vmi.GetName())
			}, 30*time.Second).Should(
				WithTransform(
					CleanMACAddressesFromStatus,
					ConsistOf(interfaceStatusFromInterfaceNames(ifaceName))))
			Expect(assertHotpluggedIfaceExists(vmi, vmIfaceName)).To(Succeed())
		})
	})
})

func vmiCurrentInterfaces(virtClient kubecli.KubevirtClient, vmiNamespace, vmiName string) []v1.VirtualMachineInstanceNetworkInterface {
	vmi, err := virtClient.VirtualMachineInstance(vmiNamespace).Get(vmiName, &metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return secondaryInterfaces(vmi)
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

func removeIfaceOptions(networkName, ifaceName string) *v1.RemoveInterfaceOptions {
	return &v1.RemoveInterfaceOptions{
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

func createBridgeWithIPAMNetworkAttachmentDefinition(virtClient kubecli.KubevirtClient, namespace, networkName string, bridgeName string) error {
	ipam := "\\\"type\\\": \\\"host-local\\\", \\\"subnet\\\": \\\"10.10.30.0/24\\\""
	linuxBridgeWithIPAM := `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s"},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"mynet\", \"plugins\": [{\"type\": \"%s\", \"bridge\": \"%s\", \"ipam\": {%s}}]}"}}`
	nad := fmt.Sprintf(linuxBridgeWithIPAM, networkName, namespace, bridgeCNIType, bridgeName, ipam)
	raw, _ := json.MarshalIndent(nad, "", " ")
	fmt.Println(string(raw))
	return createNetworkAttachmentDefinition(
		virtClient,
		networkName,
		namespace,
		nad,
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

func CleanMACAddressesFromStatus(status []v1.VirtualMachineInstanceNetworkInterface) []v1.VirtualMachineInstanceNetworkInterface {
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
			Ready:         true,
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

func withPCISlots(numberOfPCISlots int) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.NumberPciPorts = uint8(numberOfPCISlots)
	}
}
