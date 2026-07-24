package framework

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
)

type ResourceSimulator struct {
	k8sClient              kubernetes.Interface
	deploymentInformer     cache.SharedIndexInformer
	daemonSetInformer      cache.SharedIndexInformer
	validatingWHInformer   cache.SharedIndexInformer
	mutatingWHInformer     cache.SharedIndexInformer

	deploymentHandlerReg     cache.ResourceEventHandlerRegistration
	daemonSetHandlerReg      cache.ResourceEventHandlerRegistration
	validatingWHHandlerReg   cache.ResourceEventHandlerRegistration
	mutatingWHHandlerReg     cache.ResourceEventHandlerRegistration

	mu      sync.Mutex
	handled map[string]bool

	podCounter int32
}

func NewResourceSimulator(
	k8sClient kubernetes.Interface,
	deploymentInformer cache.SharedIndexInformer,
	daemonSetInformer cache.SharedIndexInformer,
	validatingWHInformer cache.SharedIndexInformer,
	mutatingWHInformer cache.SharedIndexInformer,
) *ResourceSimulator {
	return &ResourceSimulator{
		k8sClient:            k8sClient,
		deploymentInformer:   deploymentInformer,
		daemonSetInformer:    daemonSetInformer,
		validatingWHInformer: validatingWHInformer,
		mutatingWHInformer:   mutatingWHInformer,
		handled:              make(map[string]bool),
	}
}

func (rs *ResourceSimulator) Start() {
	rs.deploymentHandlerReg, _ = rs.deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    rs.onDeploymentEvent,
		UpdateFunc: func(_, obj interface{}) { rs.onDeploymentEvent(obj) },
	})
	rs.daemonSetHandlerReg, _ = rs.daemonSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    rs.onDaemonSetEvent,
		UpdateFunc: func(_, obj interface{}) { rs.onDaemonSetEvent(obj) },
	})
	rs.validatingWHHandlerReg, _ = rs.validatingWHInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: rs.onValidatingWebhookEvent,
	})
	rs.mutatingWHHandlerReg, _ = rs.mutatingWHInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: rs.onMutatingWebhookEvent,
	})
}

func (rs *ResourceSimulator) Stop() {
	if rs.deploymentHandlerReg != nil {
		rs.deploymentInformer.RemoveEventHandler(rs.deploymentHandlerReg)
	}
	if rs.daemonSetHandlerReg != nil {
		rs.daemonSetInformer.RemoveEventHandler(rs.daemonSetHandlerReg)
	}
	if rs.validatingWHHandlerReg != nil {
		rs.validatingWHInformer.RemoveEventHandler(rs.validatingWHHandlerReg)
	}
	if rs.mutatingWHHandlerReg != nil {
		rs.mutatingWHInformer.RemoveEventHandler(rs.mutatingWHHandlerReg)
	}
}

func (rs *ResourceSimulator) onDeploymentEvent(obj interface{}) {
	depl, ok := obj.(*appsv1.Deployment)
	if !ok {
		return
	}
	if !isManagedByKubeVirt(depl.Labels) {
		return
	}

	key := depl.Namespace + "/" + depl.Name
	rs.mu.Lock()
	if rs.handled[key] {
		rs.mu.Unlock()
		return
	}
	rs.handled[key] = true
	rs.mu.Unlock()

	fmt.Printf("[ResourceSimulator] Deployment detected: %s/%s (labels: %v)\n", depl.Namespace, depl.Name, depl.Labels)
	go rs.simulateDeploymentReady(depl)
}

func (rs *ResourceSimulator) onDaemonSetEvent(obj interface{}) {
	ds, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return
	}
	if !isManagedByKubeVirt(ds.Labels) {
		return
	}

	key := ds.Namespace + "/" + ds.Name
	rs.mu.Lock()
	if rs.handled[key] {
		rs.mu.Unlock()
		return
	}
	rs.handled[key] = true
	rs.mu.Unlock()

	fmt.Printf("[ResourceSimulator] DaemonSet detected: %s/%s\n", ds.Namespace, ds.Name)
	go rs.simulateDaemonSetReady(ds)
}

