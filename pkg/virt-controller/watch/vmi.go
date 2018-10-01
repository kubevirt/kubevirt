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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package watch

import (
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
)

// Reasons for vmi events
const (
	// FailedCreatePodReason is added in an event and in a vmi controller condition
	// when a pod for a vmi controller failed to be created.
	FailedCreatePodReason = "FailedCreate"
	// SuccessfulCreatePodReason is added in an event when a pod for a vmi controller
	// is successfully created.
	SuccessfulCreatePodReason = "SuccessfulCreate"
	// FailedDeletePodReason is added in an event and in a vmi controller condition
	// when a pod for a vmi controller failed to be deleted.
	FailedDeletePodReason = "FailedDelete"
	// SuccessfulDeletePodReason is added in an event when a pod for a vmi controller
	// is successfully deleted.
	SuccessfulDeletePodReason = "SuccessfulDelete"
	// FailedHandOverPodReason is added in an event and in a vmi controller condition
	// when transferring the pod ownership from the controller to virt-hander fails.
	FailedHandOverPodReason = "FailedHandOver"
	// SuccessfulHandOverPodReason is added in an event
	// when the pod ownership transfer from the controller to virt-hander succeeds.
	SuccessfulHandOverPodReason = "SuccessfulHandOver"

	// FailedDataVolumeImportReason is added in an event when a dynamically generated
	// dataVolume reaches the failed status phase.
	FailedDataVolumeImportReason = "FailedDataVolumeImport"
	// FailedDataVolumeCreateReason is added in an event when posting a dynamically
	// generated dataVolume to the cluster fails.
	FailedDataVolumeCreateReason = "FailedDataVolumeCreate"
	// FailedDataVolumeDeleteReason is added in an event when deleting a dynamically
	// generated dataVolume in the cluster fails.
	FailedDataVolumeDeleteReason = "FailedDataVolumeDelete"
	// SuccessfulDataVolumeCreateReason is added in an event when a dynamically generated
	// dataVolume is successfully created
	SuccessfulDataVolumeCreateReason = "SuccessfulDataVolumeCreate"
	// SuccessfulDataVolumeImportReason is added in an event when a dynamically generated
	// dataVolume is successfully imports its data
	SuccessfulDataVolumeImportReason = "SuccessfulDataVolumeImport"
	// SuccessfulDataVolumeDeleteReason is added in an event when a dynamically generated
	// dataVolume is successfully deleted
	SuccessfulDataVolumeDeleteReason = "SuccessfulDataVolumeDelete"
	// FailedGuaranteePodResourcesReason is added in an event and in a vmi controller condition
	// when a pod has been created without a Guaranteed resources.
	FailedGuaranteePodResourcesReason = "FailedGuaranteeResources"
	// FailedPvcNotFoundReason is added in an event
	// when a PVC for a volume was not found.
	FailedPvcNotFoundReason = "FailedPvcNotFound"
)

func NewVMIController(templateService services.TemplateService,
	vmiInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	configMapInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer) *VMIController {

	c := &VMIController{
		templateService:      templateService,
		Queue:                workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:          vmiInformer,
		podInformer:          podInformer,
		recorder:             recorder,
		clientset:            clientset,
		podExpectations:      controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		handoverExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		configMapInformer:    configMapInformer,
		dataVolumeInformer:   dataVolumeInformer,
	}

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		DeleteFunc: c.deletePod,
		UpdateFunc: c.updatePod,
	})

	c.dataVolumeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDataVolume,
		DeleteFunc: c.deleteDataVolume,
		UpdateFunc: c.updateDataVolume,
	})

	return c
}

type syncError interface {
	error
	Reason() string
}

type syncErrorImpl struct {
	err    error
	reason string
}

func (e *syncErrorImpl) Error() string {
	return e.err.Error()
}

func (e *syncErrorImpl) Reason() string {
	return e.reason
}

type VMIController struct {
	templateService      services.TemplateService
	clientset            kubecli.KubevirtClient
	Queue                workqueue.RateLimitingInterface
	vmiInformer          cache.SharedIndexInformer
	podInformer          cache.SharedIndexInformer
	recorder             record.EventRecorder
	podExpectations      *controller.UIDTrackingControllerExpectations
	handoverExpectations *controller.UIDTrackingControllerExpectations
	configMapInformer    cache.SharedIndexInformer
	dataVolumeInformer   cache.SharedIndexInformer
}

