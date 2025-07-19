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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG(" SRIOV nic-hotplug", Serial, decorators.SRIOV, func() {
	sriovResourceName := readSRIOVResourceName()

	BeforeEach(func() {
		// Check if the hardware supports SRIOV
		Expect(validateSRIOVSetup(sriovResourceName, 1)).To(Succeed(),
			"Sriov is not enabled in this environment: %v. Skip these tests using - export FUNC_TEST_ARGS='--label-filter=!SRIOV'")

	})

	BeforeEach(func() {
		virtClient := kubevirt.Client()
		originalKv := libkubevirt.GetCurrentKv(virtClient)
		updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(v1.VMRolloutStrategyLiveUpdate)
		patchWorkloadUpdateMethodAndRolloutStrategy(originalKv.Name, virtClient, updateStrategy, rolloutStrategy)

		currentKv := libkubevirt.GetCurrentKv(virtClient)
		config.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)
	})

	Context("a running VM", func() {
		const (
			ifaceName = "iface1"
			nadName   = "skynet"
		)

		var hotPluggedVM *v1.VirtualMachine
		var hotPluggedVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("Creating a VM")
			vmi := libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			hotPluggedVM = libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			var err error
			hotPluggedVM, err = kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), hotPluggedVM, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(matcher.ThisVM(hotPluggedVM)).WithTimeout(6 * time.Minute).WithPolling(3 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			hotPluggedVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), hotPluggedVM.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(console.LoginToAlpine(hotPluggedVMI)).To(Succeed())

			By("Creating a NAD")
			Expect(createSRIOVNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), nadName, sriovResourceName)).To(Succeed())

			By("Hotplugging an interface to the VM")
			Expect(addSRIOVInterface(hotPluggedVM, ifaceName, nadName)).To(Succeed())
		})

		It("can hotplug a network interface", func() {
			libnet.WaitForSingleHotPlugIfaceOnVMISpec(hotPluggedVMI, ifaceName, nadName)

			virtClient := kubevirt.Client()

			By("Waiting for MigrationRequired condition to appear")
			Eventually(matcher.ThisVMI(hotPluggedVMI), 1*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRequired))

			By("Ensuring live-migration started")
			var migration *v1.VirtualMachineInstanceMigration
			Eventually(func() *v1.VirtualMachineInstanceMigration {
				migrations, err := virtClient.VirtualMachineInstanceMigration(hotPluggedVMI.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s", v1.MigrationSelectorLabel, hotPluggedVMI.Name),
				})
				Expect(err).ToNot(HaveOccurred())

				if len(migrations.Items) > 0 {
					migration = migrations.Items[0].DeepCopy()
					return migration
				}
				return nil
			}).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(Not(BeNil()))

			libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration)
			libmigration.ConfirmVMIPostMigration(kubevirt.Client(), hotPluggedVMI, migration)

			hotPluggedVMI = verifySriovDynamicInterfaceChange(hotPluggedVMI)

			const guestSecondaryIfaceName = "eth1"
			Expect(libnet.InterfaceExists(hotPluggedVMI, guestSecondaryIfaceName)).To(Succeed())

			updatedVM, err := kubevirt.Client().VirtualMachine(hotPluggedVM.Namespace).Get(context.Background(), hotPluggedVM.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmIfaceSpec := vmispec.LookupInterfaceByName(updatedVM.Spec.Template.Spec.Domain.Devices.Interfaces, ifaceName)
			Expect(vmIfaceSpec).NotTo(BeNil(), "VM spec should contain the new interface")
			Expect(vmIfaceSpec.MacAddress).NotTo(BeEmpty(), "VM iface spec should have MAC address")

			Eventually(func(g Gomega) {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(hotPluggedVMI.Namespace).Get(context.Background(), hotPluggedVMI.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				vmiIfaceStatus := vmispec.LookupInterfaceStatusByName(updatedVMI.Status.Interfaces, ifaceName)
				g.Expect(vmiIfaceStatus).NotTo(BeNil(), "VMI status should report the hotplugged interface")

				g.Expect(vmiIfaceStatus.MAC).To(Equal(vmIfaceSpec.MacAddress),
					"hot-plugged iface in VMI status should have a MAC address as specified in VM template spec")
			}, time.Second*30, time.Second*3).Should(Succeed())
		})
	})
}))

func createSRIOVNetworkAttachmentDefinition(namespace, networkName, sriovResourceName string) error {
	netAttachDef := libnet.NewSriovNetAttachDef(networkName, 0)
	netAttachDef.Annotations = map[string]string{libnet.ResourceNameAnnotation: sriovResourceName}
	_, err := libnet.CreateNetAttachDef(context.Background(), namespace, netAttachDef)
	return err
}

func newSRIOVNetworkInterface(name, netAttachDefName string) (v1.Network, v1.Interface) {
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
		InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
	}
	return network, iface
}

func addSRIOVInterface(vm *v1.VirtualMachine, name, netAttachDefName string) error {
	newNetwork, newIface := newSRIOVNetworkInterface(name, netAttachDefName)
	mac, err := libnet.GenerateRandomMac()
	if err != nil {
		return err
	}
	newIface.MacAddress = mac.String()
	return libnet.PatchVMWithNewInterface(vm, newNetwork, newIface)
}

func verifySriovDynamicInterfaceChange(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	const queueCount = 0
	return libnet.VerifyDynamicInterfaceChange(vmi, queueCount)
}
