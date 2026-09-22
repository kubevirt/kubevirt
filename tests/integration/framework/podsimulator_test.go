package framework

import (
	"context"
	"fmt"
	"testing"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

func TestPodSimulatorRetriesWithoutAnotherInformerEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "launcher", Namespace: "test",
			Labels: map[string]string{virtv1.AppLabel: "virt-launcher"},
		},
		Spec: k8sv1.PodSpec{NodeName: testNodeName},
	}
	client := fake.NewSimpleClientset(pod)
	patches := 0
	client.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patches++
		if patches == 1 {
			return true, nil, fmt.Errorf("transient status failure")
		}
		return false, nil, nil
	})
	informer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
	if err := informer.GetStore().Add(pod); err != nil {
		t.Fatal(err)
	}
	ps := NewPodSimulator(client, informer, testNodeName)
	defer ps.Stop()
	// Bound the blocking queue reads if a regression prevents a retry.
	go func() {
		<-ctx.Done()
		ps.queue.ShutDown()
	}()
	ps.enqueuePod(pod)
	ps.processNextPod(ctx)
	ps.processNextPod(ctx)
	updated, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if patches != 2 || updated.Status.Phase != k8sv1.PodRunning {
		t.Fatalf("expected retry to make pod Running; patches=%d, phase=%s", patches, updated.Status.Phase)
	}
	if retries := ps.queue.NumRequeues(pod.Namespace + "/" + pod.Name); retries != 0 {
		t.Fatalf("successful pod still has %d retries", retries)
	}
}