func (rs *ResourceSimulator) simulateDeploymentReady(depl *appsv1.Deployment) {
	ctx := context.Background()

	var replicas int32 = 1
	if depl.Spec.Replicas != nil {
		replicas = *depl.Spec.Replicas
	}

	depl, err := rs.k8sClient.AppsV1().Deployments(depl.Namespace).Get(ctx, depl.Name, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("[ResourceSimulator] failed to get deployment %s: %v\n", depl.Name, err)
		return
	}

	depl.Status.Replicas = replicas
	depl.Status.ReadyReplicas = replicas
	depl.Status.AvailableReplicas = replicas
	depl.Status.UpdatedReplicas = replicas
	_, err = rs.k8sClient.AppsV1().Deployments(depl.Namespace).UpdateStatus(ctx, depl, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("[ResourceSimulator] failed to update deployment status %s: %v\n", depl.Name, err)
		return
	}
	fmt.Printf("[ResourceSimulator] Deployment %s/%s status updated (replicas=%d, ready=%d)\n", depl.Namespace, depl.Name, replicas, replicas)

	rs.createInfrastructurePod(depl.Namespace, depl.Name, depl.Spec.Template)
	fmt.Printf("[ResourceSimulator] Pod created for %s (template annotations: %v)\n", depl.Name, depl.Spec.Template.Annotations)
}

func (rs *ResourceSimulator) simulateDaemonSetReady(ds *appsv1.DaemonSet) {
	ctx := context.Background()

	ds, _ = rs.k8sClient.AppsV1().DaemonSets(ds.Namespace).Get(ctx, ds.Name, metav1.GetOptions{})
	if ds == nil {
		return
	}

	ds.Status.DesiredNumberScheduled = 1
	ds.Status.CurrentNumberScheduled = 1
	ds.Status.NumberReady = 1
	ds.Status.NumberAvailable = 1
	ds.Status.UpdatedNumberScheduled = 1
	rs.k8sClient.AppsV1().DaemonSets(ds.Namespace).UpdateStatus(ctx, ds, metav1.UpdateOptions{})

	rs.createInfrastructurePod(ds.Namespace, ds.Name, ds.Spec.Template)
}

func (rs *ResourceSimulator) createInfrastructurePod(namespace, ownerName string, template k8sv1.PodTemplateSpec) {
	ctx := context.Background()
	podNum := atomic.AddInt32(&rs.podCounter, 1)

	podSpec := *template.Spec.DeepCopy()
	podSpec.PriorityClassName = ""

	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-pod-%d", ownerName, podNum),
			Namespace:   namespace,
			Labels:      template.Labels,
			Annotations: template.Annotations,
		},
		Spec: podSpec,
		Status: k8sv1.PodStatus{
			Phase: k8sv1.PodRunning,
		},
	}

	for _, c := range template.Spec.Containers {
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, k8sv1.ContainerStatus{
			Name:  c.Name,
			Ready: true,
			State: k8sv1.ContainerState{Running: &k8sv1.ContainerStateRunning{}},
		})
	}

	_, err := rs.k8sClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("[ResourceSimulator] failed to create pod %s/%s: %v\n", namespace, pod.Name, err)
	}
}

func (rs *ResourceSimulator) onValidatingWebhookEvent(obj interface{}) {
	whc, ok := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration)
	if !ok || !isManagedByKubeVirt(whc.Labels) {
		return
	}
	go rs.setWebhookFailurePolicyIgnore(whc.Name)
}

func (rs *ResourceSimulator) onMutatingWebhookEvent(obj interface{}) {
	whc, ok := obj.(*admissionregistrationv1.MutatingWebhookConfiguration)
	if !ok || !isManagedByKubeVirt(whc.Labels) {
		return
	}
	go rs.setMutatingWebhookFailurePolicyIgnore(whc.Name)
}

func (rs *ResourceSimulator) setWebhookFailurePolicyIgnore(name string) {
	ctx := context.Background()
	whc, err := rs.k8sClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return
	}
	ignore := admissionregistrationv1.Ignore
	changed := false
	for i := range whc.Webhooks {
		if whc.Webhooks[i].FailurePolicy == nil || *whc.Webhooks[i].FailurePolicy != ignore {
			whc.Webhooks[i].FailurePolicy = &ignore
			changed = true
		}
	}
	if changed {
		rs.k8sClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(ctx, whc, metav1.UpdateOptions{})
	}
}

func (rs *ResourceSimulator) setMutatingWebhookFailurePolicyIgnore(name string) {
	ctx := context.Background()
	whc, err := rs.k8sClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return
	}
	ignore := admissionregistrationv1.Ignore
	changed := false
	for i := range whc.Webhooks {
		if whc.Webhooks[i].FailurePolicy == nil || *whc.Webhooks[i].FailurePolicy != ignore {
			whc.Webhooks[i].FailurePolicy = &ignore
			changed = true
		}
	}
	if changed {
		rs.k8sClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(ctx, whc, metav1.UpdateOptions{})
	}
}

func isManagedByKubeVirt(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	return labels[virtv1.ManagedByLabel] == virtv1.ManagedByLabelOperatorValue
}
