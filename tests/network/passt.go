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

	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG(" VirtualMachineInstance with passt network binding", Serial, func() {

	var err error
	const (
		port1234 = 1234
		tcp      = "TCP"
		udp      = "UDP"
	)

	BeforeEach(OncePerOrdered, func() {
		config.EnableFeatureGate(featuregate.PasstBinding)
	})

	It("should apply the interface configuration", func() {
		const testMACAddr = "02:02:02:02:02:02"
		const testPCIAddr = "0000:01:00.0"
		passtIface := passtBindingInterfaceWithPort(tcp, port1234)
		passtIface.MacAddress = testMACAddr
		passtIface.PciAddress = testPCIAddr
		vmi := libvmifact.NewAlpineWithTestTooling(
			libvmi.WithInterface(passtIface),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		waitUntilVMIsReady(console.LoginToAlpine, vmi)

		Expect(vmi.Status.Interfaces).To(HaveLen(1))
		Expect(vmi.Status.Interfaces[0].IPs).NotTo(BeEmpty())
		Expect(vmi.Status.Interfaces[0].IP).NotTo(BeEmpty())
		Expect(vmi.Status.Interfaces[0].MAC).To(Equal(testMACAddr))

		guestIfaceName := vmi.Status.Interfaces[0].InterfaceName
		cmd := fmt.Sprintf("ls /sys/bus/pci/devices/%s/virtio0/net/%s", testPCIAddr, guestIfaceName)
		Expect(console.RunCommand(vmi, cmd, time.Second*5)).To(Succeed())
	})

	Context("TCP without port specification", Ordered, decorators.OncePerOrderedCleanup, func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const highTCPPort = 8080

		BeforeAll(func() {
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtBindingInterfaceWithPort(tcp, port1234)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding(v1.DefaultPodNetwork().Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitUntilVMIsReady(console.LoginToAlpine, clientVMI, serverVMI)

			vmnetserver.StartTCPServer(serverVMI, highTCPPort, console.LoginToAlpine)
		})
		DescribeTable("connectivity", func(ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
			Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())
			Expect(console.RunCommand(clientVMI, connectToServerCmd(serverIP, highTCPPort), 30*time.Second)).To(Succeed())
		},
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol),
		)
	})

	Context("TCP with port specification", Ordered, decorators.OncePerOrderedCleanup, func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const highTCPPort = 8080

		BeforeAll(func() {
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtBindingInterfaceWithPort(tcp, port1234)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			//ports := []v1.Port{{Name: "http", Port: highTCPPort, Protocol: "TCP"}}
			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtBindingInterfaceWithPort(tcp, highTCPPort)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitUntilVMIsReady(console.LoginToAlpine, clientVMI, serverVMI)

			vmnetserver.StartTCPServer(serverVMI, highTCPPort, console.LoginToAlpine)
			By("starting a TCP server on a port not specified on the VM spec")
			vmnetserver.StartTCPServer(serverVMI, highTCPPort+1, console.LoginToAlpine)

		})
		DescribeTable("connectivity", func(ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
			Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())

			By("Connecting from the client VM")
			Expect(console.RunCommand(clientVMI, connectToServerCmd(serverIP, highTCPPort), 30*time.Second)).To(Succeed())

			By("Connecting from the client VM to a port not specified on the VM spec")
			Expect(console.RunCommand(clientVMI, connectToServerCmd(serverIP, highTCPPort+1), 30)).NotTo(Succeed())
		},
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol),
		)
	})

	Context("TCP with low port specification", Ordered, decorators.OncePerOrderedCleanup, func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const lowHTTPPort = 80

		BeforeAll(func() {
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtBindingInterfaceWithPort(tcp, port1234)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtBindingInterfaceWithPort(tcp, lowHTTPPort)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitUntilVMIsReady(console.LoginToAlpine, clientVMI, serverVMI)

			vmnetserver.StartTCPServer(serverVMI, lowHTTPPort, console.LoginToAlpine)
			By("starting a TCP server on a port not specified on the VM spec")
			vmnetserver.StartTCPServer(serverVMI, lowHTTPPort+1, console.LoginToAlpine)
		})

		DescribeTable("connectivity", func(ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
			Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())

			By("Connecting from the client VM")
			Expect(console.RunCommand(clientVMI, connectToServerCmd(serverIP, lowHTTPPort), 30*time.Second)).To(Succeed())

			By("Connecting from the client VM to a port not specified on the VM spec")
			Expect(console.RunCommand(clientVMI, connectToServerCmd(serverIP, lowHTTPPort+1), 30)).NotTo(Succeed())
		},
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol),
		)
	})

	Context("UDP", Ordered, decorators.OncePerOrderedCleanup, func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const udpPortForIPv4 = 1700
		const udpPortForIPv6 = 1701

		BeforeAll(func() {

			By("Starting server VMI")
			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtBindingInterfaceWithPort(udp, udpPortForIPv4, udpPortForIPv6)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting client VMI")
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding(v1.DefaultPodNetwork().Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitUntilVMIsReady(console.LoginToAlpine, serverVMI, clientVMI)
		})

		DescribeTable("connectivity", func(udpPort int, ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			By("Starting a UDP server")
			vmnetserver.StartPythonUDPServer(serverVMI, udpPort, ipFamily)

			By("Starting and verifying UDP client")
			// Due to a passt bug, at least one UDPv6 message has to be sent from a machine before it can receive UDPv6 messages
			// Tracking bug - https://bugs.passt.top/show_bug.cgi?id=16
			if ipFamily == k8sv1.IPv6Protocol {
				clientIP := libnet.GetVmiPrimaryIPByFamily(clientVMI, ipFamily)
				Expect(libnet.PingFromVMConsole(serverVMI, clientIP)).To(Succeed())
			}
			serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
			Expect(startAndVerifyUDPClient(clientVMI, serverIP, udpPort, ipFamily)).To(Succeed())
		},
			Entry("[IPv4]", udpPortForIPv4, k8sv1.IPv4Protocol),
			Entry("[IPv6]", udpPortForIPv6, k8sv1.IPv6Protocol),
		)
	})

	Context("egress connectivity", Ordered, decorators.OncePerOrderedCleanup, func() {
		var vmi *v1.VirtualMachineInstance

		BeforeAll(func() {
			vmi = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtBindingInterfaceWithPort(tcp, port1234)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitUntilVMIsReady(console.LoginToAlpine, vmi)
		})

		It("should be able to reach the outside world [IPv4]", Label("RequiresOutsideConnectivity"), func() {
			libnet.SkipWhenClusterNotSupportIpv4()
			ipv4Address := "8.8.8.8"
			if flags.IPV4ConnectivityCheckAddress != "" {
				ipv4Address = flags.IPV4ConnectivityCheckAddress
			}
			dns := "google.com"
			if flags.ConnectivityCheckDNS != "" {
				dns = flags.ConnectivityCheckDNS
			}

			By("Checking ping (IPv4)")
			Expect(libnet.PingFromVMConsole(vmi, ipv4Address, "-c 5", "-w 15")).To(Succeed())
			Expect(libnet.PingFromVMConsole(vmi, dns, "-c 5", "-w 15")).To(Succeed())
		})

		It("should be able to reach the outside world", Label("RequiresOutsideConnectivity", "IPv6"), func() {
			libnet.SkipWhenClusterNotSupportIpv6()
			// Cluster nodes subnet (docker network gateway)
			// Docker network subnet cidr definition:
			// https://github.com/kubevirt/project-infra/blob/master/github/ci/shared-deployments/files/docker-daemon-mirror.conf#L5
			ipv6Address := "2001:db8:1::1"
			if flags.IPV6ConnectivityCheckAddress != "" {
				ipv6Address = flags.IPV6ConnectivityCheckAddress
			}

			By("Checking ping (IPv6) from VM to cluster nodes gateway")
			Expect(libnet.PingFromVMConsole(vmi, ipv6Address)).To(Succeed())
		})
	})

	Context("migration", Ordered, decorators.OncePerOrderedCleanup, func() {
		var migrateVMI *v1.VirtualMachineInstance
		var anotherVMI *v1.VirtualMachineInstance

		BeforeAll(func() {
			By("Starting a VMI")
			migrateVMI = startPasstBindingVMI()

			By("Starting another VMI")
			anotherVMI = startPasstBindingVMI()

			waitUntilVMIsReady(console.LoginToFedora, migrateVMI, anotherVMI)
		})

		DescribeTable("connectivity should be preserved", func(ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			By("Verify the VMIs can ping each other")
			migrateVmiBeforeMigIP := libnet.GetVmiPrimaryIPByFamily(migrateVMI, ipFamily)
			anotherVmiIP := libnet.GetVmiPrimaryIPByFamily(anotherVMI, ipFamily)
			Expect(libnet.PingFromVMConsole(migrateVMI, anotherVmiIP)).To(Succeed())
			Expect(libnet.PingFromVMConsole(anotherVMI, migrateVmiBeforeMigIP)).To(Succeed())

			beforeMigNodeName := migrateVMI.Status.NodeName

			By("Perform migration")
			migration := libmigration.New(migrateVMI.Name, migrateVMI.Namespace)
			migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)
			migrateVMI = libmigration.ConfirmVMIPostMigration(kubevirt.Client(), migrateVMI, migrationUID)

			By("Verify all the containers in the source pod were terminated")
			labelSelector := fmt.Sprintf("%s=%s", v1.CreatedByLabel, string(migrateVMI.GetUID()))
			fieldSelector := fmt.Sprintf("spec.nodeName==%s", beforeMigNodeName)

			assertSourcePodContainersTerminate(labelSelector, fieldSelector, migrateVMI)

			By("Verify the VMI new IP is propagated to the status")
			var migrateVmiAfterMigIP string
			Eventually(func() string {
				migrateVMI, err = kubevirt.Client().VirtualMachineInstance(migrateVMI.Namespace).Get(context.Background(), migrateVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "should have been able to retrieve the VMI instance")
				migrateVmiAfterMigIP = libnet.GetVmiPrimaryIPByFamily(migrateVMI, ipFamily)
				return migrateVmiAfterMigIP
			}, 30*time.Second).ShouldNot(Equal(migrateVmiBeforeMigIP), "the VMI status should get a new IP after migration")

			By("Verify the VMIs can ping each other after migration")
			Expect(libnet.PingFromVMConsole(migrateVMI, anotherVmiIP)).To(Succeed())
			Expect(libnet.PingFromVMConsole(anotherVMI, migrateVmiAfterMigIP)).To(Succeed())
		},
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol, decorators.Quarantine),
		)
	})
}))

func passtBindingInterfaceWithPort(protocol string, port ...int) v1.Interface {
	passtIface := libvmi.InterfaceDeviceWithPasstBinding(v1.DefaultPodNetwork().Name)
	var ports []v1.Port
	for _, p := range port {
		ports = append(ports, v1.Port{Port: int32(p), Protocol: protocol})
	}
	passtIface.Ports = ports
	return passtIface
}

func startPasstBindingVMI() *v1.VirtualMachineInstance {
	networkData, err := cloudinit.NewNetworkData(
		cloudinit.WithEthernet("eth0",
			cloudinit.WithDHCP4Enabled(),
			cloudinit.WithDHCP6Enabled(),
		),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	vmi := libvmifact.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding(v1.DefaultPodNetwork().Name)),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
	)
	vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return vmi
}
