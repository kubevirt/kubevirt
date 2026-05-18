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

package compute

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Domain resilience", func() {
	It("VMI should survive virt-handler restart when launcher socket is unreachable", Serial, func() {
		virtClient := kubevirt.Client
		namespace := testsuite.GetTestNamespace(nil)

		By("Starting an Alpine VMI")
		vmi := libvmifact.NewAlpine()
		vmi, err := virtClient().VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		vmi, err = virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		nodeName := vmi.Status.NodeName

		By("Finding the launcher pod and its socket path")
		launcherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, namespace)
		Expect(err).ToNot(HaveOccurred())
		socketPath := fmt.Sprintf("/pods/%s/volumes/kubernetes.io~empty-dir/sockets/launcher-sock", launcherPod.UID)

		By("Replacing the launcher socket with a regular file to simulate unreachable launcher")
		virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient(), nodeName)
		Expect(err).ToNot(HaveOccurred())
		_, err = exec.ExecuteCommandOnPod(virtHandlerPod, "virt-handler", []string{"mv", socketPath, socketPath + ".bak"})
		Expect(err).ToNot(HaveOccurred())
		_, err = exec.ExecuteCommandOnPod(virtHandlerPod, "virt-handler", []string{"touch", socketPath})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			vh, err := libnode.GetVirtHandlerPod(virtClient(), nodeName)
			if err != nil {
				return
			}
			exec.ExecuteCommandOnPod(vh, "virt-handler", []string{"sh", "-c", fmt.Sprintf("test -f %s.bak && rm -f %s && mv %s.bak %s; true", socketPath, socketPath, socketPath, socketPath)})
		})

		By("Deleting the virt-handler pod to trigger informer restart and listAllKnownDomains")
		err = virtClient().CoreV1().Pods(virtHandlerPod.Namespace).Delete(context.Background(), virtHandlerPod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the new virt-handler pod to be ready")
		Eventually(func() (*k8sv1.Pod, error) {
			return libnode.GetVirtHandlerPod(virtClient(), nodeName)
		}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))

		By("Verifying the VMI is still running - Unknown domain status prevented spurious deletion")
		currentVMI, err := virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(currentVMI.Status.Phase).To(Equal(v1.Running), "VMI should still be Running")

		By("Restoring the original launcher socket")
		Eventually(func() error {
			vh, err := libnode.GetVirtHandlerPod(virtClient(), nodeName)
			if err != nil {
				return err
			}
			_, err = exec.ExecuteCommandOnPod(vh, "virt-handler", []string{"sh", "-c", fmt.Sprintf("rm -f %s && mv %s.bak %s", socketPath, socketPath, socketPath)})
			return err
		}).WithTimeout(60 * time.Second).WithPolling(5 * time.Second).Should(Succeed())

		By("Restarting virt-handler to re-discover the restored launcher socket")
		virtHandlerPod, err = libnode.GetVirtHandlerPod(virtClient(), nodeName)
		Expect(err).ToNot(HaveOccurred())
		err = virtClient().CoreV1().Pods(virtHandlerPod.Namespace).Delete(context.Background(), virtHandlerPod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() (*k8sv1.Pod, error) {
			return libnode.GetVirtHandlerPod(virtClient(), nodeName)
		}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))

		By("Pausing the VMI to prove virt-handler resumes active domain processing")
		err = virtClient().VirtualMachineInstance(namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
	})
}))
