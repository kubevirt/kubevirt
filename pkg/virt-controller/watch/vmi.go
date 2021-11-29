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

package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/pkg/controller"
	kubevirttypes "kubevirt.io/kubevirt/pkg/util/types"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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

	// UnauthorizedDataVolumeCreateReason is added in an event when the DataVolume
	// ServiceAccount doesn't have permission to create a DataVolume
	UnauthorizedDataVolumeCreateReason = "UnauthorizedDataVolumeCreate"
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
	// FailedGatherhingClusterTopologyHints is added if the cluster topology hints can't be collected for a VMI by virt-controller
	FailedGatherhingClusterTopologyHints = "FailedGatherhingClusterTopologyHints"
	// FailedPvcNotFoundReason is added in an event
	// when a PVC for a volume was not found.
	FailedPvcNotFoundReason = "FailedPvcNotFound"
	// FailedDataVolumeNotFoundReason is added in an event
	// when a DataVolume for a volume was not found.
	FailedDataVolumeNotFoundReason = "FailedDataVolumeNotFound"
	// SuccessfulMigrationReason is added when a migration attempt completes successfully
	SuccessfulMigrationReason = "SuccessfulMigration"
	// FailedMigrationReason is added when a migration attempt fails
	FailedMigrationReason = "FailedMigration"
	// SuccessfulAbortMigrationReason is added when an attempt to abort migration completes successfully
	SuccessfulAbortMigrationReason = "SuccessfulAbortMigration"
	// FailedAbortMigrationReason is added when an attempt to abort migration fails
	FailedAbortMigrationReason = "FailedAbortMigration"
	// MissingAttachmentPodReason is set when we have a hotplugged volume, but the attachment pod is missing
	MissingAttachmentPodReason = "MissingAttachmentPod"
	// PVCNotReadyReason is set when the PVC is not ready to be hot plugged.
	PVCNotReadyReason = "PVCNotReady"
	// FailedHotplugSyncReason is set when a hotplug specific failure occurs during sync
	FailedHotplugSyncReason = "FailedHotplugSync"
	// ErrImagePullReason is set when an error has occured while pulling an image for a containerDisk VM volume.
	ErrImagePullReason = "ErrImagePull"
	// ImagePullBackOffReason is set when an error has occured while pulling an image for a containerDisk VM volume,
	// and that kubelet is backing off before retrying.
	ImagePullBackOffReason = "ImagePullBackOff"
	// NoSuitableNodesForHostModelMigration is set when a VMI with host-model CPU mode tries to migrate but no node
	// is suitable for migration (since CPU model / required features are not supported)
	NoSuitableNodesForHostModelMigration = "NoSuitableNodesForHostModelMigration"
)

const failedToRenderLaunchManifestErrFormat = "failed to render launch manifest: %v"

var failedToFindCdi error = errors.New("No CDIConfig named config")

func NewVMIController(templateService services.TemplateService,
	vmiInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	dataVolumeInformer cache.SharedIndexInformer,
	cdiInformer cache.SharedIndexInformer,
	cdiConfigInformer cache.SharedIndexInformer,
	clusterConfig *virtconfig.ClusterConfig,
	topologyHinter topology.Hinter,
) *VMIController {

	c := &VMIController{
		templateService:    templateService,
		Queue:              workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-vmi"),
		vmiInformer:        vmiInformer,
		vmInformer:         vmInformer,
		podInformer:        podInformer,
		pvcInformer:        pvcInformer,
		recorder:           recorder,
		clientset:          clientset,
		podExpectations:    controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		vmiExpectations:    controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		dataVolumeInformer: dataVolumeInformer,
		cdiInformer:        cdiInformer,
		cdiConfigInformer:  cdiConfigInformer,
		clusterConfig:      clusterConfig,
		topologyHinter:     topologyHinter,
	}

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
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

	c.pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updatePVC,
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
	vmInformer         cache.SharedIndexInformer
	podInformer        cache.SharedIndexInformer
	pvcInformer        cache.SharedIndexInformer
	topologyHinter     topology.Hinter
	recorder           record.EventRecorder
	podExpectations    *controller.UIDTrackingControllerExpectations
	vmiExpectations    *controller.UIDTrackingControllerExpectations
	dataVolumeInformer cache.SharedIndexInformer
	cdiInformer        cache.SharedIndexInformer
	cdiConfigInformer  cache.SharedIndexInformer
	clusterConfig      *virtconfig.ClusterConfig
}

func (c *VMIController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting vmi controller.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmiInformer.HasSynced, c.podInformer.HasSynced, c.dataVolumeInformer.HasSynced, c.cdiConfigInformer.HasSynced, c.cdiInformer.HasSynced)

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
		c.vmiExpectations.DeleteExpectations(key)
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
		_, err = c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(vmi)
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
	pod, err := controller.CurrentVMIPod(vmi, c.podInformer)
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

	syncErr := c.sync(vmi, pod, dataVolumes)

	err = c.updateStatus(vmi, pod, dataVolumes, syncErr)
	if err != nil {
		return err
	}

	if syncErr != nil {
		return syncErr
	}

	return nil

}

func (c *VMIController) setLauncherContainerInfo(vmi *virtv1.VirtualMachineInstance, curPodImage string) *virtv1.VirtualMachineInstance {

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

func (c *VMIController) hasOwnerVM(vmi *virtv1.VirtualMachineInstance) bool {
	controllerRef := v1.GetControllerOf(vmi)
	if controllerRef == nil || controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return false
	}

	obj, exists, _ := c.vmInformer.GetStore().GetByKey(vmi.Namespace + "/" + controllerRef.Name)
	if !exists {
		return false
	}

	ownerVM := obj.(*virtv1.VirtualMachine)
	if controllerRef.UID == ownerVM.UID {
		return true
	}

	return false
}

