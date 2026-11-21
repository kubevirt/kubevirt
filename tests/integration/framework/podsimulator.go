package framework

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

const (
	// ipRangeStart is the first usable octet (skipping .0 network and .1 gateway).
	ipRangeStart = 2
	// ipRangeSize is the number of usable octets in [ipRangeStart, 254] (excluding .255 broadcast).
	ipRangeSize = 253
)

type PodAction string

const (
	ActionDefault  PodAction = "default"
	ActionSkip     PodAction = "skip"
	ActionFail     PodAction = "fail"
	ActionNotReady PodAction = "not_ready"
)

type PodSimulationHook func(pod *k8sv1.Pod) PodSimulationResult

type PodSimulationResult struct {
	Action       PodAction
	Delay        time.Duration
	Reason       string
	Message      string
	CustomStatus *k8sv1.PodStatus
}

type PodSimulator struct {
	k8sClient kubernetes.Interface
	informer  cache.SharedIndexInformer
	nodeName  string

	handlerReg cache.ResourceEventHandlerRegistration

	queue     workqueue.TypedRateLimitingInterface[string]
	ipCounter atomic.Int32
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	hook      PodSimulationHook
}

func (ps *PodSimulator) SetHook(hook PodSimulationHook) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.hook = hook
}

func (ps *PodSimulator) ResetHook() {
	ps.SetHook(nil)
}

func NewPodSimulator(k8sClient kubernetes.Interface, podInformer cache.SharedIndexInformer, nodeName string) *PodSimulator {
	return &PodSimulator{
		k8sClient: k8sClient,
		informer:  podInformer,
		nodeName:  nodeName,
		queue:     workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]()),
	}
}

func (ps *PodSimulator) Start(ctx context.Context) {
	GinkgoHelper()
	ctx, ps.cancel = context.WithCancel(ctx)
	reg, err := ps.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ps.enqueuePod,
		UpdateFunc: func(_, obj any) {
			ps.enqueuePod(obj)
		},
	})
	Expect(err).NotTo(HaveOccurred(), "failed to add pod event handler")
	ps.handlerReg = reg
	ps.wg.Go(func() {
		<-ctx.Done()
		ps.queue.ShutDown()
	})
	ps.wg.Go(func() {
		for ps.processNextPod(ctx) {
		}
	})
}

func (ps *PodSimulator) Stop() {
	if ps.cancel != nil {
		ps.cancel()
	}
	ps.queue.ShutDown()
	ps.wg.Wait()
	if ps.handlerReg != nil {
		_ = ps.informer.RemoveEventHandler(ps.handlerReg)
	}
}

func (ps *PodSimulator) enqueuePod(obj any) {
	pod, ok := obj.(*k8sv1.Pod)
	if !ok {
		return
	}
	if !isVirtLauncherPod(pod) || pod.DeletionTimestamp != nil || isPodSimulationComplete(pod) {
		return
	}

	ps.queue.Add(pod.Namespace + "/" + pod.Name)
}

func (ps *PodSimulator) processNextPod(ctx context.Context) bool {
	key, shutdown := ps.queue.Get()
	if shutdown {
		return false
	}
	defer ps.queue.Done(key)
	obj, exists, err := ps.informer.GetStore().GetByKey(key)
	if err != nil {
		ps.queue.AddRateLimited(key)
		return true
	}
	if !exists {
		ps.queue.Forget(key)
		return true
	}
	pod := obj.(*k8sv1.Pod)
	if pod.DeletionTimestamp != nil || isPodSimulationComplete(pod) {
		ps.queue.Forget(key)
		return true
	}
	if ps.bindAndSetReady(ctx, pod) {
		ps.queue.Forget(key)
	} else {
		ps.queue.AddRateLimited(key)
	}
	return true
}

