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

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	linuxBridgeName = "supadupabr"
	ifaceName       = "iface1"
	networkName     = "skynet"
	vmIfaceName     = "eth1"
)

type hotplugMethod string

const (
	migrationBased hotplugMethod = "migrationBased"
	inPlace        hotplugMethod = "inPlace"
)

var _ = SIGDescribe("nic-hotplug", func() {

	BeforeEach(func() {
		Expect(checks.HasFeature(virtconfig.HotplugNetworkIfacesGate)).To(BeTrue())
	})

	Context("a running VM", func() {
		var hotPluggedVM *v1.VirtualMachine
		var hotPluggedVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("Creating a VM")
			hotPluggedVM = newVMWithOneInterface()
			var err error
			hotPluggedVM, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), hotPluggedVM)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error {
				var err error
				hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), hotPluggedVM.GetName(), &metav1.GetOptions{})
				return err
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
			libwait.WaitUntilVMIReady(hotPluggedVMI, console.LoginToAlpine)

			By("Creating a NAD")
			Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), networkName, linuxBridgeName)).To(Succeed())

			By("Hotplugging an interface to the VM")
			Expect(kubevirt.Client().VirtualMachine(hotPluggedVM.GetNamespace()).AddInterface(
				context.Background(),
				hotPluggedVM.GetName(),
				addIfaceOptions(networkName, ifaceName),
			),
			).To(Succeed())
		})

		DescribeTable("can be hotplugged a network interface", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			Expect(libnet.InterfaceExists(hotPluggedVMI, vmIfaceName)).To(Succeed())

			updatedVM, err := kubevirt.Client().VirtualMachine(hotPluggedVM.Namespace).Get(context.Background(), hotPluggedVM.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmIfaceSpec := vmispec.LookupInterfaceByName(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces, ifaceName)
			Expect(vmIfaceSpec).NotTo(BeNil(), "VM spec should contain the new interface")
			Expect(vmIfaceSpec.MacAddress).NotTo(BeEmpty(), "VM iface spec should have MAC address")

			Eventually(func(g Gomega) {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(hotPluggedVMI.Namespace).Get(context.Background(), hotPluggedVMI.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				vmiIfaceStatus := vmispec.LookupInterfaceStatusByName(updatedVMI.Status.Interfaces, ifaceName)
				g.Expect(vmiIfaceStatus).NotTo(BeNil(), "VMI status should report the hotplugged interface")
				g.Expect(vmiIfaceStatus.MAC).NotTo(BeEmpty(), "VMI hotplugged iface status should report MAC address")

				g.Expect(vmiIfaceStatus.MAC).To(Equal(vmIfaceSpec.MacAddress),
					"hot-plugged iface in VMI status should have a MAC address as specified in VM template spec")
			}, time.Second*30, time.Second*3).Should(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)

		DescribeTable("hotplugged interfaces are available after the VM is restarted", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			By("restarting the VM")
			Expect(kubevirt.Client().VirtualMachine(hotPluggedVM.GetNamespace()).Restart(
				context.Background(),
				hotPluggedVM.GetName(),
				&v1.RestartOptions{},
			)).To(Succeed())

			By("asserting a new VMI is created, and running")
			Eventually(func() v1.VirtualMachineInstancePhase {
				newVMI, err := kubevirt.Client().VirtualMachineInstance(hotPluggedVM.GetNamespace()).Get(context.Background(), hotPluggedVM.Name, &metav1.GetOptions{})
				if err != nil || hotPluggedVMI.UID == newVMI.UID {
					hotPluggedVMI.GetNamespace()
					return v1.VmPhaseUnset
				}
				return newVMI.Status.Phase
			}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))
			var err error
			hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(hotPluggedVM.GetNamespace()).Get(context.Background(), hotPluggedVM.GetName(), &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitUntilVMIReady(hotPluggedVMI, console.LoginToAlpine)

			hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(hotPluggedVM.GetNamespace()).Get(context.Background(), hotPluggedVM.GetName(), &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(libnet.InterfaceExists(hotPluggedVMI, vmIfaceName)).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)

		DescribeTable("can migrate a VMI with hotplugged interfaces", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyDynamicInterfaceChange(hotPluggedVMI, plugMethod)

			migrate(hotPluggedVMI)
			Expect(libnet.InterfaceExists(hotPluggedVMI, vmIfaceName)).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)

		DescribeTable("has connectivity over the secondary network", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyDynamicInterfaceChange(hotPluggedVMI, plugMethod)

			const subnetMask = "/24"
			const ip1 = "10.1.1.1"
			const ip2 = "10.1.1.2"

			By("Configuring static IP address on the hotplugged interface inside the guest")
			Expect(configInterface(hotPluggedVMI, vmIfaceName, ip1+subnetMask)).To(Succeed())

			By("creating another VM connected to the same secondary network")
			net := v1.Network{
				Name: ifaceName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: networkName,
					},
				},
			}

			iface := v1.Interface{
				Name: ifaceName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
			}

			anotherVmi := libvmi.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(iface),
				libvmi.WithNetwork(&net),
				libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByDevice("eth1", ip2+subnetMask)))
			anotherVmi = tests.CreateVmiOnNode(anotherVmi, hotPluggedVMI.Status.NodeName)
			libwait.WaitUntilVMIReady(anotherVmi, console.LoginToFedora)

			By("Ping from the VM with hotplugged interface to the other VM")
			Expect(libnet.PingFromVMConsole(hotPluggedVMI, ip2)).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)

		DescribeTable("is able to hotplug multiple network interfaces", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			By("hotplugging the second interface")
			const secondHotpluggedIfaceName = "iface2"
			Expect(
				kubevirt.Client().VirtualMachine(hotPluggedVM.GetNamespace()).AddInterface(
					context.Background(),
					hotPluggedVMI.GetName(),
					addIfaceOptions(networkName, secondHotpluggedIfaceName),
				),
			).To(Succeed())

			By("wait for the second network to appear in the VMI spec")
			EventuallyWithOffset(1, func() []v1.Network {
				var err error
				hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(hotPluggedVMI.GetNamespace()).Get(context.Background(), hotPluggedVMI.GetName(), &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return hotPluggedVMI.Spec.Networks
			}, 30*time.Second).Should(
				ConsistOf(
					*v1.DefaultPodNetwork(),
					v1.Network{
						Name: secondHotpluggedIfaceName,
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
							NetworkName: networkName,
						}},
					},
					v1.Network{
						Name: ifaceName,
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
							NetworkName: networkName,
						}},
					},
				))
			hotPluggedVMI = verifyDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			Expect(libnet.InterfaceExists(hotPluggedVMI, "eth2")).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)
	})

	Context("a stopped VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			var err error

			vm = newStoppedVMWithOneInterface()
			vm, err = kubevirt.Client().VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), vm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("cannot be subject to **hot** plug, but will mutate the template.Spec on behalf of the user", func() {
			Expect(
				kubevirt.Client().VirtualMachine(vm.GetNamespace()).AddInterface(
					context.Background(),
					vm.GetName(),
					addIfaceOptions(networkName, ifaceName),
				),
			).To(Succeed())

			Eventually(func() []v1.Network {
				var err error
				vm, err = kubevirt.Client().VirtualMachine(vm.GetNamespace()).Get(context.Background(), vm.GetName(), &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vm.Spec.Template.Spec.Networks
			}, 30*time.Second).Should(
				ConsistOf(
					*v1.DefaultPodNetwork(),
					v1.Network{
						Name: ifaceName,
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
							NetworkName: networkName,
						}},
					},
				))
			// we need to clean up the MACs because KubeMacPool sets MACs for VM interfaces ...
			Expect(cleanMACAddressesFromSpec(vm.Spec.Template.Spec.Domain.Devices.Interfaces)).To(
				ConsistOf(
					*v1.DefaultMasqueradeNetworkInterface(),
					v1.Interface{
						Name: ifaceName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{},
						},
					},
				))
		})
	})
})

