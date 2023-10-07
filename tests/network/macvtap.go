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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	macvtapNetworkConfNAD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s", "annotations": {"k8s.v1.cni.cncf.io/resourceName": "macvtap.network.kubevirt.io/%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"%s\", \"type\": \"macvtap\"}"}}`
)

var _ = SIGDescribe("Macvtap", decorators.Macvtap, func() {
	var virtClient kubecli.KubevirtClient
	var macvtapLowerDevice string
	var macvtapNetworkName string

	createMacvtapNetworkAttachmentDefinition := func(namespace, networkName, macvtapLowerDevice string) error {
		macvtapNad := fmt.Sprintf(macvtapNetworkConfNAD, networkName, namespace, macvtapLowerDevice, networkName)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, macvtapNad)
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		macvtapLowerDevice = "eth0"
		macvtapNetworkName = "net1"
	})

	BeforeEach(func() {
		Expect(createMacvtapNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil), macvtapNetworkName, macvtapLowerDevice)).
			To(Succeed(), "A macvtap network named %s should be provisioned", macvtapNetworkName)
	})

	newAlpineVMIWithMacvtapNetwork := func(macvtapNetworkName string) *v1.VirtualMachineInstance {
		return libvmi.NewAlpine(
			libvmi.WithInterface(
				*v1.DefaultMacvtapNetworkInterface(macvtapNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName, macvtapNetworkName)))
	}

	newAlpineVMIWithExplicitMac := func(macvtapNetworkName string, mac string) *v1.VirtualMachineInstance {
		return libvmi.NewAlpineWithTestTooling(
			libvmi.WithInterface(
				*libvmi.InterfaceWithMac(
					v1.DefaultMacvtapNetworkInterface(macvtapNetworkName), mac)),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName, macvtapNetworkName)))
	}

	newFedoraVMIWithExplicitMacAndGuestAgent := func(macvtapNetworkName string, mac string) *v1.VirtualMachineInstance {
		return libvmi.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithInterface(
				*libvmi.InterfaceWithMac(
					v1.DefaultMacvtapNetworkInterface(macvtapNetworkName), mac)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetworkName, macvtapNetworkName)))
	}

	createAlpineVMIStaticIPOnNode := func(nodeName string, networkName string, ifaceName string, ipCIDR string, mac *string) *v1.VirtualMachineInstance {
		var vmi *v1.VirtualMachineInstance
		if mac != nil {
			vmi = newAlpineVMIWithExplicitMac(networkName, *mac)
		} else {
			vmi = newAlpineVMIWithMacvtapNetwork(networkName)
		}
		vmi = libwait.WaitUntilVMIReady(
			tests.CreateVmiOnNode(vmi, nodeName),
			console.LoginToAlpine)
		// configure the client VMI
		Expect(configInterface(vmi, ifaceName, ipCIDR)).To(Succeed())
		return vmi
	}

	createAlpineVMIRandomNode := func(networkName string, mac string) (*v1.VirtualMachineInstance, error) {
		runningVMI := tests.RunVMIAndExpectLaunch(
			newAlpineVMIWithExplicitMac(networkName, mac),
			180,
		)
		err := console.LoginToAlpine(runningVMI)
		return runningVMI, err
	}

	createFedoraVMIRandomNode := func(networkName string, mac string) (*v1.VirtualMachineInstance, error) {
		runningVMI := tests.RunVMIAndExpectLaunch(
			newFedoraVMIWithExplicitMacAndGuestAgent(networkName, mac),
			180,
		)
		err := console.LoginToFedora(runningVMI)
		return runningVMI, err
	}

	Context("a virtual machine with one macvtap interface, with a custom MAC address", func() {
		var serverVMI *v1.VirtualMachineInstance
		var chosenMAC string
		var nodeList *k8sv1.NodeList
		var nodeName string
		var serverIP string

		BeforeEach(func() {
			nodeList = libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodeList.Items).NotTo(BeEmpty(), "schedulable kubernetes nodes must be present")
			nodeName = nodeList.Items[0].Name
			chosenMACHW, err := GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			chosenMAC = chosenMACHW.String()
			serverCIDR := "192.0.2.102/24"

			serverIP, err = libnet.CidrToIP(serverCIDR)
			Expect(err).ToNot(HaveOccurred())

			serverVMI = createAlpineVMIStaticIPOnNode(nodeName, macvtapNetworkName, "eth0", serverCIDR, &chosenMAC)
		})

		It("should have the specified MAC address reported back via the API", func() {
			Expect(serverVMI.Status.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(serverVMI.Status.Interfaces[0].MAC).To(Equal(chosenMAC), "the expected MAC address should be set in the VMI")
		})

		Context("and another virtual machine connected to the same network", func() {
			var clientVMI *v1.VirtualMachineInstance
			BeforeEach(func() {
				clientVMI = createAlpineVMIStaticIPOnNode(nodeName, macvtapNetworkName, "eth0", "192.0.2.101/24", nil)
			})
			It("can communicate with the virtual machine in the same network", func() {
				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())
			})
		})
	})

	Context("VMI migration", func() {
		var clientVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			checks.SkipIfMigrationIsNotPossible()
		})

		BeforeEach(func() {
			macAddressHW, err := GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			macAddress := macAddressHW.String()
			clientVMI, err = createAlpineVMIRandomNode(macvtapNetworkName, macAddress)
			Expect(err).NotTo(HaveOccurred(), "must succeed creating a VMI on a random node")
		})

		It("should be successful when the VMI MAC address is defined in its spec", func() {
			By("starting the migration")
			migration := tests.NewRandomMigration(clientVMI.Name, clientVMI.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, clientVMI, migration)
		})

		Context("with live traffic", func() {
			var serverVMI *v1.VirtualMachineInstance
			var serverVMIPodName string
			var serverIP string

			macvtapIfaceIPReportTimeout := 4 * time.Minute

			waitVMMacvtapIfaceIPReport := func(vmi *v1.VirtualMachineInstance, macAddress string, timeout time.Duration) (string, error) {
				var vmiIP string
				err := wait.PollImmediate(time.Second, timeout, func() (done bool, err error) {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{})
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

			waitForPodCompleted := func(podNamespace string, podName string) error {
				pod, err := virtClient.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, k8smetav1.GetOptions{})
				if err != nil {
					return err
				}
				if pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
					return nil
				}
				return fmt.Errorf("pod hasn't completed, current Phase: %s", pod.Status.Phase)
			}

			BeforeEach(func() {
				macAddressHW, err := GenerateRandomMac()
				Expect(err).ToNot(HaveOccurred())
				macAddress := macAddressHW.String()

				serverVMI, err = createFedoraVMIRandomNode(macvtapNetworkName, macAddress)
				Expect(err).NotTo(HaveOccurred(), "must have succeeded creating a fedora VMI on a random node")
				Expect(serverVMI.Status.Interfaces).NotTo(BeEmpty(), "a migrate-able VMI must have network interfaces")
				serverVMIPodName = tests.GetVmPodName(virtClient, serverVMI)

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
				migration := tests.NewRandomMigration(serverVMI.Name, serverVMI.GetNamespace())
				_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
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
