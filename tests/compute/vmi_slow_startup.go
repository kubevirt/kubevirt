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

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Slow QEMU startup", func() {
	It("should not prematurely delete a domain when qemu startup is delayed", Serial, func() {
		virtClient := kubevirt.Client()
		namespace := testsuite.GetTestNamespace(nil)

		By("Selecting a schedulable node")
		nodes := libnode.GetAllSchedulableNodes(virtClient)
		Expect(nodes.Items).ToNot(BeEmpty())
		targetNode := nodes.Items[0].Name

		By("Launching a privileged interceptor pod that SIGSTOPs qemu on the target node")
		interceptorPod := createQemuInterceptorPod(virtClient, targetNode)

		By("Creating a guestless VMI pinned to the target node")
		vmi := libvmifact.NewGuestless(libvmi.WithNodeAffinityFor(targetNode))
		vmi, err := virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the interceptor to catch and stop qemu")
		Eventually(func(g Gomega) {
			pod, err := virtClient.CoreV1().Pods(interceptorPod.Namespace).Get(
				context.Background(), interceptorPod.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(pod.Status.Phase).To(Equal(k8sv1.PodSucceeded))
		}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(Succeed())

		By("Verifying the VMI is not deleted across multiple virt-handler reconcile cycles")
		Consistently(func(g Gomega) {
			current, err := virtClient.VirtualMachineInstance(namespace).Get(
				context.Background(), vmi.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(current.Status.Phase).ToNot(Equal(v1.Failed))
			g.Expect(current.DeletionTimestamp).To(BeNil())
		}).WithTimeout(60 * time.Second).WithPolling(5 * time.Second).Should(Succeed())

		By("Verifying no SignalDeletion event was emitted")
		eventList, err := virtClient.CoreV1().Events(namespace).List(context.Background(),
			metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s,reason=SignalDeletion", vmi.Name),
			})
		Expect(err).ToNot(HaveOccurred())
		Expect(eventList.Items).To(BeEmpty())

		By("Resuming qemu-kvm via virt-handler exec")
		virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, targetNode)
		Expect(err).ToNot(HaveOccurred())
		_, err = exec.ExecuteCommandOnPod(virtHandlerPod, "virt-handler",
			[]string{"sh", "-c", "pgrep -u 107 qemu-kvm | xargs -r kill -CONT"})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the VMI to reach Running after resume")
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(180 * time.Second).WithPolling(2 * time.Second).Should(matcher.BeRunning())
	})
}))

func createQemuInterceptorPod(virtClient kubecli.KubevirtClient, nodeName string) *k8sv1.Pod {
	const script = `while true; do
  PIDS=$(pgrep -u 107 qemu-kvm 2>/dev/null)
  if [ -n "$PIDS" ]; then
    for PID in $PIDS; do kill -STOP $PID 2>/dev/null; done
    exit 0
  fi
  sleep 0.05
done`
	pod := libpod.RenderPrivilegedPod("qemu-interceptor-", []string{"/bin/bash", "-c"}, []string{script})
	pod.Spec.NodeName = nodeName

	pod, err := virtClient.CoreV1().Pods(pod.Namespace).Create(
		context.Background(), pod, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	EventuallyWithOffset(1, func(g Gomega) {
		p, err := virtClient.CoreV1().Pods(pod.Namespace).Get(
			context.Background(), pod.Name, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(p.Status.Phase).To(Equal(k8sv1.PodRunning))
	}).WithTimeout(60 * time.Second).WithPolling(time.Second).Should(Succeed())

	return pod
}
