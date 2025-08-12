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
 * Copyright The KubeVirt Authors.
 *
 */

package vmi

import (
	"context"
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/trace"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
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
	netMigrationEvaluator migrationEvaluator,
	allPodInformer cache.SharedIndexInformer,
	namespaceInformer cache.SharedIndexInformer,
	nodeInformer cache.SharedIndexInformer,
	resourceQuotaInformer cache.SharedIndexInformer,
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
		netMigrationEvaluator:   netMigrationEvaluator,

		allPodIndexer:    allPodInformer.GetIndexer(),
		namespaceIndexer: namespaceInformer.GetIndexer(),
		nodeIndexer:      nodeInformer.GetIndexer(),

		resourceQuotaIndexer: resourceQuotaInformer.GetIndexer(),
	}

	c.hasSynced = func() bool {
		return vmInformer.HasSynced() && vmiInformer.HasSynced() && podInformer.HasSynced() &&
			dataVolumeInformer.HasSynced() && cdiConfigInformer.HasSynced() && cdiInformer.HasSynced() &&
			pvcInformer.HasSynced() && storageClassInformer.HasSynced() && storageProfileInformer.HasSynced() &&
			allPodInformer.HasSynced() && namespaceInformer.HasSynced() && nodeInformer.HasSynced()
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

	_, err = resourceQuotaInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateResourceQuota,
		DeleteFunc: c.deleteResourceQuota,
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

type migrationEvaluator interface {
	// Evaluate determines if a VMI should request an automatic migration.
	//
	// The method returns one of three values:
	//   * ConditionUnknown: No action needed; the VMI should not be marked for auto-migration.
	//   * ConditionTrue: Mark the VMI for immediate migration.
	//   * ConditionFalse: Mark the VMI for pending migration.
	Evaluate(vmi *virtv1.VirtualMachineInstance) k8sv1.ConditionStatus
}

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
	netMigrationEvaluator   migrationEvaluator

	allPodIndexer    cache.Indexer
	namespaceIndexer cache.Indexer
	nodeIndexer      cache.Indexer

	resourceQuotaIndexer cache.Indexer
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
			c.vmiExpectations.SetExpectations(key, 0, 0)
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

// When a resourceQuota is updated, check if there are any VirtualMachineInstances (VMI) in the namespace
// that have not had their Pods created due to quota limitations.
// If such VMIs exist, enqueue them to expedite the Pod creation process.
func (c *Controller) updateResourceQuota(_, cur interface{}) {
	curResourceQuota := cur.(*k8sv1.ResourceQuota)
	log.Log.V(4).Object(curResourceQuota).Infof("ResourceQuota updated")
	objs, _ := c.vmiIndexer.ByIndex(cache.NamespaceIndex, curResourceQuota.Namespace)
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		if vmi.Status.Conditions == nil {
			continue
		}
		for _, cond := range vmi.Status.Conditions {
			if cond.Type == virtv1.VirtualMachineInstanceSynchronized && cond.Reason == controller.FailedCreatePodReason {
				c.enqueueVirtualMachine(vmi)
			}
		}
	}
	return
}

// When a resourceQuota is deleted, check if there are any VirtualMachineInstances (VMI) in the namespace
// that have not had their Pods created due to quota limitations.
// If such VMIs exist, enqueue them to expedite the Pod creation process.
func (c *Controller) deleteResourceQuota(obj interface{}) {
	resourceQuota := obj.(*k8sv1.ResourceQuota)
	log.Log.V(4).Object(resourceQuota).Infof("ResourceQuota deleted")
	objs, _ := c.vmiIndexer.ByIndex(cache.NamespaceIndex, resourceQuota.Namespace)
	for _, obj := range objs {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		if vmi.Status.Conditions == nil {
			continue
		}
		for _, cond := range vmi.Status.Conditions {
			if cond.Type == virtv1.VirtualMachineInstanceSynchronized && cond.Reason == controller.FailedCreatePodReason {
				c.enqueueVirtualMachine(vmi)
			}
		}
	}
	return
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
	c.vmiExpectations.SetExpectations(key, 0, 0)
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

func (c *Controller) volumeStatusContainsVolumeAndPod(volumeStatus []virtv1.VolumeStatus, volume *virtv1.Volume) bool {
	for _, status := range volumeStatus {
		if status.Name == volume.Name && status.HotplugVolume != nil && status.HotplugVolume.AttachPodName != "" {
			return true
		}
	}
	return false
}