var _ = SIGDescribe("nic-hotunplug", func() {
	const (
		linuxBridgeNetworkName1 = "red"
		linuxBridgeNetworkName2 = "blue"
	)

	BeforeEach(func() {
		Expect(checks.HasFeature(virtconfig.HotplugNetworkIfacesGate)).To(BeTrue())
	})

	Context("a running VM", func() {
		var vm *v1.VirtualMachine
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("creating a NAD")
			Expect(createBridgeNetworkAttachmentDefinition(
				testsuite.GetTestNamespace(nil), networkName, linuxBridgeName)).To(Succeed())

			By("running a VM")
			opts := append(
				libvmi.WithMasqueradeNetworking(),
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName1, networkName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName2, networkName)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName1)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName2)),
			)
			vmi = libvmi.NewAlpineWithTestTooling(opts...)
			vm = tests.NewRandomVirtualMachine(vmi, true)

			var err error
			vm, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error {
				vmi, err = kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				return err
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)
		})

		DescribeTable("hot-unplug network interface succeed", func(plugMethod hotplugMethod) {
			Expect(kubevirt.Client().VirtualMachine(vm.Namespace).RemoveInterface(context.Background(), vm.Name,
				&v1.RemoveInterfaceOptions{Name: linuxBridgeNetworkName2},
			)).To(Succeed())

			By("wait for requested interface VM spec have 'absent' state or to be removed")
			Eventually(func() bool {
				var err error
				vm, err = kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				iface := vmispec.LookupInterfaceByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces, linuxBridgeNetworkName2)
				return iface == nil || iface.State == v1.InterfaceStateAbsent
			}, 30*time.Second).Should(BeTrue())

			By("wait for remove iface request remove from VM status")
			Eventually(func() []v1.VirtualMachineInterfaceRequest {
				var err error
				vm, err = kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vm.Status.InterfaceRequests
			}, 30*time.Second).Should(BeEmpty())

			By("verify unplugged interface is not reported in the VMI status")
			vmi = verifyDynamicInterfaceChange(vmi, plugMethod)

			By("verify unplugged iface cleared from VM & VMI spec")
			Eventually(func(g Gomega) {
				updatedVM, err := kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				g.Expect(vmispec.LookupInterfaceByName(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces, linuxBridgeNetworkName2)).To(BeNil(),
					"unplugged iface should be cleared from VM spec")
				g.Expect(libnet.LookupNetworkByName(updatedVM.Spec.Template.Spec.Networks, linuxBridgeNetworkName2)).To(BeNil(),
					"unplugged iface corresponding network should be cleared from VM spec")
				g.Expect(vmispec.LookupInterfaceByName(updatedVMI.Spec.Domain.Devices.Interfaces, linuxBridgeNetworkName2)).To(BeNil(),
					"unplugged iface should be cleared from VMI spec")
				g.Expect(libnet.LookupNetworkByName(updatedVMI.Spec.Networks, linuxBridgeNetworkName2)).To(BeNil(),
					"unplugged iface corresponding network should be cleared from VMI spec")
			}, 30*time.Second, 3*time.Second).Should(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)
	})

	Context("a stopped VM", func() {
		var vm *v1.VirtualMachine
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("create stopped VM")
			opts := append(
				libvmi.WithMasqueradeNetworking(),
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName1, networkName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName2, networkName)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName1)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName2)),
			)
			vmi = libvmi.NewAlpineWithTestTooling(opts...)
			vm = tests.NewRandomVirtualMachine(vmi, false)

			var err error
			vm, err = kubevirt.Client().VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), vm)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() v1.VirtualMachinePrintableStatus {
				virtualMachine, err := kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return virtualMachine.Status.PrintableStatus
			}, 10*time.Second, 1*time.Second).Should(Equal(v1.VirtualMachineStatusStopped))
		})

		It("cannot be subject to **hot** unplug, but will mutate the template.Spec on behalf of the user", func() {
			removeIfaceOpts := &v1.RemoveInterfaceOptions{
				Name: linuxBridgeNetworkName2,
			}

			Expect(kubevirt.Client().VirtualMachine(vm.Namespace).RemoveInterface(context.Background(), vm.Name, removeIfaceOpts)).To(Succeed())

			By("wait for requested interface be cleared from VM spec")
			Eventually(func(g Gomega) {
				var err error
				vm, err = kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				iface := vmispec.LookupInterfaceByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces, linuxBridgeNetworkName2)
				g.Expect(iface).To(BeNil(), "absent iface should be removed from stopped VM spec")
			}, 30*time.Second).Should(Succeed())

			iface := libnet.LookupNetworkByName(vm.Spec.Template.Spec.Networks, linuxBridgeNetworkName2)
			Expect(iface).To(BeNil(), "associated network of an absent interface should be removed from a stopped VM spec")
		})
	})
})

