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
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
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
	// SuccessfulMigrationReason is added when a migration attempt completes successfully
	SuccessfulMigrationReason = "SuccessfulMigration"
	// FailedMigrationReason is added when a migration attempt fails
	FailedMigrationReason = "FailedMigration"
)

func NewVMIController(templateService services.TemplateService,
	vmiInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	configMapInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer) *VMIController {

	c := &VMIController{
		templateService:    templateService,
		Queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:        vmiInformer,
		podInformer:        podInformer,
		recorder:           recorder,
		clientset:          clientset,
		podExpectations:    controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		configMapInformer:  configMapInformer,
		dataVolumeInformer: dataVolumeInformer,
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
	templateService    services.TemplateService
	clientset          kubecli.KubevirtClient
	Queue              workqueue.RateLimitingInterface
	vmiInformer        cache.SharedIndexInformer
	podInformer        cache.SharedIndexInformer
	recorder           record.EventRecorder
	podExpectations    *controller.UIDTrackingControllerExpectations
	configMapInformer  cache.SharedIndexInformer
	dataVolumeInformer cache.SharedIndexInformer
}

func (c *VMIController) Run(threadiness int, stopCh <-chan struct{}) {
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
		return nil
	}
	vmi := obj.(*virtv1.VirtualMachineInstance)

	logger := log.Log.Object(vmi)

	// Only consider pods which belong to this vmi
	// excluding unfinalized migration targets from this list.
	pod, err := c.currentPod(vmi)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch pods for namespace from cache.")
		return err
	}

	// Get all dataVolumes associated with this vmi
	dataVolumes, err := c.listMatchingDataVolumes(vmi)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch dataVolumes for namespace from cache.")
		return err
	}

	// If needsSync is true (expectations fulfilled) we can make save assumptions if virt-handler or virt-controller owns the pod
	needsSync := c.podExpectations.SatisfiedExpectations(key)

	var syncErr syncError = nil
	if needsSync {
		syncErr = c.sync(vmi, pod, dataVolumes)
	}
	err = c.updateStatus(vmi, pod, dataVolumes, syncErr)
	if err != nil {
		return err
	}

	if syncErr != nil {
		return syncErr
	}

	return nil

}

