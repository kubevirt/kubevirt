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

	expect "github.com/google/goexpect"

	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
)

var _ = SIGDescribe("[crit:high][arm64][vendor:cnv-qe@redhat.com][level:component]", func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	})

	Describe("[crit:high][vendor:cnv-qe@redhat.com][level:component]Creating a VirtualMachineInstance", func() {
		Context("when virt-handler is responsive", func() {
			It("[Serial]VMIs with Bridge Networking shouldn't fail after the kubelet restarts", func() {
				bridgeVMI := vmi
				// Remove the masquerade interface to use the default bridge one
				bridgeVMI.Spec.Domain.Devices.Interfaces = nil
				bridgeVMI.Spec.Networks = nil
				v1.SetDefaults_NetworkInterface(bridgeVMI)
				Expect(bridgeVMI.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())

				By("starting a VMI with bridged network on a node")
				bridgeVMI, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(bridgeVMI)
				Expect(err).To(BeNil(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance with bridged networking
				nodeName := tests.WaitForSuccessfulVMIStart(bridgeVMI)

				tests.VerifyDummyNicForBridgeNetwork(bridgeVMI)

				By("restarting kubelet")
				pod := renderPkillAllPod("kubelet")
				pod.Spec.NodeName = nodeName
				_, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("starting another VMI on the same node, to verify kubelet is running again")
				newVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				newVMI.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				Eventually(func() error {
					newVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(newVMI)
					Expect(err).To(BeNil())
					return nil
				}, 100, 10).Should(Succeed(), "Should be able to start a new VM")

				By("checking if the VMI with bridged networking is still running, it will verify the CNI didn't cause the pod to be killed")
				tests.WaitForSuccessfulVMIStart(bridgeVMI)
			})

			It("VMIs with Bridge Networking should work with Duplicate Address Detection (DAD)", func() {
				bridgeVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				// Remove the masquerade interface to use the default bridge one
				bridgeVMI.Spec.Domain.Devices.Interfaces = nil
				bridgeVMI.Spec.Networks = nil
				v1.SetDefaults_NetworkInterface(bridgeVMI)
				Expect(bridgeVMI.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())

				By("starting a VMI with bridged network on a node")
				bridgeVMI, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(bridgeVMI)
				Expect(err).To(BeNil(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance with bridged networking
				By("Waiting the VirtualMachineInstance start")
				tests.WaitUntilVMIReady(bridgeVMI, console.LoginToCirros)
				tests.VerifyDummyNicForBridgeNetwork(bridgeVMI)

				// Update the VMI object so we get the IP address
				bridgeVMI, err = virtClient.VirtualMachineInstance(bridgeVMI.Namespace).Get(bridgeVMI.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmIP := libnet.GetVmiPrimaryIpByFamily(bridgeVMI, k8sv1.IPv4Protocol)
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
