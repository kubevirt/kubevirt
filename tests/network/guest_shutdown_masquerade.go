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
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	"context"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Guest shutdown with masquerade binding", decorators.Networking, func() {
	const tcpPort = 8080

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("should preserve declared-port connectivity after in-guest shutdown with runStrategy Always", decorators.WgS390x, func() {
		libnet.SkipWhenClusterNotSupportIpv4()

		const declaredPortName = "http"
		serverVM := libvmi.NewVirtualMachine(
			libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(v1.Port{Name: declaredPortName, Port: tcpPort})),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		serverVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(serverVM)).Create(context.Background(), serverVM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVM(serverVM)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())

		serverVMI, err := virtClient.VirtualMachineInstance(serverVM.Namespace).Get(context.Background(), serverVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToAlpine)

		clientVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(
			context.Background(),
			libvmifact.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			metav1.CreateOptions{},
		)
		Expect(err).ToNot(HaveOccurred())
		clientVMI = libwait.WaitUntilVMIReady(clientVMI, console.LoginToCirros)

		verifyMasqueradePortConnectivity := func(server *v1.VirtualMachineInstance) {
			Expect(console.LoginToAlpine(server)).To(Succeed())
			vmnetserver.StartTCPServer(server, tcpPort, console.LoginToAlpine)

			serverIP := libnet.GetVmiPrimaryIPByFamily(server, k8sv1.IPv4Protocol)
			Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())
			Expect(console.SafeExpectBatch(clientVMI, createExpectConnectToServer(serverIP, tcpPort, true), 30)).To(Succeed())
		}

		By("Verifying declared-port connectivity before guest shutdown")
		verifyMasqueradePortConnectivity(serverVMI)

		By("Shutting down the guest from inside the VM")
		guestPowerOff(serverVMI)

		By("Waiting for the controller to replace the shut-down VMI with a new instance")
		Eventually(matcher.ThisVMI(serverVMI), 240*time.Second, time.Second).Should(matcher.BeRestarted(serverVMI.UID))

		restartedVMI, err := virtClient.VirtualMachineInstance(serverVM.Namespace).Get(context.Background(), serverVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the restarted VMI to be ready")
		restartedVMI = libwait.WaitUntilVMIReady(restartedVMI, console.LoginToAlpine)

		By("Verifying declared-port connectivity after guest shutdown and restart")
		verifyMasqueradePortConnectivity(restartedVMI)
	})
}))

func guestPowerOff(vmi *v1.VirtualMachineInstance) {
	expecter, _, err := console.NewExpecter(kubevirt.Client(), vmi, 60*time.Second)
	Expect(err).ToNot(HaveOccurred())

	_, err = expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "poweroff\n"},
	}, 20*time.Second)
	Expect(err).ToNot(HaveOccurred())
}
