/*
 * This file is part of the KubeVirt project
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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package ipam

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	kvirtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("Persistent IPs", func() {

	Context("with a VirtualMachine configured with a allowPersistentIPs nad", func() {
		var (
			networkInterfaceName = "multus"
			vm                   *kvirtv1.VirtualMachine
			vmi                  *kvirtv1.VirtualMachineInstance
			nad                  *nadv1.NetworkAttachmentDefinition
		)
		BeforeEach(func() {
			Expect(checks.HasFeature(virtconfig.PersistentIPsGate)).To(BeTrue(), "should have persistent ips feature gate enabled")

			nad = &nadv1.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "persistentips",
					Namespace: testsuite.GetTestNamespace(nil),
				},
				Spec: nadv1.NetworkAttachmentDefinitionSpec{
					Config: fmt.Sprintf(`
{
        "cniVersion": "0.3.0",
        "name": "persistentips",
        "type": "ovn-k8s-cni-overlay",
        "topology": "layer2",
        "subnets": "10.100.200.0/24",
        "netAttachDefName": "%s/persistentips",
        "allowPersistentIPs": true
}
`, testsuite.GetTestNamespace(nil)),
				},
			}
			By("Create NetworkAttachmentDefinition")
			_, err := kubevirt.Client().NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(testsuite.GetTestNamespace(nil)).Create(context.TODO(), nad, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			vmi = libvmifact.NewAlpine(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkInterfaceName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkInterfaceName, "persistentips")),
			)
			vm = libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())

			By("Creating VM using the nad")
			vm, err = kubevirt.Client().VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Wait for VMI to be created")
			Eventually(matcher.ThisVMIWith(vmi.Namespace, vmi.Name)).
				WithPolling(time.Second).
				WithTimeout(2 * time.Minute).
				ShouldNot(BeNil())

			By("Wait for VMI to be ready")
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			By("Check IPAMClaim get created")
			Eventually(matcher.IPAMClaimsFromNamespace(vm.Namespace)).
				WithTimeout(time.Minute).
				WithPolling(time.Second).
				ShouldNot(BeEmpty())
		})

		JustAfterEach(func() {
			By("Check no ipamclaims leftovers after VM delete")
			Expect(kubevirt.Client().VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})).To(Succeed())
			Eventually(matcher.IPAMClaimsFromNamespace(vm.Namespace)).
				WithTimeout(time.Minute).
				WithPolling(time.Second).
				Should(BeEmpty())
		})

		It("should keep ips after live migration", func() {
			vmiIPsBeforeMigration := vmi.Status.Interfaces[0].IPs

			migration := libmigration.New(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

			Expect(matcher.ThisVMI(vmi)()).Should(matcher.MatchIPsAtInterfaceByName(networkInterfaceName, ConsistOf(vmiIPsBeforeMigration)))

		})

		It("should keep ips after restart", func() {
			vmiIPsBeforeRestart := vmi.Status.Interfaces[0].IPs
			vmiUUIDBeforeRestart := vmi.UID

			By("Re-starting the VM")
			Expect(kubevirt.Client().VirtualMachine(vm.Namespace).Restart(
				context.Background(),
				vm.Name,
				&kvirtv1.RestartOptions{},
			)).To(Succeed())

			By("Wait for a new VMI to be re-started")
			Eventually(matcher.ThisVMI(vmi)).
				WithPolling(time.Second).
				WithTimeout(90 * time.Second).
				Should(matcher.BeRestarted(vmiUUIDBeforeRestart))

			By("Wait for VMI to be ready after restart")
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			Expect(matcher.ThisVMI(vmi)()).Should(matcher.MatchIPsAtInterfaceByName(networkInterfaceName, ConsistOf(vmiIPsBeforeRestart)))
		})
	})
})
