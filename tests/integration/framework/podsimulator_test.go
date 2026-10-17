/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package framework

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Pod simulator", func() {
	var (
		pod      *k8sv1.Pod
		client   *fake.Clientset
		informer cache.SharedIndexInformer
	)

	BeforeEach(func() {
		pod = &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "launcher", Namespace: "test",
				Labels: map[string]string{virtv1.AppLabel: "virt-launcher"},
			},
			Spec: k8sv1.PodSpec{
				NodeName:   testNodeName,
				Containers: []k8sv1.Container{{Name: "compute", Image: "virt-launcher:latest"}},
			},
		}
		client = fake.NewSimpleClientset(pod)
		informer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		Expect(informer.GetStore().Add(pod)).To(Succeed())
	})

	It("reports regular, completed init, and restartable init containers", func(ctx SpecContext) {
		pod.Spec.Containers = append(pod.Spec.Containers, k8sv1.Container{Name: "guest-console-log", Image: "console-log:latest"})
		pod.Spec.InitContainers = []k8sv1.Container{
			{Name: "container-disk-init", Image: "container-disk:latest"},
			{Name: "network-sidecar", Image: "network:latest", RestartPolicy: new(k8sv1.ContainerRestartPolicyAlways)},
		}
		client = fake.NewSimpleClientset(pod)
		ps := NewPodSimulator(client, informer, testNodeName)
		DeferCleanup(ps.Stop)
		Expect(ps.bindAndSetReady(ctx, pod)).To(BeTrue())

		updated, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Status.ContainerStatuses).To(HaveLen(2))
		for i, status := range updated.Status.ContainerStatuses {
			Expect(status.Name).To(Equal(pod.Spec.Containers[i].Name))
			Expect(status.Image).To(Equal(pod.Spec.Containers[i].Image))
			Expect(status.Ready).To(BeTrue())
			Expect(status.Started).To(HaveValue(BeTrue()))
			Expect(status.State.Running).NotTo(BeNil())
		}
		Expect(updated.Status.InitContainerStatuses).To(HaveLen(2))
		init := updated.Status.InitContainerStatuses[0]
		Expect(init.Name).To(Equal("container-disk-init"))
		Expect(init.State.Terminated).NotTo(BeNil())
		Expect(init.State.Terminated.ExitCode).To(BeZero())
		Expect(init.State.Running).To(BeNil())
		sidecar := updated.Status.InitContainerStatuses[1]
		Expect(sidecar.Name).To(Equal("network-sidecar"))
		Expect(sidecar.Ready).To(BeTrue())
		Expect(sidecar.Started).To(HaveValue(BeTrue()))
		Expect(sidecar.State.Running).NotTo(BeNil())
		Expect(sidecar.State.Terminated).To(BeNil())
	}, SpecTimeout(5*time.Second))

	It("retries a failed status update without another informer event", func(ctx SpecContext) {
		patches := 0
		client.PrependReactor("patch", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			patches++
			if patches == 1 {
				return true, nil, fmt.Errorf("transient status failure")
			}
			return false, nil, nil
		})
		ps := NewPodSimulator(client, informer, testNodeName)
		ps.Start(ctx)
		DeferCleanup(ps.Stop)
		ps.enqueuePod(pod)

		Eventually(ctx, func(ctx context.Context) (k8sv1.PodPhase, error) {
			updated, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			return updated.Status.Phase, nil
		}).Should(Equal(k8sv1.PodRunning))
		Expect(patches).To(Equal(2))
		Eventually(ctx, func() int {
			return ps.queue.NumRequeues(pod.Namespace + "/" + pod.Name)
		}).Should(BeZero())
	}, SpecTimeout(5*time.Second))

	It("waits for an in-flight status update before stopping", func(ctx SpecContext) {
		patchStarted := make(chan struct{})
		releasePatch := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(releasePatch) })
		client.PrependReactor("patch", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			close(patchStarted)
			<-releasePatch
			return false, nil, nil
		})
		ps := NewPodSimulator(client, informer, testNodeName)
		ps.Start(ctx)
		DeferCleanup(ps.Stop)
		DeferCleanup(unblock)
		ps.enqueuePod(pod)
		Eventually(ctx, patchStarted).Should(BeClosed())

		stopped := make(chan struct{})
		go func() {
			ps.Stop()
			close(stopped)
		}()
		Consistently(ctx, stopped, 100*time.Millisecond).ShouldNot(BeClosed())
		unblock()
		Eventually(ctx, stopped).Should(BeClosed())
	}, SpecTimeout(5*time.Second))
})