func (c *VMIController) updateStatus(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume, syncErr syncError) error {

	hasFailedDataVolume := false
	for _, dataVolume := range dataVolumes {
		if dataVolume.Status.Phase == cdiv1.Failed {
			hasFailedDataVolume = true
		}
	}

	hasWffcDataVolume := false
	// there is no reason to check for waitForFirstConsumer is there are failed DV's
	if !hasFailedDataVolume {
		for _, dataVolume := range dataVolumes {
			if dataVolume.Status.Phase == cdiv1.WaitForFirstConsumer {
				hasWffcDataVolume = true
			}
		}
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()
	podConditionManager := controller.NewPodConditionManager()
	vmiCopy := vmi.DeepCopy()
	vmiPodExists := podExists(pod) && !isTempPod(pod)
	tempPodExists := podExists(pod) && isTempPod(pod)

	vmiCopy, err := c.setActivePods(vmiCopy)
	if err != nil {
		return fmt.Errorf("Error detecting vmi pods: %v", err)
	}

	c.syncReadyConditionFromPod(vmiCopy, pod)
	if vmiPodExists {
		err := c.syncPausedConditionToPod(vmiCopy, pod)
		if err != nil {
			return fmt.Errorf("error syncing paused condition to pod: %v", err)
		}
	}

	switch {
	case vmi.IsUnprocessed():
		if vmiPodExists {
			vmiCopy.Status.Phase = virtv1.Scheduling
		} else if vmi.DeletionTimestamp != nil || hasFailedDataVolume {
			vmiCopy.Status.Phase = virtv1.Failed
		} else {
			vmiCopy.Status.Phase = virtv1.Pending
			if vmi.Status.TopologyHints == nil && c.topologyHinter.TopologyHintsRequiredForVMI(vmi) {
				if topologyHints, err := c.topologyHinter.TopologyHintsForVMI(vmi); err != nil {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedGatherhingClusterTopologyHints, err.Error())
					return &syncErrorImpl{err, FailedGatherhingClusterTopologyHints}
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
					if isPodFailedOrGoingDown(pod) {
						vmiCopy.Status.Phase = virtv1.Failed
					}
				}
			}
			if syncErr != nil &&
				(syncErr.Reason() == FailedPvcNotFoundReason || syncErr.Reason() == FailedDataVolumeNotFoundReason) {
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

			if isPodReady(pod) && vmi.DeletionTimestamp == nil {
				// fail vmi creation if CPU pinning has been requested but the Pod QOS is not Guaranteed
				podQosClass := pod.Status.QOSClass
				if podQosClass != k8sv1.PodQOSGuaranteed && vmi.IsCPUDedicated() {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedGuaranteePodResourcesReason, "failed to guarantee pod resources")
					syncErr = &syncErrorImpl{fmt.Errorf("failed to guarantee pod resources"), FailedGuaranteePodResourcesReason}
					break
				}

				hotplugPodsReady := false
				hotplugPodsReady, syncErr = c.hotplugPodsReady(vmi, pod)
				if syncErr != nil {
					break
				}
				if !hotplugPodsReady {
					log.Log.V(3).Object(vmi).Infof("Postpone the pod hand-over and await the attachment pod ready")
					break
				}

				// Initialize the volume status field with information
				// about the PVCs that the VMI is consuming. This prevents
				// virt-handler from needing to make API calls to GET the pvc
				// during reconcile
				c.updateVolumeStatus(vmiCopy, pod)
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
			} else if isPodDownOrGoingDown(pod) {
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
			break
		}

		c.updateVolumeStatus(vmiCopy, pod)

		var foundImage string
		for _, container := range pod.Spec.Containers {
			if container.Name == "compute" {
				foundImage = container.Image
				break
			}
		}
		vmiCopy = c.setLauncherContainerInfo(vmiCopy, foundImage)

	case vmi.IsScheduled():
		// Nothing here
		break
	default:
		return fmt.Errorf("unknown vmi phase %v", vmi.Status.Phase)
	}

	// VMI is owned by virt-handler, so patch instead of update
	if vmi.IsRunning() || vmi.IsScheduled() {
		patchBytes, err := prepareVMIPatch(vmi, vmiCopy)
		if err != nil {
			return fmt.Errorf("error preparing VMI patch: %v", err)
		}

		if len(patchBytes) > 0 {
			_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(vmi.Name, types.JSONPatchType, []byte(patchBytes), &v1.PatchOptions{})
			// We could not retry if the "test" fails but we have no sane way to detect that right now: https://github.com/kubernetes/kubernetes/issues/68202 for details
			// So just retry like with any other errors
			if err != nil {
				return fmt.Errorf("patching of vmi conditions and activePods failed: %v", err)
			}
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
	vmiChanged := !reflect.DeepEqual(vmi.Status, vmiCopy.Status) || !reflect.DeepEqual(vmi.Finalizers, vmiCopy.Finalizers) || !reflect.DeepEqual(vmi.Annotations, vmiCopy.Annotations) || !reflect.DeepEqual(vmi.Labels, vmiCopy.Labels)
	if vmiChanged {
		key := controller.VirtualMachineInstanceKey(vmi)
		c.vmiExpectations.SetExpectations(key, 1, 0)
		_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Update(vmiCopy)
		if err != nil {
			c.vmiExpectations.LowerExpectations(key, 1, 0)
			return err
		}
	}

	return nil
}

func preparePodPatch(oldPod, newPod *k8sv1.Pod) ([]byte, error) {
	var patchOps []string

	podConditions := controller.NewPodConditionManager()
	if !podConditions.ConditionsEqual(oldPod, newPod) {

		newConditions, err := json.Marshal(newPod.Status.Conditions)
		if err != nil {
			return nil, err
		}
		oldConditions, err := json.Marshal(oldPod.Status.Conditions)
		if err != nil {
			return nil, err
		}

		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "test", "path": "/status/conditions", "value": %s }`, string(oldConditions)))
		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/status/conditions", "value": %s }`, string(newConditions)))
	}

	if len(patchOps) == 0 {
		return nil, nil
	}
	return controller.GeneratePatchBytes(patchOps), nil
}

