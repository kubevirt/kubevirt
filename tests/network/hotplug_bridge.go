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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

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

const linuxBridgeName = "supadupabr"

var _ = SIGDescribe("bridge nic-hotplug", func() {

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
			hotPluggedVM, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), hotPluggedVM, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error {
				var err error
				hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), hotPluggedVM.GetName(), metav1.GetOptions{})
				return err
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
			libwait.WaitUntilVMIReady(hotPluggedVMI, console.LoginToAlpine)

			By("Creating a NAD")
			Expect(createBridgeNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), nadName, linuxBridgeName)).To(Succeed())

			By("Hotplugging an interface to the VM")
			Expect(addBridgeInterface(hotPluggedVM, ifaceName, nadName)).To(Succeed())
		})

		DescribeTable("can be hotplugged a network interface", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyBridgeDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			Expect(libnet.InterfaceExists(hotPluggedVMI, vmIfaceName)).To(Succeed())

			updatedVM, err := kubevirt.Client().VirtualMachine(hotPluggedVM.Namespace).Get(context.Background(), hotPluggedVM.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmIfaceSpec := vmispec.LookupInterfaceByName(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces, ifaceName)
			Expect(vmIfaceSpec).NotTo(BeNil(), "VM spec should contain the new interface")
			Expect(vmIfaceSpec.MacAddress).NotTo(BeEmpty(), "VM iface spec should have MAC address")

			Eventually(func(g Gomega) {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(hotPluggedVMI.Namespace).Get(context.Background(), hotPluggedVMI.Name, metav1.GetOptions{})
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
			hotPluggedVMI = verifyBridgeDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			By("restarting the VM")
			Expect(kubevirt.Client().VirtualMachine(hotPluggedVM.GetNamespace()).Restart(
				context.Background(),
				hotPluggedVM.GetName(),
				&v1.RestartOptions{},
			)).To(Succeed())

			By("asserting a new VMI is created, and running")
			Eventually(func() v1.VirtualMachineInstancePhase {
				newVMI, err := kubevirt.Client().VirtualMachineInstance(hotPluggedVM.GetNamespace()).Get(context.Background(), hotPluggedVM.Name, metav1.GetOptions{})
				if err != nil || hotPluggedVMI.UID == newVMI.UID {
					hotPluggedVMI.GetNamespace()
					return v1.VmPhaseUnset
				}
				return newVMI.Status.Phase
			}, 90*time.Second, 1*time.Second).Should(Equal(v1.Running))
			var err error
			hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(hotPluggedVM.GetNamespace()).Get(context.Background(), hotPluggedVM.GetName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitUntilVMIReady(hotPluggedVMI, console.LoginToAlpine)

			hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(hotPluggedVM.GetNamespace()).Get(context.Background(), hotPluggedVM.GetName(), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(libnet.InterfaceExists(hotPluggedVMI, vmIfaceName)).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)

		DescribeTable("can migrate a VMI with hotplugged interfaces", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyBridgeDynamicInterfaceChange(hotPluggedVMI, plugMethod)

			migrate(hotPluggedVMI)
			Expect(libnet.InterfaceExists(hotPluggedVMI, vmIfaceName)).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)

		DescribeTable("has connectivity over the secondary network", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyBridgeDynamicInterfaceChange(hotPluggedVMI, plugMethod)

			const subnetMask = "/24"
			const ip1 = "10.1.1.1"
			const ip2 = "10.1.1.2"

			By("Configuring static IP address on the hotplugged interface inside the guest")
			Expect(libnet.AddIPAddress(hotPluggedVMI, vmIfaceName, ip1+subnetMask)).To(Succeed())
			Expect(libnet.SetInterfaceUp(hotPluggedVMI, vmIfaceName)).To(Succeed())

			By("creating another VM connected to the same secondary network")
			net := v1.Network{
				Name: ifaceName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: nadName,
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
				libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkDataWithStaticIPsByDevice("eth1", ip2+subnetMask)),
				libvmi.WithNodeAffinityFor(hotPluggedVMI.Status.NodeName))
			anotherVmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(anotherVmi)).Create(context.Background(), anotherVmi, metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(anotherVmi, console.LoginToFedora)

			By("Ping from the VM with hotplugged interface to the other VM")
			Expect(libnet.PingFromVMConsole(hotPluggedVMI, ip2)).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)

		DescribeTable("is able to hotplug multiple network interfaces", func(plugMethod hotplugMethod) {
			waitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI)
			hotPluggedVMI = verifyBridgeDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			By("hotplugging the second interface")
			var err error
			hotPluggedVM, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), hotPluggedVM.GetName(), &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			const secondHotpluggedIfaceName = "iface2"
			Expect(addBridgeInterface(hotPluggedVM, secondHotpluggedIfaceName, nadName)).To(Succeed())

			By("wait for the second network to appear in the VMI spec")
			EventuallyWithOffset(1, func() []v1.Network {
				var err error
				hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(hotPluggedVMI.GetNamespace()).Get(context.Background(), hotPluggedVMI.GetName(), metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return hotPluggedVMI.Spec.Networks
			}, 30*time.Second).Should(
				ConsistOf(
					*v1.DefaultPodNetwork(),
					v1.Network{
						Name: secondHotpluggedIfaceName,
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
							NetworkName: nadName,
						}},
					},
					v1.Network{
						Name: ifaceName,
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
							NetworkName: nadName,
						}},
					},
				))
			hotPluggedVMI = verifyBridgeDynamicInterfaceChange(hotPluggedVMI, plugMethod)
			Expect(libnet.InterfaceExists(hotPluggedVMI, "eth2")).To(Succeed())
		},
			Entry("In place", decorators.InPlaceHotplugNICs, inPlace),
			Entry("Migration based", decorators.MigrationBasedHotplugNICs, migrationBased),
		)
	})
})

