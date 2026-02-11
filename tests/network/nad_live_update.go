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
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("NAD name live update", decorators.RequiresTwoSchedulableNodes, Serial, func() {
	const (
		staticVM1Name      = "static-vm-1"
		staticVM2Name      = "static-vmi-2"
		testVMName         = "vm-under-test"
		secondaryIfaceName = "secondaryIface"
		nad1               = "nad-1"
		nad2               = "nad-2"
		br1                = "br-1"
		br2                = "br-2"
		staticVM1IP        = "10.1.1.10"
		staticVM2IP        = "10.1.2.10"
		testVMIP1          = "10.1.1.100"
		testVMIP2          = "10.1.2.100"
		subnetMask         = "/24"
		pollingInterval    = 2 * time.Second
		timeoutInterval    = 5 * time.Minute
		minNoOfNodesNeeded = 2
	)
	var testNamespace string

	BeforeEach(func() {
		virtClient := kubevirt.Client()

		config.EnableFeatureGate("LiveUpdateNADRef")

		updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(v1.VMRolloutStrategyLiveUpdate)
		err := config.RegisterKubevirtConfigChange(
			config.WithWorkloadUpdateStrategy(updateStrategy),
			config.WithVMRolloutStrategy(rolloutStrategy),
		)
		Expect(err).ToNot(HaveOccurred())

		currentKv := libkubevirt.GetCurrentKv(virtClient)
		config.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)

		testNamespace = testsuite.GetTestNamespace(nil)

		netAttachDef1 := libnet.NewBridgeNetAttachDef(nad1, br1)
		_, err = libnet.CreateNetAttachDef(context.Background(), testNamespace, netAttachDef1)
		Expect(err).NotTo(HaveOccurred())

		netAttachDef2 := libnet.NewBridgeNetAttachDef(nad2, br2)
		_, err = libnet.CreateNetAttachDef(context.Background(), testNamespace, netAttachDef2)
		Expect(err).NotTo(HaveOccurred())

		nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
		Expect(len(nodes.Items)).To(BeNumerically(">=", minNoOfNodesNeeded))
		nodeName1 := nodes.Items[0].Name

		By("Creating a static VM connected to first NAD")
		var staticVMI1 *v1.VirtualMachineInstance
		staticVMI1, err = newVMWithAffinity(
			staticVM1Name,
			testNamespace, secondaryIfaceName, nad1, staticVM1IP, subnetMask, nodeName1)
		Expect(err).ToNot(HaveOccurred())
		staticVM1 := libvmi.NewVirtualMachine(staticVMI1, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		staticVM1, err = kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), staticVM1, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVM(staticVM1)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		staticVMI1, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), staticVM1Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToAlpine(staticVMI1)).To(Succeed())

		By("Creating a VM under test connected to first NAD")
		var testVMI *v1.VirtualMachineInstance
		testVMI, err = newVMWithAffinity(
			testVMName,
			testNamespace, secondaryIfaceName, nad1, testVMIP1, subnetMask, nodeName1)
		Expect(err).ToNot(HaveOccurred())
		testVM := libvmi.NewVirtualMachine(testVMI, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		testVM, err = kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), testVM, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVM(testVM)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		testVMI, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), testVMName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToAlpine(testVMI)).To(Succeed())

		By("Verifying initial connectivity on first NAD")
		testVMI, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), testVMName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(libnet.PingFromVMConsole(testVMI, staticVM1IP)).To(Succeed())
	})

	It("should succeed when The FG is enabled", func() {
		var testVM *v1.VirtualMachine

		By("Enabling the FG LiveUpdateNADRef")
		testVMI, err := kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), testVMName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		testVM, err = kubevirt.Client().VirtualMachine(testNamespace).Get(context.Background(), testVMName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Changing the NAD name of VM under test")
		err = updateNADNameAndRemoveAffinityRules(testVM, nad2)
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the VM to auto migrate")
		Eventually(matcher.ThisVMI(testVMI), timeoutInterval, pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRequired))

		Eventually(matcher.ThisVMI(testVMI), timeoutInterval, pollingInterval).
			Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceMigrationRequired))

		By("Creating a VM on the node that the VM under test migrated to")
		testVMI, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), testVMName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		testVMNode := testVMI.Status.NodeName

		var staticVMI2 *v1.VirtualMachineInstance
		staticVMI2, err = newVMWithAffinity(
			staticVM2Name,
			testNamespace, secondaryIfaceName, nad2, staticVM2IP, subnetMask, testVMNode)
		Expect(err).ToNot(HaveOccurred())
		staticVM2 := libvmi.NewVirtualMachine(staticVMI2, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		staticVM2, err = kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), staticVM2, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVM(staticVM2)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		staticVMI2, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), staticVM1Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToAlpine(staticVMI2)).To(Succeed())

		By("Verifying connectivity on second NAD")
		err = configureStaticIP(testVMI, "eth0", testVMIP2+subnetMask)
		Expect(err).NotTo(HaveOccurred())
		testVMI, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), testVMName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(libnet.PingFromVMConsole(testVMI, staticVM2IP)).To(Succeed())
		Expect(libnet.PingFromVMConsole(testVMI, staticVM1IP)).ToNot(Succeed())
	})
}))

func configureStaticIP(vmi *v1.VirtualMachineInstance, iface, ip string) error {
	err := libnet.AddIPAddress(vmi, iface, ip)
	if err != nil {
		return err
	}
	err = libnet.SetInterfaceUp(vmi, iface)
	if err != nil {
		return err
	}
	return nil
}

func updateNADNameAndRemoveAffinityRules(vm *v1.VirtualMachine, newNAD string) error {
	patchData, err := patch.New(
		patch.WithRemove("/spec/template/spec/affinity"),
		patch.WithReplace("/spec/template/spec/networks/0/multus/networkName", newNAD),
	).GeneratePayload()
	if err != nil {
		return err
	}
	_, err = kubevirt.Client().VirtualMachine(vm.Namespace).Patch(
		context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func newVMWithAffinity(name, namespace, ifacename, nad, ip, subnetMask, node string) (*v1.VirtualMachineInstance, error) {
	networkData1, err := cloudinit.NewNetworkData(
		cloudinit.WithEthernet("eth0",
			cloudinit.WithAddresses(ip+subnetMask),
		),
	)
	if err != nil {
		return nil, err
	}
	vmi := libvmifact.NewAlpineWithTestTooling(
		libvmi.WithName(name),
		libvmi.WithNamespace(namespace),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(ifacename)),
		libvmi.WithNetwork(libvmi.MultusNetwork(ifacename, nad)),
		libvmi.WithNodeAffinityFor(node),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData1)),
	)
	return vmi, nil
}
