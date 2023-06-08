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
	"context"
	"fmt"
	"strings"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"

	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = SIGDescribe("[crit:high][arm64][vendor:cnv-qe@redhat.com][level:component]", func() {
	var virtClient kubecli.KubevirtClient
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		vmi = libvmi.NewAlpine()
	})

	Describe("[crit:high][vendor:cnv-qe@redhat.com][level:component]Creating a VirtualMachineInstance", func() {
		Context("when virt-handler is responsive", func() {
			DescribeTable("[Serial]VMIs shouldn't fail after the kubelet restarts", func(bridgeNetworking bool) {
				if bridgeNetworking {
					libnet.SkipWhenClusterNotSupportIpv4()
					// Remove the masquerade interface to use the default bridge one
					vmi.Spec.Domain.Devices.Interfaces = nil
					vmi.Spec.Networks = nil
					v1.SetDefaults_NetworkInterface(vmi)
					Expect(vmi.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())
				}

				By("starting a VMI on a node")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				nodeName := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName

				if bridgeNetworking {
					verifyDummyNicForBridgeNetwork(vmi)
				}

				By("restarting kubelet")
				pod := renderPkillAllPod("kubelet")
				pod.Spec.NodeName = nodeName
				pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() k8sv1.PodPhase {
					pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Get(context.Background(), pod.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pod.Status.Phase
				}, 50, 5).Should(Equal(k8sv1.PodSucceeded))

				By("starting another VMI on the same node, to verify kubelet is running again")
				newVMI := libvmi.NewCirros()
				newVMI.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				Eventually(func() error {
					newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(newVMI)).Create(context.Background(), newVMI)
					Expect(err).ToNot(HaveOccurred())
					return nil
				}, 100, 10).Should(Succeed(), "Should be able to start a new VM")
				libwait.WaitForSuccessfulVMIStart(newVMI)

				By("checking if the VMI with bridged networking is still running, it will verify the CNI didn't cause the pod to be killed")
				libwait.WaitForSuccessfulVMIStart(vmi)
			},
				Entry("[sig-network]with bridge networking", Serial, decorators.SigNetwork, true),
				Entry("[sig-compute]with default networking", Serial, decorators.SigCompute, false),
			)

			It("VMIs with Bridge Networking should work with Duplicate Address Detection (DAD)", decorators.Networking, func() {
				libnet.SkipWhenClusterNotSupportIpv4()
				bridgeVMI := libvmi.NewCirros()
				// Remove the masquerade interface to use the default bridge one
				bridgeVMI.Spec.Domain.Devices.Interfaces = nil
				bridgeVMI.Spec.Networks = nil
				v1.SetDefaults_NetworkInterface(bridgeVMI)
				Expect(bridgeVMI.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())

				By("starting a VMI with bridged network on a node")
				bridgeVMI = tests.RunVMI(bridgeVMI, 40)

				// Start a VirtualMachineInstance with bridged networking
				By("Waiting the VirtualMachineInstance start")
				bridgeVMI = libwait.WaitUntilVMIReady(bridgeVMI, console.LoginToCirros)
				verifyDummyNicForBridgeNetwork(bridgeVMI)

				vmIP := libnet.GetVmiPrimaryIPByFamily(bridgeVMI, k8sv1.IPv4Protocol)
				dadCommand := fmt.Sprintf("sudo /usr/sbin/arping -D -I eth0 -c 2 %s | grep Received | cut -d ' ' -f 2\n", vmIP)

				Expect(console.SafeExpectBatch(bridgeVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},

					&expect.BSnd{S: dadCommand},
					&expect.BExp{R: "0"},
				}, 600)).To(Succeed())
			})
		})
	})
})

func renderPkillAllPod(processName string) *k8sv1.Pod {
	return tests.RenderPrivilegedPod("vmi-killer", []string{"pkill"}, []string{"-9", processName})
}

func verifyDummyNicForBridgeNetwork(vmi *v1.VirtualMachineInstance) {
	output := tests.RunCommandOnVmiPod(vmi, []string{tests.BinBash, "-c", "/usr/sbin/ip link show|grep DOWN|grep -c eth0"})
	ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("1"))

	output = tests.RunCommandOnVmiPod(vmi, []string{tests.BinBash, "-c", "/usr/sbin/ip link show|grep UP|grep -c eth0-nic"})
	ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("1"))
}