var _ = SIGDescribe("bridge nic-hotunplug", func() {
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
				testsuite.GetTestNamespace(nil), nadName, linuxBridgeName)).To(Succeed())

			By("running a VM")
			opts := append(
				libvmi.WithMasqueradeNetworking(),
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName1, nadName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName2, nadName)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName1)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName2)),
			)
			vmi = libvmi.NewAlpineWithTestTooling(opts...)
			vm = libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())

			var err error
			vm, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() error {
				vmi, err = kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				return err
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)
		})

		DescribeTable("hot-unplug network interface succeed", func(plugMethod hotplugMethod) {
			Expect(removeInterface(vm, linuxBridgeNetworkName2)).To(Succeed())

			By("wait for requested interface VMI spec to have 'absent' state or to be removed")
			Eventually(func() bool {
				var err error
				vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				iface := vmispec.LookupInterfaceByName(vmi.Spec.Domain.Devices.Interfaces, linuxBridgeNetworkName2)
				return iface == nil || iface.State == v1.InterfaceStateAbsent
			}, 30*time.Second).Should(BeTrue())

			By("verify unplugged interface is not reported in the VMI status")
			vmi = verifyBridgeDynamicInterfaceChange(vmi, plugMethod)

			vm, vmi = verifyUnpluggedIfaceClearedFromVMandVMI(vm.Namespace, vm.Name, linuxBridgeNetworkName2)

			By("Unplug the last secondary interface")
			Expect(removeInterface(vm, linuxBridgeNetworkName1)).To(Succeed())

			if plugMethod == migrationBased {
				migrate(vmi)
			}

			By("verify unplugged iface cleared from VM & VMI")
			verifyUnpluggedIfaceClearedFromVMandVMI(vm.Namespace, vm.Name, linuxBridgeNetworkName1)
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
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName1, nadName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(linuxBridgeNetworkName2, nadName)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName1)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(linuxBridgeNetworkName2)),
			)
			vmi = libvmi.NewAlpineWithTestTooling(opts...)
			vm = libvmi.NewVirtualMachine(vmi)

			var err error
			vm, err = kubevirt.Client().VirtualMachine(util.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("cannot be subject to **hot** unplug, but will mutate the template.Spec on behalf of the user", func() {
			previousVMTemplateSpec := vm.Spec.Template.Spec.DeepCopy()
			Expect(removeInterface(vm, linuxBridgeNetworkName2)).To(Succeed())

			By("wait for requested interface VM spec have 'absent' state")
			Eventually(func() v1.InterfaceState {
				var err error
				vm, err = kubevirt.Client().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				iface := vmispec.LookupInterfaceByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces, linuxBridgeNetworkName2)
				return iface.State
			}, 30*time.Second).Should(Equal(v1.InterfaceStateAbsent))

			Expect(previousVMTemplateSpec.Networks).To(Equal(vm.Spec.Template.Spec.Networks), "network spec should not change")
		})
	})
})

