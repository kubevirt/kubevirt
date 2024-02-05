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
 * Copyright 2024 Red Hat, Inc.
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
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkvconfig"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("VirtualMachineInstance with macvtap network binding plugin", decorators.Macvtap, decorators.NetCustomBindingPlugins, Serial, func() {
	const (
		macvtapLowerDevice = "eth0"
		macvtapNetworkName = "net1"
	)

	BeforeEach(func() {
		tests.EnableFeatureGate(virtconfig.NetworkBindingPlugingsGate)
	})

	BeforeEach(func() {
		const macvtapBindingName = "macvtap"
		err := libkvconfig.WithNetBindingPlugin(macvtapBindingName, v1.InterfaceBindingPlugin{
			DomainAttachmentType: v1.Tap,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		ns := testsuite.GetTestNamespace(nil)
		Expect(libnet.CreateMacvtapNetworkAttachmentDefinition(ns, macvtapNetworkName, macvtapLowerDevice)).To(Succeed(),
			"A macvtap network named %s should be provisioned", macvtapNetworkName)
	})

	It("should successfully create a VM with macvtap interface with custom MAC address", func() {
		const mac = "02:00:00:00:00:02"
		vmi, err := createAlpineVMIRandomNode(macvtapNetworkName, mac)
		Expect(err).NotTo(HaveOccurred())

		Expect(vmi.Status.Interfaces).To(HaveLen(1), "should have a single interface")
		Expect(vmi.Status.Interfaces[0].MAC).To(Equal(mac), "the expected MAC address should be set in the VMI")
	})

	It("two VMs with macvtap interface should be able to communicate over macvtap network", func() {
		nodeList := libnode.GetAllSchedulableNodes(kubevirt.Client())
		Expect(nodeList.Items).NotTo(BeEmpty(), "schedulable kubernetes nodes must be present")
		nodeName := nodeList.Items[0].Name

		chosenMACHW, err := GenerateRandomMac()
		Expect(err).ToNot(HaveOccurred())
		chosenMAC := chosenMACHW.String()

		serverCIDR := "192.0.2.102/24"
		serverIP, err := libnet.CidrToIP(serverCIDR)
		Expect(err).ToNot(HaveOccurred())

		_ = createAlpineVMIStaticIPOnNode(nodeName, macvtapNetworkName, "eth0", serverCIDR, &chosenMAC)
		clientVMI := createAlpineVMIStaticIPOnNode(nodeName, macvtapNetworkName, "eth0", "192.0.2.101/24", nil)

		Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())
	})

	Context("VMI migration", func() {
		var clientVMI *v1.VirtualMachineInstance

		BeforeEach(checks.SkipIfMigrationIsNotPossible)

		BeforeEach(func() {
			macAddressHW, err := GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			macAddress := macAddressHW.String()
			clientVMI, err = createAlpineVMIRandomNode(macvtapNetworkName, macAddress)
			Expect(err).NotTo(HaveOccurred(), "must succeed creating a VMI on a random node")
		})

		It("should be successful when the VMI MAC address is defined in its spec", func() {
			By("starting the migration")
			migration := libmigration.New(clientVMI.Name, clientVMI.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(kubevirt.Client(), clientVMI, migration)
		})

		Context("with live traffic", func() {
			var serverVMI *v1.VirtualMachineInstance
			var serverVMIPodName string
			var serverIP string

			const macvtapIfaceIPReportTimeout = 4 * time.Minute

			BeforeEach(func() {
				macAddressHW, err := GenerateRandomMac()
				Expect(err).ToNot(HaveOccurred())
				macAddress := macAddressHW.String()

				serverVMI, err = createFedoraVMIRandomNode(macvtapNetworkName, macAddress)
				Expect(err).NotTo(HaveOccurred(), "must have succeeded creating a fedora VMI on a random node")
				Expect(serverVMI.Status.Interfaces).NotTo(BeEmpty(), "a migrate-able VMI must have network interfaces")
				serverVMIPodName = tests.GetVmPodName(kubevirt.Client(), serverVMI)

				serverIP, err = waitVMMacvtapIfaceIPReport(serverVMI, macAddress, macvtapIfaceIPReportTimeout)
				Expect(err).NotTo(HaveOccurred(), "should have managed to figure out the IP of the server VMI")
			})

			BeforeEach(func() {
				// TODO test also the IPv6 address (issue- https://github.com/kubevirt/kubevirt/issues/7506)
				libnet.SkipWhenClusterNotSupportIpv4()
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *before* migrating the VMI")
			})

			It("should keep connectivity after a migration", func() {
				const containerCompletionWaitTime = 60
				migration := libmigration.New(serverVMI.Name, serverVMI.GetNamespace())
				_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
				// In case of clientVMI and serverVMI running on the same node before migration, the serverVMI
				// will be reachable only when the original launcher pod terminates.
				Eventually(func() error {
					return waitForPodCompleted(serverVMI.Namespace, serverVMIPodName)
				}, containerCompletionWaitTime, time.Second).Should(Succeed(), fmt.Sprintf("all containers should complete in source virt-launcher pod: %s", serverVMIPodName))
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed(), "connectivity is expected *after* migrating the VMI")
			})
		})
	})
})

func createAlpineVMIStaticIPOnNode(nodeName string, networkName string, ifaceName string, ipCIDR string, mac *string) *v1.VirtualMachineInstance {
	var vmi *v1.VirtualMachineInstance
	if mac != nil {
		vmi = libvmi.NewAlpineWithTestTooling(
			libvmi.WithInterface(*libvmi.InterfaceWithMac(libvmi.InterfaceWithMacvtapBindingPlugin(networkName), *mac)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkName)),
			libvmi.WithNodeAffinityFor(nodeName),
		)
	} else {
		vmi = libvmi.NewAlpine(
			libvmi.WithInterface(*libvmi.InterfaceWithMacvtapBindingPlugin(networkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkName)),
			libvmi.WithNodeAffinityFor(nodeName),
		)
	}
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)
	// configure the client VMI
	Expect(libnet.AddIPAddress(vmi, ifaceName, ipCIDR)).To(Succeed())
	return vmi
}

func createAlpineVMIRandomNode(networkName string, mac string) (*v1.VirtualMachineInstance, error) {
	runningVMI := tests.RunVMIAndExpectLaunch(
		libvmi.NewAlpineWithTestTooling(
			libvmi.WithInterface(*libvmi.InterfaceWithMac(libvmi.InterfaceWithMacvtapBindingPlugin(networkName), mac)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkName)),
		),
		180,
	)
	err := console.LoginToAlpine(runningVMI)
	return runningVMI, err
}

func createFedoraVMIRandomNode(networkName string, mac string) (*v1.VirtualMachineInstance, error) {
	runningVMI := tests.RunVMIAndExpectLaunch(
		newFedoraVMIWithExplicitMacAndGuestAgent(networkName, mac),
		180,
	)
	err := console.LoginToFedora(runningVMI)
	return runningVMI, err
}

func newFedoraVMIWithExplicitMacAndGuestAgent(macvtapNetworkName string, mac string) *v1.VirtualMachineInstance {
	return libvmi.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithInterface(
			*libvmi.InterfaceWithMac(
				libvmi.InterfaceWithMacvtapBindingPlugin(macvtapNetworkName), mac)),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName, macvtapNetworkName)))
}

func waitForPodCompleted(podNamespace string, podName string) error {
	pod, err := kubevirt.Client().CoreV1().Pods(podNamespace).Get(context.Background(), podName, k8smetav1.GetOptions{})
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
	err := wait.PollImmediate(time.Second, timeout, func() (done bool, err error) {
		vmi, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
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
