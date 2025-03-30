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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package replicaset

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
)

const failedRsKeyExtraction = "Failed to extract rsKey from replicaset."

// Reasons for replicaset events
const (

	// SuccessfulPausedReplicaSetReason is added in an event when the replica set discovered that it
	// should be paused. The event is triggered after it successfully managed to add the Paused Condition
	// to itself.
	SuccessfulPausedReplicaSetReason = "SuccessfulPaused"
	// SuccessfulResumedReplicaSetReason is added in an event when the replica set discovered that it
	// should be resumed. The event is triggered after it successfully managed to remove the Paused Condition
	// from itself.
	SuccessfulResumedReplicaSetReason = "SuccessfulResumed"
)

func NewController(vmiInformer cache.SharedIndexInformer, vmiRSInformer cache.SharedIndexInformer, recorder record.EventRecorder, clientset kubecli.KubevirtClient, burstReplicas uint) (*Controller, error) {
	c := &Controller{
		Queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-replicaset"},
		),
		vmiIndexer:    vmiInformer.GetIndexer(),
		vmiRSIndexer:  vmiRSInformer.GetIndexer(),
		recorder:      recorder,
		clientset:     clientset,
		expectations:  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		burstReplicas: burstReplicas,
	}

	c.hasSynced = func() bool {
		return vmiInformer.HasSynced() && vmiRSInformer.HasSynced()
	}

	_, err := vmiRSInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addReplicaSet,
		DeleteFunc: c.deleteReplicaSet,
		UpdateFunc: c.updateReplicaSet,
	})
	if err != nil {
		return nil, err
	}

	_, err = vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

type Controller struct {
	clientset     kubecli.KubevirtClient
	Queue         workqueue.TypedRateLimitingInterface[string]
	vmiIndexer    cache.Indexer
	vmiRSIndexer  cache.Indexer
	recorder      record.EventRecorder
	expectations  *controller.UIDTrackingControllerExpectations
	burstReplicas uint
	hasSynced     func() bool
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachineInstanceReplicaSet controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VirtualMachineInstanceReplicaSet controller.")
}

func (c *Controller) runWorker() {
	for c.Execute() {
	}
}

