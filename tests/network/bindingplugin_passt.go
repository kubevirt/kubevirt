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

	expect "github.com/google/goexpect"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG(" VirtualMachineInstance with passt network binding plugin", decorators.NetCustomBindingPlugins, Serial, func() {
	const passtNetAttDefName = "netbindingpasst"

	var err error

	BeforeEach(func() {
		const passtBindingName = "passt"

		passtComputeMemoryOverheadWhenAllPortsAreForwarded := resource.MustParse("500Mi")

		passtSidecarImage := libregistry.GetUtilityImageFromRegistry("network-passt-binding")

		err := config.WithNetBindingPlugin(passtBindingName, v1.InterfaceBindingPlugin{
			SidecarImage:                passtSidecarImage,
			NetworkAttachmentDefinition: passtNetAttDefName,
			Migration:                   &v1.InterfaceBindingMigration{},
			ComputeResourceOverhead: &v1.ResourceRequirementsWithoutClaims{
				Requests: map[k8sv1.ResourceName]resource.Quantity{
					k8sv1.ResourceMemory: passtComputeMemoryOverheadWhenAllPortsAreForwarded,
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		netAttachDef := libnet.NewPasstNetAttachDef(passtNetAttDefName)
		_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should apply the interface configuration", func() {
		const testMACAddr = "02:02:02:02:02:02"
		const testPCIAddr = "0000:01:00.0"
		passtIface := libvmi.InterfaceWithPasstBindingPlugin()
		passtIface.Ports = []v1.Port{{Port: 1234, Protocol: "TCP"}}
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

	Context("TCP without port specification", func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const highTCPPort = 8080

		BeforeEach(func() {
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithPasstInterfaceWithPort(),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
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

	Context("TCP with port specification", func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const highTCPPort = 8080

		BeforeEach(func() {
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithPasstInterfaceWithPort(),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			ports := []v1.Port{{Name: "http", Port: highTCPPort, Protocol: "TCP"}}
			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin(ports...)),
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

	Context("TCP with low port specification", func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const lowTCPPort = 80

		BeforeEach(func() {
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithPasstInterfaceWithPort(),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			ports := []v1.Port{{Name: "http", Port: lowTCPPort, Protocol: "TCP"}}
			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin(ports...)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitUntilVMIsReady(console.LoginToAlpine, clientVMI, serverVMI)

			vmnetserver.StartTCPServer(serverVMI, lowTCPPort, console.LoginToAlpine)
			By("starting a TCP server on a port not specified on the VM spec")
			vmnetserver.StartTCPServer(serverVMI, lowTCPPort+1, console.LoginToAlpine)
		})

		DescribeTable("connectivity", func(ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
			Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())

			By("Connecting from the client VM")
			Expect(console.RunCommand(clientVMI, connectToServerCmd(serverIP, lowTCPPort), 30*time.Second)).To(Succeed())

			By("Connecting from the client VM to a port not specified on the VM spec")
			Expect(console.RunCommand(clientVMI, connectToServerCmd(serverIP, lowTCPPort+1), 30)).NotTo(Succeed())
		},
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol),
		)
	})

	Context("UDP", func() {
		var clientVMI *v1.VirtualMachineInstance
		var serverVMI *v1.VirtualMachineInstance

		const udpPort = 1700

		BeforeEach(func() {
			var ports = []v1.Port{{Port: udpPort, Protocol: "UDP"}}

			By("Starting server VMI")
			serverVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin(ports...)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting client VMI")
			clientVMI = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitUntilVMIsReady(console.LoginToAlpine, serverVMI, clientVMI)
		})
		DescribeTable("connectivity", func(ipFamily k8sv1.IPFamily) {
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
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol),
		)
	})

	Context("egress connectivity", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithPasstInterfaceWithPort(),
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

	Context("migration", func() {
		var migrateVMI *v1.VirtualMachineInstance
		var anotherVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("Starting a VMI")
			migrateVMI = startPasstVMI()

			By("Starting another VMI")
			anotherVMI = startPasstVMI()

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

			By("Verify the VMI new IP is propogated to the status")
			var migrateVmiAfterMigIP string
			Eventually(func() string {
				migrateVMI, err = kubevirt.Client().VirtualMachineInstance(migrateVMI.Namespace).Get(context.Background(), migrateVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "should have been able to retrive the VMI instance")
				migrateVmiAfterMigIP = libnet.GetVmiPrimaryIPByFamily(migrateVMI, ipFamily)
				return migrateVmiAfterMigIP
			}, 30*time.Second).ShouldNot(Equal(migrateVmiBeforeMigIP), "the VMI status should get a new IP after migration")

			By("Verify the VMIs can ping each other after migration")
			Expect(libnet.PingFromVMConsole(migrateVMI, anotherVmiIP)).To(Succeed())
			Expect(libnet.PingFromVMConsole(anotherVMI, migrateVmiAfterMigIP)).To(Succeed())
		},
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol),
		)
	})
}))

func assertSourcePodContainersTerminate(labelSelector, fieldSelector string, vmi *v1.VirtualMachineInstance) bool {
	return Eventually(func() k8sv1.PodPhase {
		pods, err := kubevirt.Client().CoreV1().Pods(vmi.Namespace).List(context.Background(),
			metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))

		return pods.Items[0].Status.Phase
	}, 30*time.Second).Should(Equal(k8sv1.PodSucceeded))
}

func startPasstVMI() *v1.VirtualMachineInstance {
	networkData, err := cloudinit.NewNetworkData(
		cloudinit.WithEthernet("eth0",
			cloudinit.WithDHCP4Enabled(),
			cloudinit.WithDHCP6Enabled(),
		),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	vmi := libvmifact.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
	)
	vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return vmi
}

func waitUntilVMIsReady(loginTo console.LoginToFunction, vmis ...*v1.VirtualMachineInstance) {
	for idx, vmi := range vmis {
		*vmis[idx] = *libwait.WaitUntilVMIReady(
			vmi,
			loginTo,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(180),
		)
	}
}

func connectToServerCmd(serverIP string, port int) string {
	return fmt.Sprintf("echo test | nc %s %d -i 1 -w 1 1> /dev/null", serverIP, port)
}

func startAndVerifyUDPClient(vmi *v1.VirtualMachineInstance, serverIP string, serverPort int, ipFamily k8sv1.IPFamily) error {
	var inetSuffix string
	if ipFamily == k8sv1.IPv6Protocol {
		inetSuffix = "6"
	}

	createClientScript := fmt.Sprintf(`cat >udp_client.py <<EOL
import socket
try:
  client = socket.socket(socket.AF_INET%s, socket.SOCK_DGRAM);
  client.sendto("Hello Server".encode(), ("%s",%d));
  client.settimeout(5);
  response = client.recv(1024);
  print(response.decode("utf-8"));

except socket.timeout:
    client.close();
EOL`, inetSuffix, serverIP, serverPort)
	runClient := "python3 udp_client.py"
	return console.ExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: fmt.Sprintf("%s\n", createClientScript)},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: fmt.Sprintf("%s\n", runClient)},
		&expect.BExp{R: console.RetValue("Hello Client")},
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: console.ShellSuccess},
	}, 60*time.Second)
}
