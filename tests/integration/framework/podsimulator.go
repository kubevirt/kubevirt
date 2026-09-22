package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
)

type PodSimulator struct {
	k8sClient kubernetes.Interface
	informer  cache.SharedIndexInformer
	nodeName  string

	handlerReg cache.ResourceEventHandlerRegistration

	mu        sync.Mutex
	handled   map[string]bool
	ipCounter atomic.Int32
}

func NewPodSimulator(k8sClient kubernetes.Interface, podInformer cache.SharedIndexInformer, nodeName string) *PodSimulator {
	return &PodSimulator{
		k8sClient: k8sClient,
		informer:  podInformer,
		nodeName:  nodeName,
		handled:   make(map[string]bool),
	}
}

func (ps *PodSimulator) Start() {
	GinkgoHelper()
	reg, err := ps.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ps.onPodAdd,
	})
	Expect(err).NotTo(HaveOccurred(), "failed to add pod event handler")
	ps.handlerReg = reg
}

func (ps *PodSimulator) Stop() {
	if ps.handlerReg != nil {
		_ = ps.informer.RemoveEventHandler(ps.handlerReg)
	}
}

func (ps *PodSimulator) onPodAdd(obj any) {
	pod, ok := obj.(*k8sv1.Pod)
	if !ok {
		return
	}
	ps.simulatePod(pod)
}

func (ps *PodSimulator) simulatePod(pod *k8sv1.Pod) {
	if !isVirtLauncherPod(pod) {
		return
	}

	key := pod.Namespace + "/" + pod.Name
	ps.mu.Lock()
	if ps.handled[key] {
		ps.mu.Unlock()
		return
	}
	ps.handled[key] = true
	ps.mu.Unlock()

	// Bind and status-patch are network calls; run off the informer event handler goroutine.
	go ps.bindAndSetReady(pod)
}

func (ps *PodSimulator) bindAndSetReady(pod *k8sv1.Pod) {
	ctx := context.Background()

	if pod.Spec.NodeName == "" {
		binding := &k8sv1.Binding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
			Target: k8sv1.ObjectReference{
				Kind: "Node",
				Name: ps.nodeName,
			},
		}
		err := ps.k8sClient.CoreV1().Pods(pod.Namespace).Bind(ctx, binding, metav1.CreateOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "pod simulator: failed to bind pod %s/%s: %v\n", pod.Namespace, pod.Name, err)
			return
		}
	}

	pod, err := ps.k8sClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "pod simulator: failed to get pod %s/%s after bind: %v\n", pod.Namespace, pod.Name, err)
		return
	}

	// Start at .2 to avoid .0 (network) and .1 (gateway); uint8 wraps naturally within an octet.
	ipOctet := uint8(ps.ipCounter.Add(1)) + 1 // #nosec G115 intentional octet wrapping
	pod.Status = k8sv1.PodStatus{
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
		ContainerStatuses: []k8sv1.ContainerStatus{
			{
				Name:  "compute",
				Ready: true,
				State: k8sv1.ContainerState{
					Running: &k8sv1.ContainerStateRunning{},
				},
			},
		},
	}

	statusPatch, err := json.Marshal(map[string]any{"status": pod.Status})
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "pod simulator: failed to marshal pod status %s/%s: %v\n", pod.Namespace, pod.Name, err)
		return
	}
	_, err = ps.k8sClient.CoreV1().Pods(pod.Namespace).
		Patch(ctx, pod.Name, k8stypes.MergePatchType, statusPatch, metav1.PatchOptions{}, "status")
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "pod simulator: failed to patch pod status %s/%s: %v\n", pod.Namespace, pod.Name, err)
	}
}

func isVirtLauncherPod(pod *k8sv1.Pod) bool {
	return pod.Labels[virtv1.AppLabel] == "virt-launcher"
}