func (c *Controller) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineInstanceReplicaSet %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstanceReplicaSet %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *Controller) execute(key string) error {
	obj, exists, err := c.vmiRSIndexer.GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	rs := obj.(*virtv1.VirtualMachineInstanceReplicaSet)

	logger := log.Log.Object(rs)

	// this must be first step in execution. Writing the object
	// when api version changes ensures our api stored version is updated.
	if !controller.ObservedLatestApiVersionAnnotation(rs) {
		rs := rs.DeepCopy()
		controller.SetLatestApiVersionAnnotation(rs)
		_, err = c.clientset.ReplicaSet(rs.Namespace).Update(context.Background(), rs, metav1.UpdateOptions{})
		return err
	}

	// TODO default rs if necessary, the aggregated apiserver will do that in the future
	if rs.Spec.Template == nil || rs.Spec.Selector == nil || len(rs.Spec.Template.ObjectMeta.Labels) == 0 {
		logger.Error("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	selector, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector)
	if err != nil {
		logger.Reason(err).Error("Invalid selector on replicaset, will not re-enqueue.")
		return nil
	}

	if !selector.Matches(labels.Set(rs.Spec.Template.ObjectMeta.Labels)) {
		logger.Reason(err).Error("Selector does not match template labels, will not re-enqueue.")
		return nil
	}

	needsSync := c.expectations.SatisfiedExpectations(key)

	// get all potentially interesting VMIs from the cache
	vmis, err := c.listVMIsFromNamespace(rs.ObjectMeta.Namespace)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch vmis for namespace from cache.")
		return err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := c.clientset.ReplicaSet(rs.ObjectMeta.Namespace).Get(context.Background(), rs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != rs.ObjectMeta.UID {
			return nil, fmt.Errorf("original ReplicaSet %v/%v is gone: got uid %v, wanted %v", rs.Namespace, rs.Name, fresh.UID, rs.UID)
		}
		return fresh, nil
	})
	cm := controller.NewVirtualMachineControllerRefManager(controller.RealVirtualMachineControl{Clientset: c.clientset}, rs, selector, virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind, canAdoptFunc)
	vmis, err = cm.ClaimVirtualMachineInstances(vmis)
	if err != nil {
		return err
	}

	finishedVmis := append(c.filterFinishedVMIs(vmis), c.filterUnkownVMIs(vmis)...)
	activeVmis := c.filterActiveVMIs(vmis)

	var scaleErr error

	// Scale up or down, if all expected creates and deletes were report by the listener
	if needsSync && !rs.Spec.Paused && rs.ObjectMeta.DeletionTimestamp == nil {
		scaleErr = c.scale(rs, activeVmis)
		if len(finishedVmis) > 0 && scaleErr == nil {
			scaleErr = c.cleanFinishedVmis(rs, finishedVmis)
		}
	}

	if scaleErr != nil {
		logger.Reason(err).Error("Scaling the replicaset failed.")
	}

	err = c.updateStatus(rs.DeepCopy(), activeVmis, scaleErr)
	if err != nil {
		logger.Reason(err).Error("Updating the replicaset status failed.")
	}

	return scaleErr
}

func (c *Controller) scale(rs *virtv1.VirtualMachineInstanceReplicaSet, vmis []*virtv1.VirtualMachineInstance) error {
	log.Log.V(4).Object(rs).Info("Scale")
	diff := c.calcDiff(rs, vmis)

	rsKey, err := controller.KeyFunc(rs)
	if err != nil {
		log.Log.Object(rs).Reason(err).Error(failedRsKeyExtraction)
		return nil
	}

	if diff == 0 {
		return nil
	}

	// Make sure that we don't overload the cluster
	maxDiff := int(math.Min(math.Abs(float64(diff)), float64(c.burstReplicas)))

	// Every delete request can fail, give the channel enough room, to not block the go routines
	errChan := make(chan error, maxDiff)

	var wg sync.WaitGroup
	wg.Add(maxDiff)

	if diff > 0 {
		log.Log.V(4).Object(rs).Info("Delete excess VM's")
		// We have to delete VMIs, use a very simple selection strategy for now
		// TODO: Possible deletion order: not yet running VMIs < migrating VMIs < other
		deleteCandidates := vmis[0:maxDiff]
		c.expectations.ExpectDeletions(rsKey, controller.VirtualMachineInstanceKeys(deleteCandidates))
		for i := 0; i < maxDiff; i++ {
			go func(idx int) {
				defer wg.Done()
				deleteCandidate := vmis[idx]
				err := c.clientset.VirtualMachineInstance(rs.ObjectMeta.Namespace).Delete(context.Background(), deleteCandidate.ObjectMeta.Name, metav1.DeleteOptions{})
				// Don't log an error if it is already deleted
				if err != nil {
					// We can't observe a delete if it was not accepted by the server
					c.expectations.DeletionObserved(rsKey, controller.VirtualMachineInstanceKey(deleteCandidate))
					c.recorder.Eventf(rs, k8score.EventTypeWarning, common.FailedDeleteVirtualMachineReason, "Error deleting virtual machine instance %s: %v", deleteCandidate.ObjectMeta.Name, err)
					errChan <- err
					return
				}
				c.recorder.Eventf(rs, k8score.EventTypeNormal, common.SuccessfulDeleteVirtualMachineReason, "Stopped the virtual machine by deleting the virtual machine instance %v", deleteCandidate.ObjectMeta.UID)
			}(i)
		}

	} else if diff < 0 {
		log.Log.V(4).Object(rs).Info("Add missing VM's")
		// We have to create VMIs
		c.expectations.ExpectCreations(rsKey, maxDiff)
		basename := c.getVirtualMachineBaseName(rs)
		for range maxDiff {
			go func() {
				defer wg.Done()
				vmi := virtv1.NewVMIReferenceFromNameWithNS(rs.ObjectMeta.Namespace, "")
				vmi.ObjectMeta = rs.Spec.Template.ObjectMeta
				vmi.ObjectMeta.Name = ""
				vmi.ObjectMeta.GenerateName = basename
				vmi.Spec = rs.Spec.Template.Spec
				// TODO check if vmi labels exist, and when make sure that they match. For now just override them
				vmi.ObjectMeta.Labels = rs.Spec.Template.ObjectMeta.Labels
				vmi.ObjectMeta.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmi, err := c.clientset.VirtualMachineInstance(rs.ObjectMeta.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				if err != nil {
					c.expectations.CreationObserved(rsKey)
					c.recorder.Eventf(rs, k8score.EventTypeWarning, common.FailedCreateVirtualMachineReason, "Error creating virtual machine instance: %v", err)
					errChan <- err
					return
				}
				c.recorder.Eventf(rs, k8score.EventTypeNormal, common.SuccessfulCreateVirtualMachineReason, "Started the virtual machine by creating the new virtual machine instance %v", vmi.ObjectMeta.Name)
			}()
		}
	}
	wg.Wait()

	select {
	case err := <-errChan:
		// Only return the first error which occurred, the others will most likely be equal errors
		return err
	default:
	}
	return nil
}

// filterActiveVMIs takes a list of VMIs and returns all VMIs which are not in a final state, not terminating and not unknown
func (c *Controller) filterActiveVMIs(vmis []*virtv1.VirtualMachineInstance) []*virtv1.VirtualMachineInstance {
	return filter(vmis, func(vmi *virtv1.VirtualMachineInstance) bool {
		return !vmi.IsFinal() && vmi.DeletionTimestamp == nil &&
			!controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatusAndReason(vmi, virtv1.VirtualMachineInstanceConditionType(k8score.PodReady), k8score.ConditionFalse, virtv1.PodTerminatingReason)
	})
}

// filterReadyVMIs takes a list of VMIs and returns all VMIs which are in ready state.
func (c *Controller) filterReadyVMIs(vmis []*virtv1.VirtualMachineInstance) []*virtv1.VirtualMachineInstance {
	return filter(vmis, func(vmi *virtv1.VirtualMachineInstance) bool {
		return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceConditionType(k8score.PodReady), k8score.ConditionTrue)
	})
}

// filterFinishedVMIs takes a list of VMIs and returns all VMIs which are in final state.
func (c *Controller) filterFinishedVMIs(vmis []*virtv1.VirtualMachineInstance) []*virtv1.VirtualMachineInstance {
	return filter(vmis, func(vmi *virtv1.VirtualMachineInstance) bool {
		return vmi.IsFinal() && vmi.DeletionTimestamp == nil
	})
}

// filterUnknownVMIs takes a list of VMIs and returns all VMIs which are in an unknown and not yet terminating stage
func (c *Controller) filterUnkownVMIs(vmis []*virtv1.VirtualMachineInstance) []*virtv1.VirtualMachineInstance {
	return filter(vmis, func(vmi *virtv1.VirtualMachineInstance) bool {
		return !vmi.IsFinal() && vmi.DeletionTimestamp == nil &&
			controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatusAndReason(vmi, virtv1.VirtualMachineInstanceConditionType(k8score.PodReady), k8score.ConditionFalse, virtv1.PodTerminatingReason)
	})
}

func filter(vmis []*virtv1.VirtualMachineInstance, f func(vmi *virtv1.VirtualMachineInstance) bool) []*virtv1.VirtualMachineInstance {
	filtered := []*virtv1.VirtualMachineInstance{}
	for _, vmi := range vmis {
		if f(vmi) {
			filtered = append(filtered, vmi)
		}
	}
	return filtered
}

// listVMIsFromNamespace takes a namespace and returns all VMIs from the VirtualMachineInstance cache which run in this namespace
func (c *Controller) listVMIsFromNamespace(namespace string) ([]*virtv1.VirtualMachineInstance, error) {
	objs, err := c.vmiIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vmis := []*virtv1.VirtualMachineInstance{}
	for _, obj := range objs {
		vmis = append(vmis, obj.(*virtv1.VirtualMachineInstance))
	}
	return vmis, nil
}

// listControllerFromNamespace takes a namespace and returns all VMIReplicaSets from the ReplicaSet cache which run in this namespace
func (c *Controller) listControllerFromNamespace(namespace string) ([]*virtv1.VirtualMachineInstanceReplicaSet, error) {
	objs, err := c.vmiRSIndexer.ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	replicaSets := []*virtv1.VirtualMachineInstanceReplicaSet{}
	for _, obj := range objs {
		rs := obj.(*virtv1.VirtualMachineInstanceReplicaSet)
		replicaSets = append(replicaSets, rs)
	}
	return replicaSets, nil
}

// getMatchingController returns the first VMIReplicaSet which matches the labels of the VirtualMachineInstance from the listener cache.
// If there are no matching controllers, a NotFound error is returned.
func (c *Controller) getMatchingControllers(vmi *virtv1.VirtualMachineInstance) (rss []*virtv1.VirtualMachineInstanceReplicaSet) {
	logger := log.Log
	controllers, err := c.listControllerFromNamespace(vmi.ObjectMeta.Namespace)
	if err != nil {
		return nil
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, rs := range controllers {
		selector, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector)
		if err != nil {
			logger.Object(rs).Reason(err).Error("Failed to parse label selector from replicaset.")
			continue
		}

		if selector.Matches(labels.Set(vmi.ObjectMeta.Labels)) {
			rss = append(rss, rs)
		}

	}
	return rss
}

// When a vmi is created, enqueue the replica set that manages it and update its expectations.
func (c *Controller) addVirtualMachine(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)

	if vmi.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new vmi shows up in a state that
		// is already pending deletion. Prevent the vmi from being a creation observation.
		c.deleteVirtualMachine(vmi)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(vmi); controllerRef != nil {
		rs := c.resolveControllerRef(vmi.Namespace, controllerRef)
		if rs == nil {
			return
		}
		rsKey, err := controller.KeyFunc(rs)
		if err != nil {
			return
		}
		log.Log.V(4).Object(vmi).Infof("VirtualMachineInstance created")
		c.expectations.CreationObserved(rsKey)
		c.enqueueReplicaSet(rs)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching ReplicaSets and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	rss := c.getMatchingControllers(vmi)
	if len(rss) == 0 {
		return
	}
	log.Log.V(4).Object(vmi).Infof("Orphan VirtualMachineInstance created")
	for _, rs := range rss {
		c.enqueueReplicaSet(rs)
	}
}

// When a vmi is updated, figure out what replica set/s manage it and wake them
// up. If the labels of the vmi have changed we need to awaken both the old
// and new replica set. old and cur must be *metav1.VirtualMachineInstance types.
func (c *Controller) updateVirtualMachine(old, cur interface{}) {
	curVMI := cur.(*virtv1.VirtualMachineInstance)
	oldVMI := old.(*virtv1.VirtualMachineInstance)
	if curVMI.ResourceVersion == oldVMI.ResourceVersion {
		// Periodic resync will send update events for all known vmis.
		// Two different versions of the same vmi will always have different RVs.
		return
	}

	labelChanged := !equality.Semantic.DeepEqual(curVMI.Labels, oldVMI.Labels)
	if curVMI.DeletionTimestamp != nil {
		// when a vmi is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
		// and after such time has passed, the virt-handler actually deletes it from the store. We receive an update
		// for modification of the deletion timestamp and expect an rs to create more replicas asap, not wait
		// until the virt-handler actually deletes the vmi. This is different from the Phase of a vmi changing, because
		// an rs never initiates a phase change, and so is never asleep waiting for the same.
		c.deleteVirtualMachine(curVMI)
		if labelChanged {
			// we don't need to check the oldVMI.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deleteVirtualMachine(oldVMI)
		}
		return
	}

	curControllerRef := metav1.GetControllerOf(curVMI)
	oldControllerRef := metav1.GetControllerOf(oldVMI)
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if rs := c.resolveControllerRef(oldVMI.Namespace, oldControllerRef); rs != nil {
			c.enqueueReplicaSet(rs)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		rs := c.resolveControllerRef(curVMI.Namespace, curControllerRef)
		if rs == nil {
			return
		}
		log.Log.V(4).Object(curVMI).Infof("VirtualMachineInstance updated")
		c.enqueueReplicaSet(rs)
		// TODO: MinReadySeconds in the VirtualMachineInstance will generate an Available condition to be added in
		// Update once we support the available conect on the rs
		return
	}

	// Otherwise, it's an orphan. If anything changed, sync matching controllers
	// to see if anyone wants to adopt it now.
	if labelChanged || controllerRefChanged {
		rss := c.getMatchingControllers(curVMI)
		if len(rss) == 0 {
			return
		}
		log.Log.V(4).Object(curVMI).Infof("Orphan VirtualMachineInstance updated")
		for _, rs := range rss {
			c.enqueueReplicaSet(rs)
		}
	}
}

// When a vmi is deleted, enqueue the replica set that manages the vmi and update its expectations.
// obj could be an *metav1.VirtualMachineInstance, or a DeletionFinalStateUnknown marker item.
func (c *Controller) deleteVirtualMachine(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a vmi in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the vmi
	// changed labels the new ReplicaSet will not be woken up till the periodic resync.
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

	controllerRef := metav1.GetControllerOf(vmi)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	rs := c.resolveControllerRef(vmi.Namespace, controllerRef)
	if rs == nil {
		return
	}
	rsKey, err := controller.KeyFunc(rs)
	if err != nil {
		return
	}
	c.expectations.DeletionObserved(rsKey, controller.VirtualMachineInstanceKey(vmi))
	c.enqueueReplicaSet(rs)
}

func (c *Controller) addReplicaSet(obj interface{}) {
	c.enqueueReplicaSet(obj)
}

func (c *Controller) deleteReplicaSet(obj interface{}) {
	c.enqueueReplicaSet(obj)
}

func (c *Controller) updateReplicaSet(_, curr interface{}) {
	c.enqueueReplicaSet(curr)
}

func (c *Controller) enqueueReplicaSet(obj interface{}) {
	logger := log.Log
	rs := obj.(*virtv1.VirtualMachineInstanceReplicaSet)
	key, err := controller.KeyFunc(rs)
	if err != nil {
		logger.Object(rs).Reason(err).Error(failedRsKeyExtraction)
		return
	}
	c.Queue.Add(key)
}

func (c *Controller) hasCondition(rs *virtv1.VirtualMachineInstanceReplicaSet, cond virtv1.VirtualMachineInstanceReplicaSetConditionType) bool {
	for _, c := range rs.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (c *Controller) removeCondition(rs *virtv1.VirtualMachineInstanceReplicaSet, cond virtv1.VirtualMachineInstanceReplicaSetConditionType) {
	var conds []virtv1.VirtualMachineInstanceReplicaSetCondition
	for _, c := range rs.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	rs.Status.Conditions = conds
}

func (c *Controller) updateStatus(rs *virtv1.VirtualMachineInstanceReplicaSet, vmis []*virtv1.VirtualMachineInstance, scaleErr error) error {
	diff := c.calcDiff(rs, vmis)
	readyReplicas := int32(len(c.filterReadyVMIs(vmis)))
	labelSelector, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector)
	if err != nil {
		return err
	}

	// check if we have reached the equilibrium
	statesMatch := int32(len(vmis)) == rs.Status.Replicas && readyReplicas == rs.Status.ReadyReplicas

	// check if we need to update because of appeared or disappeared errors
	errorsMatch := (scaleErr != nil) == c.hasCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaFailure)

	// check if we need to update because pause was modified
	pausedMatch := rs.Spec.Paused == c.hasCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaPaused)

	// check if the label selector changed
	labelSelectorMatch := labelSelector.String() == rs.Status.LabelSelector

	// in case the replica count matches and the scaleErr and the error condition equal, don't update
	if statesMatch && errorsMatch && pausedMatch && labelSelectorMatch {
		return nil
	}

	rs.Status.LabelSelector = labelSelector.String()
	rs.Status.Replicas = int32(len(vmis))
	rs.Status.ReadyReplicas = readyReplicas

	// Add/Remove Paused condition
	c.checkPaused(rs)

	// Add/Remove Failure condition if necessary
	c.checkFailure(rs, diff, scaleErr)

	_, err = c.clientset.ReplicaSet(rs.Namespace).UpdateStatus(context.Background(), rs, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	// Finally trigger resumed or paused events
	if !pausedMatch {
		if rs.Spec.Paused {
			c.recorder.Eventf(rs, k8score.EventTypeNormal, SuccessfulPausedReplicaSetReason, "Paused")
		} else {
			c.recorder.Eventf(rs, k8score.EventTypeNormal, SuccessfulResumedReplicaSetReason, "Resumed")
		}
	}

	return nil
}

