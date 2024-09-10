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
	"sort"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/virt-controller/network"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/vsock"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

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
	netadmitter "kubevirt.io/kubevirt/pkg/network/admitter"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/descheduler"
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
	storageClassInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	dataVolumeInformer cache.SharedIndexInformer,
	storageProfileInformer cache.SharedIndexInformer,
	cdiInformer cache.SharedIndexInformer,
	cdiConfigInformer cache.SharedIndexInformer,
	clusterConfig *virtconfig.ClusterConfig,
	topologyHinter topology.Hinter,
) (*Controller, error) {

	c := &Controller{
		templateService:   templateService,
		Queue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-vmi"),
		vmiIndexer:        vmiInformer.GetIndexer(),
		vmStore:           vmInformer.GetStore(),
		podIndexer:        podInformer.GetIndexer(),
		pvcIndexer:        pvcInformer.GetIndexer(),
		recorder:          recorder,
		clientset:         clientset,
		podExpectations:   controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		vmiExpectations:   controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		dataVolumeIndexer: dataVolumeInformer.GetIndexer(),
		cdiStore:          cdiInformer.GetStore(),
		cdiConfigStore:    cdiConfigInformer.GetStore(),
		clusterConfig:     clusterConfig,
		topologyHinter:    topologyHinter,
		cidsMap:           vsock.NewCIDsMap(),
		backendStorage:    backendstorage.NewBackendStorage(clientset, clusterConfig, storageClassInformer.GetStore(), storageProfileInformer.GetStore(), pvcInformer.GetIndexer()),
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
		DeleteFunc: c.deletePod,
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

type Controller struct {
	templateService   services.TemplateService
	clientset         kubecli.KubevirtClient
	Queue             workqueue.RateLimitingInterface
	vmiIndexer        cache.Indexer
	vmStore           cache.Store
	podIndexer        cache.Indexer
	pvcIndexer        cache.Indexer
	topologyHinter    topology.Hinter
	recorder          record.EventRecorder
	podExpectations   *controller.UIDTrackingControllerExpectations
	vmiExpectations   *controller.UIDTrackingControllerExpectations
	dataVolumeIndexer cache.Indexer
	cdiStore          cache.Store
	cdiConfigStore    cache.Store
	clusterConfig     *virtconfig.ClusterConfig
	cidsMap           vsock.Allocator
	backendStorage    *backendstorage.BackendStorage
	hasSynced         func() bool
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

	virtControllerVMIWorkQueueTracer.StartTrace(key.(string), "virt-controller VMI workqueue", trace.Field{Key: "Workqueue Key", Value: key})
	defer virtControllerVMIWorkQueueTracer.StopTrace(key.(string))

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
	needsSync := c.podExpectations.SatisfiedExpectations(key) && c.vmiExpectations.SatisfiedExpectations(key)

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

	syncErr := c.sync(vmi, pod, dataVolumes)

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
		if podAnnotationValue, keyExist := pod.Annotations[key]; !keyExist || (keyExist && podAnnotationValue != newValue) {
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

	obj, exists, _ := c.vmStore.GetByKey(vmi.Namespace + "/" + controllerRef.Name)
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

	c.aggregateDataVolumesConditions(vmiCopy, dataVolumes)

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
		switch {
		case vmiPodExists:
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

				if err := c.updateInterfaceStatus(vmiCopy, pod); err != nil {
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
		case !vmiPodExists:
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

		if err := c.updateInterfaceStatus(vmiCopy, pod); err != nil {
			log.Log.Errorf("failed to update the interface status: %v", err)
		}

		if c.requireCPUHotplug(vmiCopy) {
			c.syncHotplugCondition(vmiCopy, virtv1.VirtualMachineInstanceVCPUChange)
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

func (c *Controller) sync(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) common.SyncError {
	key := controller.VirtualMachineInstanceKey(vmi)
	defer virtControllerVMIWorkQueueTracer.StepTrace(key, "sync", trace.Field{Key: "VMI Name", Value: vmi.Name})

	if vmi.DeletionTimestamp != nil {
		err := c.deleteAllMatchingPods(vmi)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to delete pod: %v", err), controller.FailedDeletePodReason)
		}
		return nil
	}

	if vmi.IsFinal() {
		err := c.deleteAllAttachmentPods(vmi)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to delete attachment pods: %v", err), controller.FailedHotplugSyncReason)
		}
		return nil
	}

	if err := c.deleteOrphanedAttachmentPods(vmi); err != nil {
		log.Log.Reason(err).Errorf("failed to delete orphaned attachment pods %s: %v", controller.VirtualMachineInstanceKey(vmi), err)
		// do not return; just log the error
	}

	err := c.backendStorage.CreateIfNeededAndUpdateVolumeStatus(vmi)
	if err != nil {
		return common.NewSyncError(err, controller.FailedBackendStorageCreateReason)
	}

	dataVolumesReady, isWaitForFirstConsumer, syncErr := c.handleSyncDataVolumes(vmi, dataVolumes)
	if syncErr != nil {
		return syncErr
	}

	if !controller.PodExists(pod) {
		// If we came ever that far to detect that we already created a pod, we don't create it again
		if !vmi.IsUnprocessed() {
			return nil
		}
		// let's check if we already have topology hints or if we are still waiting for them
		if vmi.Status.TopologyHints == nil && c.topologyHinter.IsTscFrequencyRequired(vmi) {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation until topology hints are set")
			return nil
		}

		// ensure that all dataVolumes associated with the VMI are ready before creating the pod
		if !dataVolumesReady {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation while DataVolume populates or while we wait for PVCs to appear.")
			return nil
		}

		// Ensure the backend storage PVC is ready
		var backendStorageReady bool
		backendStorageReady, err = c.backendStorage.IsPVCReady(vmi)
		if err != nil {
			return common.NewSyncError(err, controller.FailedBackendStorageProbeReason)
		}
		if !backendStorageReady {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation while backend storage populates.")
			return common.NewSyncError(fmt.Errorf("PVC pending"), controller.BackendStorageNotReadyReason)
		}

		var templatePod *k8sv1.Pod
		var err error
		if isWaitForFirstConsumer {
			log.Log.V(3).Object(vmi).Infof("Scheduling temporary pod for WaitForFirstConsumer DV")
			templatePod, err = c.templateService.RenderLaunchManifestNoVm(vmi)
		} else {
			templatePod, err = c.templateService.RenderLaunchManifest(vmi)
		}
		if _, ok := err.(storagetypes.PvcNotFoundError); ok {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedPvcNotFoundReason, services.FailedToRenderLaunchManifestErrFormat, err)
			return &informalSyncError{fmt.Errorf(services.FailedToRenderLaunchManifestErrFormat, err), controller.FailedPvcNotFoundReason}
		} else if err != nil {
			return common.NewSyncError(fmt.Errorf(services.FailedToRenderLaunchManifestErrFormat, err), controller.FailedCreatePodReason)
		}

		netValidator := netadmitter.NewValidator(k8sfield.NewPath("spec"), &vmi.Spec, c.clusterConfig)
		var validateErrors []error
		for _, cause := range netValidator.ValidateCreation() {
			validateErrors = append(validateErrors, errors.New(cause.String()))
		}
		if validateErr := errors.Join(validateErrors...); validateErrors != nil {
			return common.NewSyncError(fmt.Errorf("failed create validation: %v", validateErr), "FailedCreateValidation")
		}

		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		c.podExpectations.ExpectCreations(vmiKey, 1)
		pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), templatePod, v1.CreateOptions{})
		if k8serrors.IsForbidden(err) && strings.Contains(err.Error(), "violates PodSecurity") {
			psaErr := fmt.Errorf("failed to create pod for vmi %s/%s, it needs a privileged namespace to run: %w", vmi.GetNamespace(), vmi.GetName(), err)
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, services.FailedToRenderLaunchManifestErrFormat, psaErr)
			return common.NewSyncError(psaErr, controller.FailedCreatePodReason)
		}
		if err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, "Error creating pod: %v", err)
			c.podExpectations.CreationObserved(vmiKey)
			return common.NewSyncError(fmt.Errorf("failed to create virtual machine pod: %v", err), controller.FailedCreatePodReason)
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulCreatePodReason, "Created virtual machine pod %s", pod.Name)
		return nil
	}

	if !isWaitForFirstConsumer {
		err := c.cleanupWaitForFirstConsumerTemporaryPods(vmi, pod)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to clean up temporary pods: %v", err), controller.FailedHotplugSyncReason)
		}
	}

	if !isTempPod(pod) && controller.IsPodReady(pod) {
		newAnnotations := map[string]string{descheduler.EvictOnlyAnnotation: ""}
		maps.Copy(newAnnotations, network.GeneratePodAnnotations(vmi.Spec.Networks, vmi.Spec.Domain.Devices.Interfaces, pod.Annotations[networkv1.NetworkStatusAnnot], c.clusterConfig.GetNetworkBindings()))

		patchedPod, err := c.syncPodAnnotations(pod, newAnnotations)
		if err != nil {
			return common.NewSyncError(err, controller.FailedPodPatchReason)
		}
		*pod = *patchedPod

		hotplugVolumes := controller.GetHotplugVolumes(vmi, pod)
		hotplugAttachmentPods, err := controller.AttachmentPods(pod, c.podIndexer)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("failed to get attachment pods: %v", err), controller.FailedHotplugSyncReason)
		}

		if pod.DeletionTimestamp == nil && c.needsHandleHotplug(hotplugVolumes, hotplugAttachmentPods) {
			var hotplugSyncErr common.SyncError = nil
			hotplugSyncErr = c.handleHotplugVolumes(hotplugVolumes, hotplugAttachmentPods, vmi, pod, dataVolumes)
			if hotplugSyncErr != nil {
				if hotplugSyncErr.Reason() == controller.MissingAttachmentPodReason {
					// We are missing an essential hotplug pod. Delete all pods associated with the VMI.
					if err := c.deleteAllMatchingPods(vmi); err != nil {
						log.Log.Warningf("failed to deleted VMI %s pods: %v", vmi.GetUID(), err)
					}
				} else {
					return hotplugSyncErr
				}
			}
		}

		if vmiSpecIfaces, vmiSpecNets, dynamicIfacesExist := network.CalculateInterfacesAndNetworksForMultusAnnotationUpdate(vmi); dynamicIfacesExist {
			if err := c.updateMultusAnnotation(vmi.Namespace, vmiSpecIfaces, vmiSpecNets, pod); err != nil {
				return common.NewSyncError(fmt.Errorf("failed to hot{un}plug network interfaces for vmi [%s/%s]: %w", vmi.GetNamespace(), vmi.GetName(), err), controller.FailedHotplugSyncReason)
			}
		}
	}
	return nil
}