func prepareVMIPatch(oldVMI, newVMI *virtv1.VirtualMachineInstance) ([]byte, error) {
	var patchOps []string

	if !reflect.DeepEqual(newVMI.Status.VolumeStatus, oldVMI.Status.VolumeStatus) {
		// VolumeStatus changed which means either removed or added volumes.
		newVolumeStatus, err := json.Marshal(newVMI.Status.VolumeStatus)
		if err != nil {
			return nil, err
		}
		oldVolumeStatus, err := json.Marshal(oldVMI.Status.VolumeStatus)
		if err != nil {
			return nil, err
		}
		if string(oldVolumeStatus) == "null" {
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "add", "path": "/status/volumeStatus", "value": %s }`, string(newVolumeStatus)))
		} else {
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "test", "path": "/status/volumeStatus", "value": %s }`, string(oldVolumeStatus)))
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/status/volumeStatus", "value": %s }`, string(newVolumeStatus)))
		}
		log.Log.V(3).Object(oldVMI).Infof("Patching Volume Status")
	}
	// We don't own the object anymore, so patch instead of update
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	if !vmiConditions.ConditionsEqual(oldVMI, newVMI) {

		newConditions, err := json.Marshal(newVMI.Status.Conditions)
		if err != nil {
			return nil, err
		}
		oldConditions, err := json.Marshal(oldVMI.Status.Conditions)
		if err != nil {
			return nil, err
		}

		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "test", "path": "/status/conditions", "value": %s }`, string(oldConditions)))
		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/status/conditions", "value": %s }`, string(newConditions)))

		log.Log.V(3).Object(oldVMI).Infof("Patching VMI conditions")
	}

	if !reflect.DeepEqual(newVMI.Status.ActivePods, oldVMI.Status.ActivePods) {
		newPods, err := json.Marshal(newVMI.Status.ActivePods)
		if err != nil {
			return nil, err
		}
		oldPods, err := json.Marshal(oldVMI.Status.ActivePods)
		if err != nil {
			return nil, err
		}

		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "test", "path": "/status/activePods", "value": %s }`, string(oldPods)))
		patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/status/activePods", "value": %s }`, string(newPods)))

		log.Log.V(3).Object(oldVMI).Infof("Patching VMI activePods")
	}

	if newVMI.Status.LauncherContainerImageVersion != oldVMI.Status.LauncherContainerImageVersion {
		if oldVMI.Status.LauncherContainerImageVersion == "" {
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "add", "path": "/status/launcherContainerImageVersion", "value": "%s" }`, newVMI.Status.LauncherContainerImageVersion))
		} else {
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "test", "path": "/status/launcherContainerImageVersion", "value": "%s" }`, oldVMI.Status.LauncherContainerImageVersion))
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/status/launcherContainerImageVersion", "value": "%s" }`, newVMI.Status.LauncherContainerImageVersion))
		}
	}

	if !reflect.DeepEqual(oldVMI.Labels, newVMI.Labels) {
		newLabelBytes, err := json.Marshal(newVMI.Labels)
		if err != nil {
			return nil, err
		}
		oldLabelBytes, err := json.Marshal(oldVMI.Labels)
		if err != nil {
			return nil, err
		}

		if oldVMI.Labels == nil {
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "add", "path": "/metadata/labels", "value": %s }`, string(newLabelBytes)))
		} else {
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "test", "path": "/metadata/labels", "value": %s }`, string(oldLabelBytes)))
			patchOps = append(patchOps, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/labels", "value": %s }`, string(newLabelBytes)))

		}
	}

	if len(patchOps) == 0 {
		return nil, nil
	}

	return controller.GeneratePatchBytes(patchOps), nil
}

func (c *VMIController) syncReadyConditionFromPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) {
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

	} else if isPodDownOrGoingDown(pod) {
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

func (c *VMIController) syncPausedConditionToPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
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
	patchBytes, err := preparePodPatch(pod, podCopy)
	if err != nil {
		return fmt.Errorf("error preparing pod patch: %v", err)
	}

	if len(patchBytes) > 0 {
		log.Log.V(3).Object(pod).Infof("Patching pod conditions")

		_, err = c.clientset.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.JSONPatchType, []byte(patchBytes), v1.PatchOptions{}, "status")
		// We could not retry if the "test" fails but we have no sane way to detect that right now:
		// https://github.com/kubernetes/kubernetes/issues/68202 for details
		// So just retry like with any other errors
		if err != nil {
			log.Log.Object(pod).Errorf("Patching of pod conditions failed: %v", err)
			return fmt.Errorf("patching of pod conditions failed: %v", err)
		}
	}

	return nil
}

// checkForContainerImageError checks if an error has occured while handling the image of any of the pod's containers
// (including init containers), and returns a syncErr with the details of the error, or nil otherwise.
func checkForContainerImageError(pod *k8sv1.Pod) syncError {
	containerStatuses := append(append([]k8sv1.ContainerStatus{},
		pod.Status.InitContainerStatuses...),
		pod.Status.ContainerStatuses...)

	for _, containerStatus := range containerStatuses {
		if containerStatus.State.Waiting == nil {
			continue
		}

		reason := containerStatus.State.Waiting.Reason
		if reason == ErrImagePullReason || reason == ImagePullBackOffReason {
			return &syncErrorImpl{
				reason: reason,
				err:    fmt.Errorf(containerStatus.State.Waiting.Message),
			}
		}
	}

	return nil
}

// isPodReady treats the pod as ready to be handed over to virt-handler, as soon as all pods except
// the compute pod are ready.
func isPodReady(pod *k8sv1.Pod) bool {
	if isPodDownOrGoingDown(pod) {
		return false
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		// The compute container potentially holds a readiness probe for the VMI. Therefore
		// don't wait for the compute container to become ready (the VMI later on will trigger the change to ready)
		// and only check that the container started
		if containerStatus.Name == "compute" {
			if containerStatus.State.Running == nil {
				return false
			}
		} else if containerStatus.Name != "istio-proxy" && containerStatus.Ready == false {
			// When using istio the istio-proxy container will not be ready
			// until there is a service pointing to this pod.
			// We need to start the VM anyway
			return false
		}
	}

	return pod.Status.Phase == k8sv1.PodRunning
}

func isPodPending(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodPending
}

func isPodDownOrGoingDown(pod *k8sv1.Pod) bool {
	return podIsDown(pod) || isComputeContainerDown(pod) || pod.DeletionTimestamp != nil
}