func (c *Controller) calcDiff(rs *virtv1.VirtualMachineInstanceReplicaSet, vmis []*virtv1.VirtualMachineInstance) int {
	// TODO default this on the aggregated api server
	wantedReplicas := int32(1)
	if rs.Spec.Replicas != nil {
		wantedReplicas = *rs.Spec.Replicas
	}

	return len(vmis) - int(wantedReplicas)
}

func (c *Controller) getVirtualMachineBaseName(replicaset *virtv1.VirtualMachineInstanceReplicaSet) string {
	// TODO defaulting should make sure that the right field is set, instead of doing this
	if len(replicaset.Spec.Template.ObjectMeta.Name) > 0 {
		return replicaset.Spec.Template.ObjectMeta.Name
	}
	if len(replicaset.Spec.Template.ObjectMeta.GenerateName) > 0 {
		return replicaset.Spec.Template.ObjectMeta.GenerateName
	}
	return replicaset.ObjectMeta.Name
}

func (c *Controller) checkPaused(rs *virtv1.VirtualMachineInstanceReplicaSet) {
	if rs.Spec.Paused == true && !c.hasCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaPaused) {
		rs.Status.Conditions = append(rs.Status.Conditions, virtv1.VirtualMachineInstanceReplicaSetCondition{
			Type:               virtv1.VirtualMachineInstanceReplicaSetReplicaPaused,
			Reason:             "Paused",
			Message:            "Controller got paused",
			LastTransitionTime: metav1.Now(),
			Status:             k8score.ConditionTrue,
		})
	} else if rs.Spec.Paused == false && c.hasCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaPaused) {
		c.removeCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaPaused)
	}
}