func verifyDynamicInterfaceChange(vmi *v1.VirtualMachineInstance, plugMethod hotplugMethod) *v1.VirtualMachineInstance {
	if plugMethod == migrationBased {
		migrate(vmi)
	}

	vmi, err := kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).Get(context.Background(), vmi.GetName(), &metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	nonAbsentIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	nonAbsentNets := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, nonAbsentIfaces)
	var secondaryNetworksNames []string
	for _, net := range vmispec.FilterMultusNonDefaultNetworks(nonAbsentNets) {
		secondaryNetworksNames = append(secondaryNetworksNames, net.Name)
	}
	ExpectWithOffset(1, secondaryNetworksNames).NotTo(BeEmpty())
	EventuallyWithOffset(1, func() []v1.VirtualMachineInstanceNetworkInterface {
		return cleanMACAddressesFromStatus(vmiCurrentInterfaces(vmi.GetNamespace(), vmi.GetName()))
	}, 30*time.Second).Should(
		ConsistOf(interfaceStatusFromInterfaceNames(secondaryNetworksNames...)))

	vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).Get(context.Background(), vmi.GetName(), &metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return vmi
}

func waitForSingleHotPlugIfaceOnVMISpec(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	EventuallyWithOffset(1, func() []v1.Network {
		var err error
		vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.GetNamespace()).Get(context.Background(), vmi.GetName(), &metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return vmi.Spec.Networks
	}, 30*time.Second).Should(
		ConsistOf(
			*v1.DefaultPodNetwork(),
			v1.Network{
				Name: ifaceName,
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
					NetworkName: networkName,
				}},
			},
		))
	return vmi
}

