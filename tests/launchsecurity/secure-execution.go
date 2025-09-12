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
 * Copyright The KubeVirt Authors
 *
 */

package launchsecurity

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libconfigmap"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kubevirtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("[sig-compute]IBM Secure Execution", decorators.SecureExecution, decorators.SigCompute, decorators.RequiresS390X, Serial, func() {
	Context("Node Labels", func() {
		It("Should have nodes with Secure Execution Label", func() {
			nodes := libnode.GetAllSchedulableNodes(k8s.Client())
			hasNodeWithSELabel := false
			for _, node := range nodes.Items {
				if _, ok := node.Labels[kubevirtv1.SecureExecutionLabel]; ok {
					hasNodeWithSELabel = true
					break
				}
			}
			Expect(hasNodeWithSELabel).To(BeTrue())
		})
	})
	Context("Ensure cluster can run Secure Execution VMs", func() {
		var vmi *kubevirtv1.VirtualMachineInstance

		const commandTimeout = 10 * time.Second
		BeforeEach(func() {
			config.EnableFeatureGate(featuregate.SecureExecution)
			By("Reading the hostkey from the 'secex-hostkey' configmap in the 'kubevirt-prow-jobs' namespace")
			cm, err := k8s.Client().CoreV1().ConfigMaps("kubevirt-prow-jobs").Get(context.Background(), "secex-hostkey", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating a configmap containing the hostkey")
			cm = libconfigmap.New(cm.Name, cm.Data)
			cm, err = k8s.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(cm)).Create(context.Background(), cm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi = libvmifact.NewFedora(
				libvmi.WithConfigMapDisk(cm.Name, cm.Name),
			)
			// Enabling launchsecurity here won't prevent starting non-SE VMs
			vmi.Spec.Domain.LaunchSecurity = &kubevirtv1.LaunchSecurity{}

			By("Launching a non-SE VM to convert it to Secure Execution")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Logging in to non-SE VM")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Mounting the secure execution hostkey iso")
			Expect(console.RunCommand(vmi, "mount /dev/vdb /mnt/", commandTimeout)).To(Succeed())

			By("Writing the kernel cmdline to a file")
			Expect(console.RunCommand(vmi, "cat /proc/cmdline > /tmp/parmfile.txt", commandTimeout)).To(Succeed())

			By("Creating the encrypted boot image")
			Expect(console.RunCommand(vmi, "genprotimg --no-verify -o /boot/sdboot -i /boot/vmlinuz-*.s390x -r /boot/initramfs-*.s390x.img -p /tmp/parmfile.txt -k /mnt/secex-hostkey.crt", commandTimeout)).To(Succeed())

			By("Calling zipl to boot from the encrypted image")
			Expect(console.RunCommand(vmi, "zipl -t /boot -i /boot/sdboot", commandTimeout)).To(Succeed())

			By("Rebooting VM into Secure Execution mode")
			Expect(kubevirt.Client().VirtualMachineInstance(vmi.Namespace).SoftReboot(context.Background(), vmi.Name)).To(Succeed())
		})

		It("Should launch a Secure Execution VM", func() {
			By("Verifying that the VM is running in Secure Execution Mode")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			output, err := console.RunCommandAndStoreOutput(vmi, "cat /sys/firmware/uv/prot_virt_guest", commandTimeout)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal("1"))
		})
	})
})