func (c *VMIController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting vmi controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced, c.podInformer.HasSynced, c.configMapInformer.HasSynced, c.dataVolumeInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping vmi controller.")
}

func (c *VMIController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMIController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *VMIController) execute(key string) error {

	// Fetch the latest Vm state from cache
	obj, exists, err := c.vmiInformer.GetStore().GetByKey(key)

	if err != nil {
		return err
	}

	// Once all finalizers are removed the vmi gets deleted and we can clean all expectations
	if !exists {
		c.podExpectations.DeleteExpectations(key)
		c.handoverExpectations.DeleteExpectations(key)
		return nil
	}
	vmi := obj.(*virtv1.VirtualMachineInstance)

	logger := log.Log.Object(vmi)

	// Get all pods from the namespace
	pods, err := c.listPodsFromNamespace(vmi.Namespace)

	if err != nil {
		logger.Reason(err).Error("Failed to fetch pods for namespace from cache.")
		return err
	}

	// Only consider pods which belong to this vmi
	pods, err = c.filterMatchingPods(vmi, pods)
	if err != nil {
		return err
	}

	if len(pods) > 1 {
		logger.Reason(fmt.Errorf("More than one pod detected")).Error("That should not be possible, will not requeue")
		return nil
	}

	// Get all dataVolumes associated with this vmi
	dataVolumes, err := c.listMatchingDataVolumes(vmi)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch dataVolumes for namespace from cache.")
		return err
	}

	// If needsSync is true (expectations fulfilled) we can make save assumptions if virt-handler or virt-controller owns the pod
	needsSync := c.podExpectations.SatisfiedExpectations(key) && c.handoverExpectations.SatisfiedExpectations(key)

	var syncErr syncError = nil
	if needsSync {
		syncErr = c.sync(vmi, pods, dataVolumes)
	}
	err = c.updateStatus(vmi, pods, dataVolumes, syncErr)
	if err != nil {
		return err
	}

	if syncErr != nil {
		return syncErr
	}

	return nil

}