func (c *Controller) handleSyncDataVolumes(vmi *virtv1.VirtualMachineInstance, dataVolumes []*cdiv1.DataVolume) (bool, bool, common.SyncError) {

	ready := true
	wffc := false

	for _, volume := range vmi.Spec.Volumes {
		// Check both DVs and PVCs
		if volume.VolumeSource.DataVolume != nil || volume.VolumeSource.PersistentVolumeClaim != nil {
			volumeReady, volumeWffc, err := storagetypes.VolumeReadyToAttachToNode(vmi.Namespace, volume, dataVolumes, c.dataVolumeIndexer, c.pvcIndexer)
			if err != nil {
				if _, ok := err.(storagetypes.PvcNotFoundError); ok {
					// due to the eventually consistent nature of controllers, CDI or users may need some time to actually crate the PVC.
					// We wait for them to appear.
					c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.FailedPvcNotFoundReason, "PVC %s/%s does not exist, waiting for it to appear", vmi.Namespace, storagetypes.PVCNameFromVirtVolume(&volume))
					return false, false, &informalSyncError{err: fmt.Errorf("PVC %s/%s does not exist, waiting for it to appear", vmi.Namespace, storagetypes.PVCNameFromVirtVolume(&volume)), reason: controller.FailedPvcNotFoundReason}
				} else {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedPvcNotFoundReason, "Error determining if volume is ready: %v", err)
					return false, false, common.NewSyncError(fmt.Errorf("Error determining if volume is ready %v", err), controller.FailedDataVolumeImportReason)
				}
			}
			wffc = wffc || volumeWffc
			// Ready only becomes false if WFFC is also false.
			ready = ready && (volumeReady || volumeWffc)
		}
	}

	return ready, wffc, nil
}