func isPodFailedOrGoingDown(pod *k8sv1.Pod) bool {
	return isPodFailed(pod) || isComputeContainerFailed(pod) || pod.DeletionTimestamp != nil
}

func isComputeContainerDown(pod *k8sv1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "compute" {
			return containerStatus.State.Terminated != nil
		}
	}
	return false
}

func isComputeContainerFailed(pod *k8sv1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "compute" {
			return containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0
		}
	}
	return false
}

func podIsDown(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed
}

func isPodFailed(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodFailed
}

func podExists(pod *k8sv1.Pod) bool {
	if pod != nil {
		return true
	}
	return false
}

func (c *VMIController) hotplugPodsReady(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) (bool, syncError) {
	if controller.VMIHasHotplugVolumes(vmi) {
		hotplugAttachmentPods, err := controller.AttachmentPods(virtLauncherPod, c.podInformer)
		if err != nil {
			return false, &syncErrorImpl{fmt.Errorf("failed to get attachment pods: %v", err), FailedHotplugSyncReason}
		}
		for _, attachmentPod := range hotplugAttachmentPods {
			if isPodReady(attachmentPod) && attachmentPod.DeletionTimestamp == nil && attachmentPod.Spec.NodeName == virtLauncherPod.Spec.NodeName {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func (c *VMIController) sync(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) syncError {
	if vmi.DeletionTimestamp != nil {
		err := c.deleteAllMatchingPods(vmi)
		if err != nil {
			return &syncErrorImpl{fmt.Errorf("failed to delete pod: %v", err), FailedDeletePodReason}
		}
		return nil
	}

	if vmi.IsFinal() {
		err := c.deleteAllAttachmentPods(vmi)
		if err != nil {
			return &syncErrorImpl{fmt.Errorf("failed to delete attachment pods: %v", err), FailedHotplugSyncReason}
		}
		return nil
	}

	if err := c.deleteOrphanedAttachmentPods(vmi); err != nil {
		log.Log.Reason(err).Errorf("failed to delete orphaned attachment pods %s: %v", controller.VirtualMachineInstanceKey(vmi), err)
		// do not return; just log the error
	}

	dataVolumesReady, isWaitForFirstConsumer, syncErr := c.handleSyncDataVolumes(vmi, dataVolumes)
	if syncErr != nil {
		return syncErr
	}

	if !podExists(pod) {
		// If we came ever that far to detect that we already created a pod, we don't create it again
		if !vmi.IsUnprocessed() {
			return nil
		}
		// let's check if we already have topology hints or if we are still waiting for them
		if vmi.Status.TopologyHints == nil && c.topologyHinter.TopologyHintsRequiredForVMI(vmi) {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation until topology hints are set")
			return nil
		}

		// ensure that all dataVolumes associated with the VMI are ready before creating the pod
		if !dataVolumesReady {
			log.Log.V(3).Object(vmi).Infof("Delaying pod creation while DataVolume populates")
			return nil
		}
		var templatePod *k8sv1.Pod
		var err error
		if isWaitForFirstConsumer {
			log.Log.V(3).Object(vmi).Infof("Scheduling temporary pod for WaitForFirstConsumer DV")
			templatePod, err = c.templateService.RenderLaunchManifestNoVm(vmi)
		} else {
			templatePod, err = c.templateService.RenderLaunchManifest(vmi)
		}
		if _, ok := err.(services.PvcNotFoundError); ok {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedPvcNotFoundReason, failedToRenderLaunchManifestErrFormat, err)
			return &syncErrorImpl{fmt.Errorf(failedToRenderLaunchManifestErrFormat, err), FailedPvcNotFoundReason}
		} else if err != nil {
			return &syncErrorImpl{fmt.Errorf(failedToRenderLaunchManifestErrFormat, err), FailedCreatePodReason}
		}

		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		c.podExpectations.ExpectCreations(vmiKey, 1)
		pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), templatePod, v1.CreateOptions{})
		if err != nil {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating pod: %v", err)
			c.podExpectations.CreationObserved(vmiKey)
			return &syncErrorImpl{fmt.Errorf("failed to create virtual machine pod: %v", err), FailedCreatePodReason}
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created virtual machine pod %s", pod.Name)
		return nil
	}

	if !isWaitForFirstConsumer {
		err := c.cleanupWaitForFirstConsumerTemporaryPods(vmi, pod)
		if err != nil {
			return &syncErrorImpl{fmt.Errorf("failed to clean up temporary pods: %v", err), FailedHotplugSyncReason}
		}
	}

	if !isTempPod(pod) && isPodReady(pod) {
		hotplugVolumes := getHotplugVolumes(vmi, pod)
		hotplugAttachmentPods, err := controller.AttachmentPods(pod, c.podInformer)
		if err != nil {
			return &syncErrorImpl{fmt.Errorf("failed to get attachment pods: %v", err), FailedHotplugSyncReason}
		}

		if pod.DeletionTimestamp == nil && c.needsHandleHotplug(hotplugVolumes, hotplugAttachmentPods) {
			var hotplugSyncErr syncError = nil
			hotplugSyncErr = c.handleHotplugVolumes(hotplugVolumes, hotplugAttachmentPods, vmi, pod, dataVolumes)
			if hotplugSyncErr != nil {
				if hotplugSyncErr.Reason() == MissingAttachmentPodReason {
					// We are missing an essential hotplug pod. Delete all pods associated with the VMI.
					c.deleteAllMatchingPods(vmi)
				} else {
					return hotplugSyncErr
				}
			}
		}
	}
	return nil
}

func (c *VMIController) handleSyncDataVolumes(vmi *virtv1.VirtualMachineInstance, dataVolumes []*cdiv1.DataVolume) (bool, bool, syncError) {

	ready := true
	wffc := false

	for _, volume := range vmi.Spec.Volumes {
		// Check both DVs and PVCs
		if volume.VolumeSource.DataVolume != nil || volume.VolumeSource.PersistentVolumeClaim != nil {
			volumeReady, volumeWffc, err := c.volumeReadyToAttachToNode(vmi.Namespace, volume, dataVolumes)
			if err != nil {
				// Keep existing behavior of missing PVC = ready. This in turn triggers template render, which sets conditions and events, and fails appropriately
				if _, ok := err.(services.PvcNotFoundError); ok {
					continue
				} else if _, ok := err.(services.DataVolumeNotFoundError); ok {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDataVolumeNotFoundReason, "DataVolume is referenced by VMI but doesn't exist: %v", err)
					return false, false, &syncErrorImpl{fmt.Errorf("DataVolume is referenced by VMI but doesn't exist: %v", err), FailedDataVolumeNotFoundReason}
				} else {
					c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedPvcNotFoundReason, "Error determining if volume is ready: %v", err)
					return false, false, &syncErrorImpl{fmt.Errorf("Error determining if volume is ready %v", err), FailedDataVolumeImportReason}
				}
			}
			wffc = wffc || volumeWffc
			// Ready only becomes false if WFFC is also false.
			ready = ready && (volumeReady || volumeWffc)
		}
	}

	return ready, wffc, nil
}

