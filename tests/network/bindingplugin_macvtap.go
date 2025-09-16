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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VirtualMachineInstance with macvtap network binding plugin", decorators.Macvtap, decorators.NetCustomBindingPlugins, Serial, func() {
	const (
		macvtapLowerDevice = "eth0"
		macvtapNetworkName = "net1"
	)

	BeforeEach(func() {
		const macvtapBindingName = "macvtap"
		err := config.RegisterKubevirtConfigChange(
			config.WithNetBindingPlugin(macvtapBindingName, v1.InterfaceBindingPlugin{
				DomainAttachmentType: v1.Tap,
			}),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		netAttachDef := libnet.NewMacvtapNetAttachDef(macvtapNetworkName, macvtapLowerDevice)
		_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
		Expect(err).NotTo(HaveOccurred())
	})

	var serverMAC, clientMAC string
	BeforeEach(func() {
		mac, err := libnet.GenerateRandomMac()
		serverMAC = mac.String()
		Expect(err).NotTo(HaveOccurred())
		mac, err = libnet.GenerateRandomMac()
		Expect(err).NotTo(HaveOccurred())
		clientMAC = mac.String()
	})

	It("two VMs with macvtap interface should be able to communicate over macvtap network", func() {
		const (
			guestIfaceName = "eth0"
			serverIPAddr   = "192.0.2.102"
			serverCIDR     = serverIPAddr + "/24"
			clientCIDR     = "192.0.2.101/24"
		)
		nodeList := libnode.GetAllSchedulableNodes(kubevirt.Client())
		Expect(nodeList.Items).NotTo(BeEmpty(), "schedulable kubernetes nodes must be present")
		nodeName := nodeList.Items[0].Name

		opts := []libvmi.Option{
			libvmi.WithInterface(*libvmi.InterfaceWithMacvtapBindingPlugin(macvtapNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName, macvtapNetworkName)),
			libvmi.WithNodeAffinityFor(nodeName),
		}
		serverVMI := libvmifact.NewAlpineWithTestTooling(opts...)
		clientVMI := libvmifact.NewAlpineWithTestTooling(opts...)

		var err error
		ns := testsuite.GetTestNamespace(nil)
		serverVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Create(context.Background(), serverVMI, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		clientVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Create(context.Background(), clientVMI, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToAlpine)
		clientVMI = libwait.WaitUntilVMIReady(clientVMI, console.LoginToAlpine)

		Expect(libnet.AddIPAddress(serverVMI, guestIfaceName, serverCIDR)).To(Succeed())
		Expect(libnet.AddIPAddress(clientVMI, guestIfaceName, clientCIDR)).To(Succeed())

		Expect(libnet.PingFromVMConsole(clientVMI, serverIPAddr)).To(Succeed())
	})

	Context("VMI migration", decorators.RequiresTwoSchedulableNodes, func() {
		var clientVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(*libvmi.InterfaceWithMac(
					libvmi.InterfaceWithMacvtapBindingPlugin("test"), clientMAC)),
				libvmi.WithNetwork(libvmi.MultusNetwork("test", macvtapNetworkName)),
			)
			var err error
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "should create VMI successfully")
			clientVMI = libwait.WaitUntilVMIReady(clientVMI, console.LoginToAlpine)
		})

		It("should be successful when the VMI MAC address is defined in its spec", func() {
			Expect(clientVMI.Status.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(clientVMI.Status.Interfaces[0].MAC).To(Equal(clientMAC), "the expected MAC address should be set in the VMI")

			By("starting the migration")
			migration := libmigration.New(clientVMI.Name, clientVMI.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(kubevirt.Client(), clientVMI, migration)
		})

		Context("with live traffic", func() {
			var serverVMI *v1.VirtualMachineInstance
			var serverIP string

			const macvtapIfaceIPReportTimeout = 4 * time.Minute

			BeforeEach(func() {
				serverVMI = libvmifact.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithInterface(*libvmi.InterfaceWithMac(
						libvmi.InterfaceWithMacvtapBindingPlugin(macvtapNetworkName), serverMAC)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName, macvtapNetworkName)),
				)
				var err error
				serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "should create VMI successfully")
				serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToFedora)

				Expect(serverVMI.Status.Interfaces).NotTo(BeEmpty(), "a migrate-able VMI must have network interfaces")

				serverIP, err = waitVMMacvtapIfaceIPReport(serverVMI, serverMAC, macvtapIfaceIPReportTimeout)
				Expect(err).NotTo(HaveOccurred(), "should have managed to figure out the IP of the server VMI")
			})

			BeforeEach(func() {
				// TODO test also the IPv6 address (issue- https://github.com/kubevirt/kubevirt/issues/7506)
				libnet.SkipWhenClusterNotSupportIpv4()
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *before* migrating the VMI")
			})

			It("should keep connectivity after a migration", func() {
				const containerCompletionWaitTime = 60
				serverVmiPod, err := libpod.GetPodByVirtualMachineInstance(serverVMI, testsuite.GetTestNamespace(nil))
				Expect(err).ToNot(HaveOccurred())
				migration := libmigration.New(serverVMI.Name, serverVMI.GetNamespace())
				_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
				// In case of clientVMI and serverVMI running on the same node before migration, the serverVMI
				// will be reachable only when the original launcher pod terminates.
				Eventually(func() error {
					return waitForPodCompleted(serverVMI.Namespace, serverVmiPod.Name)
				}, containerCompletionWaitTime, time.Second).Should(Succeed(), fmt.Sprintf("all containers should complete in source virt-launcher pod: %s", serverVMI.Name))
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *after* migrating the VMI")
			})
		})
	})
}))

func waitForPodCompleted(podNamespace, podName string) error {
	pod, err := kubevirt.Client().CoreV1().Pods(podNamespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
		return nil
	}
	return fmt.Errorf("pod hasn't completed, current Phase: %s", pod.Status.Phase)
}

func waitVMMacvtapIfaceIPReport(vmi *v1.VirtualMachineInstance, macAddress string, timeout time.Duration) (string, error) {
	var vmiIP string
	err := virtwait.PollImmediately(time.Second, timeout, func(ctx context.Context) (done bool, err error) {
		vmi, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(ctx, vmi.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, iface := range vmi.Status.Interfaces {
			if iface.MAC == macAddress {
				if ip := iface.IP; ip != "" {
					vmiIP = ip
					return true, nil
				}
				return false, nil
			}
		}

		return false, nil
	})
	if err != nil {
		return "", err
	}

	return vmiIP, nil
}