func (ps *PodSimulator) bindAndSetReady(ctx context.Context, pod *k8sv1.Pod) bool {
	namespace, name := pod.Namespace, pod.Name
	// A previous attempt may have bound the pod before failing to update status.
	pod, err := ps.k8sClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "pod simulator: failed to get pod %s/%s: %v\n", namespace, name, err)
		return false
	}
	if pod.DeletionTimestamp != nil || isPodSimulationComplete(pod) {
		return true
	}

	ps.mu.RLock()
	hook := ps.hook
	ps.mu.RUnlock()
	var result PodSimulationResult
	if hook != nil {
		result = hook(pod.DeepCopy())
	}
	if result.Delay > 0 {
		timer := time.NewTimer(result.Delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return true
		case <-timer.C:
		}
	}
	if result.Action == ActionSkip {
		return true
	}

	if pod.Spec.NodeName == "" {
		binding := &k8sv1.Binding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Target: k8sv1.ObjectReference{
				Kind: "Node",
				Name: ps.nodeName,
			},
		}
		err = ps.k8sClient.CoreV1().Pods(namespace).Bind(ctx, binding, metav1.CreateOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "pod simulator: failed to bind pod %s/%s: %v\n", namespace, name, err)
			return false
		}
	}

	pod.Status = ps.buildStatus(pod, result)
	statusPatch, err := patch.New(patch.WithAdd("/status", pod.Status)).GeneratePayload()
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "pod simulator: failed to generate pod status patch %s/%s: %v\n", namespace, name, err)
		return false
	}
	_, err = ps.k8sClient.CoreV1().Pods(namespace).
		Patch(ctx, name, k8stypes.JSONPatchType, statusPatch, metav1.PatchOptions{}, "status")
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "pod simulator: failed to patch pod status %s/%s: %v\n", namespace, name, err)
		return false
	}
	return true
}

func (ps *PodSimulator) buildStatus(pod *k8sv1.Pod, result PodSimulationResult) k8sv1.PodStatus {
	if result.CustomStatus != nil {
		return *result.CustomStatus.DeepCopy()
	}
	if result.Action == ActionFail {
		reason := result.Reason
		if reason == "" {
			reason = "OOMKilled"
		}
		const failureExitCode = 137
		return k8sv1.PodStatus{
			Phase: k8sv1.PodFailed, Reason: reason, Message: result.Message,
			Conditions: []k8sv1.PodCondition{
				{Type: k8sv1.PodScheduled, Status: k8sv1.ConditionTrue},
				{Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse},
			},
			ContainerStatuses: []k8sv1.ContainerStatus{{
				Name: "compute",
				State: k8sv1.ContainerState{Terminated: &k8sv1.ContainerStateTerminated{
					ExitCode: failureExitCode, Reason: reason, Message: result.Message,
				}},
			}},
		}
	}
	// Allocate IPs in 10.244.0.2-254, avoiding .0 (network), .1 (gateway), .255 (broadcast).
	n := ps.ipCounter.Add(1)
	ipOctet := byte((n-1)%ipRangeSize + ipRangeStart) //nolint:gosec // G115: modulo bounds result to [2,254], safe truncation
	status := k8sv1.PodStatus{
		Phase: k8sv1.PodRunning,
		PodIP: fmt.Sprintf("10.244.0.%d", ipOctet),
		Conditions: []k8sv1.PodCondition{
			{
				Type:   k8sv1.PodReady,
				Status: k8sv1.ConditionTrue,
			},
			{
				Type:   k8sv1.PodScheduled,
				Status: k8sv1.ConditionTrue,
			},
		},
	}
	for _, container := range pod.Spec.Containers {
		status.ContainerStatuses = append(status.ContainerStatuses, runningContainerStatus(container))
	}
	for _, container := range pod.Spec.InitContainers {
		containerStatus := runningContainerStatus(container)
		if container.RestartPolicy == nil || *container.RestartPolicy != k8sv1.ContainerRestartPolicyAlways {
			// Ordinary init containers finish before the pod can run; restartable sidecars keep running.
			containerStatus.State = k8sv1.ContainerState{Terminated: &k8sv1.ContainerStateTerminated{ExitCode: 0, Reason: "Completed"}}
			containerStatus.Started = new(false)
		}
		status.InitContainerStatuses = append(status.InitContainerStatuses, containerStatus)
	}
	if result.Action == ActionNotReady {
		status.Conditions[0].Status = k8sv1.ConditionFalse
		for i := range status.ContainerStatuses {
			status.ContainerStatuses[i].Ready = false
		}
	}
	return status
}

func isPodSimulationComplete(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodRunning || pod.Status.Phase == k8sv1.PodFailed || pod.Status.Phase == k8sv1.PodSucceeded
}

func runningContainerStatus(container k8sv1.Container) k8sv1.ContainerStatus {
	return k8sv1.ContainerStatus{
		Name:    container.Name,
		Image:   container.Image,
		Ready:   true,
		Started: new(true),
		State:   k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
	}
}

func isVirtLauncherPod(pod *k8sv1.Pod) bool {
	return pod.Labels[virtv1.AppLabel] == "virt-launcher"
}