func vmiCurrentInterfaces(vmiNamespace, vmiName string) []v1.VirtualMachineInstanceNetworkInterface {
	vmi, err := kubevirt.Client().VirtualMachineInstance(vmiNamespace).Get(context.Background(), vmiName, &metav1.GetOptions{})
	ExpectWithOffset(2, err).NotTo(HaveOccurred())
	return secondaryInterfaces(vmi)
}

func addIfaceOptions(networkName, ifaceName string) *v1.AddInterfaceOptions {
	return &v1.AddInterfaceOptions{
		NetworkAttachmentDefinitionName: networkName,
		Name:                            ifaceName,
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
			InfoSource: vmispec.NewInfoSource(
				vmispec.InfoSourceDomain, vmispec.InfoSourceGuestAgent, vmispec.InfoSourceMultusStatus),
			QueueCount: 1,
		})
	}
	return ifaceStatus
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

func newVMWithOneInterface() *v1.VirtualMachine {
	vm := tests.NewRandomVirtualMachine(libvmi.NewAlpineWithTestTooling(), true)
	vm.Spec.Template.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	vm.Spec.Template.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
	return vm
}

func newStoppedVMWithOneInterface() *v1.VirtualMachine {
	vm := newVMWithOneInterface()
	stopped := false
	vm.Spec.Running = &stopped
	return vm
}

func cleanMACAddressesFromSpec(status []v1.Interface) []v1.Interface {
	for i := range status {
		status[i].MacAddress = ""
	}
	return status
}

func migrate(vmi *v1.VirtualMachineInstance) {
	By("migrating the VMI")
	migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
	migrationUID := tests.RunMigrationAndExpectCompletion(kubevirt.Client(), migration, tests.MigrationWaitTime)
	tests.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migrationUID)
}
