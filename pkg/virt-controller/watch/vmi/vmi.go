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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package vmi

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/trace"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/descheduler"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vsock"
)

const (
	deleteNotifFailed        = "Failed to process delete notification"
	tombstoneGetObjectErrFmt = "couldn't get object from tombstone %+v"
)

func NewController(templateService services.TemplateService,
	vmiInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	migrationInformer cache.SharedIndexInformer,
	storageClassInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	dataVolumeInformer cache.SharedIndexInformer,
	storageProfileInformer cache.SharedIndexInformer,
	cdiInformer cache.SharedIndexInformer,
	cdiConfigInformer cache.SharedIndexInformer,
	clusterConfig *virtconfig.ClusterConfig,
	topologyHinter topology.Hinter,
	netAnnotationsGenerator annotationsGenerator,
	netStatusUpdater statusUpdater,
	netSpecValidator specValidator,
) (*Controller, error) {

	c := &Controller{
		templateService: templateService,
		Queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-vmi"},
		),
		vmiIndexer:              vmiInformer.GetIndexer(),
		vmStore:                 vmInformer.GetStore(),
		podIndexer:              podInformer.GetIndexer(),
		pvcIndexer:              pvcInformer.GetIndexer(),
		migrationIndexer:        migrationInformer.GetIndexer(),
		recorder:                recorder,
		clientset:               clientset,
		podExpectations:         controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		vmiExpectations:         controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		pvcExpectations:         controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		dataVolumeIndexer:       dataVolumeInformer.GetIndexer(),
		cdiStore:                cdiInformer.GetStore(),
		cdiConfigStore:          cdiConfigInformer.GetStore(),
		clusterConfig:           clusterConfig,
		topologyHinter:          topologyHinter,
		cidsMap:                 vsock.NewCIDsMap(),
		backendStorage:          backendstorage.NewBackendStorage(clientset, clusterConfig, storageClassInformer.GetStore(), storageProfileInformer.GetStore(), pvcInformer.GetIndexer()),
		netAnnotationsGenerator: netAnnotationsGenerator,
		updateNetworkStatus:     netStatusUpdater,
		validateNetworkSpec:     netSpecValidator,
	}

	c.hasSynced = func() bool {
		return vmInformer.HasSynced() && vmiInformer.HasSynced() && podInformer.HasSynced() &&
			dataVolumeInformer.HasSynced() && cdiConfigInformer.HasSynced() && cdiInformer.HasSynced() &&
			pvcInformer.HasSynced() && storageClassInformer.HasSynced() && storageProfileInformer.HasSynced()
	}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})
	if err != nil {
		return nil, err
	}

	_, err = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		DeleteFunc: c.onPodDelete,
		UpdateFunc: c.updatePod,
	})
	if err != nil {
		return nil, err
	}

	_, err = dataVolumeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDataVolume,
		DeleteFunc: c.deleteDataVolume,
		UpdateFunc: c.updateDataVolume,
	})
	if err != nil {
		return nil, err
	}

	_, err = pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPVC,
		UpdateFunc: c.updatePVC,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

type informalSyncError struct {
	err    error
	reason string
}

func (i informalSyncError) Error() string {
	return i.err.Error()
}

func (i informalSyncError) Reason() string {
	return i.reason
}

func (i informalSyncError) RequiresRequeue() bool {
	return false
}

type annotationsGenerator interface {
	GenerateFromActivePod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) map[string]string
}

type statusUpdater func(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error

type specValidator func(*k8sfield.Path, *virtv1.VirtualMachineInstanceSpec, *virtconfig.ClusterConfig) []v1.StatusCause

type Controller struct {
	templateService         services.TemplateService
	clientset               kubecli.KubevirtClient
	Queue                   workqueue.TypedRateLimitingInterface[string]
	vmiIndexer              cache.Indexer
	vmStore                 cache.Store
	podIndexer              cache.Indexer
	pvcIndexer              cache.Indexer
	migrationIndexer        cache.Indexer
	topologyHinter          topology.Hinter
	recorder                record.EventRecorder
	podExpectations         *controller.UIDTrackingControllerExpectations
	vmiExpectations         *controller.UIDTrackingControllerExpectations
	pvcExpectations         *controller.UIDTrackingControllerExpectations
	dataVolumeIndexer       cache.Indexer
	cdiStore                cache.Store
	cdiConfigStore          cache.Store
	clusterConfig           *virtconfig.ClusterConfig
	cidsMap                 vsock.Allocator
	backendStorage          *backendstorage.BackendStorage
	hasSynced               func() bool
	netAnnotationsGenerator annotationsGenerator
	updateNetworkStatus     statusUpdater
	validateNetworkSpec     specValidator
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting vmi controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// Sync the CIDs from exist VMIs
	var vmis []*virtv1.VirtualMachineInstance
	for _, obj := range c.vmiIndexer.List() {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		vmis = append(vmis, vmi)
	}
	c.cidsMap.Sync(vmis)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping vmi controller.")
}

func (c *Controller) runWorker() {
	for c.Execute() {
	}
}

var virtControllerVMIWorkQueueTracer = &traceUtils.Tracer{Threshold: time.Second}

func (c *Controller) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	virtControllerVMIWorkQueueTracer.StartTrace(key, "virt-controller VMI workqueue", trace.Field{Key: "Workqueue Key", Value: key})
	defer virtControllerVMIWorkQueueTracer.StopTrace(key)

	defer c.Queue.Done(key)
	err := c.execute(key)

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *Controller) execute(key string) error {

	// Fetch the latest Vm state from cache
	obj, exists, err := c.vmiIndexer.GetByKey(key)

	if err != nil {
		return err
	}

	// Once all finalizers are removed the vmi gets deleted and we can clean all expectations
	if !exists {
		c.podExpectations.DeleteExpectations(key)
		c.vmiExpectations.DeleteExpectations(key)
		c.cidsMap.Remove(key)
		return nil
	}
	vmi := obj.(*virtv1.VirtualMachineInstance)

	logger := log.Log.Object(vmi)

	// this must be first step in execution. Writing the object
	// when api version changes ensures our api stored version is updated.
	if !controller.ObservedLatestApiVersionAnnotation(vmi) {
		vmi := vmi.DeepCopy()
		controller.SetLatestApiVersionAnnotation(vmi)
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err = c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmi, v1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
		return nil
	}

	// If needsSync is true (expectations fulfilled) we can make save assumptions if virt-handler or virt-controller owns the pod
	needsSync := c.podExpectations.SatisfiedExpectations(key) && c.vmiExpectations.SatisfiedExpectations(key) && c.pvcExpectations.SatisfiedExpectations(key)

	if !needsSync {
		return nil
	}

	// Only consider pods which belong to this vmi
	// excluding unfinalized migration targets from this list.
	pod, err := controller.CurrentVMIPod(vmi, c.podIndexer)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch pods for namespace from cache.")
		return err
	}

	// Get all dataVolumes associated with this vmi
	dataVolumes, err := storagetypes.ListDataVolumesFromVolumes(vmi.Namespace, vmi.Spec.Volumes, c.dataVolumeIndexer, c.pvcIndexer)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch dataVolumes for namespace from cache.")
		return err
	}

	syncErr, pod := c.sync(vmi, pod, dataVolumes)

	err = c.updateStatus(vmi, pod, dataVolumes, syncErr)
	if err != nil {
		return err
	}

	if syncErr != nil && syncErr.RequiresRequeue() {
		return syncErr
	}

	return nil
}

