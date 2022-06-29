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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"
	"io"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const skipIPv6Message = "port-forwarding over ipv6 is not supported yet. Tracking issue https://github.com/kubevirt/kubevirt/issues/7276"

var _ = SIGDescribe("Port-forward", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("VMI With masquerade binding", func() {
		var (
			localPort         int
			portForwardCmd    *exec.Cmd
			vmiHttpServerPort int
			vmiDeclaredPorts  []v1.Port
		)

		setup := func(ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(virtClient, ipFamily)

			if ipFamily == k8sv1.IPv6Protocol {
				Skip(skipIPv6Message)
			}

			vmi := createCirrosVMIWithPortsAndBlockUntilReady(virtClient, vmiDeclaredPorts)
			tests.StartHTTPServerWithSourceIp(vmi, vmiHttpServerPort, getMasqueradeInternalAddress(ipFamily), console.LoginToCirros)

			localPort = 1500 + GinkgoParallelProcess()
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
			Expect(vmiPod).ToNot(BeNil())
			portForwardCmd, err = portForwardCommand(vmiPod, localPort, vmiHttpServerPort)
			Expect(err).NotTo(HaveOccurred())

			stdout, err := portForwardCmd.StdoutPipe()
			Expect(err).NotTo(HaveOccurred())
			Expect(portForwardCmd.Start()).To(Succeed())
			waitForPortForwardCmd(ipFamily, stdout, localPort, vmiHttpServerPort)
		}

		AfterEach(func() {
			Expect(killPortForwardCommand(portForwardCmd)).To(Succeed())
		})

		When("performing port-forward from a local port to a VMI's declared port", func() {
			const declaredPort = 1501
			BeforeEach(func() {
				vmiDeclaredPorts = []v1.Port{{Port: declaredPort}}
				vmiHttpServerPort = declaredPort
			})

			DescribeTable("should reach the vmi", func(ipFamily k8sv1.IPFamily) {
				setup(ipFamily)
				By(fmt.Sprintf("checking that service running on port %d can be reached", declaredPort))
				Expect(testConnectivityThroughLocalPort(ipFamily, localPort)).To(Succeed())
			},
				Entry("IPv4", k8sv1.IPv4Protocol),
				Entry("IPv6", k8sv1.IPv6Protocol),
			)
		})

		When("performing port-forward from a local port to a VMI with no declared ports", func() {
			const nonDeclaredPort = 1502
			BeforeEach(func() {
				vmiDeclaredPorts = []v1.Port{}
				vmiHttpServerPort = nonDeclaredPort
			})

			DescribeTable("should reach the vmi", func(ipFamily k8sv1.IPFamily) {
				setup(ipFamily)
				By(fmt.Sprintf("checking that service running on port %d can be reached", nonDeclaredPort))
				Expect(testConnectivityThroughLocalPort(ipFamily, localPort)).To(Succeed())
			},
				Entry("IPv4", k8sv1.IPv4Protocol),
				Entry("IPv6", k8sv1.IPv6Protocol),
			)
		})

		When("performing port-forward from a local port to a VMI's non-declared port", func() {
			const nonDeclaredPort = 1502
			const declaredPort = 1501
			BeforeEach(func() {
				vmiDeclaredPorts = []v1.Port{{Port: declaredPort}}
				vmiHttpServerPort = nonDeclaredPort
			})

			DescribeTable("should not reach the vmi", func(ipFamily k8sv1.IPFamily) {
				setup(ipFamily)
				By(fmt.Sprintf("checking that service running on port %d can not be reached", nonDeclaredPort))
				Expect(testConnectivityThroughLocalPort(ipFamily, localPort)).ToNot(Succeed())
			},
				Entry("IPv4", k8sv1.IPv4Protocol),
				Entry("IPv6", k8sv1.IPv6Protocol),
			)
		})
	})
})

func portForwardCommand(pod *k8sv1.Pod, sourcePort, targetPort int) (*exec.Cmd, error) {
	_, cmd, err := clientcmd.CreateCommandWithNS(pod.Namespace, clientcmd.GetK8sCmdClient(), "port-forward", pod.Name, fmt.Sprintf("%d:%d", sourcePort, targetPort))

	return cmd, err
}

func killPortForwardCommand(portForwardCmd *exec.Cmd) error {
	if portForwardCmd == nil {
		return nil
	}

	portForwardCmd.Process.Kill()
	_, err := portForwardCmd.Process.Wait()
	return err
}

func createCirrosVMIWithPortsAndBlockUntilReady(virtClient kubecli.KubevirtClient, ports []v1.Port) *v1.VirtualMachineInstance {
	vmi := libvmi.NewCirros(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(ports...)),
	)

	vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
	Expect(err).ToNot(HaveOccurred())
	vmi = tests.WaitUntilVMIReady(vmi, console.LoginToCirros)

	return vmi
}

func testConnectivityThroughLocalPort(ipFamily k8sv1.IPFamily, portNumber int) error {
	return exec.Command("curl", fmt.Sprintf("%s:%d", libnet.GetLoopbackAddressForURL(ipFamily), portNumber)).Run()
}

func waitForPortForwardCmd(ipFamily k8sv1.IPFamily, stdout io.ReadCloser, src, dst int) {
	Eventually(func() string {
		tmp := make([]byte, 1024)
		_, err := stdout.Read(tmp)
		Expect(err).NotTo(HaveOccurred())
		return string(tmp)
	}, 30*time.Second, 1*time.Second).Should(ContainSubstring(fmt.Sprintf("Forwarding from %s:%d -> %d", libnet.GetLoopbackAddressForURL(ipFamily), src, dst)))
}

func getMasqueradeInternalAddress(ipFamily k8sv1.IPFamily) string {
	if ipFamily == k8sv1.IPv4Protocol {
		return "10.0.2.2"
	}
	return "fd10:0:2::2"
}