func (c *Controller) addPVC(obj interface{}) {
	pvc := obj.(*k8sv1.PersistentVolumeClaim)
	if pvc.DeletionTimestamp != nil {
		return
	}

	vmis, err := c.listVMIsMatchingDV(pvc.Namespace, pvc.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(pvc).Infof("PVC created for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

func (c *Controller) updatePVC(old, cur interface{}) {
	curPVC := cur.(*k8sv1.PersistentVolumeClaim)
	oldPVC := old.(*k8sv1.PersistentVolumeClaim)
	if curPVC.ResourceVersion == oldPVC.ResourceVersion {
		// Periodic resync will send update events for all known PVCs.
		// Two different versions of the same PVC will always
		// have different RVs.
		return
	}
	if curPVC.DeletionTimestamp != nil {
		return
	}
	if equality.Semantic.DeepEqual(curPVC.Status.Capacity, oldPVC.Status.Capacity) {
		// We only do something when the capacity changes
		return
	}

	vmis, err := c.listVMIsMatchingDV(curPVC.Namespace, curPVC.Name)
	if err != nil {
		log.Log.Object(curPVC).Errorf("Error encountered getting VMIs for DataVolume: %v", err)
		return
	}

	for _, vmi := range vmis {
		log.Log.V(4).Object(curPVC).Infof("PVC updated for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

func (c *Controller) addDataVolume(obj interface{}) {
	dataVolume := obj.(*cdiv1.DataVolume)
	if dataVolume.DeletionTimestamp != nil {
		c.deleteDataVolume(dataVolume)
		return
	}

	vmis, err := c.listVMIsMatchingDV(dataVolume.Namespace, dataVolume.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(dataVolume).Infof("DataVolume created for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

func (c *Controller) updateDataVolume(old, cur interface{}) {
	curDataVolume := cur.(*cdiv1.DataVolume)
	oldDataVolume := old.(*cdiv1.DataVolume)
	if curDataVolume.ResourceVersion == oldDataVolume.ResourceVersion {
		// Periodic resync will send update events for all known DataVolumes.
		// Two different versions of the same dataVolume will always
		// have different RVs.
		return
	}
	if curDataVolume.DeletionTimestamp != nil {
		labelChanged := !equality.Semantic.DeepEqual(curDataVolume.Labels, oldDataVolume.Labels)
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

	vmis, err := c.listVMIsMatchingDV(curDataVolume.Namespace, curDataVolume.Name)
	if err != nil {
		log.Log.Object(curDataVolume).Errorf("Error encountered during datavolume update: %v", err)
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(curDataVolume).Infof("DataVolume updated for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}
func (c *Controller) deleteDataVolume(obj interface{}) {
	dataVolume, ok := obj.(*cdiv1.DataVolume)
	// When a delete is dropped, the relist will notice a dataVolume in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the dataVolume
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf(tombstoneGetObjectErrFmt, obj)).Error(deleteNotifFailed)
			return
		}
		dataVolume, ok = tombstone.Obj.(*cdiv1.DataVolume)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a dataVolume %#v", obj)).Error(deleteNotifFailed)
			return
		}
	}
	vmis, err := c.listVMIsMatchingDV(dataVolume.Namespace, dataVolume.Name)
	if err != nil {
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(dataVolume).Infof("DataVolume deleted for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// When a pod is created, enqueue the vmi that manages it and update its podExpectations.
func (c *Controller) addPod(obj interface{}) {
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
		c.deletePod(curPod)
		if labelChanged {
			// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deletePod(oldPod)
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
func (c *Controller) deletePod(obj interface{}) {
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
		obj, exists, err := c.podIndexer.GetByKey(namespace + "/" + controllerRef.Name)
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
	vmi, exists, err := c.vmiIndexer.GetByKey(namespace + "/" + controllerRef.Name)
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

func (c *Controller) listVMIsMatchingDV(namespace string, dvName string) ([]*virtv1.VirtualMachineInstance, error) {
	// TODO - refactor if/when dv/pvc do not have the same name
	vmis := []*virtv1.VirtualMachineInstance{}
	for _, indexName := range []string{"dv", "pvc"} {
		objs, err := c.vmiIndexer.ByIndex(indexName, namespace+"/"+dvName)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			vmi := obj.(*virtv1.VirtualMachineInstance)
			vmis = append(vmis, vmi.DeepCopy())
		}
	}
	return vmis, nil
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
		if pod.DeletionTimestamp != nil {
			continue
		}

		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		c.podExpectations.ExpectDeletions(vmiKey, []string{controller.PodKey(pod)})
		err := c.clientset.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), pod.Name, v1.DeleteOptions{})
		if err != nil {
			c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedDeletePodReason, "Failed to delete virtual machine pod %s", pod.Name)
			return err
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulDeletePodReason, "Deleted virtual machine pod %s", pod.Name)
	}
	return nil
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
		c.podExpectations.ExpectDeletions(vmiKey, []string{controller.PodKey(pod)})
		err := c.clientset.CoreV1().Pods(pod.GetNamespace()).Delete(context.Background(), pod.Name, v1.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if err != nil {
			c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(pod))
			return err
		}
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

func (c *Controller) needsHandleHotplug(hotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod) bool {
	if len(hotplugAttachmentPods) > 1 {
		return true
	}
	// Determine if the ready volumes have changed compared to the current pod
	if len(hotplugAttachmentPods) == 1 && c.podVolumesMatchesReadyVolumes(hotplugAttachmentPods[0], hotplugVolumes) {
		return false
	}
	return len(hotplugVolumes) > 0 || len(hotplugAttachmentPods) > 0
}

func (c *Controller) getActiveAndOldAttachmentPods(readyHotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod) (*k8sv1.Pod, []*k8sv1.Pod) {
	var currentPod *k8sv1.Pod
	oldPods := make([]*k8sv1.Pod, 0)
	for _, attachmentPod := range hotplugAttachmentPods {
		if !c.podVolumesMatchesReadyVolumes(attachmentPod, readyHotplugVolumes) {
			oldPods = append(oldPods, attachmentPod)
		} else {
			currentPod = attachmentPod
		}
	}
	sort.Slice(oldPods, func(i, j int) bool {
		return oldPods[i].CreationTimestamp.Time.After(oldPods[j].CreationTimestamp.Time)
	})
	return currentPod, oldPods
}

// cleanupAttachmentPods deletes all old attachment pods when the following is true
// 1. There is a currentPod that is running. (not nil and phase.Status == Running)
// 2. There are no readyVolumes (numReadyVolumes == 0)
// 3. The newest oldPod is not running and not marked for deletion.
// If any of those are true, it will not delete the newest oldPod, since that one is the latest
// pod that is closest to the desired state.
func (c *Controller) cleanupAttachmentPods(currentPod *k8sv1.Pod, oldPods []*k8sv1.Pod, vmi *virtv1.VirtualMachineInstance, numReadyVolumes int) common.SyncError {
	foundRunning := false
	for _, attachmentPod := range oldPods {
		if !foundRunning &&
			attachmentPod.Status.Phase == k8sv1.PodRunning &&
			attachmentPod.DeletionTimestamp == nil &&
			numReadyVolumes > 0 &&
			(currentPod == nil || currentPod.Status.Phase != k8sv1.PodRunning) {
			foundRunning = true
			continue
		}
		if err := c.deleteAttachmentPodForVolume(vmi, attachmentPod); err != nil {
			return common.NewSyncError(fmt.Errorf("Error deleting attachment pod %v", err), controller.FailedDeletePodReason)
		}
	}
	return nil
}

func hasPendingPods(pods []*k8sv1.Pod) bool {
	for _, pod := range pods {
		if pod.Status.Phase == k8sv1.PodRunning || pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
			continue
		}
		return true
	}
	return false
}

func (c *Controller) requeueAfter(oldPods []*k8sv1.Pod, threshold time.Duration) (bool, time.Duration) {
	if len(oldPods) > 0 && oldPods[0].CreationTimestamp.Time.After(time.Now().Add(-1*threshold)) {
		return true, threshold - time.Since(oldPods[0].CreationTimestamp.Time)
	}
	return false, 0
}

func (c *Controller) handleHotplugVolumes(hotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod, vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) common.SyncError {
	logger := log.Log.Object(vmi)

	readyHotplugVolumes := make([]*virtv1.Volume, 0)
	// Find all ready volumes
	for _, volume := range hotplugVolumes {
		var err error
		ready, wffc, err := storagetypes.VolumeReadyToAttachToNode(vmi.Namespace, *volume, dataVolumes, c.dataVolumeIndexer, c.pvcIndexer)
		if err != nil {
			return common.NewSyncError(fmt.Errorf("Error determining volume status %v", err), controller.PVCNotReadyReason)
		}
		if wffc {
			// Volume in WaitForFirstConsumer, it has not been populated by CDI yet. create a dummy pod
			logger.V(1).Infof("Volume %s/%s is in WaitForFistConsumer, triggering population", vmi.Namespace, volume.Name)
			syncError := c.triggerHotplugPopulation(volume, vmi, virtLauncherPod)
			if syncError != nil {
				return syncError
			}
			continue
		}
		if !ready {
			// Volume not ready, skip until it is.
			logger.V(3).Infof("Skipping hotplugged volume: %s, not ready", volume.Name)
			continue
		}
		readyHotplugVolumes = append(readyHotplugVolumes, volume)
	}

	currentPod, oldPods := c.getActiveAndOldAttachmentPods(readyHotplugVolumes, hotplugAttachmentPods)
	if currentPod == nil && !hasPendingPods(oldPods) && len(readyHotplugVolumes) > 0 {
		if rateLimited, waitTime := c.requeueAfter(oldPods, time.Duration(len(readyHotplugVolumes)/-10)); rateLimited {
			key, err := controller.KeyFunc(vmi)
			if err != nil {
				logger.Object(vmi).Reason(err).Error("failed to extract key from virtualmachine.")
				return common.NewSyncError(fmt.Errorf("failed to extract key from virtualmachine. %v", err), controller.FailedHotplugSyncReason)
			}
			c.Queue.AddAfter(key, waitTime)
		} else {
			if newPod, err := c.createAttachmentPod(vmi, virtLauncherPod, readyHotplugVolumes); err != nil {
				return err
			} else {
				currentPod = newPod
			}
		}
	}
	if err := c.cleanupAttachmentPods(currentPod, oldPods, vmi, len(readyHotplugVolumes)); err != nil {
		return err
	}

	return nil
}

func (c *Controller) podVolumesMatchesReadyVolumes(attachmentPod *k8sv1.Pod, volumes []*virtv1.Volume) bool {
	// -2 for empty dir and token
	if len(attachmentPod.Spec.Volumes)-2 != len(volumes) {
		return false
	}
	podVolumeMap := make(map[string]k8sv1.Volume)
	for _, volume := range attachmentPod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			podVolumeMap[volume.Name] = volume
		}
	}
	for _, volume := range volumes {
		delete(podVolumeMap, volume.Name)
	}
	return len(podVolumeMap) == 0
}

func (c *Controller) createAttachmentPod(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, volumes []*virtv1.Volume) (*k8sv1.Pod, common.SyncError) {
	attachmentPodTemplate, _ := c.createAttachmentPodTemplate(vmi, virtLauncherPod, volumes)
	if attachmentPodTemplate == nil {
		return nil, nil
	}
	vmiKey := controller.VirtualMachineInstanceKey(vmi)
	c.podExpectations.ExpectCreations(vmiKey, 1)

	pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), attachmentPodTemplate, v1.CreateOptions{})
	if err != nil {
		c.podExpectations.CreationObserved(vmiKey)
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, "Error creating attachment pod: %v", err)
		return nil, common.NewSyncError(fmt.Errorf("Error creating attachment pod %v", err), controller.FailedCreatePodReason)
	}
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulCreatePodReason, "Created attachment pod %s", pod.Name)
	return pod, nil
}

func (c *Controller) triggerHotplugPopulation(volume *virtv1.Volume, vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) common.SyncError {
	populateHotplugPodTemplate, err := c.createAttachmentPopulateTriggerPodTemplate(volume, virtLauncherPod, vmi)
	if err != nil {
		return common.NewSyncError(fmt.Errorf("Error creating trigger pod template %v", err), controller.FailedCreatePodReason)
	}
	if populateHotplugPodTemplate != nil { // nil means the PVC is not populated yet.
		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		c.podExpectations.ExpectCreations(vmiKey, 1)

		_, err = c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), populateHotplugPodTemplate, v1.CreateOptions{})
		if err != nil {
			c.podExpectations.CreationObserved(vmiKey)
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedCreatePodReason, "Error creating hotplug population trigger pod for volume %s: %v", volume.Name, err)
			return common.NewSyncError(fmt.Errorf("Error creating hotplug population trigger pod %v", err), controller.FailedCreatePodReason)
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulCreatePodReason, "Created hotplug trigger pod for volume %s", volume.Name)
	}
	return nil
}

func (c *Controller) volumeStatusContainsVolumeAndPod(volumeStatus []virtv1.VolumeStatus, volume *virtv1.Volume) bool {
	for _, status := range volumeStatus {
		if status.Name == volume.Name && status.HotplugVolume != nil && status.HotplugVolume.AttachPodName != "" {
			return true
		}
	}
	return false
}

func (c *Controller) getNewHotplugVolumes(hotplugAttachmentPods []*k8sv1.Pod, hotplugVolumes []*virtv1.Volume) []*virtv1.Volume {
	var newVolumes []*virtv1.Volume
	hotplugVolumeMap := make(map[string]*virtv1.Volume)
	for _, volume := range hotplugVolumes {
		hotplugVolumeMap[volume.Name] = volume
	}
	// Remove all the volumes that we have a pod for.
	for _, pod := range hotplugAttachmentPods {
		for _, volume := range pod.Spec.Volumes {
			delete(hotplugVolumeMap, volume.Name)
		}
	}
	// Any remaining volumes are new.
	for _, v := range hotplugVolumeMap {
		newVolumes = append(newVolumes, v)
	}
	return newVolumes
}

func (c *Controller) getDeletedHotplugVolumes(hotplugPods []*k8sv1.Pod, hotplugVolumes []*virtv1.Volume) []k8sv1.Volume {
	var deletedVolumes []k8sv1.Volume
	hotplugVolumeMap := make(map[string]*virtv1.Volume)
	for _, volume := range hotplugVolumes {
		hotplugVolumeMap[volume.Name] = volume
	}
	for _, pod := range hotplugPods {
		for _, volume := range pod.Spec.Volumes {
			if _, ok := hotplugVolumeMap[volume.Name]; !ok && volume.PersistentVolumeClaim != nil {
				deletedVolumes = append(deletedVolumes, volume)
			}
		}
	}
	return deletedVolumes
}

func (c *Controller) deleteAttachmentPodForVolume(vmi *virtv1.VirtualMachineInstance, attachmentPod *k8sv1.Pod) error {
	vmiKey := controller.VirtualMachineInstanceKey(vmi)
	zero := int64(0)

	if attachmentPod.DeletionTimestamp != nil {
		return nil
	}

	c.podExpectations.ExpectDeletions(vmiKey, []string{controller.PodKey(attachmentPod)})
	err := c.clientset.CoreV1().Pods(attachmentPod.GetNamespace()).Delete(context.Background(), attachmentPod.Name, v1.DeleteOptions{
		GracePeriodSeconds: &zero,
	})
	if err != nil {
		c.podExpectations.DeletionObserved(vmiKey, controller.PodKey(attachmentPod))
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, controller.FailedDeletePodReason, "Failed to delete attachment pod %s", attachmentPod.Name)
		return err
	}
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.SuccessfulDeletePodReason, "Deleted attachment pod %s", attachmentPod.Name)
	return nil
}