func (c *VMIController) updateStatus(vmi *virtv1.VirtualMachineInstance, pods []*k8sv1.Pod, dataVolumes []*cdiv1.DataVolume, syncErr syncError) error {

	var pod *k8sv1.Pod = nil
	podExists := len(pods) > 0
	if podExists {
		pod = pods[0]
	}

	hasFailedDataVolume := false
	for _, dataVolume := range dataVolumes {
		if dataVolume.Status.Phase == cdiv1.Failed {
			hasFailedDataVolume = true
		}
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()
	vmiCopy := vmi.DeepCopy()

	switch {

	case vmi.IsUnprocessed():
		if podExists {
			vmiCopy.Status.Phase = virtv1.Scheduling
		} else if vmi.DeletionTimestamp != nil {
			vmiCopy.Status.Phase = virtv1.Failed
		} else if hasFailedDataVolume {
			vmiCopy.Status.Phase = virtv1.Failed
		} else {
			vmiCopy.Status.Phase = virtv1.Pending
			if syncErr != nil && syncErr.Reason() == FailedPvcNotFoundReason {
				condition := virtv1.VirtualMachineInstanceCondition{
					Type:    virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
					Reason:  k8sv1.PodReasonUnschedulable,
					Message: syncErr.Error(),
					Status:  k8sv1.ConditionFalse,
				}
				vmiCopy.Status.Conditions = append(vmiCopy.Status.Conditions, condition)
			}
		}
	case vmi.IsScheduling():
		switch {
		case podExists:
			// Add PodScheduled False condition to the VM
			if cond := conditionManager.GetPodCondition(pod, k8sv1.PodScheduled, k8sv1.ConditionFalse); cond != nil {
				conditionManager.AddPodCondition(vmiCopy, cond)
			} else if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)) {
				// Remove PodScheduling condition from the VM
				conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
			}
			if isPodOwnedByHandler(pod) {
				// vmi is still owned by the controller but pod is already handed over,
				// so let's hand over the vmi too
				vmiCopy.Status.Interfaces = []virtv1.VirtualMachineInstanceNetworkInterface{
					{
						IP: pod.Status.PodIP,
					},
				}
				vmiCopy.Status.Phase = virtv1.Scheduled
				if vmiCopy.Labels == nil {
					vmiCopy.Labels = map[string]string{}
				}
				vmiCopy.ObjectMeta.Labels[virtv1.NodeNameLabel] = pod.Spec.NodeName
				vmiCopy.Status.NodeName = pod.Spec.NodeName
			} else if isPodDownOrGoingDown(pod) {
				vmiCopy.Status.Phase = virtv1.Failed
			}
		case !podExists:
			// someone other than the controller deleted the pod unexpectedly
			vmiCopy.Status.Phase = virtv1.Failed
		}
	case vmi.IsFinal():
		if !podExists {
			controller.RemoveFinalizer(vmiCopy, virtv1.VirtualMachineInstanceFinalizer)
		}
	case vmi.IsRunning() || vmi.IsScheduled():
		// Don't process states where the vmi is clearly owned by virt-handler
		return nil
	default:
		return fmt.Errorf("unknown vmi phase %v", vmi.Status.Phase)
	}

	reason := ""
	if syncErr != nil {
		reason = syncErr.Reason()
	}

	conditionManager.CheckFailure(vmiCopy, syncErr, reason)

	// If we detect a change on the vmi we update the vmi
	if !reflect.DeepEqual(vmi.Status, vmiCopy.Status) ||
		!reflect.DeepEqual(vmi.Finalizers, vmiCopy.Finalizers) ||
		!reflect.DeepEqual(vmi.Annotations, vmiCopy.Annotations) {
		_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Update(vmiCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

func isPodReady(pod *k8sv1.Pod) bool {
	if isPodDownOrGoingDown(pod) {
		return false
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Ready == false {
			return false
		}
	}

	return pod.Status.Phase == k8sv1.PodRunning
}

func isPodDownOrGoingDown(pod *k8sv1.Pod) bool {
	return podIsDown(pod) || isComputeContainerDown(pod) || pod.DeletionTimestamp != nil
}

func isComputeContainerDown(pod *k8sv1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "compute" {
			return containerStatus.State.Terminated != nil
		}
	}
	return false
}

func podIsDown(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed
}

func (c *VMIController) sync(vmi *virtv1.VirtualMachineInstance, pods []*k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) (err syncError) {

	var pod *k8sv1.Pod = nil
	podExists := len(pods) > 0
	if podExists {
		pod = pods[0]
	}

	vmiKey := controller.VirtualMachineKey(vmi)

	if vmi.DeletionTimestamp != nil {
		if !podExists {
			return nil
		} else if pod.DeletionTimestamp == nil {
			c.podExpectations.ExpectDeletions(vmiKey, []string{controller.PodKey(pod)})
			err := c.clientset.CoreV1().Pods(vmi.Namespace).Delete(pod.Name, &v1.DeleteOptions{})
			if err != nil {
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDeletePodReason, "Error deleting pod: %v", err)
				c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))
				return &syncErrorImpl{fmt.Errorf("failed to delete virtual machine pod: %v", err), FailedDeletePodReason}
			}
			c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted virtual machine pod %s", pod.Name)
			return nil
		}
		return nil
	} else if vmi.IsFinal() {
		return nil
	}

	if !podExists {
		// If we came ever that far to detect that we already created a pod, we don't create it again
		if !vmi.IsUnprocessed() {
			return nil
		}

		// ensure that all dataVolumes associated with the VMI are ready before creating the pod
		dataVolumesReady, syncErr := c.handleSyncDataVolumes(vmi, dataVolumes)
		if syncErr != nil {
			return syncErr
		} else if !dataVolumesReady {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation while DataVolume populates")
			return nil
		}

		c.podExpectations.ExpectCreations(vmiKey, 1)
		templatePod, err := c.templateService.RenderLaunchManifest(vmi)
		if _, ok := err.(services.PvcNotFoundError); ok {
			return &syncErrorImpl{fmt.Errorf("failed to render launch manifest: %v", err), FailedPvcNotFoundReason}
		} else if err != nil {
			return &syncErrorImpl{fmt.Errorf("failed to render launch manifest: %v", err), FailedCreatePodReason}
		}
		pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(templatePod)
		if err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating pod: %v", err)
			c.podExpectations.CreationObserved(vmiKey)
			return &syncErrorImpl{fmt.Errorf("failed to create virtual machine pod: %v", err), FailedCreatePodReason}
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created virtual machine pod %s", pod.Name)
		return nil
	} else if isPodReady(pod) && !isPodOwnedByHandler(pod) {
		pod := pod.DeepCopy()

		// fail vmi creation if CPU pinning has been requested but the Pod QOS is not Guaranteed
		podQosClass := pod.Status.QOSClass
		if podQosClass != k8sv1.PodQOSGuaranteed && vmi.IsCPUDedicated() {
			c.handoverExpectations.CreationObserved(controller.VirtualMachineKey(vmi))
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedGuaranteePodResourcesReason, "failed to guarantee pod resources")
			return &syncErrorImpl{fmt.Errorf("failed to guarantee pod resources"), FailedGuaranteePodResourcesReason}
		}

		pod.Annotations[virtv1.OwnedByAnnotation] = "virt-handler"
		c.handoverExpectations.ExpectCreations(controller.VirtualMachineKey(vmi), 1)
		_, err := c.clientset.CoreV1().Pods(vmi.Namespace).Update(pod)
		if err != nil {
			c.handoverExpectations.CreationObserved(controller.VirtualMachineKey(vmi))
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedHandOverPodReason, "Error on handing over pod: %v", err)
			return &syncErrorImpl{fmt.Errorf("failed to hand over pod to virt-handler: %v", err), FailedHandOverPodReason}
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulHandOverPodReason, "Pod ownership transferred to the node %s", pod.Name)
	}
	return nil
}

