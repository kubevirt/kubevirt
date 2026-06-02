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
	"strings"
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
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("[crit:high][vendor:cnv-qe@redhat.com][level:component]", decorators.WgArm64, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("[crit:high][vendor:cnv-qe@redhat.com][level:component]Creating a VirtualMachineInstance", func() {
		Context("when virt-handler is responsive", func() {
			DescribeTable("VMIs shouldn't fail after the kubelet restarts", decorators.Disruptive, func(bridgeNetworking bool) {
				var vmiOptions []libvmi.Option

				if bridgeNetworking {
					libnet.SkipWhenClusterNotSupportIpv4()

					vmiOptions = []libvmi.Option{
						libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
					}
				}

				vmi := libvmifact.NewAlpine(vmiOptions...)

				By("starting a VMI on a node")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				nodeName := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName

				if bridgeNetworking {
					verifyDummyNicForBridgeNetwork(vmi)
				}

				By("restarting kubelet")
				pod := libpod.RenderPrivilegedPod("vmi-killer", []string{"pkill"}, []string{"-9", "kubelet"})
				pod.Spec.NodeName = nodeName
				pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() k8sv1.PodPhase {
					pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Get(context.Background(), pod.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pod.Status.Phase
				}, 50, 5).Should(Equal(k8sv1.PodSucceeded))

				By("starting another VMI on the same node, to verify kubelet is running again")
				newVMI := libvmifact.NewAlpine()
				newVMI.Spec.NodeSelector = map[string]string{k8sv1.LabelHostname: nodeName}
				Eventually(func() error {
					newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(newVMI)).Create(context.Background(), newVMI, metav1.CreateOptions{})
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
				bridgeVMI := libvmifact.NewAlpineWithTestTooling(
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)

				By("creating a VMI with bridged network on a node")
				bridgeVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(bridgeVMI)).Create(context.Background(), bridgeVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting the VirtualMachineInstance start")
				// Alpine on arm64 UEFI—often needs more than the default 180s console login timeout.
				loginToAlpine := func(vmi *v1.VirtualMachineInstance, _ ...time.Duration) error {
					return console.LoginToAlpine(vmi, 10*time.Minute)
				}
				bridgeVMI = libwait.WaitUntilVMIReady(bridgeVMI, loginToAlpine)
				verifyDummyNicForBridgeNetwork(bridgeVMI)

				// Alpine does not auto-bring-up eth0 or run DHCP. arping -D needs eth0 up
				// and a configured guest stack; -n exits after one lease attempt (bare udhcpc would hang).
				By("Configuring network inside Alpine VM")
				Expect(console.RunCommand(bridgeVMI, "ip link set eth0 up", 30*time.Second)).To(Succeed())
				Expect(console.RunCommand(bridgeVMI, "udhcpc -i eth0 -n -q", 60*time.Second)).To(Succeed())

				vmIP := libnet.GetVmiPrimaryIPByFamily(bridgeVMI, k8sv1.IPv4Protocol)
				dadCommand := fmt.Sprintf("arping -D -I eth0 -c 2 %s | grep Received | cut -d ' ' -f 2\n", vmIP)

				Expect(console.SafeExpectBatch(bridgeVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: ""},

					&expect.BSnd{S: dadCommand},
					&expect.BExp{R: "0"},
				}, 600)).To(Succeed())
			})
		})
	})
}))

func verifyDummyNicForBridgeNetwork(vmi *v1.VirtualMachineInstance) {
	output := libpod.RunCommandOnVmiPod(vmi, []string{"/bin/bash", "-c", "/usr/sbin/ip link show|grep DOWN|grep -c eth0"})
	ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("1"))

	output = libpod.RunCommandOnVmiPod(vmi, []string{"/bin/bash", "-c", "/usr/sbin/ip link show|grep UP|grep -c eth0-nic"})
	ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("1"))
}
