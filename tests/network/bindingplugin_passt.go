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

	expect "github.com/google/goexpect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkvconfig"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("[Serial] VirtualMachineInstance with passt network binding plugin", decorators.NetCustomBindingPlugins, Serial, func() {
	var err error

	BeforeEach(func() {
		tests.EnableFeatureGate(virtconfig.NetworkBindingPlugingsGate)
	})

	BeforeEach(func() {
		const passtBindingName = "passt"
		const passtSidecarImage = "registry:5000/kubevirt/network-passt-binding:devel"

		err := libkvconfig.WithNetBindingPlugin(passtBindingName, v1.InterfaceBindingPlugin{
			SidecarImage:                passtSidecarImage,
			NetworkAttachmentDefinition: libnet.PasstNetAttDef,
			Migration:                   &v1.InterfaceBindingMigration{Method: v1.LinkRefresh},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		Expect(libnet.CreatePasstNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil))).To(Succeed())
	})

	It("should apply the interface configuration", func() {
		const testMACAddr = "02:02:02:02:02:02"
		const testPCIAddr = "0000:01:00.0"
		passtIface := libvmi.InterfaceWithPasstBindingPlugin()
		passtIface.Ports = []v1.Port{{Port: 1234, Protocol: "TCP"}}
		passtIface.MacAddress = testMACAddr
		passtIface.PciAddress = testPCIAddr
		vmi := libvmi.NewAlpineWithTestTooling(
			libvmi.WithInterface(passtIface),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitUntilVMIReady(
			vmi,
			console.LoginToAlpine,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(180),
		)

		Expect(vmi.Status.Interfaces).To(HaveLen(1))
		Expect(vmi.Status.Interfaces[0].IPs).NotTo(BeEmpty())
		Expect(vmi.Status.Interfaces[0].IP).NotTo(BeEmpty())
		Expect(vmi.Status.Interfaces[0].MAC).To(Equal(testMACAddr))

		guestIfaceName := vmi.Status.Interfaces[0].InterfaceName
		cmd := fmt.Sprintf("ls /sys/bus/pci/devices/%s/virtio0/net/%s", testPCIAddr, guestIfaceName)
		Expect(console.RunCommand(vmi, cmd, time.Second*5)).To(Succeed())
	})

	Context("should allow regular network connection", func() {
		Context("should have client server connectivity", func() {
			var clientVMI *v1.VirtualMachineInstance
			var serverVMI *v1.VirtualMachineInstance

			startServerVMI := func(ports []v1.Port) {
				passtIface := libvmi.InterfaceWithPasstBindingPlugin(ports...)
				serverVMI = libvmi.NewAlpineWithTestTooling(
					libvmi.WithInterface(passtIface),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)

				serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serverVMI = libwait.WaitUntilVMIReady(
					serverVMI,
					console.LoginToAlpine,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				)
			}

			Context("TCP", func() {
				checkConnectionToServer := func(serverIP string, port int, expectSuccess bool) []expect.Batcher {
					expectResult := console.ShellFail
					if expectSuccess {
						expectResult = console.ShellSuccess
					}

					clientCommand := fmt.Sprintf("echo test | nc %s %d -i 1 -w 1 1> /dev/null\n", serverIP, port)

					return []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: clientCommand},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: expectResult},
					}
				}

				verifyClientServerConnectivity := func(clientVMI *v1.VirtualMachineInstance, serverVMI *v1.VirtualMachineInstance, tcpPort int, ipFamily k8sv1.IPFamily) error {
					serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
					err := libnet.PingFromVMConsole(clientVMI, serverIP)
					if err != nil {
						return err
					}

					By("Connecting from the client VM")
					err = console.SafeExpectBatch(clientVMI, checkConnectionToServer(serverIP, tcpPort, true), 30)
					if err != nil {
						return err
					}

					return nil
				}

				startClientVMI := func() {
					clientVMI = libvmi.NewAlpineWithTestTooling(
						libvmi.WithPasstInterfaceWithPort(),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
					)

					clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					clientVMI = libwait.WaitUntilVMIReady(clientVMI,
						console.LoginToAlpine,
						libwait.WithFailOnWarnings(false),
						libwait.WithTimeout(180),
					)
				}
				DescribeTable("Client server connectivity", func(ports []v1.Port, tcpPort int, ipFamily k8sv1.IPFamily) {
					libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

					By("starting a client VMI")
					startClientVMI()

					By("starting a server VMI")
					startServerVMI(ports)

					By("starting a TCP server")
					tests.StartTCPServer(serverVMI, tcpPort, console.LoginToAlpine)

					Expect(verifyClientServerConnectivity(clientVMI, serverVMI, tcpPort, ipFamily)).To(Succeed())

					if len(ports) != 0 {
						By("starting a TCP server on a port not specified on the VM spec")
						vmPort := int(ports[0].Port)
						serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)

						tests.StartTCPServer(serverVMI, vmPort+1, console.LoginToAlpine)

						By("Connecting from the client VM to a port not specified on the VM spec")
						Expect(console.SafeExpectBatch(clientVMI, checkConnectionToServer(serverIP, tcpPort+1, true), 30)).To(Not(Succeed()))
					}
				},
					Entry("with a specific port number [IPv4]", []v1.Port{{Name: "http", Port: 8080, Protocol: "TCP"}}, 8080, k8sv1.IPv4Protocol),
					Entry("with a specific lower port number [IPv4]", []v1.Port{{Name: "http", Port: 80, Protocol: "TCP"}}, 80, k8sv1.IPv4Protocol),
					Entry("without a specific port number [IPv4]", []v1.Port{}, 8080, k8sv1.IPv4Protocol),
					Entry("with a specific port number [IPv6]", []v1.Port{{Name: "http", Port: 8080, Protocol: "TCP"}}, 8080, k8sv1.IPv6Protocol),
					Entry("without a specific port number [IPv6]", []v1.Port{}, 8080, k8sv1.IPv6Protocol),
				)
			})

			Context("UDP", func() {
				startAndVerifyUDPClient := func(vmi *v1.VirtualMachineInstance, serverIP string, serverPort int, ipFamily k8sv1.IPFamily) error {
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
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.ShellSuccess},
					}, 60*time.Second)
				}
				DescribeTable("Client server connectivity", func(ipFamily k8sv1.IPFamily) {
					libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

					const SERVER_PORT = 1700

					By("Starting server VMI")
					startServerVMI([]v1.Port{{Port: SERVER_PORT, Protocol: "UDP"}})
					serverVMI = libwait.WaitUntilVMIReady(serverVMI,
						console.LoginToAlpine,
						libwait.WithFailOnWarnings(false),
						libwait.WithTimeout(180),
					)

					By("Starting a UDP server")
					tests.StartPythonUDPServer(serverVMI, SERVER_PORT, ipFamily)

					By("Starting client VMI")
					clientVMI = libvmi.NewAlpineWithTestTooling(
						libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
					)
					clientVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					clientVMI = libwait.WaitUntilVMIReady(clientVMI,
						console.LoginToAlpine,
						libwait.WithFailOnWarnings(false),
						libwait.WithTimeout(180),
					)

					By("Starting and verifying UDP client")
					// Due to a passt bug, at least one UDPv6 message has to be sent from a machine before it can receive UDPv6 messages
					// Tracking bug - https://bugs.passt.top/show_bug.cgi?id=16
					if ipFamily == k8sv1.IPv6Protocol {
						clientIP := libnet.GetVmiPrimaryIPByFamily(clientVMI, ipFamily)
						Expect(libnet.PingFromVMConsole(serverVMI, clientIP)).To(Succeed())
					}
					serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
					Expect(startAndVerifyUDPClient(clientVMI, serverIP, SERVER_PORT, ipFamily)).To(Succeed())
				},
					Entry("[IPv4]", k8sv1.IPv4Protocol),
					Entry("[IPv6]", k8sv1.IPv6Protocol),
				)
			})
		})

		It("[outside_connectivity]should be able to reach the outside world [IPv4]", func() {
			libnet.SkipWhenClusterNotSupportIpv4()
			ipv4Address := "8.8.8.8"
			if flags.IPV4ConnectivityCheckAddress != "" {
				ipv4Address = flags.IPV4ConnectivityCheckAddress
			}
			dns := "google.com"
			if flags.ConnectivityCheckDNS != "" {
				dns = flags.ConnectivityCheckDNS
			}

			vmi := libvmi.NewAlpineWithTestTooling(
				libvmi.WithPasstInterfaceWithPort(),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi,
				console.LoginToAlpine,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)

			By("Checking ping (IPv4)")
			Expect(libnet.PingFromVMConsole(vmi, ipv4Address, "-c 5", "-w 15")).To(Succeed())
			Expect(libnet.PingFromVMConsole(vmi, dns, "-c 5", "-w 15")).To(Succeed())
		})

		It("[outside_connectivity]should be able to reach the outside world [IPv6]", func() {
			libnet.SkipWhenClusterNotSupportIpv6()
			// Cluster nodes subnet (docker network gateway)
			// Docker network subnet cidr definition:
			// https://github.com/kubevirt/project-infra/blob/master/github/ci/shared-deployments/files/docker-daemon-mirror.conf#L5
			ipv6Address := "2001:db8:1::1"
			if flags.IPV6ConnectivityCheckAddress != "" {
				ipv6Address = flags.IPV6ConnectivityCheckAddress
			}

			vmi := libvmi.NewAlpineWithTestTooling(
				libvmi.WithPasstInterfaceWithPort(),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithAnnotation("kubevirt.io/libvirt-log-filters", "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"),
			)
			Expect(err).ToNot(HaveOccurred())
			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi,
				console.LoginToAlpine,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)

			By("Checking ping (IPv6) from VM to cluster nodes gateway")
			Expect(libnet.PingFromVMConsole(vmi, ipv6Address)).To(Succeed())
		})

		DescribeTable("Connectivity should be preserved after migration", func(ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			By("Starting a VMI")
			migrateVMI := startPasstVMI(libvmi.NewFedora, console.LoginToFedora)
			beforeMigNodeName := migrateVMI.Status.NodeName

			By("Starting another VMI")
			anotherVMI := startPasstVMI(libvmi.NewAlpine, console.LoginToAlpine)

			By("Verify the VMIs can ping each other")
			migrateVmiBeforeMigIP := libnet.GetVmiPrimaryIPByFamily(migrateVMI, ipFamily)
			anotherVmiIP := libnet.GetVmiPrimaryIPByFamily(anotherVMI, ipFamily)
			Expect(libnet.PingFromVMConsole(migrateVMI, anotherVmiIP)).To(Succeed())
			Expect(libnet.PingFromVMConsole(anotherVMI, migrateVmiBeforeMigIP)).To(Succeed())

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
				migrateVMI, err = kubevirt.Client().VirtualMachineInstance(migrateVMI.Namespace).Get(context.Background(), migrateVMI.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "should have been able to retrive the VMI instance")
				migrateVmiAfterMigIP = libnet.GetVmiPrimaryIPByFamily(migrateVMI, ipFamily)
				return migrateVmiAfterMigIP
			}, 30*time.Second).ShouldNot(Equal(migrateVmiBeforeMigIP), "the VMI status should get a new IP after migration")

			By("Verify the guest renewed the IP")
			/// We renew the IP to avoid issues such as
			// another pod in the cluster gets the IP of the old pod
			// a user wants the internal and exteranl IP on the VM to be same
			ipValidation := fmt.Sprintf("ip addr show eth0 | grep %s", migrateVmiAfterMigIP)
			Eventually(func() error {
				return console.RunCommand(migrateVMI, ipValidation, time.Second*5)
			}, 30*time.Second).Should(Succeed())

			By("Verify the VMIs can ping each other after migration")
			Expect(libnet.PingFromVMConsole(migrateVMI, anotherVmiIP)).To(Succeed())
			Expect(libnet.PingFromVMConsole(anotherVMI, migrateVmiAfterMigIP)).To(Succeed())
		},
			Entry("[IPv4]", k8sv1.IPv4Protocol),
			Entry("[IPv6]", k8sv1.IPv6Protocol),
		)
	})
})

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

func startPasstVMI(vmiBuilder func(opts ...libvmi.Option) *v1.VirtualMachineInstance, loginTo console.LoginToFunction) *v1.VirtualMachineInstance {
	networkData, err := libnet.NewNetworkData(
		libnet.WithEthernet("eth0",
			libnet.WithDHCP4Enabled(),
			libnet.WithDHCP6Enabled(),
		),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	vmi := libvmi.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloudNetworkData(networkData),
	)
	vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	vmi = libwait.WaitUntilVMIReady(vmi,
		console.LoginToFedora,
		libwait.WithFailOnWarnings(false),
		libwait.WithTimeout(180),
	)
	return vmi
}
