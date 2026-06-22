package inplaceresize

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/trace"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

type Controller struct {
	clusterConfig   *virtconfig.ClusterConfig
	recorder        record.EventRecorder
	clientset       kubecli.KubevirtClient
	podIndexer      cache.Indexer
	vmiIndexer      cache.Indexer
	queue           workqueue.TypedRateLimitingInterface[string]
	hasSynced       func() bool
	templateService services.TemplateService
	logger          *log.FilteredLogger
	vmiConditions   *controller.VirtualMachineInstanceConditionManager
	podConditions   *controller.PodConditionManager
}

func NewInPlaceResizeController(
	clusterConfig *virtconfig.ClusterConfig,
	vmiInformer,
	podInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	templateService services.TemplateService) (*Controller, error) {
	c := &Controller{
		clusterConfig:   clusterConfig,
		recorder:        recorder,
		clientset:       clientset,
		templateService: templateService,
		logger:          log.Log.With("controller", "inplaceresize"),
		vmiConditions:   controller.NewVirtualMachineInstanceConditionManager(),
		podConditions:   controller.NewPodConditionManager(),

		podIndexer: podInformer.GetIndexer(),
		vmiIndexer: vmiInformer.GetIndexer(),

		queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "inplaceresize-controller"},
		),
	}

	c.hasSynced = func() bool {
		return vmiInformer.HasSynced() && podInformer.HasSynced()
	}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})
	if err != nil {
		return nil, err
	}

	_, err = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		UpdateFunc: c.updatePod,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Controller) addVirtualMachineInstance(obj interface{}) {
	if vmi, ok := obj.(*v1.VirtualMachineInstance); ok {
		c.enqueueVirtualMachineInstance(vmi)
		return
	}
	c.logger.Error("Failed to cast object to VirtualMachineInstance")
}

func (c *Controller) updateVirtualMachineInstance(oldObj, newObj interface{}) {
	_, oldOk := oldObj.(*v1.VirtualMachineInstance)
	newVmi, newOk := newObj.(*v1.VirtualMachineInstance)

	if oldOk && newOk {
		c.enqueueVirtualMachineInstance(newVmi)
		return
	}
	c.logger.Error("Failed to cast object to VirtualMachineInstance")
}

func (c *Controller) addPod(obj interface{}) {
	pod, ok := obj.(*k8sv1.Pod)
	if ok {
		vmi := controller.ResolveControllerRef(pod.Namespace, controller.GetControllerOf(pod), c.podIndexer, c.vmiIndexer)
		if vmi != nil {
			c.enqueueVirtualMachineInstance(vmi)
		}
	} else {
		c.logger.Error("Failed to cast object to Pod")
	}
}

func (c *Controller) updatePod(oldObj interface{}, newObj interface{}) {
	_, oldOk := oldObj.(*k8sv1.Pod)
	newPod, newOk := newObj.(*k8sv1.Pod)

	if oldOk && newOk {
		vmi := controller.ResolveControllerRef(newPod.Namespace, controller.GetControllerOf(newPod), c.podIndexer, c.vmiIndexer)
		if vmi != nil {
			c.enqueueVirtualMachineInstance(vmi)
		}
	} else {
		c.logger.Error("Failed to cast object to Pod")
	}
}

func (c *Controller) enqueueVirtualMachineInstance(vmi *v1.VirtualMachineInstance) {
	key, err := controller.KeyFunc(vmi)
	if err != nil {
		c.logger.Object(vmi).Reason(err).Error("Failed to extract key from VirtualMachineInstance.")
	}
	c.queue.Add(key)
}

func (c *Controller) Run(threadiness int, ctx context.Context) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting InPlaceResize controller")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(ctx.Done(), c.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
	log.Log.Info("Stopping DRA Status controller")
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.Execute(ctx) {
	}
}

var controllerWorkQueueTracer = &traceUtils.Tracer{Threshold: time.Second}