func (c *Controller) createAttachmentPodTemplate(vmi *virtv1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod, volumes []*virtv1.Volume) (*k8sv1.Pod, error) {
	logger := log.Log.Object(vmi)
	var pod *k8sv1.Pod
	var err error

	volumeNamesPVCMap, err := storagetypes.VirtVolumesToPVCMap(volumes, c.pvcIndexer, virtlauncherPod.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get PVC map: %v", err)
	}
	for volumeName, pvc := range volumeNamesPVCMap {
		//Verify the PVC is ready to be used.
		populated, err := cdiv1.IsSucceededOrPendingPopulation(pvc, func(name, namespace string) (*cdiv1.DataVolume, error) {
			dv, exists, _ := c.dataVolumeIndexer.GetByKey(fmt.Sprintf("%s/%s", namespace, name))
			if !exists {
				return nil, fmt.Errorf("unable to find datavolume %s/%s", namespace, name)
			}
			return dv.(*cdiv1.DataVolume), nil
		})
		if err != nil {
			return nil, err
		}
		if !populated {
			logger.Infof("Unable to hotplug, claim %s found, but not ready", pvc.Name)
			delete(volumeNamesPVCMap, volumeName)
		}
	}

	if len(volumeNamesPVCMap) > 0 {
		pod, err = c.templateService.RenderHotplugAttachmentPodTemplate(volumes, virtlauncherPod, vmi, volumeNamesPVCMap)
	}
	return pod, err
}