// These "dynamic" labels are Pod labels which may diverge from the VMI over time that we want to keep in sync.
func (c *Controller) syncDynamicLabelsToPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	patchSet := patch.New()

	dynamicLabels := []string{
		virtv1.NodeNameLabel,
		virtv1.OutdatedLauncherImageLabel,
	}

	podMeta := pod.ObjectMeta.DeepCopy()
	if podMeta.Labels == nil {
		podMeta.Labels = map[string]string{}
	}

	changed := false
	for _, key := range dynamicLabels {
		vmiVal, vmiLabelExists := vmi.Labels[key]
		podVal, podLabelExists := podMeta.Labels[key]
		if vmiLabelExists == podLabelExists && vmiVal == podVal {
			continue
		}

		changed = true
		if !vmiLabelExists {
			delete(podMeta.Labels, key)
		} else {
			podMeta.Labels[key] = vmiVal
		}
	}

	if !changed {
		return nil
	}

	if pod.ObjectMeta.Labels == nil {
		patchSet.AddOption(patch.WithAdd("/metadata/labels", podMeta.Labels))
	} else {
		patchSet.AddOption(
			patch.WithTest("/metadata/labels", pod.ObjectMeta.Labels),
			patch.WithReplace("/metadata/labels", podMeta.Labels),
		)
	}

	if patchSet.IsEmpty() {
		return nil
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}

	if _, err := c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.Background(), pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{}); err != nil {
		log.Log.Object(pod).Errorf("failed to sync dynamic pod labels during sync: %v", err)
		return err
	}
	return nil
}

func (c *Controller) syncPodAnnotations(pod *k8sv1.Pod, newAnnotations map[string]string) (*k8sv1.Pod, error) {
	patchSet := patch.New()
	for key, newValue := range newAnnotations {
		if podAnnotationValue, keyExist := pod.Annotations[key]; !keyExist || podAnnotationValue != newValue {
			patchSet.AddOption(
				patch.WithAdd(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(key)), newValue),
			)
		}
	}
	if patchSet.IsEmpty() {
		return pod, nil
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return pod, fmt.Errorf("failed to generate patch payload: %w", err)
	}

	patchedPod, err := c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.Background(), pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		log.Log.Object(pod).Errorf("failed to sync pod annotations during sync: %v", err)
		return nil, err
	}
	return patchedPod, nil
}

func (c *Controller) setLauncherContainerInfo(vmi *virtv1.VirtualMachineInstance, curPodImage string) *virtv1.VirtualMachineInstance {

	if curPodImage != "" && curPodImage != c.templateService.GetLauncherImage() {
		if vmi.Labels == nil {
			vmi.Labels = map[string]string{}
		}
		vmi.Labels[virtv1.OutdatedLauncherImageLabel] = ""
	} else {
		if vmi.Labels != nil {
			delete(vmi.Labels, virtv1.OutdatedLauncherImageLabel)
		}
	}

	vmi.Status.LauncherContainerImageVersion = curPodImage

	return vmi

}

func (c *Controller) hasOwnerVM(vmi *virtv1.VirtualMachineInstance) bool {
	controllerRef := v1.GetControllerOf(vmi)
	if controllerRef == nil || controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return false
	}

	obj, exists, _ := c.vmStore.GetByKey(controller.NamespacedKey(vmi.Namespace, controllerRef.Name))
	if !exists {
		return false
	}

	ownerVM := obj.(*virtv1.VirtualMachine)
	return controllerRef.UID == ownerVM.UID
}

