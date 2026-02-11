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
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	"kubevirt.io/kubevirt/tests/libnode"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("NAD name live update", Serial, func() {
	const (
		nad1           = "nad-1"
		nad2           = "nad-2"
		nadNonExistent = "nadNA"
		br1            = "br-1"
		br2            = "br-2"
		ip1            = "10.1.1.10"
		ip2            = "10.1.2.10"
		ipOld          = "10.1.1.100"
		ipNew          = "10.1.2.100"
		subnetMask     = "/24"
		timeout30s     = 30 * time.Second
		timeout2s      = 2 * time.Second
		timeout5s      = 5 * time.Second
		timeout1m      = 1 * time.Minute
		timeout2m      = 2 * time.Minute
		timeout5m      = 5 * time.Minute
	)
	var testNamespace string

	BeforeEach(func() {
		virtClient := kubevirt.Client()
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
	})

	It("should succeed when The FG is enabled", func() {
		By("Enabling the FG LiveUpdateNADRef")
		config.EnableFeatureGate("LiveUpdateNADRef")

		nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
		Expect(nodes.Items).NotTo(BeEmpty())
		nodeName1 := nodes.Items[0].Name

		By("Creating 2 VMs")
		_, vmi1 := createAndVerifyLoginNewVMWithAffinity(nodeName1, nad1, testNamespace)
		configureStaticIP(vmi1, "eth0", ip1+subnetMask)

		vmTest, vmiTest := createAndVerifyLoginNewVMWithAffinity(nodeName1, nad1, testNamespace)
		configureStaticIP(vmiTest, "eth0", ipOld+subnetMask)

		By("Verifying initial connectivity on first NAD")
		Expect(libnet.PingFromVMConsole(vmiTest, ip1)).To(Succeed())

		By("Changing the NAD name of VM under test")
		patchData, err := patch.New(
			patch.WithRemove("/spec/template/spec/affinity"),
			patch.WithReplace("/spec/template/spec/networks/0/multus/networkName", nad2),
		).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())
		_, err = kubevirt.Client().VirtualMachine(vmTest.Namespace).Patch(
			context.Background(), vmTest.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the VM to auto migrate")
		Eventually(matcher.ThisVMI(vmiTest), timeout1m, timeout2s).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRequired))

		var migration *v1.VirtualMachineInstanceMigration
		Eventually(func() *v1.VirtualMachineInstanceMigration {
			var migrations *v1.VirtualMachineInstanceMigrationList
			migrations, err = kubevirt.Client().VirtualMachineInstanceMigration(testNamespace).
				List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			for i, mig := range migrations.Items {
				if mig.Spec.VMIName == vmTest.Name {
					migration = migrations.Items[i].DeepCopy()
					return migration
				}
			}
			return nil
		}, timeout2m, timeout2s).ShouldNot(BeNil())
		libmigration.ExpectMigrationToSucceedWithDefaultTimeout(kubevirt.Client(), migration)

		By("Creating a VM on the node that the VM under test migrated to")
		vmiTest, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vmTest.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		newNode := vmiTest.Status.NodeName

		_, vmi2 := createAndVerifyLoginNewVMWithAffinity(newNode, nad2, testNamespace)
		configureStaticIP(vmi2, "eth0", ip2+subnetMask)

		By("Verifying connectivity on second NAD")
		configureStaticIP(vmiTest, "eth0", ipNew+subnetMask)
		Expect(libnet.PingFromVMConsole(vmiTest, ip2)).To(Succeed())
		Expect(libnet.PingFromVMConsole(vmiTest, ip1)).ToNot(Succeed())
	})

	It("should fail at migration if updated to non existent NAD ref", func() {
		By("Enabling the FG LiveUpdateNADRef")
		config.EnableFeatureGate("LiveUpdateNADRef")

		nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
		Expect(nodes.Items).NotTo(BeEmpty())
		nodeName1 := nodes.Items[0].Name

		By("Creating a VM")
		vmTest, vmiTest := createAndVerifyLoginNewVMWithAffinity(nodeName1, nad1, testNamespace)

		By("Changing the NAD name of VM under test")
		patchData, err := patch.New(
			patch.WithRemove("/spec/template/spec/affinity"),
			patch.WithReplace("/spec/template/spec/networks/0/multus/networkName", nadNonExistent),
		).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())
		_, err = kubevirt.Client().VirtualMachine(vmTest.Namespace).Patch(
			context.Background(), vmTest.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the migration to fail due to missing NAD")
		Eventually(matcher.ThisVMI(vmiTest), timeout1m, timeout2s).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRequired))

		var migration *v1.VirtualMachineInstanceMigration
		Eventually(func(g Gomega) {
			var migrations *v1.VirtualMachineInstanceMigrationList
			migrations, err = kubevirt.Client().VirtualMachineInstanceMigration(testNamespace).
				List(context.Background(), metav1.ListOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			for i, mig := range migrations.Items {
				if mig.Spec.VMIName == vmTest.Name {
					migration = migrations.Items[i].DeepCopy()
					break
				}
			}
			g.Expect(migration).ToNot(BeNil())
		}, timeout5m, timeout5s).Should(Succeed())

		Consistently(func(g Gomega) {
			currentMig, err := kubevirt.Client().VirtualMachineInstanceMigration(testNamespace).
				Get(context.Background(), migration.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(currentMig.Status.Phase).ToNot(Equal(v1.MigrationSucceeded))
		}, timeout30s, timeout5s).Should(Succeed())

		By("Verifying the VMI spec was updated even if migration failed")
		vmiTest, _ = kubevirt.Client().VirtualMachineInstance(testNamespace).
			Get(context.Background(), vmTest.Name, metav1.GetOptions{})
		Expect(vmiTest.Spec.Networks[0].Multus.NetworkName).To(Equal(nadNonExistent))
	})

	It("should not work if FG is disabled", func() {
		By("Disabling the FG LiveUpdateNADRef")
		config.DisableFeatureGate("LiveUpdateNADRef")

		nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
		Expect(nodes.Items).NotTo(BeEmpty())
		nodeName1 := nodes.Items[0].Name

		By("Creating a VM")
		vmTest, vmiTest := createAndVerifyLoginNewVMWithAffinity(nodeName1, nad1, testNamespace)
		configureStaticIP(vmiTest, "eth0", ipOld+subnetMask)

		By("Changing the NAD name of VM under test")
		patchData, err := patch.New(
			patch.WithRemove("/spec/template/spec/affinity"),
			patch.WithReplace("/spec/template/spec/networks/0/multus/networkName", nad2),
		).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())
		_, err = kubevirt.Client().VirtualMachine(vmTest.Namespace).Patch(
			context.Background(), vmTest.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Verifying no migration required condition appears")
		Consistently(matcher.ThisVMI(vmiTest), timeout30s, timeout5s).
			ShouldNot(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRequired))

		By("Verifying that the NAD ref in VMI spec has not changed")
		Consistently(func(g Gomega) {
			vmi, err := kubevirt.Client().VirtualMachineInstance(testNamespace).
				Get(context.Background(), vmTest.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vmi.Spec.Networks[0].Multus.NetworkName).To(Equal(nad1))
		}, timeout30s, timeout5s).Should(Succeed())
	})
}))

func createAndVerifyLoginNewVMWithAffinity(node, nad, ns string) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	const (
		timeout2s = 2 * time.Second
		timeout5m = 5 * time.Minute
	)
	secondaryIfaceName := "secondaryIface"
	vmi := libvmifact.NewAlpineWithTestTooling(
		libvmi.WithNodeAffinityFor(node),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryIfaceName)),
		libvmi.WithNetwork(libvmi.MultusNetwork(secondaryIfaceName, nad)),
	)
	vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
	vm, err := kubevirt.Client().VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	Eventually(matcher.ThisVM(vm)).WithTimeout(timeout5m).WithPolling(timeout2s).
		Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
	vmi, err = kubevirt.Client().VirtualMachineInstance(ns).Get(context.Background(), vm.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(console.LoginToAlpine(vmi)).To(Succeed())
	return vm, vmi
}

func configureStaticIP(vmi *v1.VirtualMachineInstance, iface, ip string) {
	Expect(libnet.AddIPAddress(vmi, iface, ip)).To(Succeed())
	Expect(libnet.SetInterfaceUp(vmi, iface)).To(Succeed())
}