func (c *Controller) createAttachmentPopulateTriggerPodTemplate(volume *virtv1.Volume, virtlauncherPod *k8sv1.Pod, vmi *virtv1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	claimName := storagetypes.PVCNameFromVirtVolume(volume)
	if claimName == "" {
		return nil, errors.New("Unable to hotplug, claim not PVC or Datavolume")
	}

	pvc, exists, isBlock, err := storagetypes.IsPVCBlockFromStore(c.pvcIndexer, virtlauncherPod.Namespace, claimName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Unable to trigger hotplug population, claim %s not found", claimName)
	}
	pod, err := c.templateService.RenderHotplugAttachmentTriggerPodTemplate(volume, virtlauncherPod, vmi, pvc.Name, isBlock, true)
	return pod, err
}

func (c *Controller) deleteAllAttachmentPods(vmi *virtv1.VirtualMachineInstance) error {
	virtlauncherPod, err := controller.CurrentVMIPod(vmi, c.podIndexer)
	if err != nil {
		return err
	}
	if virtlauncherPod != nil {
		attachmentPods, err := controller.AttachmentPods(virtlauncherPod, c.podIndexer)
		if err != nil {
			return err
		}
		for _, attachmentPod := range attachmentPods {
			err := c.deleteAttachmentPodForVolume(vmi, attachmentPod)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) deleteOrphanedAttachmentPods(vmi *virtv1.VirtualMachineInstance) error {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return fmt.Errorf("failed to list pods from namespace %s: %v", vmi.Namespace, err)
	}

	for _, pod := range pods {
		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		if !controller.PodIsDown(pod) {
			continue
		}

		attachmentPods, err := controller.AttachmentPods(pod, c.podIndexer)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get attachment pods %s: %v", controller.PodKey(pod), err)
			// do not return; continue the cleanup...
			continue
		}

		for _, attachmentPod := range attachmentPods {
			if err := c.deleteAttachmentPodForVolume(vmi, attachmentPod); err != nil {
				log.Log.Reason(err).Errorf("failed to delete attachment pod %s: %v", controller.PodKey(attachmentPod), err)
				// do not return; continue the cleanup...
			}
		}
	}

	return nil
}

