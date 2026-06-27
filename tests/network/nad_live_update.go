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

	k8sv1 "k8s.io/api/core/v1"
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

const peerLabel = "nad-live-update-peer"

var _ = Describe(SIG("NAD name live update", decorators.RequiresTwoSchedulableNodes, Serial, func() {
	const (
		vmName          = "migrating-vm"
		vmIP            = "10.1.1.100"
		sourceNAD       = "nad-1"
		targetNAD       = "nad-2"
		pollingInterval = 2 * time.Second
		timeoutInterval = 5 * time.Minute

		staticVMI1Name = "static-vmi-1"
		staticVMI1IP   = "10.1.1.10"
		staticVMI2Name = "static-vmi-2"
		staticVMI2IP   = "10.1.1.20"
		subnetMask     = "/24"
	)
	var testNamespace string

	BeforeEach(func() {
		updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(v1.VMRolloutStrategyLiveUpdate)
		err := config.RegisterKubevirtConfigChange(
			config.WithWorkloadUpdateStrategy(updateStrategy),
			config.WithVMRolloutStrategy(rolloutStrategy),
		)
		Expect(err).ToNot(HaveOccurred())

		currentKv := libkubevirt.GetCurrentKv(kubevirt.Client())
		config.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)
	})

	BeforeEach(func() {
		testNamespace = testsuite.GetTestNamespace(nil)
		const br1 = "br-1"
		netAttachDef1 := libnet.NewBridgeNetAttachDef(sourceNAD, br1)
		_, err := libnet.CreateNetAttachDef(context.Background(), testNamespace, netAttachDef1)
		Expect(err).NotTo(HaveOccurred())

		const br2 = "br-2"
		netAttachDef2 := libnet.NewBridgeNetAttachDef(targetNAD, br2)
		_, err = libnet.CreateNetAttachDef(context.Background(), testNamespace, netAttachDef2)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		antiAffinityTerm := k8sv1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: peerLabel, Operator: metav1.LabelSelectorOpExists},
				},
			},
			TopologyKey: k8sv1.LabelHostname,
		}

		staticVMI1, err := newVMI(
			staticVMI1Name,
			sourceNAD,
			staticVMI1IP+subnetMask,
			libvmi.WithLabel(peerLabel, staticVMI1Name),
			libvmi.WithRequiredPodAntiAffinity(antiAffinityTerm),
		)
		Expect(err).ToNot(HaveOccurred())

		staticVMI1, err = kubevirt.Client().VirtualMachineInstance(testNamespace).
			Create(context.Background(), staticVMI1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		staticVMI2, err := newVMI(
			staticVMI2Name,
			targetNAD,
			staticVMI2IP+subnetMask,
			libvmi.WithLabel(peerLabel, staticVMI2Name),
			libvmi.WithRequiredPodAntiAffinity(antiAffinityTerm),
		)
		Expect(err).ToNot(HaveOccurred())

		staticVMI2, err = kubevirt.Client().VirtualMachineInstance(testNamespace).
			Create(context.Background(), staticVMI2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		affinityToStaticVMI1 := k8sv1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{peerLabel: staticVMI1Name},
			},
			TopologyKey: k8sv1.LabelHostname,
		}

		var vmi *v1.VirtualMachineInstance
		vmi, err = newVMI(
			vmName,
			sourceNAD,
			vmIP+subnetMask,
			libvmi.WithRequiredPodAffinity(affinityToStaticVMI1),
		)
		Expect(err).ToNot(HaveOccurred())

		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err = kubevirt.Client().VirtualMachine(testNamespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(matcher.ThisVMI(staticVMI1)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Eventually(matcher.ThisVMI(staticVMI2)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Eventually(matcher.ThisVM(vm)).WithTimeout(timeoutInterval).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Expect(console.LoginToAlpine(staticVMI1)).To(Succeed())

		Expect(libnet.PingFromVMConsole(staticVMI1, vmIP)).To(Succeed())
	})

	It("should modify VM network", func() {
		vm, err := kubevirt.Client().VirtualMachine(testNamespace).Get(context.Background(), vmName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = updateNADNameAndAffinity(vm, targetNAD, staticVMI2Name)
		Expect(err).NotTo(HaveOccurred())

		var vmi *v1.VirtualMachineInstance
		vmi, err = kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), vmName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for migration condition to appear and disappear")
		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(timeoutInterval).
			WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRequired))

		Eventually(matcher.ThisVMI(vmi)).
			WithTimeout(timeoutInterval).
			WithPolling(pollingInterval).
			Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceMigrationRequired))

		staticVMI2, err := kubevirt.Client().VirtualMachineInstance(testNamespace).Get(context.Background(), staticVMI2Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(console.LoginToAlpine(staticVMI2)).To(Succeed())

		Expect(libnet.PingFromVMConsole(staticVMI2, vmIP)).To(Succeed())
	})
}))

func updateNADNameAndAffinity(vm *v1.VirtualMachine, targetNAD, targetPeer string) error {
	affinityToTarget := &k8sv1.Affinity{
		PodAffinity: &k8sv1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{peerLabel: targetPeer},
					},
					TopologyKey: k8sv1.LabelHostname,
				},
			},
		},
	}

	patchData, err := patch.New(
		patch.WithReplace("/spec/template/spec/affinity", affinityToTarget),
		patch.WithReplace("/spec/template/spec/networks/0/multus/networkName", targetNAD),
	).GeneratePayload()
	if err != nil {
		return err
	}
	_, err = kubevirt.Client().VirtualMachine(vm.Namespace).Patch(
		context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	return err
}

func newVMI(name, nad, ip string, opts ...libvmi.Option) (*v1.VirtualMachineInstance, error) {
	const ifaceName = "net1"
	networkData, err := cloudinit.NewNetworkData(
		cloudinit.WithEthernet("eth0",
			cloudinit.WithAddresses(ip),
		),
	)
	if err != nil {
		return nil, err
	}
	vmi := libvmifact.NewAlpineWithTestTooling(append(opts,
		libvmi.WithName(name),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(ifaceName)),
		libvmi.WithNetwork(libvmi.MultusNetwork(ifaceName, nad)),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
	)...)
	return vmi, nil
}