func (c *Controller) updateStatus(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume, syncErr common.SyncError) error {
	key := controller.VirtualMachineInstanceKey(vmi)
	defer virtControllerVMIWorkQueueTracer.StepTrace(key, "updateStatus", trace.Field{Key: "VMI Name", Value: vmi.Name})

	hasFailedDataVolume := storagetypes.HasFailedDataVolumes(dataVolumes)

	hasWffcDataVolume := false
	// there is no reason to check for waitForFirstConsumer is there are failed DV's
	if !hasFailedDataVolume {
		hasWffcDataVolume = storagetypes.HasWFFCDataVolumes(dataVolumes)
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()
	podConditionManager := controller.NewPodConditionManager()
	vmiCopy := vmi.DeepCopy()
	vmiPodExists := controller.PodExists(pod) && !isTempPod(pod)
	tempPodExists := controller.PodExists(pod) && isTempPod(pod)

	vmiCopy, err := c.setActivePods(vmiCopy)
	if err != nil {
		return fmt.Errorf("Error detecting vmi pods: %v", err)
	}

	c.syncReadyConditionFromPod(vmiCopy, pod)
	if vmiPodExists {
		var foundImage string
		for _, container := range pod.Spec.Containers {
			if container.Name == "compute" {
				foundImage = container.Image
				break
			}
		}
		vmiCopy = c.setLauncherContainerInfo(vmiCopy, foundImage)

		if err := c.syncPausedConditionToPod(vmiCopy, pod); err != nil {
			return fmt.Errorf("error syncing paused condition to pod: %v", err)
		}

		if err := c.syncDynamicLabelsToPod(vmiCopy, pod); err != nil {
			return fmt.Errorf("error syncing labels to pod: %v", err)
		}
	}

	aggregateDataVolumesConditions(vmiCopy, dataVolumes)

	if pvc := backendstorage.PVCForVMI(c.pvcIndexer, vmi); pvc != nil {
		c.backendStorage.UpdateVolumeStatus(vmiCopy, pvc)
	}

	switch {
	case vmi.IsUnprocessed():
		if vmiPodExists {
			vmiCopy.Status.Phase = virtv1.Scheduling
		} else if vmi.DeletionTimestamp != nil || hasFailedDataVolume {
			vmiCopy.Status.Phase = virtv1.Failed
		} else {
			vmiCopy.Status.Phase = virtv1.Pending
			if vmi.Status.TopologyHints == nil {
				if topologyHints, tscRequirement, err := c.topologyHinter.TopologyHintsForVMI(vmi); err != nil && tscRequirement == topology.RequiredForBoot {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedGatherhingClusterTopologyHints, err.Error())
					return common.NewSyncError(err, controller.FailedGatherhingClusterTopologyHints)
				} else if topologyHints != nil {
					vmiCopy.Status.TopologyHints = topologyHints
				}
			}
			if hasWffcDataVolume {
				condition := virtv1.VirtualMachineInstanceCondition{
					Type:   virtv1.VirtualMachineInstanceProvisioning,
					Status: k8sv1.ConditionTrue,
				}
				if !conditionManager.HasCondition(vmiCopy, condition.Type) {
					vmiCopy.Status.Conditions = append(vmiCopy.Status.Conditions, condition)
				}
				if tempPodExists {
					// Add PodScheduled False condition to the VM
					if podConditionManager.HasConditionWithStatus(pod, k8sv1.PodScheduled, k8sv1.ConditionFalse) {
						conditionManager.AddPodCondition(vmiCopy, podConditionManager.GetCondition(pod, k8sv1.PodScheduled))
					} else if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)) {
						// Remove PodScheduling condition from the VM
						conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
					}
					if controller.IsPodFailedOrGoingDown(pod) {
						vmiCopy.Status.Phase = virtv1.Failed
					}
				}
			}
			if syncErr != nil &&
				(syncErr.Reason() == controller.FailedPvcNotFoundReason) {
				condition := virtv1.VirtualMachineInstanceCondition{
					Type:    virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
					Reason:  k8sv1.PodReasonUnschedulable,
					Message: syncErr.Error(),
					Status:  k8sv1.ConditionFalse,
				}
				cm := controller.NewVirtualMachineInstanceConditionManager()
				if cm.HasCondition(vmiCopy, condition.Type) {
					cm.RemoveCondition(vmiCopy, condition.Type)
				}
				vmiCopy.Status.Conditions = append(vmiCopy.Status.Conditions, condition)
			}
		}
	case vmi.IsScheduling():
		// Remove InstanceProvisioning condition from the VM
		if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceProvisioning) {
			conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceProvisioning)
		}
		if vmiPodExists {
			// ensure that the QOS class on the VMI matches to Pods QOS class
			if pod.Status.QOSClass == "" {
				vmiCopy.Status.QOSClass = nil
			} else {
				vmiCopy.Status.QOSClass = &pod.Status.QOSClass
			}

			// Add PodScheduled False condition to the VM
			if podConditionManager.HasConditionWithStatus(pod, k8sv1.PodScheduled, k8sv1.ConditionFalse) {
				conditionManager.AddPodCondition(vmiCopy, podConditionManager.GetCondition(pod, k8sv1.PodScheduled))
			} else if conditionManager.HasCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled)) {
				// Remove PodScheduling condition from the VM
				conditionManager.RemoveCondition(vmiCopy, virtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
			}

			if imageErr := checkForContainerImageError(pod); imageErr != nil {
				// only overwrite syncErr if imageErr != nil
				syncErr = imageErr
			}

			if controller.IsPodReady(pod) && vmi.DeletionTimestamp == nil {
				// fail vmi creation if CPU pinning has been requested but the Pod QOS is not Guaranteed
				podQosClass := pod.Status.QOSClass
				if podQosClass != k8sv1.PodQOSGuaranteed && vmi.IsCPUDedicated() {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedGuaranteePodResourcesReason, "failed to guarantee pod resources")
					syncErr = common.NewSyncError(fmt.Errorf("failed to guarantee pod resources"), controller.FailedGuaranteePodResourcesReason)
					break
				}

				// Initialize the volume status field with information
				// about the PVCs that the VMI is consuming. This prevents
				// virt-handler from needing to make API calls to GET the pvc
				// during reconcile
				if err := c.updateVolumeStatus(vmiCopy, pod); err != nil {
					return err
				}

				if err := c.updateNetworkStatus(vmiCopy, pod); err != nil {
					log.Log.Errorf("failed to update the interface status: %v", err)
				}

				// vmi is still owned by the controller but pod is already ready,
				// so let's hand over the vmi too
				vmiCopy.Status.Phase = virtv1.Scheduled
				if vmiCopy.Labels == nil {
					vmiCopy.Labels = map[string]string{}
				}
				vmiCopy.ObjectMeta.Labels[virtv1.NodeNameLabel] = pod.Spec.NodeName
				vmiCopy.Status.NodeName = pod.Spec.NodeName

				// Set the VMI migration transport now before the VMI can be migrated
				// This status field is needed to support the migration of legacy virt-launchers
				// to newer ones. In an absence of this field on the vmi, the target launcher
				// will set up a TCP proxy, as expected by a legacy virt-launcher.
				if shouldSetMigrationTransport(pod) {
					vmiCopy.Status.MigrationTransport = virtv1.MigrationTransportUnix
				}

				// Allocate the CID if VSOCK is enabled.
				if util.IsAutoAttachVSOCK(vmiCopy) {
					if err := c.cidsMap.Allocate(vmiCopy); err != nil {
						return err
					}
				}
			} else if controller.IsPodDownOrGoingDown(pod) {
				vmiCopy.Status.Phase = virtv1.Failed
			}
		} else {
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
			if vmiCopy.Labels != nil {
				delete(vmiCopy.Labels, virtv1.OutdatedLauncherImageLabel)
			}
			vmiCopy.Status.LauncherContainerImageVersion = ""
		}

		if !c.hasOwnerVM(vmi) && len(vmiCopy.Finalizers) > 0 {
			// if there's no owner VM around still, then remove the VM controller's finalizer if it exists
			controller.RemoveFinalizer(vmiCopy, virtv1.VirtualMachineControllerFinalizer)
		}

	case vmi.IsRunning():
		if !vmiPodExists {
			vmiCopy.Status.Phase = virtv1.Failed
			break
		}

		if err := c.updateVolumeStatus(vmiCopy, pod); err != nil {
			return err
		}

		if err := c.updateNetworkStatus(vmiCopy, pod); err != nil {
			log.Log.Errorf("failed to update the interface status: %v", err)
		}

		if c.requireCPUHotplug(vmiCopy) {
			syncHotplugCondition(vmiCopy, virtv1.VirtualMachineInstanceVCPUChange)
		}

		if c.requireMemoryHotplug(vmiCopy) {
			c.syncMemoryHotplug(vmiCopy)
		}

		if c.requireVolumesUpdate(vmiCopy) {
			c.syncVolumesUpdate(vmiCopy)
		}

	case vmi.IsScheduled():
		if !vmiPodExists {
			vmiCopy.Status.Phase = virtv1.Failed
		}
	default:
		return fmt.Errorf("unknown vmi phase %v", vmi.Status.Phase)
	}

	// VMI is owned by virt-handler, so patch instead of update
	if vmi.IsRunning() || vmi.IsScheduled() {
		patchSet := prepareVMIPatch(vmi, vmiCopy)
		if patchSet.IsEmpty() {
			return nil
		}
		patchBytes, err := patchSet.GeneratePayload()
		if err != nil {
			return fmt.Errorf("error preparing VMI patch: %v", err)
		}

		_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
		// We could not retry if the "test" fails but we have no sane way to detect that right now: https://github.com/kubernetes/kubernetes/issues/68202 for details
		// So just retry like with any other errors
		if err != nil {
			return fmt.Errorf("patching of vmi conditions and activePods failed: %v", err)
		}

		return nil
	}

	reason := ""
	if syncErr != nil {
		reason = syncErr.Reason()
	}

	conditionManager.CheckFailure(vmiCopy, syncErr, reason)

	controller.SetVMIPhaseTransitionTimestamp(vmi, vmiCopy)

	// If we detect a change on the vmi we update the vmi
	vmiChanged := !equality.Semantic.DeepEqual(vmi.Status, vmiCopy.Status) || !equality.Semantic.DeepEqual(vmi.Finalizers, vmiCopy.Finalizers) || !equality.Semantic.DeepEqual(vmi.Annotations, vmiCopy.Annotations) || !equality.Semantic.DeepEqual(vmi.Labels, vmiCopy.Labels)
	if vmiChanged {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Update(context.Background(), vmiCopy, v1.UpdateOptions{})
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	return nil
}

