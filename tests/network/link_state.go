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
 * Copyright the KubeVirt Authors.
 *
 */

package network

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("interface state up/down", func() {
	const (
		primaryLogicalNetName    = "default"
		secondary2LogicalNetName = "bridge2"
		nadName                  = "bridge-nad"
	)
	const (
		waitForGAConnectedTimout        = 6 * time.Minute
		waitForGAConnectedRetryInterval = 3 * time.Second
	)
	const waitForExpectedIfaceStatusesTimeout = 60 * time.Second
	BeforeEach(func() {
		testNamespace := testsuite.GetTestNamespace(nil)
		_, err := libnet.CreateNetAttachDef(context.Background(), testNamespace,
			libnet.NewBridgeNetAttachDef(nadName, "br02"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("status and guest should show correct iface state", func() {
		testNamespace := testsuite.GetTestNamespace(nil)
		mac1, err := libnet.GenerateRandomMac()
		Expect(err).NotTo(HaveOccurred())
		mac2, err := libnet.GenerateRandomMac()
		Expect(err).NotTo(HaveOccurred())

		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(v1.Interface{
				Name: primaryLogicalNetName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
				MacAddress: mac1.String(),
				State:      v1.InterfaceStateLinkUp,
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(v1.Interface{
				Name: secondary2LogicalNetName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
				MacAddress: mac2.String(),
				State:      v1.InterfaceStateLinkDown,
			}),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondary2LogicalNetName, nadName)),
		)

		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err = kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVM(vm)).
			WithTimeout(waitForGAConnectedTimout).
			WithPolling(waitForGAConnectedRetryInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		vmi, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		expectedIfaceStatuses := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: primaryLogicalNetName, LinkState: string(v1.InterfaceStateLinkUp)},
			{Name: secondary2LogicalNetName, LinkState: string(v1.InterfaceStateLinkDown)},
		}

		Eventually(func() ([]v1.VirtualMachineInstanceNetworkInterface, error) {
			fetchedVMI, err := kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return normalizeIfaceStatuses(fetchedVMI.Status.Interfaces), nil
		}).WithTimeout(waitForExpectedIfaceStatusesTimeout).Should(ConsistOf(expectedIfaceStatuses))

		timeout := 5 * time.Second
		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac1.String(), v1.InterfaceStateLinkUp), timeout)).To(Succeed())
		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac2.String(), v1.InterfaceStateLinkDown), timeout)).To(Succeed())

		By("flipping the state of both interfaces")

		Expect(patchToggleVMInterfacesStates(vm)).To(Succeed())

		Consistently(matcher.ThisVM(vm)).
			WithTimeout(15 * time.Second).
			WithPolling(1 * time.Second).
			Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineRestartRequired))

		expectedIfaceStatuses = []v1.VirtualMachineInstanceNetworkInterface{
			{Name: primaryLogicalNetName, LinkState: string(v1.InterfaceStateLinkDown)},
			{Name: secondary2LogicalNetName, LinkState: string(v1.InterfaceStateLinkUp)},
		}

		Eventually(func() ([]v1.VirtualMachineInstanceNetworkInterface, error) {
			fetchedVMI, err := kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return normalizeIfaceStatuses(fetchedVMI.Status.Interfaces), nil
		}).WithTimeout(waitForExpectedIfaceStatusesTimeout).Should(ConsistOf(expectedIfaceStatuses))

		timeout = 30 * time.Second
		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac1.String(), v1.InterfaceStateLinkDown), timeout)).To(Succeed())
		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac2.String(), v1.InterfaceStateLinkUp), timeout)).To(Succeed())
	})

	It("status and guest should show iface is down when vm with ifaces down is migrated", func() {
		testNamespace := testsuite.GetTestNamespace(nil)
		mac1, err := libnet.GenerateRandomMac()
		Expect(err).NotTo(HaveOccurred())
		mac2, err := libnet.GenerateRandomMac()
		Expect(err).NotTo(HaveOccurred())

		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(v1.Interface{
				Name: primaryLogicalNetName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
				MacAddress: mac1.String(),
				State:      v1.InterfaceStateLinkDown,
			}),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(v1.Interface{
				Name: secondary2LogicalNetName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Bridge: &v1.InterfaceBridge{},
				},
				MacAddress: mac2.String(),
				State:      v1.InterfaceStateLinkDown,
			}),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondary2LogicalNetName, nadName)),
		)

		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err = kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVM(vm)).
			WithTimeout(waitForGAConnectedTimout).
			WithPolling(waitForGAConnectedRetryInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		vmi, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		expectedIfaceStatuses := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: primaryLogicalNetName, LinkState: string(v1.InterfaceStateLinkDown)},
			{Name: secondary2LogicalNetName, LinkState: string(v1.InterfaceStateLinkDown)},
		}

		Eventually(func() ([]v1.VirtualMachineInstanceNetworkInterface, error) {
			fetchedVMI, err := kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return normalizeIfaceStatuses(fetchedVMI.Status.Interfaces), nil
		}).WithTimeout(waitForExpectedIfaceStatusesTimeout).Should(ConsistOf(expectedIfaceStatuses))

		timeout := 5 * time.Second
		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac1.String(), v1.InterfaceStateLinkDown), timeout)).To(Succeed())
		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac2.String(), v1.InterfaceStateLinkDown), timeout)).To(Succeed())

		By("migrating the vmi")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
		libmigration.ConfirmVMIPostMigration(kubevirt.Client(), vmi, migration)

		vmi, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() ([]v1.VirtualMachineInstanceNetworkInterface, error) {
			fetchedVMI, err := kubevirt.Client().VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return normalizeIfaceStatuses(fetchedVMI.Status.Interfaces), nil
		}).WithTimeout(waitForExpectedIfaceStatusesTimeout).Should(ConsistOf(expectedIfaceStatuses))

		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac1.String(), v1.InterfaceStateLinkDown), timeout)).To(Succeed())
		Expect(console.RunCommand(vmi, libnet.NewLinkStateAssersionCmd(mac2.String(), v1.InterfaceStateLinkDown), timeout)).To(Succeed())

	})
}))

func normalizeIfaceStatuses(ifaceStatuses []v1.VirtualMachineInstanceNetworkInterface) []v1.VirtualMachineInstanceNetworkInterface {
	var result []v1.VirtualMachineInstanceNetworkInterface
	for _, ifaceStatus := range ifaceStatuses {
		result = append(result, v1.VirtualMachineInstanceNetworkInterface{Name: ifaceStatus.Name, LinkState: ifaceStatus.LinkState})
	}
	return result
}

func patchToggleVMInterfacesStates(vm *v1.VirtualMachine) error {
	vmIfaces := vm.Spec.Template.Spec.Domain.Devices.Interfaces

	patchOptions := make([]patch.PatchOption, len(vmIfaces))

	for i, iface := range vmIfaces {
		patchOptions[i] = patch.WithReplace(
			fmt.Sprintf("/spec/template/spec/domain/devices/interfaces/%d/state", i),
			flipState(iface.State))
	}
	patchData, err := patch.New(patchOptions...).GeneratePayload()
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

func flipState(curState v1.InterfaceState) v1.InterfaceState {
	if curState == v1.InterfaceStateLinkDown {
		return v1.InterfaceStateLinkUp
	}
	return v1.InterfaceStateLinkDown
}