func (c *VMIController) updateStatus(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume, syncErr syncError) error {

	hasFailedDataVolume := false
	for _, dataVolume := range dataVolumes {
		if dataVolume.Status.Phase == cdiv1.Failed {
			hasFailedDataVolume = true
		}
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()
	vmiCopy := vmi.DeepCopy()
	podExists := podExists(pod)

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
			if cond := conditionManager.GetPodConditionWithStatus(pod, k8sv1.PodScheduled, k8sv1.ConditionFalse); cond != nil {
				conditionManager.AddPodCondition(vmiCopy, cond)
			} else if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)) {
				// Remove PodScheduling condition from the VM
				conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
			}
			if isPodReady(pod) && vmi.DeletionTimestamp == nil {
				// fail vmi creation if CPU pinning has been requested but the Pod QOS is not Guaranteed
				podQosClass := pod.Status.QOSClass
				if podQosClass != k8sv1.PodQOSGuaranteed && vmi.IsCPUDedicated() {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedGuaranteePodResourcesReason, "failed to guarantee pod resources")
					syncErr = &syncErrorImpl{fmt.Errorf("failed to guarantee pod resources"), FailedGuaranteePodResourcesReason}
				} else {

					// vmi is still owned by the controller but pod is already ready,
					// so let's hand over the vmi too
					interfaces := make([]virtv1.VirtualMachineInstanceNetworkInterface, 0)
					for _, network := range vmi.Spec.Networks {
						if network.NetworkSource.Pod != nil {
							ifc := virtv1.VirtualMachineInstanceNetworkInterface{Name: network.Name, IP: pod.Status.PodIP}
							interfaces = append(interfaces, ifc)
						}
					}
					vmiCopy.Status.Interfaces = interfaces

					vmiCopy.Status.Phase = virtv1.Scheduled
					if vmiCopy.Labels == nil {
						vmiCopy.Labels = map[string]string{}
					}
					vmiCopy.ObjectMeta.Labels[virtv1.NodeNameLabel] = pod.Spec.NodeName
					vmiCopy.Status.NodeName = pod.Spec.NodeName
				}
			} else if isPodDownOrGoingDown(pod) {
				vmiCopy.Status.Phase = virtv1.Failed
			}
		case !podExists:
			// someone other than the controller deleted the pod unexpectedly
			vmiCopy.Status.Phase = virtv1.Failed
		}
	case vmi.IsFinal():
		allDeleted, err := c.allPodsDeleted(vmi)
		if err != nil {
			return err
		}

		if allDeleted {
			log.Log.V(3).Object(vmi).Infof("All pods have been deleted, removing finalizer")
			controller.RemoveFinalizer(vmiCopy, virtv1.VirtualMachineInstanceFinalizer)
		}
	case vmi.IsRunning():
		// Keep PodReady condition in sync with the VMI
		if !podExists {
			// Remove PodScheduling condition from the VM
			conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
		} else if cond := conditionManager.GetPodCondition(pod, k8sv1.PodReady); cond != nil {
			conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
			conditionManager.AddPodCondition(vmiCopy, cond)
		} else if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodReady)) {
			// Remove PodScheduling condition from the VM
			conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodReady))
		}

		// We don't own the object anymore, so patch instead of update
		if !reflect.DeepEqual(vmiCopy.Status.Conditions, vmi.Status.Conditions) {
			newConditions, err := json.Marshal(vmiCopy.Status.Conditions)
			if err != nil {
				return err
			}
			oldConditions, err := json.Marshal(vmi.Status.Conditions)
			if err != nil {
				return err
			}
			test := fmt.Sprintf(`{ "op": "test", "path": "/status/conditions", "value": %s }`, string(oldConditions))
			patch := fmt.Sprintf(`{ "op": "replace", "path": "/status/conditions", "value": %s }`, string(newConditions))
			log.Log.V(3).Object(vmi).Infof("Patching VMI conditions")
			_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(vmi.Name, types.JSONPatchType, []byte(fmt.Sprintf("[ %s, %s ]", test, patch)))
			// We could not retry if the "test" fails but we have no sane way to detect that right now: https://github.com/kubernetes/kubernetes/issues/68202 for details
			// So just retry like with any other errors
			if err != nil {
				return fmt.Errorf("patching vmi conditions failed: %v", err)
			}
		}
		return nil
	case vmi.IsScheduled():
		// Don't process states where the vmi is clearly owned by virt-handler
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