func (c *Controller) updateVolumeStatus(vmi *virtv1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod) error {
	oldStatus := vmi.Status.DeepCopy().VolumeStatus
	oldStatusMap := make(map[string]virtv1.VolumeStatus)
	for _, status := range oldStatus {
		oldStatusMap[status.Name] = status
	}

	hotplugVolumes := controller.GetHotplugVolumes(vmi, virtlauncherPod)
	hotplugVolumesMap := make(map[string]*virtv1.Volume)
	for _, volume := range hotplugVolumes {
		hotplugVolumesMap[volume.Name] = volume
	}

	attachmentPods, err := controller.AttachmentPods(virtlauncherPod, c.podIndexer)
	if err != nil {
		return err
	}

	attachmentPod, _ := c.getActiveAndOldAttachmentPods(hotplugVolumes, attachmentPods)

	newStatus := make([]virtv1.VolumeStatus, 0)

	if backendStorage, ok := oldStatusMap[backendstorage.PVCForVMI(vmi)]; ok {
		newStatus = append(newStatus, backendStorage)
	}

	for i, volume := range vmi.Spec.Volumes {
		status := virtv1.VolumeStatus{}
		if _, ok := oldStatusMap[volume.Name]; ok {
			// Already have the status, modify if needed
			status = oldStatusMap[volume.Name]
		} else {
			status.Name = volume.Name
		}
		// Remove from map so I can detect existing volumes that have been removed from spec.
		delete(oldStatusMap, volume.Name)
		if _, ok := hotplugVolumesMap[volume.Name]; ok {
			// Hotplugged volume
			if status.HotplugVolume == nil {
				status.HotplugVolume = &virtv1.HotplugVolumeStatus{}
			}
			if volume.MemoryDump != nil && status.MemoryDumpVolume == nil {
				status.MemoryDumpVolume = &virtv1.DomainMemoryDumpInfo{
					ClaimName: volume.Name,
				}
			}
			if attachmentPod == nil {
				if !c.volumeReady(status.Phase) {
					status.HotplugVolume.AttachPodUID = ""
					// Volume is not hotplugged in VM and Pod is gone, or hasn't been created yet, check for the PVC associated with the volume to set phase and message
					phase, reason, message := c.getVolumePhaseMessageReason(&vmi.Spec.Volumes[i], vmi.Namespace)
					status.Phase = phase
					status.Message = message
					status.Reason = reason
				}
			} else {
				status.HotplugVolume.AttachPodName = attachmentPod.Name
				if len(attachmentPod.Status.ContainerStatuses) == 1 && attachmentPod.Status.ContainerStatuses[0].Ready {
					status.HotplugVolume.AttachPodUID = attachmentPod.UID
				} else {
					// Remove UID of old pod if a new one is available, but not yet ready
					status.HotplugVolume.AttachPodUID = ""
				}
				if c.canMoveToAttachedPhase(status.Phase) {
					status.Phase = virtv1.HotplugVolumeAttachedToNode
					status.Message = fmt.Sprintf("Created hotplug attachment pod %s, for volume %s", attachmentPod.Name, volume.Name)
					status.Reason = controller.SuccessfulCreatePodReason
					c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, status.Reason, status.Message)
				}
			}
		}

		if volume.VolumeSource.PersistentVolumeClaim != nil || volume.VolumeSource.DataVolume != nil || volume.VolumeSource.MemoryDump != nil {

			pvcName := storagetypes.PVCNameFromVirtVolume(&volume)

			pvcInterface, pvcExists, _ := c.pvcIndexer.GetByKey(fmt.Sprintf("%s/%s", vmi.Namespace, pvcName))
			if pvcExists {
				pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
				status.PersistentVolumeClaimInfo = &virtv1.PersistentVolumeClaimInfo{
					ClaimName:    pvc.Name,
					AccessModes:  pvc.Spec.AccessModes,
					VolumeMode:   pvc.Spec.VolumeMode,
					Capacity:     pvc.Status.Capacity,
					Requests:     pvc.Spec.Resources.Requests,
					Preallocated: storagetypes.IsPreallocated(pvc.ObjectMeta.Annotations),
				}
				filesystemOverhead, err := c.getFilesystemOverhead(pvc)
				if err != nil {
					log.Log.Reason(err).Errorf("Failed to get filesystem overhead for PVC %s/%s", vmi.Namespace, pvcName)
					return err
				}
				status.PersistentVolumeClaimInfo.FilesystemOverhead = &filesystemOverhead
			}
		}

		newStatus = append(newStatus, status)
	}

	// We have updated the status of current volumes, but if a volume was removed, we want to keep that status, until there is no
	// associated pod, then remove it. Any statuses left in the map are statuses without a matching volume in the spec.
	for k, v := range oldStatusMap {
		attachmentPod := c.findAttachmentPodByVolumeName(k, attachmentPods)
		if attachmentPod != nil {
			v.HotplugVolume.AttachPodName = attachmentPod.Name
			v.HotplugVolume.AttachPodUID = attachmentPod.UID
			v.Phase = virtv1.HotplugVolumeDetaching
			if attachmentPod.DeletionTimestamp != nil {
				v.Message = fmt.Sprintf("Deleted hotplug attachment pod %s, for volume %s", attachmentPod.Name, k)
				v.Reason = controller.SuccessfulDeletePodReason
				c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, v.Reason, v.Message)
			}
			// If the pod exists, we keep the status.
			newStatus = append(newStatus, v)
		}
	}
	sort.SliceStable(newStatus, func(i, j int) bool {
		return strings.Compare(newStatus[i].Name, newStatus[j].Name) == -1
	})
	vmi.Status.VolumeStatus = newStatus
	return nil
}