func (c *Controller) checkFailure(rs *virtv1.VirtualMachineInstanceReplicaSet, diff int, scaleErr error) {
	if scaleErr != nil && !c.hasCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaFailure) {
		var reason string
		if diff < 0 {
			reason = "FailedCreate"
		} else {
			reason = "FailedDelete"
		}

		rs.Status.Conditions = append(rs.Status.Conditions, virtv1.VirtualMachineInstanceReplicaSetCondition{
			Type:               virtv1.VirtualMachineInstanceReplicaSetReplicaFailure,
			Reason:             reason,
			Message:            scaleErr.Error(),
			LastTransitionTime: metav1.Now(),
			Status:             k8score.ConditionTrue,
		})

	} else if scaleErr == nil && c.hasCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaFailure) {
		c.removeCondition(rs, virtv1.VirtualMachineInstanceReplicaSetReplicaFailure)
	}
}

func OwnerRef(rs *virtv1.VirtualMachineInstanceReplicaSet) metav1.OwnerReference {
	t := true
	gvk := virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind
	return metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               rs.ObjectMeta.Name,
		UID:                rs.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *Controller) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) *virtv1.VirtualMachineInstanceReplicaSet {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind {
		return nil
	}
	rs, exists, err := c.vmiRSIndexer.GetByKey(controller.NamespacedKey(namespace, controllerRef.Name))
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if rs.(*virtv1.VirtualMachineInstanceReplicaSet).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return rs.(*virtv1.VirtualMachineInstanceReplicaSet)
}