func (c *Controller) Execute(ctx context.Context) bool {
	if !c.clusterConfig.InPlaceResizeEnabled() {
		return false
	}
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	controllerWorkQueueTracer.StartTrace(key, "inplaceresize controller workqueue", trace.Field{Key: "Workqueue Key", Value: key})
	defer controllerWorkQueueTracer.StopTrace(key)

	defer c.queue.Done(key)
	err := c.execute(ctx, key)

	if err != nil {
		c.logger.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		c.logger.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *Controller) execute(ctx context.Context, key string) error {
	obj, exists, err := c.vmiIndexer.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	vmi, ok := obj.(*v1.VirtualMachineInstance)
	if !ok {
		c.logger.Error("failed to cast object to VirtualMachineInstance")
		return nil
	}

	if !c.clusterConfig.InPlaceResizeEnabledOnVMI(vmi) {
		c.logger.Infof("VMI %s/%s is not enabled for in-place resize", vmi.Namespace, vmi.Name)
		return nil
	}

	// Only consider pods which belong to this vmi
	// excluding unfinalized migration targets from this list.
	pod, err := controller.CurrentVMIPod(vmi, c.podIndexer)
	if err != nil {
		c.logger.Reason(err).Error("Failed to fetch pods for namespace from cache.")
		return err
	}

	err = c.sync(ctx, vmi, pod)
	if err != nil {
		c.logger.Reason(err).Error("Error syncing vmi")
		return err
	}

	return nil
}

func (c *Controller) sync(ctx context.Context, vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	vmi, err := c.syncPodResourceResizeCondition(ctx, vmi, pod)
	if err != nil {
		return fmt.Errorf("failed to sync pod resource resize condition: %v", err)
	}

	if c.vmiConditions.HasCondition(vmi, v1.VirtualMachineInstancePodResourceResizeInProgress) {
		c.logger.Infof("VMI %s/%s is being resized", vmi.Namespace, vmi.Name)
		return nil
	}

	cpuChanged := c.vmiConditions.HasConditionWithStatus(vmi, v1.VirtualMachineInstanceVCPUChange, k8sv1.ConditionTrue)
	memoryChanged := c.vmiConditions.HasConditionWithStatus(vmi, v1.VirtualMachineInstanceMemoryChange, k8sv1.ConditionTrue)

	needResize := cpuChanged || memoryChanged
	if needResize {
		pod, err = c.handleResize(ctx, vmi, pod)
		if err != nil {
			return fmt.Errorf("failed to handle resize: %v", err)
		}
	}

	return nil
}

func (c *Controller) handleResize(ctx context.Context, vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) (*k8sv1.Pod, error) {
	desiredPod, err := c.templateService.RenderLaunchManifest(vmi)
	if err != nil {
		return pod, fmt.Errorf("failed to render launch manifest: %v", err)
	}

	if len(desiredPod.Spec.Containers) != len(pod.Spec.Containers) {
		return pod, fmt.Errorf("number of containers changed")
	}

	return c.resizePod(ctx, pod, desiredPod, isDynamicCoresHotplug(vmi), cpuChanged(vmi))
}

const computeContainerName = "d8v-compute"

func isComputeContainer(ctr k8sv1.Container) bool {
	return ctr.Name == computeContainerName
}

func (c *Controller) resizePod(ctx context.Context, currentPod *k8sv1.Pod, desiredPod *k8sv1.Pod, isDynamicCoresHotplug, cpuChanged bool) (*k8sv1.Pod, error) {
	currentComputeIndex := slices.IndexFunc(currentPod.Spec.Containers, isComputeContainer)
	desiredComputeIndex := slices.IndexFunc(desiredPod.Spec.Containers, isComputeContainer)

	if currentComputeIndex == -1 || desiredComputeIndex == -1 {
		return currentPod, fmt.Errorf("could not find compute container")
	}

	oldRequests := currentPod.Spec.Containers[currentComputeIndex].Resources.Requests
	oldLimits := currentPod.Spec.Containers[currentComputeIndex].Resources.Limits

	newRequests := desiredPod.Spec.Containers[desiredComputeIndex].Resources.Requests
	newLimits := desiredPod.Spec.Containers[desiredComputeIndex].Resources.Limits

	needUpdate := false

	if !equality.Semantic.DeepEqual(newRequests, oldRequests) {
		if newRequests.Cpu().Value() < oldRequests.Cpu().Value() && !isDynamicCoresHotplug {
			return currentPod, fmt.Errorf("cannot resize cpu to less than current cpu")
		}
		if newRequests.Memory().Value() < oldRequests.Memory().Value() && !cpuChanged {
			return currentPod, fmt.Errorf("cannot resize memory to less than current memory")
		}

		currentPod.Spec.Containers[currentComputeIndex].Resources.Requests = newRequests
		needUpdate = true
	}

	if !equality.Semantic.DeepEqual(newLimits, oldLimits) {
		if newLimits.Cpu().Value() < oldLimits.Cpu().Value() && !isDynamicCoresHotplug {
			return currentPod, fmt.Errorf("cannot resize cpu to less than current cpu")
		}
		if newLimits.Memory().Value() < oldLimits.Memory().Value() && !cpuChanged {
			return currentPod, fmt.Errorf("cannot resize memory to less than current memory")
		}

		currentPod.Spec.Containers[currentComputeIndex].Resources.Limits = newLimits
		needUpdate = true
	}

	if needUpdate {
		newPod, err := c.clientset.CoreV1().Pods(currentPod.Namespace).UpdateResize(ctx, currentPod.Name, currentPod, metav1.UpdateOptions{})
		if err != nil {
			return currentPod, fmt.Errorf("failed to resize pod: %v", err)
		}
		return newPod, nil
	}

	return currentPod, nil
}

func (c *Controller) syncPodResourceResizeCondition(ctx context.Context, vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) (*v1.VirtualMachineInstance, error) {
	podResizePendingCond := c.podConditions.GetCondition(pod, k8sv1.PodResizePending)
	podResizeInProgressCond := c.podConditions.GetCondition(pod, k8sv1.PodResizeInProgress)
	podResizePending := podResizePendingCond != nil
	podResizeInProgress := podResizeInProgressCond != nil

	condExists := c.vmiConditions.HasCondition(vmi, v1.VirtualMachineInstancePodResourceResizeInProgress)

	oldConditions := vmi.Status.DeepCopy().Conditions
	oldLabels := maps.Clone(vmi.Labels)
	newLabels := maps.Clone(vmi.Labels)

	if newLabels == nil {
		newLabels = map[string]string{}
	}

	switch {
	// if podResizePending and podResizeInProgress are False but the condition exists, that means pod resize completed
	case !(podResizePending || podResizeInProgress) && condExists:
		c.vmiConditions.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
			Type:    v1.VirtualMachineInstancePodResourceResizeInProgress,
			Status:  k8sv1.ConditionTrue,
			Reason:  v1.VirtualMachineInstanceReasonPodResizeCompleted,
			Message: "Pod resize is completed",
		})

		if c.vmiConditions.HasConditionWithStatus(vmi, v1.VirtualMachineInstanceMemoryChange, k8sv1.ConditionTrue) {
			memoryReq, err := services.GetPodMemoryRequests(pod)
			if err != nil {
				return vmi, fmt.Errorf("failed to get pod memory requests: %v", err)
			}
			newLabels[v1.VirtualMachinePodMemoryRequestsLabel] = memoryReq
		}

		if c.vmiConditions.HasConditionWithStatus(vmi, v1.VirtualMachineInstanceVCPUChange, k8sv1.ConditionTrue) {
			if vmi.IsCPUDedicated() {
				cpuLimitsCount, err := services.GetPodCPULimitsCount(pod)
				if err != nil {
					return vmi, fmt.Errorf("failed to get pod cpu limits: %v", err)
				}
				newLabels[v1.VirtualMachinePodCPULimitsLabel] = fmt.Sprintf("%d", cpuLimitsCount)
			} else {
				delete(newLabels, v1.VirtualMachinePodCPULimitsLabel)
			}
		}

	case podResizePending:
		c.vmiConditions.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
			Type:    v1.VirtualMachineInstancePodResourceResizeInProgress,
			Status:  k8sv1.ConditionTrue,
			Reason:  v1.VirtualMachineInstanceReasonPodResizePending,
			Message: fmt.Sprintf("Pod resize is pending, current resize request: %s", podResizePendingCond.Message),
		})

	case podResizeInProgress:
		c.vmiConditions.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
			Type:    v1.VirtualMachineInstancePodResourceResizeInProgress,
			Status:  k8sv1.ConditionTrue,
			Reason:  v1.VirtualMachineInstanceReasonPodResizeInProgress,
			Message: fmt.Sprintf("Pod resize is in progress, current resize request: %s", podResizeInProgressCond.Message),
		})
	}

	if equality.Semantic.DeepEqual(oldConditions, vmi.Status.Conditions) {
		if equality.Semantic.DeepEqual(oldLabels, newLabels) {
			return vmi, nil
		}
	}

	log.Log.V(4).Infof("updating vmi pod resize condition to %v", vmi.Status.Conditions)

	patchSet := patch.New()
	if !equality.Semantic.DeepEqual(oldConditions, vmi.Status.Conditions) {
		patchSet.AddOption(
			patch.WithTest("/status/conditions", oldConditions),
			patch.WithReplace("/status/conditions", vmi.Status.Conditions),
		)
	}
	if !equality.Semantic.DeepEqual(oldLabels, newLabels) {
		if oldLabels == nil {
			patchSet.AddOption(patch.WithAdd("/metadata/labels", newLabels))
		} else {
			patchSet.AddOption(
				patch.WithTest("/metadata/labels", oldLabels),
				patch.WithReplace("/metadata/labels", newLabels),
			)
		}
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return vmi, err
	}

	newVmi, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(ctx, vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return vmi, fmt.Errorf("failed to patch vmi conditions: %v", err)
	}
	return newVmi, nil
}

func isDynamicCoresHotplug(vmi *v1.VirtualMachineInstance) bool {
	_, isDynamicCoresHotplug := vmi.Annotations[v1.VCPUTopologyDynamicCoresAnnotation]
	return isDynamicCoresHotplug
}

func cpuChanged(vmi *v1.VirtualMachineInstance) bool {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	return vmiConditions.HasCondition(vmi, v1.VirtualMachineInstanceVCPUChange)
}
