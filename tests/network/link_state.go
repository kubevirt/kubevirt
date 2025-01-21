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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"
	"kubevirt.io/kubevirt/tests/decorators"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	defaultInterfaceName = "default"
	bridgeInterfaceName2 = "bridge2"
	bridgeInterfaceName3 = "bridge3"
	mac1                 = "02:4d:8b:00:00:3a"
	mac2                 = "02:4d:8b:00:00:4b"
	mac3                 = "02:4d:8b:00:00:5c"
	regexLinkStateUP     = "'state[[:space:]]+UP'"
	regexLinkStateDOWN   = "'state[[:space:]]+DOWN'"
	ipLinkTemplate       = "ip -one link | grep %s | grep -E %s\n"
)

var _ = SIGDescribe("interface state up/down", decorators.InPlaceHotplugNICs, func() {

	var vmName types.NamespacedName
	BeforeEach(func() {
		var netDefault *v1.Network
		var net2 *v1.Network
		var ifaceDefault *v1.Interface
		var iface2 *v1.Interface

		netDefault = v1.DefaultPodNetwork()
		net2 = newMultusNetwork(bridgeInterfaceName2, nadOf(bridgeInterfaceName2))

		ifaceDefault = setInterfaceStateAndMAC(
			pointer.P(libvmi.InterfaceDeviceWithMasqueradeBinding()), "", mac1)
		iface2 = setInterfaceStateAndMAC(
			pointer.P(libvmi.InterfaceDeviceWithBridgeBinding(bridgeInterfaceName2)),
			v1.InterfaceStateLinkDown, mac2)

		_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil),
			libnet.NewBridgeNetAttachDef(nadOf(bridgeInterfaceName2), "br02"))
		Expect(err).NotTo(HaveOccurred())
		_, err = libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil),
			libnet.NewBridgeNetAttachDef(nadOf(bridgeInterfaceName3), "br03"))
		Expect(err).NotTo(HaveOccurred())

		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(*ifaceDefault),
			libvmi.WithNetwork(netDefault),
			libvmi.WithInterface(*iface2),
			libvmi.WithNetwork(net2),
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(
				cloudinit.CreateNetworkDataWithStaticIPsByMac("eth1", mac2, "10.1.1.2/24"),
			)),
		)

		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisVM(vm)).WithTimeout(3 * time.Minute).WithPolling(3 * time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmName = types.NamespacedName{Name: vmi.GetName(), Namespace: vmi.GetNamespace()}
		Expect(console.LoginToFedora(vmi)).To(Succeed())
	})

	Context("VM with one link up, one link down, one hot-plugged down", Serial, func() {

		It("status and guest should show correct iface state", func() {
			vmi := WaitForStatusToShowExpectedStates(vmName, map[string]v1.InterfaceState{
				defaultInterfaceName: v1.InterfaceStateLinkUp,
				bridgeInterfaceName2: v1.InterfaceStateLinkDown,
			})

			var timeout = time.Second * 5
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac1, regexLinkStateUP), timeout)).To(Succeed())
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac2, regexLinkStateDOWN), timeout)).To(Succeed())

			By("flipping the state of both interfaces")

			vm, err := kubevirt.Client().VirtualMachine(vmName.Namespace).Get(context.Background(), vmName.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(PatchFlipVMInterfacesStates(vm)).To(Succeed())

			vmi = WaitForStatusToShowExpectedStates(vmName, map[string]v1.InterfaceState{
				defaultInterfaceName: v1.InterfaceStateLinkDown,
				bridgeInterfaceName2: v1.InterfaceStateLinkUp,
			})
			timeout = time.Second * 30
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac1, regexLinkStateDOWN), timeout)).To(Succeed())
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac2, regexLinkStateUP), timeout)).To(Succeed())
		})

		It("hot plugging an interface with link state down and migrating vmi", func() {
			hotplugNet := newMultusNetwork(bridgeInterfaceName3, nadOf(bridgeInterfaceName3))
			hotplugIface := setInterfaceStateAndMAC(
				pointer.P(libvmi.InterfaceDeviceWithBridgeBinding(bridgeInterfaceName3)),
				v1.InterfaceStateLinkDown, mac3)

			vm, err := kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmName.Name, metav1.GetOptions{})

			By("hot-plugging an interface is state down")
			Expect(err).ToNot(HaveOccurred())
			Expect(libnet.PatchVMWithNewInterface(vm, *hotplugNet, *hotplugIface)).To(Succeed())
			vmi := WaitForStatusToShowExpectedStates(vmName, map[string]v1.InterfaceState{
				defaultInterfaceName: v1.InterfaceStateLinkUp,
				bridgeInterfaceName2: v1.InterfaceStateLinkDown,
				//	bridgeInterfaceName3: v1.InterfaceStateLinkDown,
			})
			timeout := 30 * time.Second
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac1, regexLinkStateUP), timeout)).To(Succeed())
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac2, regexLinkStateDOWN), timeout)).To(Succeed())
			//	Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac3, regexLinkStateDOWN), timeout)).To(Succeed())

			By("migrating the hotplugged VMI")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
			libmigration.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migrationUID)

			vmi = WaitForStatusToShowExpectedStates(vmName, map[string]v1.InterfaceState{
				defaultInterfaceName: v1.InterfaceStateLinkUp,
				bridgeInterfaceName2: v1.InterfaceStateLinkDown,
				bridgeInterfaceName3: v1.InterfaceStateLinkDown,
			})

			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac1, regexLinkStateUP), timeout)).To(Succeed())
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac2, regexLinkStateDOWN), timeout)).To(Succeed())
			Expect(console.RunCommand(vmi, fmt.Sprintf(ipLinkTemplate, mac3, regexLinkStateDOWN), timeout)).To(Succeed())

		})
	})
})