func preparePodPatch(oldPod, newPod *k8sv1.Pod) *patch.PatchSet {
	podConditions := controller.NewPodConditionManager()
	if podConditions.ConditionsEqual(oldPod, newPod) {
		return patch.New()
	}

	return patch.New(
		patch.WithTest("/status/conditions", oldPod.Status.Conditions),
		patch.WithReplace("/status/conditions", newPod.Status.Conditions),
	)
}

func prepareVMIPatch(oldVMI, newVMI *virtv1.VirtualMachineInstance) *patch.PatchSet {
	patchSet := patch.New()

	if !equality.Semantic.DeepEqual(newVMI.Status.VolumeStatus, oldVMI.Status.VolumeStatus) {
		// VolumeStatus changed which means either removed or added volumes.
		if oldVMI.Status.VolumeStatus == nil {
			patchSet.AddOption(patch.WithAdd("/status/volumeStatus", newVMI.Status.VolumeStatus))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/volumeStatus", oldVMI.Status.VolumeStatus),
				patch.WithReplace("/status/volumeStatus", newVMI.Status.VolumeStatus),
			)
		}
		log.Log.V(3).Object(oldVMI).Infof("Patching Volume Status")
	}
	// We don't own the object anymore, so patch instead of update
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	if !vmiConditions.ConditionsEqual(oldVMI, newVMI) {
		patchSet.AddOption(
			patch.WithTest("/status/conditions", oldVMI.Status.Conditions),
			patch.WithReplace("/status/conditions", newVMI.Status.Conditions),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching VMI conditions")
	}

	if !equality.Semantic.DeepEqual(newVMI.Status.ActivePods, oldVMI.Status.ActivePods) {
		patchSet.AddOption(
			patch.WithTest("/status/activePods", oldVMI.Status.ActivePods),
			patch.WithReplace("/status/activePods", newVMI.Status.ActivePods),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching VMI activePods")
	}

	if newVMI.Status.Phase != oldVMI.Status.Phase {
		patchSet.AddOption(
			patch.WithTest("/status/phase", oldVMI.Status.Phase),
			patch.WithReplace("/status/phase", newVMI.Status.Phase),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching VMI phase")
	}

	if newVMI.Status.LauncherContainerImageVersion != oldVMI.Status.LauncherContainerImageVersion {
		if oldVMI.Status.LauncherContainerImageVersion == "" {
			patchSet.AddOption(patch.WithAdd("/status/launcherContainerImageVersion", newVMI.Status.LauncherContainerImageVersion))
		} else {
			patchSet.AddOption(
				patch.WithTest("/status/launcherContainerImageVersion", oldVMI.Status.LauncherContainerImageVersion),
				patch.WithReplace("/status/launcherContainerImageVersion", newVMI.Status.LauncherContainerImageVersion),
			)
		}
	}

	if !equality.Semantic.DeepEqual(oldVMI.Labels, newVMI.Labels) {
		if oldVMI.Labels == nil {
			patchSet.AddOption(patch.WithAdd("/metadata/labels", newVMI.Labels))
		} else {
			patchSet.AddOption(
				patch.WithTest("/metadata/labels", oldVMI.Labels),
				patch.WithReplace("/metadata/labels", newVMI.Labels),
			)
		}
	}

	if !equality.Semantic.DeepEqual(newVMI.Status.Interfaces, oldVMI.Status.Interfaces) {
		patchSet.AddOption(
			patch.WithTest("/status/interfaces", oldVMI.Status.Interfaces),
			patch.WithAdd("/status/interfaces", newVMI.Status.Interfaces),
		)
		log.Log.V(3).Object(oldVMI).Infof("Patching Interface Status")
	}

	return patchSet
}

func (c *Controller) syncReadyConditionFromPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	podConditions := controller.NewPodConditionManager()

	now := v1.Now()
	if pod == nil || isTempPod(pod) {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.PodNotExistsReason,
			Message:            "virt-launcher pod has not yet been scheduled",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})

	} else if controller.IsPodDownOrGoingDown(pod) {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.PodTerminatingReason,
			Message:            "virt-launcher pod is terminating",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})

	} else if !vmi.IsRunning() {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.GuestNotRunningReason,
			Message:            "Guest VM is not reported as running",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})

	} else if podReadyCond := podConditions.GetCondition(pod, k8sv1.PodReady); podReadyCond != nil {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             podReadyCond.Status,
			Reason:             podReadyCond.Reason,
			Message:            podReadyCond.Message,
			LastProbeTime:      podReadyCond.LastProbeTime,
			LastTransitionTime: podReadyCond.LastTransitionTime,
		})

	} else {
		vmiConditions.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceReady,
			Status:             k8sv1.ConditionFalse,
			Reason:             virtv1.PodConditionMissingReason,
			Message:            "virt-launcher pod is missing the Ready condition",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})
	}
}

