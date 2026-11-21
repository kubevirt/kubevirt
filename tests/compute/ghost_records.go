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

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// ghostRecordDir is where virt-handler persists its ghost record
// checkpoints, mounted into the virt-handler pod at the same path.
const ghostRecordDir = "/var/run/kubevirt-private/ghost-records"

var _ = Describe(SIG("Ghost record removal", func() {
	// This test exists because the feature gate has a single consumer, in
	// virt-handler's startup wiring, which no unit test reaches. A gate that
	// is enabled but never observed looks exactly like a healthy cluster:
	// VMs keep running on the ghost record path and nothing reports an
	// error. Asserting that no ghost records are written is what makes the
	// difference between "enabled" and "enabled and actually in effect"
	// visible.
	It("should stop writing ghost records once launcher socket authentication is enabled", Serial, func() {
		virtClient := kubevirt.Client()
		namespace := testsuite.GetTestNamespace(nil)

		By("Enabling the LauncherSocketAuthentication feature gate")
		config.EnableFeatureGate(featuregate.LauncherSocketAuthentication)

		By("Starting a VMI")
		vmi := libvmifact.NewAlpine()
		vmi, err := virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(120 * time.Second).WithPolling(time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		vmi, err = virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		nodeName := vmi.Status.NodeName

		// Registered before the restart below, so it runs after it: the gate
		// is only observed at startup, so disabling it without restarting
		// again would leave this node's virt-handler on VMI-based discovery
		// for every test that follows.
		DeferCleanup(func() {
			config.DisableFeatureGate(featuregate.LauncherSocketAuthentication)
			restartVirtHandler(virtClient, nodeName)
		})

		// virt-handler resolves the gate once, after its KubeVirt informer
		// syncs, so a handler that was already running when the gate was
		// flipped is still on the ghost record path. Restarting it is what
		// puts the feature into effect.
		By("Restarting virt-handler so it picks up the feature gate")
		restartVirtHandler(virtClient, nodeName)

		By("Checking that the VMI survived the restart")
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(60 * time.Second).WithPolling(time.Second).
			Should(matcher.BeRunning())

		// The restarted virt-handler deletes the checkpoint directory on
		// startup and, with discovery driven by ActivePods, never writes to
		// it again. Held over a window so that a handler that reconnects to
		// the launcher and re-adds a record is caught.
		By("Checking that no ghost records are written")
		Consistently(func() (string, error) {
			pod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
			if err != nil {
				return "", err
			}
			return exec.ExecuteCommandOnPod(pod, "virt-handler",
				[]string{"sh", "-c", fmt.Sprintf("ls -A %s 2>/dev/null || true", ghostRecordDir)})
		}).WithTimeout(30*time.Second).WithPolling(5*time.Second).
			Should(BeEmpty(), "virt-handler must not write ghost records while the gate is enabled")

		// BeRunning alone would also hold if discovery silently found
		// nothing, since the phase would simply never change. Pausing
		// forces a round trip the domain informer has to observe: the
		// Paused condition is derived from domain state, so it only
		// appears if the launcher was discovered and its domain read.
		By("Checking that domain state still round-trips through discovery")
		Expect(virtClient.VirtualMachineInstance(namespace).
			Pause(context.Background(), vmi.Name, &v1.PauseOptions{})).To(Succeed())
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(60 * time.Second).WithPolling(2 * time.Second).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

		Expect(virtClient.VirtualMachineInstance(namespace).
			Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})).To(Succeed())
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(60 * time.Second).WithPolling(2 * time.Second).
			Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
	})
}))

// restartVirtHandler deletes the virt-handler pod on the given node and
// waits for a different one to become ready. virt-handler reads the
// feature gate once at startup, so a gate change only takes effect after
// this.
func restartVirtHandler(virtClient kubecli.KubevirtClient, nodeName string) {
	pod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
	Expect(err).ToNot(HaveOccurred())
	oldPodName := pod.Name
	Expect(virtClient.CoreV1().Pods(pod.Namespace).
		Delete(context.Background(), oldPodName, metav1.DeleteOptions{})).To(Succeed())

	Eventually(func() (*k8sv1.Pod, error) {
		current, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
		if err != nil {
			return nil, err
		}
		if current.Name == oldPodName {
			return nil, fmt.Errorf("virt-handler pod %s has not been replaced yet", oldPodName)
		}
		return current, nil
	}).WithTimeout(180 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))
}