// isPodReady treats the pod as ready to be handed over to virt-handler, as soon as all pods except
// the compute pod are ready. That includes kubevirt-infra and sidecars.
func isPodReady(pod *k8sv1.Pod) bool {
	if isPodDownOrGoingDown(pod) {
		return false
	}
	infraContainerExists := false
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "kubevirt-infra" {
			infraContainerExists = true
			break
		}
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		// If there is a kubevirt-infra pod, we should not check "compute" here.
		// Otherwise all containers including "compute" should be "ready before we go on.
		if infraContainerExists {
			if containerStatus.Name != "compute" && containerStatus.Ready == false {
				return false
			}
		} else {
			if containerStatus.Ready == false {
				return false
			}
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

func podExists(pod *k8sv1.Pod) bool {
	if pod != nil {
		return true
	}
	return false
}

func (c *VMIController) sync(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) (err syncError) {

	if vmi.DeletionTimestamp != nil {
		err := c.deleteAllMatchingPods(vmi)
		if err != nil {
			return &syncErrorImpl{fmt.Errorf("failed to delete pod: %v", err), FailedDeletePodReason}
		}
		return nil
	}

	if vmi.IsFinal() {
		return nil
	}

	if !podExists(pod) {
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

		templatePod, err := c.templateService.RenderLaunchManifest(vmi)
		if _, ok := err.(services.PvcNotFoundError); ok {
			return &syncErrorImpl{fmt.Errorf("failed to render launch manifest: %v", err), FailedPvcNotFoundReason}
		} else if err != nil {
			return &syncErrorImpl{fmt.Errorf("failed to render launch manifest: %v", err), FailedCreatePodReason}
		}

		vmiKey := controller.VirtualMachineKey(vmi)
		c.podExpectations.ExpectCreations(vmiKey, 1)
		pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(templatePod)
		if err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating pod: %v", err)
			c.podExpectations.CreationObserved(vmiKey)
			return &syncErrorImpl{fmt.Errorf("failed to create virtual machine pod: %v", err), FailedCreatePodReason}
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created virtual machine pod %s", pod.Name)
		return nil
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
	if curDataVolume.DeletionTimestamp != nil {
		labelChanged := !reflect.DeepEqual(curDataVolume.Labels, oldDataVolume.Labels)
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

	controllerRef := controller.GetControllerOf(pod)
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

	if curPod.DeletionTimestamp != nil {
		labelChanged := !reflect.DeepEqual(curPod.Labels, oldPod.Labels)
		// having a pod marked for deletion is enough to count as a deletion expectation
		c.deletePod(curPod)
		if labelChanged {
			// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deletePod(oldPod)
		}
		return
	}

	curControllerRef := controller.GetControllerOf(curPod)
	oldControllerRef := controller.GetControllerOf(oldPod)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vmi := c.resolveControllerRef(oldPod.Namespace, oldControllerRef); vmi != nil {
			c.enqueueVirtualMachine(vmi)
		}
	}

	vmi := c.resolveControllerRef(curPod.Namespace, curControllerRef)
	if vmi == nil {
		return
	}
	log.Log.V(4).Object(curPod).Infof("Pod updated")
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

	controllerRef := controller.GetControllerOf(pod)
	vmi := c.resolveControllerRef(pod.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	vmiKey, err := controller.KeyFunc(vmi)
	if err != nil {
		return
	}
	c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))
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
	// Don't even try to look up by Name if it is nil or the wrong Kind.
	if controllerRef == nil || controllerRef.Kind != virtv1.VirtualMachineInstanceGroupVersionKind.Kind {
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

// takes a namespace and returns all Pods from the pod cache which run in this namespace
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

func (c *VMIController) allPodsDeleted(vmi *virtv1.VirtualMachineInstance) (bool, error) {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return false, err
	}

	for _, pod := range pods {
		if controller.IsControlledBy(pod, vmi) {
			return false, nil
		}
	}

	return true, nil

}

func (c *VMIController) deleteAllMatchingPods(vmi *virtv1.VirtualMachineInstance) error {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return err
	}

	vmiKey := controller.VirtualMachineKey(vmi)

	for _, pod := range pods {
		if pod.DeletionTimestamp != nil {
			continue
		}

		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		c.podExpectations.ExpectDeletions(vmiKey, []string{controller.PodKey(pod)})
		err := c.clientset.CoreV1().Pods(vmi.Namespace).Delete(pod.Name, &v1.DeleteOptions{})
		if err != nil {
			c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDeletePodReason, "Failed to delete virtual machine pod %s", pod.Name)
			return err
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted virtual machine pod %s", pod.Name)
	}

	return nil
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

func (c *VMIController) currentPod(vmi *virtv1.VirtualMachineInstance) (*k8sv1.Pod, error) {

	// current pod is the most recent pod created on the current VMI node
	// OR the most recent pod created if no VMI node is set.

	// Get all pods from the namespace
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return nil, err
	}

	var curPod *k8sv1.Pod = nil
	for _, pod := range pods {
		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		if vmi.Status.NodeName != "" &&
			vmi.Status.NodeName != pod.Spec.NodeName {
			// This pod isn't scheduled to the current node.
			// This can occurr during the initial migration phases when
			// a new target node is being prepared for the VMI.
			continue
		}

		if curPod == nil {
			curPod = pod
		} else if curPod.CreationTimestamp.Before(&pod.CreationTimestamp) {
			curPod = pod
		}
	}

	return curPod, nil

}
