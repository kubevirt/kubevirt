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
	"kubevirt.io/client-go/kubecli"

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
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("NAD name live update", decorators.RequiresTwoSchedulableNodes, Serial, func() {
	const (
		vmName          = "migrating-vm"
		vmIP            = "10.1.1.100"
		sourceNAD       = "nad-1"
		targetNAD       = "nad-2"
		pollingInterval = 2 * time.Second
		timeoutInterval = 5 * time.Minute
	)
	var (
		testNamespace string
		virtClient    kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

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
	})

	BeforeEach(func() {
		testNamespace = testsuite.GetTestNamespace(nil)
		const br1 = "br1"
		netAttachDef1 := libnet.NewBridgeNetAttachDef(sourceNAD, br1)
		_, err := libnet.CreateNetAttachDef(context.Background(), testNamespace, netAttachDef1)
		Expect(err).NotTo(HaveOccurred())

		netAttachDef2 := libnet.NewBridgeNetAttachDef(targetNAD, br1)
		_, err = libnet.CreateNetAttachDef(context.Background(), testNamespace, netAttachDef2)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		const (
			staticVMI1Name = "static-vmi-1"
			staticVMI1IP   = "10.1.1.10"
			subnetMask     = "/24"
		)

		staticVMI1, err := newVMI(
			staticVMI1Name,
			sourceNAD,
			staticVMI1IP+subnetMask,
		)
		Expect(err).ToNot(HaveOccurred())

		staticVMI1, err = kubevirt.Client().VirtualMachineInstance(testNamespace).
			Create(context.Background(), staticVMI1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		var vmi *v1.VirtualMachineInstance
		vmi, err = newVMI(
			vmName,
			sourceNAD,
			vmIP+subnetMask,
		)
		Expect(err).ToNot(HaveOccurred())

		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err = kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVMI(staticVMI1)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Eventually(matcher.ThisVM(vm)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Expect(console.LoginToAlpine(staticVMI1)).To(Succeed())

		Expect(libnet.PingFromVMConsole(staticVMI1, vmIP)).To(Succeed())
	})

	It("should modify VM network", func() {
		vm, err := kubevirt.Client().VirtualMachine(testNamespace).Get(context.Background(), vmName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi, err := kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vmName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		sourceNodeName := vmi.Status.NodeName
		Expect(sourceNodeName).ToNot(BeEmpty())

		err = updateNADName(vm, targetNAD)
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for migration condition to appear and disappear")
		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(timeoutInterval).
			WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRequired))

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(timeoutInterval).
			WithPolling(pollingInterval).
			Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceMigrationRequired))

		Eventually(func() (string, error) {
			vmi, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vmName, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			return vmi.Status.NodeName, nil
		}).
			WithTimeout(timeoutInterval).
			WithPolling(pollingInterval).
			Should(SatisfyAll(
				Not(BeEmpty()),
				Not(Equal(sourceNodeName)),
			))

		var staticVMI2 *v1.VirtualMachineInstance
		const (
			staticVMI2Name = "static-vmi-2"
			staticVMI2IP   = "10.1.1.20"
			subnetMask     = "/24"
		)
		staticVMI2, err = newVMI(
			staticVMI2Name,
			targetNAD,
			staticVMI2IP+subnetMask,
		)
		Expect(err).ToNot(HaveOccurred())

		staticVMI2, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(staticVMI2)).
			Create(context.Background(), staticVMI2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(matcher.ThisVMI(staticVMI2)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Expect(console.LoginToAlpine(staticVMI2)).To(Succeed())

		Expect(libnet.PingFromVMConsole(staticVMI2, vmIP)).To(Succeed())
	})
}))

func updateNADName(vm *v1.VirtualMachine, targetNAD string) error {
	patchData, err := patch.New(
		patch.WithReplace("/spec/template/spec/networks/0/multus/networkName", targetNAD),
	).GeneratePayload()
	if err != nil {
		return err
	}
	_, err = kubevirt.Client().VirtualMachine(vm.Namespace).Patch(
		context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	return err
}

func newVMI(name, nad, ip string) (*v1.VirtualMachineInstance, error) {
	const ifaceName = "net1"
	networkData1, err := cloudinit.NewNetworkData(
		cloudinit.WithEthernet("eth0",
			cloudinit.WithAddresses(ip),
		),
	)
	if err != nil {
		return nil, err
	}
	vmi := libvmifact.NewAlpineWithTestTooling(
		libvmi.WithName(name),
		libvmi.WithInterface(libvmi.NewInterface(ifaceName, libvmi.WithBridgeBinding())),
		libvmi.WithNetwork(libvmi.MultusNetwork(ifaceName, nad)),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData1)),
	)
	return vmi, nil
}