func (c *Controller) syncPausedConditionToPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	podConditions := controller.NewPodConditionManager()

	podCopy := pod.DeepCopy()
	now := v1.Now()
	if vmiConditions.HasConditionWithStatus(vmi, virtv1.VirtualMachineInstancePaused, k8sv1.ConditionTrue) {
		if podConditions.HasConditionWithStatus(pod, virtv1.VirtualMachineUnpaused, k8sv1.ConditionTrue) {
			podConditions.UpdateCondition(podCopy, &k8sv1.PodCondition{
				Type:               virtv1.VirtualMachineUnpaused,
				Status:             k8sv1.ConditionFalse,
				Reason:             "Paused",
				Message:            "the virtual machine is paused",
				LastProbeTime:      now,
				LastTransitionTime: now,
			})
		}
	} else {
		if !podConditions.HasConditionWithStatus(pod, virtv1.VirtualMachineUnpaused, k8sv1.ConditionTrue) {
			podConditions.UpdateCondition(podCopy, &k8sv1.PodCondition{
				Type:               virtv1.VirtualMachineUnpaused,
				Status:             k8sv1.ConditionTrue,
				Reason:             "NotPaused",
				Message:            "the virtual machine is not paused",
				LastProbeTime:      now,
				LastTransitionTime: now,
			})
		}
	}

	// Patch pod
	patchSet := preparePodPatch(pod, podCopy)
	if patchSet.IsEmpty() {
		return nil
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return fmt.Errorf("error preparing pod patch: %v", err)
	}
	log.Log.V(3).Object(pod).Infof("Patching pod conditions")

	_, err = c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{}, "status")
	// We could not retry if the "test" fails but we have no sane way to detect that right now:
	// https://github.com/kubernetes/kubernetes/issues/68202 for details
	// So just retry like with any other errors
	if err != nil {
		log.Log.Object(pod).Errorf("Patching of pod conditions failed: %v", err)
		return fmt.Errorf("patching of pod conditions failed: %v", err)
	}

	return nil
}

// checkForContainerImageError checks if an error has occured while handling the image of any of the pod's containers
// (including init containers), and returns a syncErr with the details of the error, or nil otherwise.
func checkForContainerImageError(pod *k8sv1.Pod) common.SyncError {
	containerStatuses := append(append([]k8sv1.ContainerStatus{},
		pod.Status.InitContainerStatuses...),
		pod.Status.ContainerStatuses...)

	for _, containerStatus := range containerStatuses {
		if containerStatus.State.Waiting == nil {
			continue
		}

		reason := containerStatus.State.Waiting.Reason
		if reason == controller.ErrImagePullReason || reason == controller.ImagePullBackOffReason {
			return common.NewSyncError(fmt.Errorf(containerStatus.State.Waiting.Message), reason)

		}
	}

	return nil
}