func createBridgeNetworkAttachmentDefinition(namespace, networkName string, bridgeName string) error {
	return libnet.CreateNetworkAttachmentDefinition(
		networkName,
		namespace,
		fmt.Sprintf(linuxBridgeNAD, networkName, namespace, bridgeCNIType, bridgeName),
	)
}

func newBridgeNetworkInterface(name, netAttachDefName string) (v1.Network, v1.Interface) {
	network := v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: netAttachDefName,
			},
		},
	}
	iface := v1.Interface{
		Name:                   name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
	}
	return network, iface
}

func addBridgeInterface(vm *v1.VirtualMachine, name, netAttachDefName string) error {
	newNetwork, newIface := newBridgeNetworkInterface(name, netAttachDefName)
	return patchVMWithNewInterface(vm, newNetwork, newIface)
}

func verifyBridgeDynamicInterfaceChange(vmi *v1.VirtualMachineInstance, plugMethod hotplugMethod) *v1.VirtualMachineInstance {
	const queueCount = 1
	return verifyDynamicInterfaceChange(vmi, plugMethod, queueCount)
}

func verifyUnpluggedIfaceClearedFromVMandVMI(namespace, vmName, netName string) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	By("verify unplugged iface cleared from VM & VMI")
	var err error
	var vmi *v1.VirtualMachineInstance
	Eventually(func(g Gomega) {
		vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(context.Background(), vmName, metav1.GetOptions{})
		Expect(err).WithOffset(1).NotTo(HaveOccurred())
		assertInterfaceUnplugedFromVMI(g, vmi, netName)
	}, 30*time.Second, 3*time.Second).WithOffset(1).Should(Succeed())

	var vm *v1.VirtualMachine
	Eventually(func(g Gomega) {
		vm, err = kubevirt.Client().VirtualMachine(namespace).Get(context.Background(), vmName, &metav1.GetOptions{})
		Expect(err).WithOffset(1).NotTo(HaveOccurred())
		assertInterfaceUnplugedFromVM(g, vm, netName)
	}, 30*time.Second, 3*time.Second).WithOffset(1).Should(Succeed())

	return vm, vmi
}

func assertInterfaceUnplugedFromVMI(g Gomega, vmi *v1.VirtualMachineInstance, name string) {
	g.Expect(vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, name)).To(BeNil(),
		"unplugged iface should be cleared from VMI status")
	g.Expect(vmispec.LookupInterfaceByName(vmi.Spec.Domain.Devices.Interfaces, name)).To(BeNil(),
		"unplugged iface should be cleared from VMI spec")
	g.Expect(libnet.LookupNetworkByName(vmi.Spec.Networks, name)).To(BeNil(),
		"unplugged iface corresponding network should be cleared from VMI spec")
}

func assertInterfaceUnplugedFromVM(g Gomega, vm *v1.VirtualMachine, name string) {
	g.Expect(vmispec.LookupInterfaceByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces, name)).To(BeNil(),
		"unplugged iface should be cleared from VM spec")
	g.Expect(libnet.LookupNetworkByName(vm.Spec.Template.Spec.Networks, name)).To(BeNil(),
		"unplugged iface corresponding network should be cleared from VM spec")
}