func (c *Controller) volumeReady(phase virtv1.VolumePhase) bool {
	return phase == virtv1.VolumeReady
}

func (c *Controller) getFilesystemOverhead(pvc *k8sv1.PersistentVolumeClaim) (virtv1.Percent, error) {
	// To avoid conflicts, we only allow having one CDI instance
	if cdiInstances := len(c.cdiStore.List()); cdiInstances != 1 {
		if cdiInstances > 1 {
			log.Log.V(3).Object(pvc).Reason(storagetypes.ErrMultipleCdiInstances).Infof(storagetypes.FSOverheadMsg)
		} else {
			log.Log.V(3).Object(pvc).Reason(storagetypes.ErrFailedToFindCdi).Infof(storagetypes.FSOverheadMsg)
		}
		return storagetypes.DefaultFSOverhead, nil
	}

	cdiConfigInterface, cdiConfigExists, err := c.cdiConfigStore.GetByKey(storagetypes.ConfigName)
	if !cdiConfigExists || err != nil {
		return "0", fmt.Errorf("Failed to find CDIConfig but CDI exists: %w", err)
	}
	cdiConfig, ok := cdiConfigInterface.(*cdiv1.CDIConfig)
	if !ok {
		return "0", fmt.Errorf("Failed to convert CDIConfig object %v to type CDIConfig", cdiConfigInterface)
	}

	return storagetypes.GetFilesystemOverhead(pvc.Spec.VolumeMode, pvc.Spec.StorageClassName, cdiConfig), nil
}

func (c *Controller) canMoveToAttachedPhase(currentPhase virtv1.VolumePhase) bool {
	return (currentPhase == "" || currentPhase == virtv1.VolumeBound || currentPhase == virtv1.VolumePending)
}

func (c *Controller) findAttachmentPodByVolumeName(volumeName string, attachmentPods []*k8sv1.Pod) *k8sv1.Pod {
	for _, pod := range attachmentPods {
		for _, podVolume := range pod.Spec.Volumes {
			if podVolume.Name == volumeName {
				return pod
			}
		}
	}
	return nil
}