func (c *Controller) sync(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) (common.SyncError, *k8sv1.Pod) {
	key := controller.VirtualMachineInstanceKey(vmi)
	defer virtControllerVMIWorkQueueTracer.StepTrace(key, "sync", trace.Field{Key: "VMI Name", Value: vmi.Name})

	if vmi.DeletionTimestamp != nil {
		err := c.deleteAllMatchingPods(vmi)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to delete pod: %v", err), controller.FailedDeletePodReason), pod
		}
		return nil, pod
	}

	if vmi.IsFinal() {
		err := c.deleteAllAttachmentPods(vmi)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to delete attachment pods: %v", err), controller.FailedHotplugSyncReason), pod
		}
		return nil, pod
	}

	if err := c.deleteOrphanedAttachmentPods(vmi); err != nil {
		log.Log.Reason(err).Errorf("failed to delete orphaned attachment pods %s: %v", controller.VirtualMachineInstanceKey(vmi), err)
		// do not return; just log the error
	}

	dataVolumesReady, isWaitForFirstConsumer, syncErr := c.areDataVolumesReady(vmi, dataVolumes)
	if syncErr != nil {
		return syncErr, pod
	}

	if !controller.PodExists(pod) {
		// If we came ever that far to detect that we already created a pod, we don't create it again
		if !vmi.IsUnprocessed() {
			return nil, pod
		}
		// let's check if we already have topology hints or if we are still waiting for them
		if vmi.Status.TopologyHints == nil && c.topologyHinter.IsTscFrequencyRequired(vmi) {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation until topology hints are set")
			return nil, pod
		}

		// ensure that all dataVolumes associated with the VMI are ready before creating the pod
		if !dataVolumesReady {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation while DataVolume populates or while we wait for PVCs to appear.")
			return nil, pod
		}

		// ensure the VMI doesn't have an unfinished migration before creating the pod
		activeMigration, err := migrations.ActiveMigrationExistsForVMI(c.migrationIndexer, vmi)
		if err != nil {
			return common.NewSyncError(err, controller.FailedCreatePodReason), pod
		}
		if activeMigration {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation because an active migration exists for the VMI.")
			// We still need to return an error to ensure the VMI gets re-enqueued
			return common.NewSyncError(fmt.Errorf("active migration exists"), controller.FailedCreatePodReason), pod
		}

		backendStoragePVCName, syncErr := c.handleBackendStorage(vmi)
		if syncErr != nil {
			return syncErr, pod
		}

		// If a backend-storage PVC was just created but not yet seen by the informer, give it time
		if !c.pvcExpectations.SatisfiedExpectations(key) {
			return nil, pod
		}

		backendStorageReady, err := c.backendStorage.IsPVCReady(vmi, backendStoragePVCName)
		if err != nil {
			return common.NewSyncError(err, controller.FailedBackendStorageProbeReason), pod
		}
		if !backendStorageReady {
			log.Log.V(2).Object(vmi).Infof("Delaying pod creation while backend storage populates.")
			return common.NewSyncError(fmt.Errorf("PVC pending"), controller.BackendStorageNotReadyReason), pod
		}

		var templatePod *k8sv1.Pod
		if isWaitForFirstConsumer {
			log.Log.V(3).Object(vmi).Infof("Scheduling temporary pod for WaitForFirstConsumer DV")
			templatePod, err = c.templateService.RenderLaunchManifestNoVm(vmi)
		} else {
			templatePod, err = c.templateService.RenderLaunchManifest(vmi)
		}
		if _, ok := err.(storagetypes.PvcNotFoundError); ok {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedPvcNotFoundReason, services.FailedToRenderLaunchManifestErrFormat, err)
			return &informalSyncError{fmt.Errorf(services.FailedToRenderLaunchManifestErrFormat, err), controller.FailedPvcNotFoundReason}, pod
		} else if err != nil {
			return common.NewSyncError(fmt.Errorf(services.FailedToRenderLaunchManifestErrFormat, err), controller.FailedCreatePodReason), pod
		}

		var validateErrors []error
		for _, cause := range c.validateNetworkSpec(k8sfield.NewPath("spec"), &vmi.Spec, c.clusterConfig) {
			validateErrors = append(validateErrors, errors.New(cause.String()))
		}
		if validateErr := errors.Join(validateErrors...); validateErrors != nil {
			return common.NewSyncError(fmt.Errorf("failed create validation: %v", validateErr), "FailedCreateValidation"), pod
		}

		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		pod, err := c.createPod(vmiKey, vmi.Namespace, templatePod)
		if k8serrors.IsForbidden(err) && strings.Contains(err.Error(), "violates PodSecurity") {
			psaErr := fmt.Errorf("failed to create pod for vmi %s/%s, it needs a privileged namespace to run: %w", vmi.GetNamespace(), vmi.GetName(), err)
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, services.FailedToRenderLaunchManifestErrFormat, psaErr)
			return common.NewSyncError(psaErr, controller.FailedCreatePodReason), nil
		}
		if err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, "Error creating pod: %v", err)
			return common.NewSyncError(fmt.Errorf("failed to create virtual machine pod: %v", err), controller.FailedCreatePodReason), nil
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulCreatePodReason, "Created virtual machine pod %s", pod.Name)
		return nil, pod
	}

	if !isWaitForFirstConsumer {
		err := c.cleanupWaitForFirstConsumerTemporaryPods(vmi, pod)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to clean up temporary pods: %v", err), controller.FailedHotplugSyncReason), pod
		}
	}

	if !isTempPod(pod) && controller.IsPodReady(pod) {
		newAnnotations := map[string]string{descheduler.EvictOnlyAnnotation: ""}
		maps.Copy(newAnnotations, c.netAnnotationsGenerator.GenerateFromActivePod(vmi, pod))

		patchedPod, err := c.syncPodAnnotations(pod, newAnnotations)
		if err != nil {
			return common.NewSyncError(err, controller.FailedPodPatchReason), pod
		}
		pod = patchedPod

		hotplugVolumes := controller.GetHotplugVolumes(vmi, pod)
		hotplugAttachmentPods, err := controller.AttachmentPods(pod, c.podIndexer)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to get attachment pods: %v", err), controller.FailedHotplugSyncReason), pod
		}

		if pod.DeletionTimestamp == nil && needsHandleHotplug(hotplugVolumes, hotplugAttachmentPods) {
			var hotplugSyncErr common.SyncError = nil
			hotplugSyncErr = c.handleHotplugVolumes(hotplugVolumes, hotplugAttachmentPods, vmi, pod, dataVolumes)
			if hotplugSyncErr != nil {
				if hotplugSyncErr.Reason() == controller.MissingAttachmentPodReason {
					// We are missing an essential hotplug pod. Delete all pods associated with the VMI.
					if err := c.deleteAllMatchingPods(vmi); err != nil {
						log.Log.Warningf("failed to deleted VMI %s pods: %v", vmi.GetUID(), err)
					}
				} else {
					return hotplugSyncErr, pod
				}
			}
		}
	}
	return nil, pod
}