func newMultusNetwork(name, netAttachDefName string) *v1.Network {
	return &v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: netAttachDefName,
			},
		},
	}
}

func setInterfaceStateAndMAC(iface *v1.Interface, state v1.InterfaceState, mac string) *v1.Interface {
	if state != "" {
		iface.State = state
	}
	if mac != "" {
		iface.MacAddress = mac
	}
	return iface
}

func nadOf(s string) string { return s + "-nad" }

func WaitForStatusToShowExpectedStates(name types.NamespacedName, ifaceNameToState map[string]v1.InterfaceState) *v1.VirtualMachineInstance {
	var vmi *v1.VirtualMachineInstance
	EventuallyWithOffset(1, func() map[string]v1.InterfaceState {
		var err error
		vmi, err = kubevirt.Client().VirtualMachineInstance(name.Namespace).Get(context.Background(), name.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		m := make(map[string]v1.InterfaceState, len(ifaceNameToState))
		for _, iface := range vmi.Status.Interfaces {
			m[iface.Name] = v1.InterfaceState(iface.LinkState)
		}
		return m
	}).
		WithPolling(10 * time.Second).
		WithTimeout(6 * time.Minute).
		Should(Equal(ifaceNameToState))
	return vmi
}

func flipState(curState v1.InterfaceState) v1.InterfaceState {
	if curState == v1.InterfaceStateLinkDown {
		return v1.InterfaceStateLinkUp
	}
	return v1.InterfaceStateLinkDown
}

func PatchFlipVMInterfacesStates(vm *v1.VirtualMachine) error {
	interfaceState := make([]v1.InterfaceState, 2)
	for i, iface := range vm.Spec.Template.Spec.Domain.Devices.Interfaces {
		interfaceState[i] = flipState(iface.State)
	}
	patchData, err := patch.New(
		patch.WithReplace("/spec/template/spec/domain/devices/interfaces/0/state", interfaceState[0]),
		patch.WithReplace("/spec/template/spec/domain/devices/interfaces/1/state", interfaceState[1])).
		GeneratePayload()
	if err != nil {
		return err
	}

	_, err = kubevirt.Client().VirtualMachine(vm.Namespace).Patch(
		context.Background(),
		vm.Name,
		types.JSONPatchType,
		patchData,
		metav1.PatchOptions{},
	)
	return err
}