func (c *Controller) getVolumePhaseMessageReason(volume *virtv1.Volume, namespace string) (virtv1.VolumePhase, string, string) {
	claimName := storagetypes.PVCNameFromVirtVolume(volume)

	pvcInterface, pvcExists, _ := c.pvcIndexer.GetByKey(fmt.Sprintf("%s/%s", namespace, claimName))
	if !pvcExists {
		return virtv1.VolumePending, controller.FailedPvcNotFoundReason, "Unable to determine PVC name"
	}
	pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
	if pvc.Status.Phase == k8sv1.ClaimPending {
		return virtv1.VolumePending, controller.PVCNotReadyReason, "PVC is in phase ClaimPending"
	} else if pvc.Status.Phase == k8sv1.ClaimBound {
		return virtv1.VolumeBound, controller.PVCNotReadyReason, "PVC is in phase Bound"
	}
	return virtv1.VolumePending, controller.PVCNotReadyReason, "PVC is in phase Lost"
}

func (c *Controller) updateMultusAnnotation(namespace string, interfaces []virtv1.Interface, networks []virtv1.Network, pod *k8sv1.Pod) error {
	podAnnotations := pod.GetAnnotations()

	indexedMultusStatusIfaces := network.NonDefaultMultusNetworksIndexedByIfaceName(pod)
	networkToPodIfaceMap := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(networks, indexedMultusStatusIfaces)
	multusAnnotations, err := multus.GenerateCNIAnnotationFromNameScheme(namespace, interfaces, networks, networkToPodIfaceMap, c.clusterConfig)
	if err != nil {
		return err
	}

	currentMultusAnnotation := podAnnotations[networkv1.NetworkAttachmentAnnot]
	log.Log.Object(pod).V(4).Infof(
		"current multus annotation for pod: %s; updated multus annotation for pod with: %s",
		currentMultusAnnotation,
		multusAnnotations,
	)

	if multusAnnotations != currentMultusAnnotation {
		newAnnotations := map[string]string{networkv1.NetworkAttachmentAnnot: multusAnnotations}
		patchedPod, err := c.syncPodAnnotations(pod, newAnnotations)
		if err != nil {
			return err
		}
		*pod = *patchedPod
	}

	return nil
}

func (c *Controller) updateInterfaceStatus(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	indexedMultusStatusIfaces := network.NonDefaultMultusNetworksIndexedByIfaceName(pod)
	ifaceNamingScheme := namescheme.CreateNetworkNameSchemeByPodNetworkStatus(vmi.Spec.Networks, indexedMultusStatusIfaces)
	for _, network := range vmi.Spec.Networks {
		vmiIfaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, network.Name)
		podIfaceName, wasFound := ifaceNamingScheme[network.Name]
		if !wasFound {
			return fmt.Errorf("could not find the pod interface name for network [%s]", network.Name)
		}

		_, exists := indexedMultusStatusIfaces[podIfaceName]
		switch {
		case exists && vmiIfaceStatus == nil:
			vmi.Status.Interfaces = append(vmi.Status.Interfaces, virtv1.VirtualMachineInstanceNetworkInterface{
				Name:       network.Name,
				InfoSource: vmispec.InfoSourceMultusStatus,
			})
		case exists && vmiIfaceStatus != nil:
			vmiIfaceStatus.InfoSource = vmispec.AddInfoSource(vmiIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
		case !exists && vmiIfaceStatus != nil:
			vmiIfaceStatus.InfoSource = vmispec.RemoveInfoSource(vmiIfaceStatus.InfoSource, vmispec.InfoSourceMultusStatus)
		}
	}

	return nil
}

func (c *Controller) syncHotplugCondition(vmi *virtv1.VirtualMachineInstance, conditionType virtv1.VirtualMachineInstanceConditionType) {
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	condition := virtv1.VirtualMachineInstanceCondition{
		Type:   conditionType,
		Status: k8sv1.ConditionTrue,
	}
	if !vmiConditions.HasCondition(vmi, condition.Type) {
		vmiConditions.UpdateCondition(vmi, &condition)
		log.Log.Object(vmi).V(4).Infof("adding hotplug condition %s", conditionType)
	}

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
	c.syncHotplugCondition(vmi, virtv1.VirtualMachineInstanceMemoryChange)
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

func (c *Controller) aggregateDataVolumesConditions(vmiCopy *virtv1.VirtualMachineInstance, dvs []*cdiv1.DataVolume) {
	if len(dvs) == 0 {
		return
	}

	dvsReadyCondition := virtv1.VirtualMachineInstanceCondition{
		Status:  k8sv1.ConditionTrue,
		Type:    virtv1.VirtualMachineInstanceDataVolumesReady,
		Reason:  virtv1.VirtualMachineInstanceReasonAllDVsReady,
		Message: "All of the VMI's DVs are bound and not running",
	}

	for _, dv := range dvs {
		cStatus := statusOfReadyCondition(dv.Status.Conditions)
		if cStatus != k8sv1.ConditionTrue {
			dvsReadyCondition.Reason = virtv1.VirtualMachineInstanceReasonNotAllDVsReady
			if cStatus == k8sv1.ConditionFalse {
				dvsReadyCondition.Status = cStatus
			} else if dvsReadyCondition.Status == k8sv1.ConditionTrue {
				dvsReadyCondition.Status = cStatus
			}
		}
	}

	if dvsReadyCondition.Status != k8sv1.ConditionTrue {
		dvsReadyCondition.Message = "Not all of the VMI's DVs are ready"
	}

	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	vmiConditions.UpdateCondition(vmiCopy, &dvsReadyCondition)
}

func statusOfReadyCondition(conditions []cdiv1.DataVolumeCondition) k8sv1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == cdiv1.DataVolumeReady {
			return condition.Status
		}
	}
	return k8sv1.ConditionUnknown
}
