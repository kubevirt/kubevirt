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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Slow QEMU startup", func() {
	// This suite verifies the fix for a race condition where virt-handler
	// would prematurely delete a domain that is still starting up.
	// Large-memory VFIO VMs can take minutes in CreateWithFlags while
	// allocating and locking guest memory. During that window the domain
	// may report Shutoff/Unknown, and the old code would trigger deletion,
	// leaving a transient domain that breaks later hotplug operations.
	//
	// The fix defers deletion when: the domain is Shutoff, the VMI is
	// still Scheduled, and the Shutoff reason is ambiguous (Unknown or
	// empty). The deferral is bounded by slowStartupDeferralTimeout
	// (5 minutes) to prevent infinite requeues for genuinely stalled VMs.

	It("should not emit deletion signals during normal VMI startup", Serial, func() {
		virtClient := kubevirt.Client
		namespace := testsuite.GetTestNamespace(nil)

		By("Creating an Alpine VMI")
		vmi := libvmifact.NewAlpine()
		vmi, err := virtClient().VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the VMI to reach Running")
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))

		vmi, err = virtClient().VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))

		By("Verifying no SignalDeletion event was emitted during startup")
		eventList, err := virtClient().CoreV1().Events(namespace).List(context.Background(),
			metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s,reason=SignalDeletion", vmi.Name),
			})
		Expect(err).ToNot(HaveOccurred())
		Expect(eventList.Items).To(BeEmpty(),
			"No SignalDeletion event should be emitted during normal VMI startup")

		By("Checking virt-handler logs for absence of premature domain deletion")
		nodeName := vmi.Status.NodeName
		virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient(), nodeName)
		Expect(err).ToNot(HaveOccurred())

		sinceSeconds := int64(120)
		logsReq := virtClient().CoreV1().Pods(virtHandlerPod.Namespace).GetLogs(
			virtHandlerPod.Name, &k8sv1.PodLogOptions{
				SinceSeconds: &sinceSeconds,
				Container:    "virt-handler",
			})
		logData, err := logsReq.DoRaw(context.Background())
		Expect(err).ToNot(HaveOccurred())

		vmiLogLines := filterLogLines(string(logData), vmi.Name)
		Expect(vmiLogLines).ToNot(ContainSubstring("Deleting inactive domain"),
			"virt-handler must not attempt domain deletion during startup")
	})

	It("should not prematurely delete a domain when qemu startup is delayed", Serial, func() {
		// This test simulates the slow QEMU startup race condition by
		// SIGSTOP-ing the qemu-kvm process immediately after it spawns.
		// This prevents the QMP handshake with libvirt, causing
		// CreateWithFlags to hang (mimicking VFIO memory allocation).
		//
		// The approach uses a privileged pod with hostPID on the target
		// node to intercept qemu before creating the VMI, following the
		// established pattern from tests/migration/migration.go.
		virtClient := kubevirt.Client()
		namespace := testsuite.GetTestNamespace(nil)

		By("Selecting a schedulable node")
		nodes := libnode.GetAllSchedulableNodes(virtClient)
		Expect(nodes.Items).ToNot(BeEmpty())
		targetNode := nodes.Items[0].Name

		By("Launching a privileged interceptor pod that will SIGSTOP qemu on the target node")
		// The pod watches for any qemu-kvm process owned by uid 107
		// (the qemu user in virt-launcher) and immediately pauses it.
		// Once it catches qemu, it prints "STOPPED" and sleeps forever
		// so it does not exit and get restarted.
		interceptorScript := `
echo "Interceptor waiting for qemu-kvm..."
while true; do
  PIDS=$(pgrep -u 107 qemu-kvm 2>/dev/null)
  if [ -n "$PIDS" ]; then
    for PID in $PIDS; do
      kill -STOP $PID 2>/dev/null
    done
    echo "STOPPED: $PIDS"
    sleep infinity
  fi
  sleep 0.05
done`

		interceptorPod := libpod.RenderPrivilegedPod("qemu-interceptor-", []string{"/bin/bash", "-c"}, []string{interceptorScript})
		interceptorPod.Spec.NodeName = targetNode
		interceptorPod, err := virtClient.CoreV1().Pods(interceptorPod.Namespace).Create(
			context.Background(), interceptorPod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			_ = virtClient.CoreV1().Pods(interceptorPod.Namespace).Delete(
				context.Background(), interceptorPod.Name, metav1.DeleteOptions{})
		})

		By("Waiting for the interceptor pod to be running")
		Eventually(func() k8sv1.PodPhase {
			pod, err := virtClient.CoreV1().Pods(interceptorPod.Namespace).Get(
				context.Background(), interceptorPod.Name, metav1.GetOptions{})
			if err != nil {
				return k8sv1.PodUnknown
			}
			return pod.Status.Phase
		}).WithTimeout(60 * time.Second).WithPolling(time.Second).Should(Equal(k8sv1.PodRunning))

		By("Creating an Alpine VMI pinned to the target node")
		vmi := libvmifact.NewAlpine(libvmi.WithNodeAffinityFor(targetNode))
		vmi, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the interceptor caught qemu-kvm")
		Eventually(func() string {
			logsReq := virtClient.CoreV1().Pods(interceptorPod.Namespace).GetLogs(
				interceptorPod.Name, &k8sv1.PodLogOptions{})
			logData, err := logsReq.DoRaw(context.Background())
			if err != nil {
				return ""
			}
			return string(logData)
		}).WithTimeout(120*time.Second).WithPolling(2*time.Second).Should(
			ContainSubstring("STOPPED:"),
			"Interceptor must catch and SIGSTOP qemu-kvm")

		By("Waiting for multiple virt-handler reconcile cycles with qemu paused")
		// virt-handler reconciles every few seconds. We wait long
		// enough for multiple cycles to have fired while qemu is
		// paused, exercising the deferral logic in the fix.
		time.Sleep(60 * time.Second)

		By("Verifying the VMI has NOT been prematurely deleted or marked Failed")
		vmi, err = virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).ToNot(Equal(v1.Failed),
			"VMI must not be marked Failed while qemu is paused during startup")
		Expect(vmi.Status.Phase).To(BeElementOf(v1.Scheduled, v1.Running),
			"VMI should remain in Scheduled or Running phase during slow startup")

		By("Verifying no SignalDeletion event was emitted")
		eventList, err := virtClient.CoreV1().Events(namespace).List(context.Background(),
			metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s,reason=SignalDeletion", vmi.Name),
			})
		Expect(err).ToNot(HaveOccurred())
		Expect(eventList.Items).To(BeEmpty(),
			"No SignalDeletion event should be emitted while qemu is paused")

		By("Deleting the interceptor pod so it cannot re-stop qemu")
		err = virtClient.CoreV1().Pods(interceptorPod.Namespace).Delete(
			context.Background(), interceptorPod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Resuming qemu-kvm via a SIGCONT privileged pod")
		resumeScript := `for PID in $(pgrep -u 107 qemu-kvm 2>/dev/null); do
  kill -CONT $PID 2>/dev/null && echo "RESUMED $PID"
done
echo "done"`
		resumePod := libpod.RenderPrivilegedPod("qemu-resume-", []string{"/bin/bash", "-c"}, []string{resumeScript})
		resumePod.Spec.NodeName = targetNode
		resumePod, err = virtClient.CoreV1().Pods(resumePod.Namespace).Create(
			context.Background(), resumePod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			_ = virtClient.CoreV1().Pods(resumePod.Namespace).Delete(
				context.Background(), resumePod.Name, metav1.DeleteOptions{})
		})

		By("Waiting for the VMI to reach Running after resume")
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(180 * time.Second).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))

		By("Confirming the VMI is Running and stable")
		vmi, err = virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
	})
}))

func filterLogLines(logs, vmiName string) string {
	var filtered []string
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, vmiName) {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}