func (c *VMIController) handleSyncDataVolumes(vmi *virtv1.VirtualMachineInstance, dataVolumes []*cdiv1.DataVolume) (bool, syncError) {

	ready := true

	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.DataVolume == nil {
			continue
		}

		exists := false
		for _, dataVolume := range dataVolumes {
			if dataVolume.Name == volume.VolumeSource.DataVolume.Name {
				exists = true
				if dataVolume.Status.Phase != cdiv1.Succeeded {
					log.Log.V(3).Object(vmi).Infof("DataVolume %s not ready. Phase=%s", dataVolume.Name, dataVolume.Status.Phase)
					ready = false
					if dataVolume.Status.Phase == cdiv1.Failed {
						c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDataVolumeImportReason, "DataVolume %s failed", dataVolume.Name)
						return ready, &syncErrorImpl{fmt.Errorf("DataVolume %s for volume %s failed", dataVolume.Name, volume.Name), FailedDataVolumeImportReason}
					}
				}
				break
			}
		}

		if !exists {
			log.Log.V(3).Object(vmi).Infof("DataVolume %s not found", volume.VolumeSource.DataVolume.Name)
			ready = false
		}
	}

	return ready, nil
}

func (c *VMIController) addDataVolume(obj interface{}) {
	dataVolume := obj.(*cdiv1.DataVolume)
	if dataVolume.DeletionTimestamp != nil {
		c.deleteDataVolume(dataVolume)
		return
	}

	vmis, err := c.listVMIsMatchingDataVolume(dataVolume.Namespace, dataVolume.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(dataVolume).Infof("DataVolume created for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}
func (c *VMIController) updateDataVolume(old, cur interface{}) {
	curDataVolume := cur.(*cdiv1.DataVolume)
	oldDataVolume := old.(*cdiv1.DataVolume)
	if curDataVolume.ResourceVersion == oldDataVolume.ResourceVersion {
		// Periodic resync will send update events for all known DataVolumes.
		// Two different versions of the same dataVolume will always
		// have different RVs.
		return
	}
	labelChanged := !reflect.DeepEqual(curDataVolume.Labels, oldDataVolume.Labels)
	if curDataVolume.DeletionTimestamp != nil {
		// having a DataVOlume marked for deletion is enough
		// to count as a deletion expectation
		c.deleteDataVolume(curDataVolume)
		if labelChanged {
			// we don't need to check the oldDataVolume.DeletionTimestamp
			// because DeletionTimestamp cannot be unset.
			c.deleteDataVolume(oldDataVolume)
		}
		return
	}

	vmis, err := c.listVMIsMatchingDataVolume(curDataVolume.Namespace, curDataVolume.Name)
	if err != nil {
		log.Log.V(4).Object(curDataVolume).Errorf("Error encountered during datavolume update: %v", err)
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(curDataVolume).Infof("DataVolume updated for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}
func (c *VMIController) deleteDataVolume(obj interface{}) {
	dataVolume, ok := obj.(*cdiv1.DataVolume)
	// When a delete is dropped, the relist will notice a dataVolume in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the dataVolume
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		dataVolume, ok = tombstone.Obj.(*cdiv1.DataVolume)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a dataVolume %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	vmis, err := c.listVMIsMatchingDataVolume(dataVolume.Namespace, dataVolume.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(dataVolume).Infof("DataVolume deleted for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// When a pod is created, enqueue the vmi that manages it and update its podExpectations.
func (c *VMIController) addPod(obj interface{}) {
	pod := obj.(*k8sv1.Pod)

	if pod.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.deletePod(pod)
		return
	}

	controllerRef := c.getControllerOf(pod)
	vmi := c.resolveControllerRef(pod.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	vmiKey, err := controller.KeyFunc(vmi)
	if err != nil {
		return
	}
	log.Log.V(4).Object(pod).Infof("Pod created")
	c.podExpectations.CreationObserved(vmiKey)
	c.enqueueVirtualMachine(vmi)
}

// When a pod is updated, figure out what vmi/s manage it and wake them
// up. If the labels of the pod have changed we need to awaken both the old
// and new vmi. old and cur must be *v1.Pod types.
func (c *VMIController) updatePod(old, cur interface{}) {
	curPod := cur.(*k8sv1.Pod)
	oldPod := old.(*k8sv1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	labelChanged := !reflect.DeepEqual(curPod.Labels, oldPod.Labels)
	if curPod.DeletionTimestamp != nil {
		// having a pod marked for deletion is enough to count as a deletion expectation
		c.deletePod(curPod)
		if labelChanged {
			// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deletePod(oldPod)
		}
		return
	}

	curControllerRef := c.getControllerOf(curPod)
	oldControllerRef := c.getControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vmi := c.resolveControllerRef(oldPod.Namespace, oldControllerRef); vmi != nil {
			c.checkHandOverExpectation(oldPod, vmi)
			c.enqueueVirtualMachine(vmi)
		}
	}

	vmi := c.resolveControllerRef(curPod.Namespace, curControllerRef)
	if vmi == nil {
		return
	}
	log.Log.V(4).Object(curPod).Infof("Pod updated")
	c.checkHandOverExpectation(curPod, vmi)
	c.enqueueVirtualMachine(vmi)
	return
}

// When a pod is deleted, enqueue the vmi that manages the pod and update its podExpectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (c *VMIController) deletePod(obj interface{}) {
	pod, ok := obj.(*k8sv1.Pod)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		pod, ok = tombstone.Obj.(*k8sv1.Pod)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pod %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}

	controllerRef := c.getControllerOf(pod)
	vmi := c.resolveControllerRef(pod.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	vmiKey, err := controller.KeyFunc(vmi)
	if err != nil {
		return
	}
	c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))
	c.checkHandOverExpectation(pod, vmi)
	c.enqueueVirtualMachine(vmi)
}

func (c *VMIController) addVirtualMachine(obj interface{}) {
	c.enqueueVirtualMachine(obj)
}

func (c *VMIController) deleteVirtualMachine(obj interface{}) {
	c.enqueueVirtualMachine(obj)
}

func (c *VMIController) updateVirtualMachine(old, curr interface{}) {
	c.enqueueVirtualMachine(curr)
}

func (c *VMIController) enqueueVirtualMachine(obj interface{}) {
	logger := log.Log
	vmi := obj.(*virtv1.VirtualMachineInstance)
	key, err := controller.KeyFunc(vmi)
	if err != nil {
		logger.Object(vmi).Reason(err).Error("Failed to extract key from virtualmachine.")
	}
	c.Queue.Add(key)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *VMIController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachineInstance {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VirtualMachineInstanceGroupVersionKind.Kind {
		return nil
	}
	vmi, exists, err := c.vmiInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if vmi.(*virtv1.VirtualMachineInstance).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return vmi.(*virtv1.VirtualMachineInstance)
}

// listPodsFromNamespace takes a namespace and returns all Pods from the pod cache which run in this namespace
func (c *VMIController) listVMIsMatchingDataVolume(namespace string, dataVolumeName string) ([]*virtv1.VirtualMachineInstance, error) {
	objs, err := c.vmiInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vmis := []*virtv1.VirtualMachineInstance{}
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		for _, volume := range vmi.Spec.Volumes {
			if volume.VolumeSource.DataVolume == nil {
				continue
			} else if volume.VolumeSource.DataVolume.Name != dataVolumeName {
				continue
			}
			vmis = append(vmis, vmi)
		}
	}
	return vmis, nil
}

func (c *VMIController) listMatchingDataVolumes(vmi *virtv1.VirtualMachineInstance) ([]*cdiv1.DataVolume, error) {

	dataVolumes := []*cdiv1.DataVolume{}
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.DataVolume == nil {
			continue
		}
		obj, exists, err := c.dataVolumeInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", vmi.Namespace, volume.VolumeSource.DataVolume.Name))

		if err != nil {
			return dataVolumes, err
		} else if exists {
			dataVolume := obj.(*cdiv1.DataVolume)
			dataVolumes = append(dataVolumes, dataVolume)
		}
	}

	return dataVolumes, nil
}

// listPodsFromNamespace takes a namespace and returns all Pods from the pod cache which run in this namespace
func (c *VMIController) listPodsFromNamespace(namespace string) ([]*k8sv1.Pod, error) {
	objs, err := c.podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	pods := []*k8sv1.Pod{}
	for _, obj := range objs {
		pod := obj.(*k8sv1.Pod)
		pods = append(pods, pod)
	}
	return pods, nil
}

func (c *VMIController) filterMatchingPods(vmi *virtv1.VirtualMachineInstance, pods []*k8sv1.Pod) ([]*k8sv1.Pod, error) {
	selector, err := v1.LabelSelectorAsSelector(&v1.LabelSelector{MatchLabels: map[string]string{virtv1.CreatedByLabel: string(vmi.UID), virtv1.AppLabel: "virt-launcher"}})
	if err != nil {
		return nil, err
	}
	matchingPods := []*k8sv1.Pod{}
	for _, pod := range pods {
		if selector.Matches(labels.Set(pod.ObjectMeta.Labels)) {
			matchingPods = append(matchingPods, pod)
		}
	}
	return matchingPods, nil
}

func isPodOwnedByHandler(pod *k8sv1.Pod) bool {
	if pod.Annotations != nil && pod.Annotations[virtv1.OwnedByAnnotation] == "virt-handler" {
		return true
	}
	return false
}

// checkHandOverExpectation checks if a pod is owned by virt-handler and marks the
// handover expectation as observed, if so.
func (c *VMIController) checkHandOverExpectation(pod *k8sv1.Pod, vmi *virtv1.VirtualMachineInstance) {
	if isPodOwnedByHandler(pod) {
		c.handoverExpectations.CreationObserved(controller.VirtualMachineKey(vmi))
	}
}

func (c *VMIController) getControllerOf(pod *k8sv1.Pod) *v1.OwnerReference {
	t := true
	return &v1.OwnerReference{
		Kind:               virtv1.VirtualMachineInstanceGroupVersionKind.Kind,
		Name:               pod.Annotations[virtv1.DomainAnnotation],
		UID:                types.UID(pod.Labels[virtv1.CreatedByLabel]),
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
}

func (c *VMIController) getControllerOfDataVolume(dataVolume *cdiv1.DataVolume) *v1.OwnerReference {
	t := true
	return &v1.OwnerReference{
		Kind:               virtv1.VirtualMachineInstanceGroupVersionKind.Kind,
		Name:               dataVolume.Annotations[virtv1.DomainAnnotation],
		UID:                types.UID(dataVolume.Annotations[virtv1.CreatedByLabel]),
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
}