// When a pod is created, enqueue the vmi that manages it and update its podExpectations.
func (c *Controller) addPod(obj interface{}) {
	pod := obj.(*k8sv1.Pod)

	if pod.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.onPodDelete(pod)
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
func (c *Controller) updatePod(old, cur interface{}) {
	curPod := cur.(*k8sv1.Pod)
	oldPod := old.(*k8sv1.Pod)
	if curPod.ResourceVersion == oldPod.ResourceVersion {
		// Periodic resync will send update events for all known pods.
		// Two different versions of the same pod will always have different RVs.
		return
	}

	if curPod.DeletionTimestamp != nil {
		labelChanged := !equality.Semantic.DeepEqual(curPod.Labels, oldPod.Labels)
		// having a pod marked for deletion is enough to count as a deletion expectation
		c.onPodDelete(curPod)
		if labelChanged {
			// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.onPodDelete(oldPod)
		}
		return
	}

	curControllerRef := controller.GetControllerOf(curPod)
	oldControllerRef := controller.GetControllerOf(oldPod)
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
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
}

// When a pod is deleted, enqueue the vmi that manages the pod and update its podExpectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (c *Controller) onPodDelete(obj interface{}) {
	pod, ok := obj.(*k8sv1.Pod)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf(tombstoneGetObjectErrFmt, obj)).Error(deleteNotifFailed)
			return
		}
		pod, ok = tombstone.Obj.(*k8sv1.Pod)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pod %#v", obj)).Error(deleteNotifFailed)
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

func (c *Controller) addVirtualMachineInstance(obj interface{}) {
	c.lowerVMIExpectation(obj)
	c.enqueueVirtualMachine(obj)
}

func (c *Controller) deleteVirtualMachineInstance(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a vmi in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf(tombstoneGetObjectErrFmt, obj)).Error(deleteNotifFailed)
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vmi %#v", obj)).Error(deleteNotifFailed)
			return
		}
	}
	c.lowerVMIExpectation(vmi)
	c.enqueueVirtualMachine(vmi)
}

func (c *Controller) updateVirtualMachineInstance(_, curr interface{}) {
	c.lowerVMIExpectation(curr)
	c.enqueueVirtualMachine(curr)
}

func (c *Controller) lowerVMIExpectation(curr interface{}) {
	key, err := controller.KeyFunc(curr)
	if err != nil {
		return
	}
	c.vmiExpectations.LowerExpectations(key, 1, 0)
}

func (c *Controller) enqueueVirtualMachine(obj interface{}) {
	logger := log.Log
	vmi := obj.(*virtv1.VirtualMachineInstance)
	key, err := controller.KeyFunc(vmi)
	if err != nil {
		logger.Object(vmi).Reason(err).Error("Failed to extract key from virtualmachine.")
		return
	}
	c.Queue.Add(key)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *Controller) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachineInstance {
	if controllerRef != nil && controllerRef.Kind == "Pod" {
		// This could be an attachment pod, look up the pod, and check if it is owned by a VMI.
		obj, exists, err := c.podIndexer.GetByKey(controller.NamespacedKey(namespace, controllerRef.Name))
		if err != nil {
			return nil
		}
		if !exists {
			return nil
		}
		pod, _ := obj.(*k8sv1.Pod)
		controllerRef = controller.GetControllerOf(pod)
	}
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it is nil or the wrong Kind.
	if controllerRef == nil || controllerRef.Kind != virtv1.VirtualMachineInstanceGroupVersionKind.Kind {
		return nil
	}
	vmi, exists, err := c.vmiIndexer.GetByKey(controller.NamespacedKey(namespace, controllerRef.Name))
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

func (c *Controller) allPodsDeleted(vmi *virtv1.VirtualMachineInstance) (bool, error) {
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

func (c *Controller) deleteAllMatchingPods(vmi *virtv1.VirtualMachineInstance) error {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return err
	}

	vmiKey := controller.VirtualMachineInstanceKey(vmi)

	for _, pod := range pods {
		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		if pod.DeletionTimestamp != nil && !isPodFinal(pod) {
			continue
		}

		if err = c.deletePod(vmiKey, pod, v1.DeleteOptions{}); err != nil {
			if !k8serrors.IsNotFound(err) { // Skip the warning if the pod was already deleted, as this is an expected condition.
				c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedDeletePodReason, "Failed to delete virtual machine pod %s", pod.Name)
				return err
			}
		} else {
			c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulDeletePodReason, "Deleted virtual machine pod %s", pod.Name)
		}
	}
	return nil
}

func isPodFinal(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed
}

// listPodsFromNamespace takes a namespace and returns all Pods from the pod cache which run in this namespace
func (c *Controller) listPodsFromNamespace(namespace string) ([]*k8sv1.Pod, error) {
	objs, err := c.podIndexer.ByIndex(cache.NamespaceIndex, namespace)
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

func (c *Controller) setActivePods(vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstance, error) {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return nil, err
	}

	activePods := make(map[types.UID]string)
	count := 0
	for _, pod := range pods {
		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		count++
		activePods[pod.UID] = pod.Spec.NodeName
	}

	if count == 0 && vmi.Status.ActivePods == nil {
		return vmi, nil
	}

	vmi.Status.ActivePods = activePods
	return vmi, nil

}

func isTempPod(pod *k8sv1.Pod) bool {
	_, ok := pod.Annotations[virtv1.EphemeralProvisioningObject]
	return ok
}