func (c *Controller) cleanFinishedVmis(rs *virtv1.VirtualMachineInstanceReplicaSet, vmis []*virtv1.VirtualMachineInstance) error {
	rsKey, err := controller.KeyFunc(rs)
	if err != nil {
		log.Log.Object(rs).Reason(err).Error(failedRsKeyExtraction)
		return nil
	}

	diff := int(math.Min(float64(len(vmis)), float64(c.burstReplicas)))

	// Every delete request can fail, give the channel enough room, to not block the go routines
	errChan := make(chan error, diff)

	var wg sync.WaitGroup
	wg.Add(diff)

	log.Log.V(4).Object(rs).Info("Delete finished VM's")
	deleteCandidates := vmis[0:diff]
	c.expectations.ExpectDeletions(rsKey, controller.VirtualMachineInstanceKeys(deleteCandidates))
	for i := 0; i < diff; i++ {
		go func(idx int) {
			defer wg.Done()
			deleteCandidate := vmis[idx]
			err := c.clientset.VirtualMachineInstance(rs.ObjectMeta.Namespace).Delete(context.Background(), deleteCandidate.ObjectMeta.Name, metav1.DeleteOptions{})
			// Don't log an error if it is already deleted
			if err != nil {
				// We can't observe a delete if it was not accepted by the server
				c.expectations.DeletionObserved(rsKey, controller.VirtualMachineInstanceKey(deleteCandidate))
				c.recorder.Eventf(rs, k8score.EventTypeWarning, common.FailedDeleteVirtualMachineReason, "Error deleting finished virtual machine %s: %v", deleteCandidate.ObjectMeta.Name, err)
				errChan <- err
				return
			}
			c.recorder.Eventf(rs, k8score.EventTypeNormal, common.SuccessfulDeleteVirtualMachineReason, "Deleted finished virtual machine: %v", deleteCandidate.ObjectMeta.UID)
		}(i)
	}
	wg.Wait()

	select {
	case err := <-errChan:
		// Only return the first error which occurred, the others will most likely be equal errors
		return err
	default:
	}
	return nil
}
