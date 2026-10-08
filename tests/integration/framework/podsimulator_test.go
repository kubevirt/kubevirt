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
			Spec: k8sv1.PodSpec{NodeName: testNodeName},
		}
		client = fake.NewSimpleClientset(pod)
		informer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		Expect(informer.GetStore().Add(pod)).To(Succeed())
	})

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