func shouldSetMigrationTransport(pod *k8sv1.Pod) bool {
	_, ok := pod.Annotations[virtv1.MigrationTransportUnixAnnotation]
	return ok
}

func (c *Controller) cleanupWaitForFirstConsumerTemporaryPods(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) error {
	triggerPods, err := c.waitForFirstConsumerTemporaryPods(vmi, virtLauncherPod)
	if err != nil {
		return err
	}

	return c.deleteRunningOrFinishedWFFCPods(vmi, triggerPods...)
}

func (c *Controller) deleteRunningOrFinishedWFFCPods(vmi *virtv1.VirtualMachineInstance, pods ...*k8sv1.Pod) error {
	for _, pod := range pods {
		err := c.deleteRunningFinishedOrFailedPod(vmi, pod)
		if err != nil && !k8serrors.IsNotFound(err) {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedDeletePodReason, "Failed to delete WaitForFirstConsumer temporary pod %s", pod.Name)
			return err
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulDeletePodReason, "Deleted WaitForFirstConsumer temporary pod %s", pod.Name)
	}
	return nil
}

func (c *Controller) deleteRunningFinishedOrFailedPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	zero := int64(0)
	if pod.Status.Phase == k8sv1.PodRunning || pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
		vmiKey := controller.VirtualMachineInstanceKey(vmi)

		return c.deletePod(vmiKey, pod, v1.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
	}
	return nil
}

func (c *Controller) waitForFirstConsumerTemporaryPods(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) ([]*k8sv1.Pod, error) {
	var temporaryPods []*k8sv1.Pod

	// Get all pods from the namespace
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return temporaryPods, err
	}

	for _, pod := range pods {
		// Cleanup candidates are temporary pods that are either controlled by the VMI or the virt launcher pod
		if !isTempPod(pod) {
			continue
		}

		if controller.IsControlledBy(pod, vmi) {
			temporaryPods = append(temporaryPods, pod)
		}

		if ownerRef := controller.GetControllerOf(pod); ownerRef != nil && ownerRef.UID == virtLauncherPod.UID {
			temporaryPods = append(temporaryPods, pod)
		}
	}

	return temporaryPods, nil
}

func (c *Controller) volumeStatusContainsVolumeAndPod(volumeStatus []virtv1.VolumeStatus, volume *virtv1.Volume) bool {
	for _, status := range volumeStatus {
		if status.Name == volume.Name && status.HotplugVolume != nil && status.HotplugVolume.AttachPodName != "" {
			return true
		}
	}
	return false
}

func (c *Controller) requireCPUHotplug(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.Status.CurrentCPUTopology == nil ||
		vmi.Spec.Domain.CPU == nil ||
		vmi.Spec.Domain.CPU.MaxSockets == 0 {
		return false
	}

	cpuTopoLogyFromStatus := &virtv1.CPU{
		Cores:   vmi.Status.CurrentCPUTopology.Cores,
		Sockets: vmi.Status.CurrentCPUTopology.Sockets,
		Threads: vmi.Status.CurrentCPUTopology.Threads,
	}

	return hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU) != hardware.GetNumberOfVCPUs(cpuTopoLogyFromStatus)
}

func (c *Controller) requireMemoryHotplug(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi.Status.Memory == nil ||
		vmi.Spec.Domain.Memory == nil ||
		vmi.Spec.Domain.Memory.Guest == nil ||
		vmi.Spec.Domain.Memory.MaxGuest == nil {
		return false
	}

	return vmi.Spec.Domain.Memory.Guest.Value() != vmi.Status.Memory.GuestRequested.Value()
}

func (c *Controller) syncMemoryHotplug(vmi *virtv1.VirtualMachineInstance) {
	syncHotplugCondition(vmi, virtv1.VirtualMachineInstanceMemoryChange)
	// store additionalGuestMemoryOverheadRatio
	overheadRatio := c.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio
	if overheadRatio != nil {
		if vmi.Labels == nil {
			vmi.Labels = map[string]string{}
		}
		vmi.Labels[virtv1.MemoryHotplugOverheadRatioLabel] = *overheadRatio
	}
}

func (c *Controller) requireVolumesUpdate(vmi *virtv1.VirtualMachineInstance) bool {
	if len(vmi.Status.MigratedVolumes) < 1 {
		return false
	}
	if controller.NewVirtualMachineInstanceConditionManager().HasCondition(vmi, virtv1.VirtualMachineInstanceVolumesChange) {
		return false
	}
	migVolsMap := make(map[string]string)
	for _, v := range vmi.Status.MigratedVolumes {
		migVolsMap[v.SourcePVCInfo.ClaimName] = v.DestinationPVCInfo.ClaimName
	}
	for _, v := range vmi.Spec.Volumes {
		claim := storagetypes.PVCNameFromVirtVolume(&v)
		if claim == "" {
			continue
		}
		if _, ok := migVolsMap[claim]; !ok {
			return true
		}
	}

	return false
}

func (c *Controller) syncVolumesUpdate(vmi *virtv1.VirtualMachineInstance) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	condition := virtv1.VirtualMachineInstanceCondition{
		Type:               virtv1.VirtualMachineInstanceVolumesChange,
		LastTransitionTime: v1.Now(),
		Status:             k8sv1.ConditionTrue,
		Message:            "migrate volumes",
	}
	vmiConditions.UpdateCondition(vmi, &condition)
}

func (c *Controller) deletePod(vmiKey string, pod *k8sv1.Pod, options v1.DeleteOptions) error {
	c.podExpectations.ExpectDeletions(vmiKey, []string{controller.PodKey(pod)})
	err := c.clientset.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, options)
	if err != nil {
		c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))

	}
	return err
}

func (c *Controller) createPod(key, namespace string, pod *k8sv1.Pod) (*k8sv1.Pod, error) {
	c.podExpectations.ExpectCreations(key, 1)
	pod, err := c.clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, v1.CreateOptions{})
	if err != nil {
		c.podExpectations.CreationObserved(key)
	}
	return pod, err
}
