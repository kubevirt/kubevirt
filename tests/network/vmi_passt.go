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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("Passt", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("VirtualMachineInstance with passt binding mechanism", func() {

		It("should report the IP to the status", func() {
			vmi := libvmi.NewCirros(
				withPasstInterfaceWithPort(),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			vmi = tests.WaitUntilVMIReady(vmi, console.LoginToCirros)

			Expect(vmi.Status.Interfaces).To(HaveLen(1))
			Expect(vmi.Status.Interfaces[0].IPs).NotTo(BeEmpty())
			Expect(vmi.Status.Interfaces[0].IP).NotTo(BeEmpty())
		})

		//Context("should allow regular network connection", func() {
		//	Context("should have client server connectivity", func() {
		//		var clientVMI *v1.VirtualMachineInstance
		//		var serverVMI *v1.VirtualMachineInstance
		//
		//		checkConnectionToServer := func(serverIP string, port int, expectSuccess bool) []expect.Batcher {
		//			expectResult := console.ShellFail
		//			if expectSuccess {
		//				expectResult = console.ShellSuccess
		//			}
		//
		//			clientCommand := fmt.Sprintf("echo test | nc %s %d -i 1 -w 1 1> /dev/null\n", serverIP, port)
		//
		//			return []expect.Batcher{
		//				&expect.BSnd{S: "\n"},
		//				&expect.BExp{R: console.PromptExpression},
		//				&expect.BSnd{S: clientCommand},
		//				&expect.BExp{R: console.PromptExpression},
		//				&expect.BSnd{S: tests.EchoLastReturnValue},
		//				&expect.BExp{R: expectResult},
		//			}
		//		}
		//
		//		verifyClientServerConnectivity := func(clientVMI *v1.VirtualMachineInstance, serverVMI *v1.VirtualMachineInstance, tcpPort int, ipFamily k8sv1.IPFamily) error {
		//			serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
		//			err := libnet.PingFromVMConsole(clientVMI, serverIP)
		//			if err != nil {
		//				return err
		//			}
		//
		//			By("Connecting from the client VM")
		//			err = console.SafeExpectBatch(clientVMI, checkConnectionToServer(serverIP, tcpPort, true), 30)
		//			if err != nil {
		//				return err
		//			}
		//
		//			return nil
		//		}
		//
		//		startServerVMI := func(ports []v1.Port) {
		//			serverVMI = libvmi.NewCirros(
		//				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding(ports...)),
		//				libvmi.WithNetwork(v1.DefaultPodNetwork()),
		//				withPasstExtendedResourceMemory(ports...),
		//			)
		//
		//			serverVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(serverVMI)
		//			Expect(err).ToNot(HaveOccurred())
		//			serverVMI = tests.WaitForSuccessfulVMIStartIgnoreWarnings(serverVMI)
		//			Expect(console.LoginToCirros(serverVMI)).To(Succeed())
		//		}
		//
		//		BeforeEach(func() {
		//			clientVMI = libvmi.NewCirros(
		//				withPasstInterfaceWithPort(),
		//				libvmi.WithNetwork(v1.DefaultPodNetwork()),
		//			)
		//
		//			clientVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(clientVMI)
		//			Expect(err).ToNot(HaveOccurred())
		//			clientVMI = tests.WaitForSuccessfulVMIStartIgnoreWarnings(clientVMI)
		//			Expect(console.LoginToCirros(clientVMI)).To(Succeed())
		//		})
		//
		//		DescribeTable("TCP", func(ports []v1.Port, tcpPort int, ipFamily k8sv1.IPFamily) {
		//			libnet.SkipWhenClusterNotSupportIPFamily(virtClient, ipFamily)
		//
		//			startServerVMI(ports)
		//
		//			By("starting a TCP server")
		//			tests.StartTCPServer(serverVMI, tcpPort, console.LoginToCirros)
		//
		//			Expect(verifyClientServerConnectivity(clientVMI, serverVMI, tcpPort, k8sv1.IPv4Protocol)).To(Succeed())
		//
		//			if len(ports) != 0 {
		//				By("starting a TCP server on a port not specified on the VM spec")
		//				vmPort := int(ports[0].Port)
		//				serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
		//
		//				tests.StartTCPServer(serverVMI, vmPort+1, console.LoginToCirros)
		//
		//				By("Connecting from the client VM to a port not specified on the VM spec")
		//				Expect(console.SafeExpectBatch(clientVMI, checkConnectionToServer(serverIP, tcpPort+1, true), 30)).To(Not(Succeed()))
		//			}
		//		},
		//			Entry("with a specific port number [IPv4]", []v1.Port{{Name: "http", Port: 8080, Protocol: "TCP"}}, 8080, k8sv1.IPv4Protocol),
		//			Entry("without a specific port number [IPv4]", []v1.Port{}, 8080, k8sv1.IPv4Protocol),
		//			Entry("with a specific port number [IPv6]", []v1.Port{{Name: "http", Port: 8080, Protocol: "TCP"}}, 8080, k8sv1.IPv6Protocol),
		//			Entry("without a specific port number [IPv6]", []v1.Port{}, 8080, k8sv1.IPv6Protocol),
		//		)
		//	})
		//
		//	It("[outside_connectivity]should be able to reach the outside world [IPv4]", func() {
		//		libnet.SkipWhenClusterNotSupportIpv4(virtClient)
		//		ipv4Address := "8.8.8.8"
		//		if flags.IPV4ConnectivityCheckAddress != "" {
		//			ipv4Address = flags.IPV4ConnectivityCheckAddress
		//		}
		//		dns := "google.com"
		//		if flags.ConnectivityCheckDNS != "" {
		//			dns = flags.ConnectivityCheckDNS
		//		}
		//
		//		vmi := libvmi.NewCirros(
		//			withPasstInterfaceWithPort(),
		//			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		//		)
		//		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		//		Expect(err).ToNot(HaveOccurred())
		//		vmi = tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
		//		Expect(console.LoginToCirros(vmi)).To(Succeed())
		//
		//		By("Checking ping (IPv4)")
		//		Expect(libnet.PingFromVMConsole(vmi, ipv4Address, "-c 5", "-w 15")).To(Succeed())
		//		Expect(libnet.PingFromVMConsole(vmi, dns, "-c 5", "-w 15")).To(Succeed())
		//	})
		//
		//	It("[outside_connectivity]should be able to reach the outside world [IPv6]", func() {
		//		libnet.SkipWhenClusterNotSupportIpv6(virtClient)
		//		// Cluster nodes subnet (docker network gateway)
		//		// Docker network subnet cidr definition:
		//		// https://github.com/kubevirt/project-infra/blob/master/github/ci/shared-deployments/files/docker-daemon-mirror.conf#L5
		//		ipv6Address := "2001:db8:1::1"
		//		if flags.IPV6ConnectivityCheckAddress != "" {
		//			ipv6Address = flags.IPV6ConnectivityCheckAddress
		//		}
		//
		//		vmi := libvmi.NewCirros(
		//			withPasstInterfaceWithPort(),
		//			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		//		)
		//		Expect(err).ToNot(HaveOccurred())
		//		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		//		Expect(err).ToNot(HaveOccurred())
		//		vmi = tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
		//		Expect(console.LoginToCirros(vmi)).To(Succeed())
		//
		//		By("Checking ping (IPv6) from VM to cluster nodes gateway")
		//		Expect(libnet.PingFromVMConsole(vmi, ipv6Address)).To(Succeed())
		//	})
		//})
	})
})

//func withPasstExtendedResourceMemory(ports ...v1.Port) libvmi.Option {
//	if len(ports) == 0 {
//		return libvmi.WithResourceMemory("2048M")
//	}
//	return func(vmi *v1.VirtualMachineInstance) {
//	}
//}

func withPasstInterfaceWithPort() libvmi.Option {
	return libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding([]v1.Port{{Port: 1234, Protocol: "TCP"}}...))
}