func dataVolumeByNameFunc(dataVolumeInformer cache.SharedIndexInformer, dataVolumes []*cdiv1.DataVolume) func(name string, namespace string) (*cdiv1.DataVolume, error) {
	return func(name, namespace string) (*cdiv1.DataVolume, error) {
		for _, dataVolume := range dataVolumes {
			if dataVolume.Name == name && dataVolume.Namespace == namespace {
				return dataVolume, nil
			}
		}
		dv, exists, _ := dataVolumeInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", namespace, name))
		if !exists {
			return nil, fmt.Errorf("Unable to find datavolume %s/%s", namespace, name)
		}
		return dv.(*cdiv1.DataVolume), nil
	}
}

func (c *VMIController) updatePVC(old, cur interface{}) {
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
	if reflect.DeepEqual(curPVC.Status.Capacity, oldPVC.Status.Capacity) {
		// We only do something when the capacity changes
		return
	}

	vmis, err := c.listVMIsMatchingPVC(curPVC.Namespace, curPVC.Name)
	if err != nil {
		log.Log.V(4).Object(curPVC).Errorf("Error encountered during pvc update: %v", err)
		return
	}
	for _, vmi := range vmis {
		log.Log.V(4).Object(curPVC).Infof("PVC updated for vmi %s", vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

func (c *VMIController) addDataVolume(obj interface{}) {
	dataVolume := obj.(*cdiv1.DataVolume)
	if dataVolume.DeletionTimestamp != nil {
		c.deleteDataVolume(dataVolume)
		return
	}

	vmis, err := c.listVMIsMatchingPVC(dataVolume.Namespace, dataVolume.Name)
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

	vmis, err := c.listVMIsMatchingPVC(curDataVolume.Namespace, curDataVolume.Name)
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
	vmis, err := c.listVMIsMatchingPVC(dataVolume.Namespace, dataVolume.Name)
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

func (c *VMIController) addVirtualMachineInstance(obj interface{}) {
	c.lowerVMIExpectation(obj)
	c.enqueueVirtualMachine(obj)
}

func (c *VMIController) deleteVirtualMachineInstance(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a vmi in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vmi %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}
	c.lowerVMIExpectation(vmi)
	c.enqueueVirtualMachine(vmi)
}

func (c *VMIController) updateVirtualMachineInstance(_, curr interface{}) {
	c.lowerVMIExpectation(curr)
	c.enqueueVirtualMachine(curr)
}

func (c *VMIController) lowerVMIExpectation(curr interface{}) {
	key, err := controller.KeyFunc(curr)
	if err != nil {
		return
	}
	c.vmiExpectations.LowerExpectations(key, 1, 0)
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
	if controllerRef != nil && controllerRef.Kind == "Pod" {
		// This could be an attachment pod, look up the pod, and check if it is owned by a VMI.
		obj, exists, err := c.podInformer.GetIndexer().GetByKey(namespace + "/" + controllerRef.Name)
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

// takes a PVC name and namespace and returns all VMIs from the VMI cache which use this PVC
func (c *VMIController) listVMIsMatchingPVC(namespace string, pvcName string) ([]*virtv1.VirtualMachineInstance, error) {
	objs, err := c.vmiInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vmis := []*virtv1.VirtualMachineInstance{}
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		for _, volume := range vmi.Spec.Volumes {
			if volume.VolumeSource.DataVolume != nil && volume.VolumeSource.DataVolume.Name == pvcName ||
				volume.VolumeSource.PersistentVolumeClaim != nil && volume.VolumeSource.PersistentVolumeClaim.ClaimName == pvcName {
				vmis = append(vmis, vmi)
			}
		}
	}
	return vmis, nil
}

func (c *VMIController) listMatchingDataVolumes(vmi *virtv1.VirtualMachineInstance) ([]*cdiv1.DataVolume, error) {

	dataVolumes := []*cdiv1.DataVolume{}
	for _, volume := range vmi.Spec.Volumes {
		dataVolumeName := c.getDataVolumeName(vmi.Namespace, volume)
		if dataVolumeName == nil {
			continue
		}

		obj, exists, err := c.dataVolumeInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", vmi.Namespace, *dataVolumeName))

		if err != nil {
			return dataVolumes, err
		} else if exists {
			dataVolume := obj.(*cdiv1.DataVolume)
			dataVolumes = append(dataVolumes, dataVolume)
		}
	}

	return dataVolumes, nil
}

func (c *VMIController) getDataVolumeName(namespace string, volume virtv1.Volume) *string {
	if volume.VolumeSource.PersistentVolumeClaim != nil {
		pvcInterface, pvcExists, _ := c.pvcInformer.GetStore().
			GetByKey(fmt.Sprintf("%s/%s", namespace, volume.VolumeSource.PersistentVolumeClaim.ClaimName))
		if pvcExists {
			pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
			pvcOwner := v1.GetControllerOf(pvc)
			if pvcOwner != nil && pvcOwner.Kind == "DataVolume" {
				return &pvcOwner.Name
			}
		}
	} else if volume.VolumeSource.DataVolume != nil {
		return &volume.VolumeSource.DataVolume.Name
	}
	return nil
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
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDeletePodReason, "Failed to delete virtual machine pod %s", pod.Name)
			return err
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted virtual machine pod %s", pod.Name)
	}
	c.deleteAllAttachmentPods(vmi)
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

func (c *VMIController) setActivePods(vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstance, error) {
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

func getHotplugVolumes(vmi *virtv1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod) []*virtv1.Volume {
	hotplugVolumes := make([]*virtv1.Volume, 0)
	podVolumes := virtlauncherPod.Spec.Volumes
	vmiVolumes := vmi.Spec.Volumes

	podVolumeMap := make(map[string]k8sv1.Volume)
	for _, podVolume := range podVolumes {
		podVolumeMap[podVolume.Name] = podVolume
	}
	for _, vmiVolume := range vmiVolumes {
		if _, ok := podVolumeMap[vmiVolume.Name]; !ok && (vmiVolume.DataVolume != nil || vmiVolume.PersistentVolumeClaim != nil) {
			hotplugVolumes = append(hotplugVolumes, vmiVolume.DeepCopy())
		}
	}
	return hotplugVolumes
}

func (c *VMIController) cleanupWaitForFirstConsumerTemporaryPods(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) error {
	triggerPods, err := c.waitForFirstConsumerTemporaryPods(vmi, virtLauncherPod)
	if err != nil {
		return err
	}

	return c.deleteRunningOrFinishedWFFCPods(vmi, triggerPods...)
}

func (c *VMIController) deleteRunningOrFinishedWFFCPods(vmi *virtv1.VirtualMachineInstance, pods ...*k8sv1.Pod) error {
	for _, pod := range pods {
		err := c.deleteRunningFinishedOrFailedPod(vmi, pod)
		if err != nil && !k8serrors.IsNotFound(err) {
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDeletePodReason, "Failed to delete WaitForFirstConsumer temporary pod %s", pod.Name)
			return err
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted WaitForFirstConsumer temporary pod %s", pod.Name)
	}
	return nil
}

func (c *VMIController) deleteRunningFinishedOrFailedPod(vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod) error {
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

func (c *VMIController) waitForFirstConsumerTemporaryPods(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) ([]*k8sv1.Pod, error) {
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

func (c *VMIController) needsHandleHotplug(hotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod) bool {
	// Determine if the ready volumes have changed compared to the current pod
	for _, attachmentPod := range hotplugAttachmentPods {
		if c.podVolumesMatchesReadyVolumes(attachmentPod, hotplugVolumes) {
			log.DefaultLogger().Infof("Don't need to handle as we have a matching attachment pod")
			return false
		}
		return true
	}
	return len(hotplugVolumes) > 0
}

func (c *VMIController) handleHotplugVolumes(hotplugVolumes []*virtv1.Volume, hotplugAttachmentPods []*k8sv1.Pod, vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, dataVolumes []*cdiv1.DataVolume) syncError {
	logger := log.Log.Object(vmi)

	readyHotplugVolumes := make([]*virtv1.Volume, 0)
	// Find all ready volumes
	for _, volume := range hotplugVolumes {
		var err error
		ready, wffc, err := c.volumeReadyToAttachToNode(vmi.Namespace, *volume, dataVolumes)
		if err != nil {
			return &syncErrorImpl{fmt.Errorf("Error determining volume status %v", err), PVCNotReadyReason}
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
	// Determine if the ready volumes have changed compared to the current pod
	currentPod := make([]*k8sv1.Pod, 0)
	oldPods := make([]*k8sv1.Pod, 0)
	for _, attachmentPod := range hotplugAttachmentPods {
		if !c.podVolumesMatchesReadyVolumes(attachmentPod, readyHotplugVolumes) {
			oldPods = append(oldPods, attachmentPod)
		} else {
			currentPod = append(currentPod, attachmentPod)
		}
	}

	if len(currentPod) == 0 && len(readyHotplugVolumes) > 0 {
		// ready volumes have changed
		// Create new attachment pod that holds all the ready volumes
		if err := c.createAttachmentPod(vmi, virtLauncherPod, readyHotplugVolumes); err != nil {
			return err
		}
	}
	// Delete old attachment pod
	for _, attachmentPod := range oldPods {
		if err := c.deleteAttachmentPodForVolume(vmi, attachmentPod); err != nil {
			return &syncErrorImpl{fmt.Errorf("Error deleting attachment pod %v", err), FailedDeletePodReason}
		}
	}
	return nil
}

func (c *VMIController) podVolumesMatchesReadyVolumes(attachmentPod *k8sv1.Pod, volumes []*virtv1.Volume) bool {
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

func (c *VMIController) createAttachmentPod(vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod, volumes []*virtv1.Volume) syncError {
	attachmentPodTemplate, _ := c.createAttachmentPodTemplate(vmi, virtLauncherPod, volumes)
	if attachmentPodTemplate == nil {
		return nil
	}
	vmiKey := controller.VirtualMachineInstanceKey(vmi)
	c.podExpectations.ExpectCreations(vmiKey, 1)

	pod, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), attachmentPodTemplate, v1.CreateOptions{})
	if err != nil {
		c.podExpectations.CreationObserved(vmiKey)
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating attachment pod: %v", err)
		return &syncErrorImpl{fmt.Errorf("Error creating attachment pod %v", err), FailedCreatePodReason}
	}
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created attachment pod %s", pod.Name)
	return nil
}

func (c *VMIController) triggerHotplugPopulation(volume *virtv1.Volume, vmi *virtv1.VirtualMachineInstance, virtLauncherPod *k8sv1.Pod) syncError {
	populateHotplugPodTemplate, err := c.createAttachmentPopulateTriggerPodTemplate(volume, virtLauncherPod, vmi)
	if err != nil {
		return &syncErrorImpl{fmt.Errorf("Error creating trigger pod template %v", err), FailedCreatePodReason}
	}
	if populateHotplugPodTemplate != nil { // nil means the PVC is not populated yet.
		vmiKey := controller.VirtualMachineInstanceKey(vmi)
		c.podExpectations.ExpectCreations(vmiKey, 1)

		_, err := c.clientset.CoreV1().Pods(vmi.GetNamespace()).Create(context.Background(), populateHotplugPodTemplate, v1.CreateOptions{})
		if err != nil {
			c.podExpectations.CreationObserved(vmiKey)
			c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedCreatePodReason, "Error creating hotplug population trigger pod for volume %s: %v", volume.Name, err)
			return &syncErrorImpl{fmt.Errorf("Error creating hotplug population trigger pod %v", err), FailedCreatePodReason}
		}
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulCreatePodReason, "Created hotplug trigger pod for volume %s", volume.Name)
	}
	return nil
}

func (c *VMIController) volumeReadyToAttachToNode(namespace string, volume virtv1.Volume, dataVolumes []*cdiv1.DataVolume) (bool, bool, error) {
	name := ""
	if volume.DataVolume != nil {
		name = volume.DataVolume.Name
	} else if volume.PersistentVolumeClaim != nil {
		name = volume.PersistentVolumeClaim.ClaimName
	}

	dataVolumeFunc := dataVolumeByNameFunc(c.dataVolumeInformer, dataVolumes)

	if volume.DataVolume != nil {
		// First, ensure DataVolume exists
		_, err := dataVolumeFunc(name, namespace)
		if err != nil {
			return false, false, services.DataVolumeNotFoundError{Reason: err.Error()}
		}
	}

	wffc := false
	ready := false
	// err is always nil
	pvcInterface, pvcExists, _ := c.pvcInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", namespace, name))
	if pvcExists {
		var err error
		pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
		ready, err = cdiv1.IsPopulated(pvc, dataVolumeFunc)
		if err != nil {
			return false, false, err
		}
		if !ready {
			waitsForFirstConsumer, err := cdiv1.IsWaitForFirstConsumerBeforePopulating(pvc, dataVolumeFunc)
			if err != nil {
				return false, false, err
			}
			if waitsForFirstConsumer {
				wffc = true
			}
		}
	} else {
		return false, false, services.PvcNotFoundError{Reason: fmt.Sprintf("didn't find PVC %v", name)}
	}
	return ready, wffc, nil
}

func (c *VMIController) volumeStatusContainsVolumeAndPod(volumeStatus []virtv1.VolumeStatus, volume *virtv1.Volume) bool {
	for _, status := range volumeStatus {
		if status.Name == volume.Name && status.HotplugVolume != nil && status.HotplugVolume.AttachPodName != "" {
			return true
		}
	}
	return false
}

func (c *VMIController) getNewHotplugVolumes(hotplugAttachmentPods []*k8sv1.Pod, hotplugVolumes []*virtv1.Volume) []*virtv1.Volume {
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

func (c *VMIController) getDeletedHotplugVolumes(hotplugPods []*k8sv1.Pod, hotplugVolumes []*virtv1.Volume) []k8sv1.Volume {
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

func (c *VMIController) deleteAttachmentPodForVolume(vmi *virtv1.VirtualMachineInstance, attachmentPod *k8sv1.Pod) error {
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
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, FailedDeletePodReason, "Failed to delete attachment pod %s", attachmentPod.Name)
		return err
	}
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, SuccessfulDeletePodReason, "Deleted attachment pod %s", attachmentPod.Name)
	return nil
}

func (c *VMIController) createAttachmentPodTemplate(vmi *virtv1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod, volumes []*virtv1.Volume) (*k8sv1.Pod, error) {
	logger := log.Log.Object(vmi)
	var pod *k8sv1.Pod
	var err error

	volumeNamesPVCMap, err := kubevirttypes.VirtVolumesToPVCMap(volumes, c.pvcInformer.GetStore(), virtlauncherPod.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get PVC map: %v", err)
	}
	for volumeName, pvc := range volumeNamesPVCMap {
		//Verify the PVC is ready to be used.
		populated, err := cdiv1.IsPopulated(pvc, func(name, namespace string) (*cdiv1.DataVolume, error) {
			dv, exists, _ := c.dataVolumeInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", namespace, name))
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
		pod, err = c.templateService.RenderHotplugAttachmentPodTemplate(volumes, virtlauncherPod, vmi, volumeNamesPVCMap, false)
	}
	return pod, err
}

func (c *VMIController) createAttachmentPopulateTriggerPodTemplate(volume *virtv1.Volume, virtlauncherPod *k8sv1.Pod, vmi *virtv1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	claimName := kubevirttypes.PVCNameFromVirtVolume(volume)
	if claimName == "" {
		return nil, errors.New("Unable to hotplug, claim not PVC or Datavolume")
	}

	pvc, exists, isBlock, err := kubevirttypes.IsPVCBlockFromStore(c.pvcInformer.GetStore(), virtlauncherPod.Namespace, claimName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Unable to trigger hotplug population, claim %s not found", claimName)
	}
	pod, err := c.templateService.RenderHotplugAttachmentTriggerPodTemplate(volume, virtlauncherPod, vmi, pvc.Name, isBlock, true)
	return pod, err
}

func (c *VMIController) deleteAllAttachmentPods(vmi *virtv1.VirtualMachineInstance) error {
	virtlauncherPod, err := controller.CurrentVMIPod(vmi, c.podInformer)
	if err != nil {
		return err
	}
	if virtlauncherPod != nil {
		attachmentPods, err := controller.AttachmentPods(virtlauncherPod, c.podInformer)
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

func (c *VMIController) deleteOrphanedAttachmentPods(vmi *virtv1.VirtualMachineInstance) error {
	pods, err := c.listPodsFromNamespace(vmi.Namespace)
	if err != nil {
		return fmt.Errorf("failed to list pods from namespace %s: %v", vmi.Namespace, err)
	}

	for _, pod := range pods {
		if !controller.IsControlledBy(pod, vmi) {
			continue
		}

		if !podIsDown(pod) {
			continue
		}

		attachmentPods, err := controller.AttachmentPods(pod, c.podInformer)
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

func (c *VMIController) updateVolumeStatus(vmi *virtv1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod) error {
	oldStatus := vmi.Status.DeepCopy().VolumeStatus
	oldStatusMap := make(map[string]virtv1.VolumeStatus)
	for _, status := range oldStatus {
		oldStatusMap[status.Name] = status
	}

	hotplugVolumes := getHotplugVolumes(vmi, virtlauncherPod)
	hotplugVolumesMap := make(map[string]*virtv1.Volume)
	for _, volume := range hotplugVolumes {
		hotplugVolumesMap[volume.Name] = volume
	}

	attachmentPods, err := controller.AttachmentPods(virtlauncherPod, c.podInformer)
	if err != nil {
		return err
	}
	newStatus := make([]virtv1.VolumeStatus, 0)
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
			attachmentPod := c.findAttachmentPodByVolumeName(volume.Name, attachmentPods)
			if attachmentPod == nil {
				status.HotplugVolume.AttachPodName = ""
				status.HotplugVolume.AttachPodUID = ""
				// Pod is gone, or hasn't been created yet, check for the PVC associated with the volume to set phase and message
				phase, reason, message := c.getVolumePhaseMessageReason(&vmi.Spec.Volumes[i], vmi.Namespace)
				status.Phase = phase
				status.Message = message
				status.Reason = reason
			} else {
				status.HotplugVolume.AttachPodName = attachmentPod.Name
				if len(attachmentPod.Status.ContainerStatuses) == 1 && attachmentPod.Status.ContainerStatuses[0].Ready {
					status.HotplugVolume.AttachPodUID = attachmentPod.UID
				}
				if c.canMoveToAttachedPhase(status.Phase) {
					status.Phase = virtv1.HotplugVolumeAttachedToNode
					status.Message = fmt.Sprintf("Created hotplug attachment pod %s, for volume %s", attachmentPod.Name, volume.Name)
					status.Reason = SuccessfulCreatePodReason
					c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, status.Reason, status.Message)
				}
			}
		}

		if volume.VolumeSource.PersistentVolumeClaim != nil || volume.VolumeSource.DataVolume != nil {

			var pvcName string
			if volume.VolumeSource.PersistentVolumeClaim != nil {
				pvcName = volume.VolumeSource.PersistentVolumeClaim.ClaimName
			} else if volume.VolumeSource.DataVolume != nil {
				pvcName = volume.VolumeSource.DataVolume.Name
			}

			pvcInterface, pvcExists, _ := c.pvcInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", vmi.Namespace, pvcName))
			if pvcExists {
				pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
				status.PersistentVolumeClaimInfo = &virtv1.PersistentVolumeClaimInfo{
					AccessModes:  pvc.Spec.AccessModes,
					VolumeMode:   pvc.Spec.VolumeMode,
					Capacity:     pvc.Status.Capacity,
					Preallocated: kubevirttypes.IsPreallocated(pvc.ObjectMeta.Annotations),
				}
				filesystemOverhead, err := c.getFilesystemOverhead(pvc)
				if errors.Is(err, failedToFindCdi) {
					filesystemOverhead = cdiv1.Percent("0.055")
					log.Log.V(3).Object(pvc).Infof("Didn't find CDI, continuing normally with filesystem overhead of 5.5%%")
				} else if err != nil {
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
				v.Reason = SuccessfulDeletePodReason
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

func (c *VMIController) getFilesystemOverhead(pvc *k8sv1.PersistentVolumeClaim) (cdiv1.Percent, error) {
	_, cdiExists, _ := c.cdiInformer.GetStore().GetByKey("cdi")
	if !cdiExists {
		return "0", failedToFindCdi
	}
	cdiConfigInterface, cdiConfigExists, err := c.cdiConfigInformer.GetStore().GetByKey("config")
	if !cdiConfigExists || err != nil {
		return "0", fmt.Errorf("Failed to find CDIConfig but CDI exists: %w", err)
	}
	cdiConfig, ok := cdiConfigInterface.(*cdiv1.CDIConfig)
	if !ok {
		return "0", fmt.Errorf("Failed to convert CDIConfig object %v to type CDIConfig", cdiConfigInterface)
	}
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == k8sv1.PersistentVolumeBlock {
		return "0", nil
	}
	scName := pvc.Spec.StorageClassName
	if scName == nil {
		return cdiConfig.Status.FilesystemOverhead.Global, nil
	}
	fsOverhead, ok := cdiConfig.Status.FilesystemOverhead.StorageClass[*scName]
	if !ok {
		return cdiConfig.Status.FilesystemOverhead.Global, nil
	}
	return fsOverhead, nil
}

func (c *VMIController) canMoveToAttachedPhase(currentPhase virtv1.VolumePhase) bool {
	return currentPhase == "" || currentPhase == virtv1.VolumeBound || currentPhase == virtv1.VolumePending ||
		currentPhase == virtv1.HotplugVolumeAttachedToNode
}

func (c *VMIController) findAttachmentPodByVolumeName(volumeName string, attachmentPods []*k8sv1.Pod) *k8sv1.Pod {
	for _, pod := range attachmentPods {
		for _, podVolume := range pod.Spec.Volumes {
			if podVolume.Name == volumeName {
				return pod
			}
		}
	}
	return nil
}

func (c *VMIController) getVolumePhaseMessageReason(volume *virtv1.Volume, namespace string) (virtv1.VolumePhase, string, string) {
	claimName := ""
	if volume.DataVolume != nil {
		// Using fact that PVC name = DV name.
		claimName = volume.DataVolume.Name
	}
	if volume.PersistentVolumeClaim != nil {
		claimName = volume.PersistentVolumeClaim.ClaimName
	}
	pvcInterface, pvcExists, _ := c.pvcInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", namespace, claimName))
	if !pvcExists {
		return virtv1.VolumePending, FailedPvcNotFoundReason, "Unable to determine PVC name"
	}
	pvc := pvcInterface.(*k8sv1.PersistentVolumeClaim)
	if pvc.Status.Phase == k8sv1.ClaimPending {
		return virtv1.VolumePending, PVCNotReadyReason, "PVC is in phase ClaimPending"
	} else if pvc.Status.Phase == k8sv1.ClaimBound {
		return virtv1.VolumeBound, PVCNotReadyReason, "PVC is in phase Bound"
	}
	return virtv1.VolumePending, PVCNotReadyReason, "PVC is in phase Lost"
}
