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

package watch

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sync"

	k8score "k8s.io/api/core/v1"

	"fmt"

	"reflect"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

// Reasons for replicaset events
const (
	// FailedCreateVirtualMachineReason is added in an event and in a replica set condition
	// when a virtual machine for a replica set is failed to be created.
	FailedCreateVirtualMachineReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineReason is added in an event when a virtual machine for a replica set
	// is successfully created.
	SuccessfulCreateVirtualMachineReason = "SuccessfulCreate"
	// FailedDeleteVirtualMachineReason is added in an event and in a replica set condition
	// when a virtual machine for a replica set is failed to be deleted.
	FailedDeleteVirtualMachineReason = "FailedDelete"
	// SuccessfulDeleteVirtualMachineReason is added in an event when a virtual machine for a replica set
	// is successfully deleted.
	SuccessfulDeleteVirtualMachineReason = "SuccessfulDelete"
	// SuccessfulPausedReplicaSetReason is added in an event when the replica set discovered that it
	// should be paused. The event is triggered after it successfully managed to add the Paused Condition
	// to itself.
	SuccessfulPausedReplicaSetReason = "SuccessfulPaused"
	// SuccessfulResumedReplicaSetReason is added in an event when the replica set discovered that it
	// should be resumed. The event is triggered after it successfully managed to remove the Paused Condition
	// from itself.
	SuccessfulResumedReplicaSetReason = "SuccessfulResumed"
)

func NewVMReplicaSet(vmInformer cache.SharedIndexInformer, vmRSInformer cache.SharedIndexInformer, recorder record.EventRecorder, clientset kubecli.KubevirtClient, burstReplicas uint) *VMReplicaSet {

	c := &VMReplicaSet{
		Queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmInformer:    vmInformer,
		vmRSInformer:  vmRSInformer,
		recorder:      recorder,
		clientset:     clientset,
		expectations:  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		burstReplicas: burstReplicas,
	}

	c.vmRSInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addReplicaSet,
		DeleteFunc: c.deleteReplicaSet,
		UpdateFunc: c.updateReplicaSet,
	})

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	return c
}

type VMReplicaSet struct {
	clientset     kubecli.KubevirtClient
	Queue         workqueue.RateLimitingInterface
	vmInformer    cache.SharedIndexInformer
	vmRSInformer  cache.SharedIndexInformer
	recorder      record.EventRecorder
	expectations  *controller.UIDTrackingControllerExpectations
	burstReplicas uint
}

func (c *VMReplicaSet) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachineReplicaSet controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmRSInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VirtualMachineReplicaSet controller.")
}

func (c *VMReplicaSet) runWorker() {
	for c.Execute() {
	}
}

func (c *VMReplicaSet) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineReplicaSet %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineReplicaSet %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *VMReplicaSet) execute(key string) error {

	obj, exists, err := c.vmRSInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	rs := obj.(*virtv1.VirtualMachineReplicaSet)

	logger := log.Log.Object(rs)

	//TODO default rs if necessary, the aggregated apiserver will do that in the future
	if rs.Spec.Template == nil || rs.Spec.Selector == nil || len(rs.Spec.Template.ObjectMeta.Labels) == 0 {
		logger.Error("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	selector, err := v1.LabelSelectorAsSelector(rs.Spec.Selector)
	if err != nil {
		logger.Reason(err).Error("Invalid selector on replicaset, will not re-enqueue.")
		return nil
	}

	if !selector.Matches(labels.Set(rs.Spec.Template.ObjectMeta.Labels)) {
		logger.Reason(err).Error("Selector does not match template labels, will not re-enqueue.")
		return nil
	}

	needsSync := c.expectations.SatisfiedExpectations(key)

	// get all potentially interesting VMs from the cache
	vms, err := c.listVMsFromNamespace(rs.ObjectMeta.Namespace)

	if err != nil {
		logger.Reason(err).Error("Failed to fetch vms for namespace from cache.")
		return err
	}

	vms = c.filterActiveVMs(vms)

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (v1.Object, error) {
		fresh, err := c.clientset.ReplicaSet(rs.ObjectMeta.Namespace).Get(rs.ObjectMeta.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != rs.ObjectMeta.UID {
			return nil, fmt.Errorf("original ReplicaSet %v/%v is gone: got uid %v, wanted %v", rs.Namespace, rs.Name, fresh.UID, rs.UID)
		}
		return fresh, nil
	})
	cm := controller.NewVirtualMachineControllerRefManager(controller.RealVirtualMachineControl{Clientset: c.clientset}, rs, selector, virtv1.VMReplicaSetGroupVersionKind, canAdoptFunc)
	vms, err = cm.ClaimVirtualMachines(vms)
	if err != nil {
		return err
	}

	var scaleErr error

	// Scale up or down, if all expected creates and deletes were report by the listener
	if needsSync && !rs.Spec.Paused && rs.ObjectMeta.DeletionTimestamp == nil {
		scaleErr = c.scale(rs, vms)
	}

	// If the controller is going to be deleted and the orphan finalizer is the next one, release the VMs. Don't update the status
	// TODO: Workaround for https://github.com/kubernetes/kubernetes/issues/56348, remove it once it is fixed
	if rs.ObjectMeta.DeletionTimestamp != nil && controller.HasFinalizer(rs, v1.FinalizerOrphanDependents) {
		return c.orphan(cm, rs, vms)
	}

	if scaleErr != nil {
		logger.Reason(err).Error("Scaling the replicaset failed.")
	}

	err = c.updateStatus(rs.DeepCopy(), vms, scaleErr)
	if err != nil {
		logger.Reason(err).Error("Updating the replicaset status failed.")
	}

	return err
}

// orphan removes the owner reference of all VMs which are owned by the controller instance.
// Workaround for https://github.com/kubernetes/kubernetes/issues/56348 to make no-cascading deletes possible
// We don't have to remove the finalizer. This part of the gc is not affected by the mentioned bug
func (c *VMReplicaSet) orphan(cm *controller.VirtualMachineControllerRefManager, rs *virtv1.VirtualMachineReplicaSet, vms []*virtv1.VirtualMachine) error {

	var wg sync.WaitGroup
	errChan := make(chan error, len(vms))
	wg.Add(len(vms))

	for _, vm := range vms {
		go func(vm *virtv1.VirtualMachine) {
			defer wg.Done()
			err := cm.ReleaseVirtualMachine(vm)
			if err != nil {
				errChan <- err
			}
		}(vm)
	}
	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
	}
	return nil
}

func (c *VMReplicaSet) scale(rs *virtv1.VirtualMachineReplicaSet, vms []*virtv1.VirtualMachine) error {

	diff := c.calcDiff(rs, vms)
	rsKey, err := controller.KeyFunc(rs)
	if err != nil {
		log.Log.Object(rs).Reason(err).Error("Failed to extract rsKey from replicaset.")
		return nil
	}

	if diff == 0 {
		return nil
	}

	// Make sure that we don't overload the cluster
	diff = limit(diff, c.burstReplicas)

	// Every delete request can fail, give the channel enough room, to not block the go routines
	errChan := make(chan error, abs(diff))

	var wg sync.WaitGroup
	wg.Add(abs(diff))

	if diff > 0 {
		// We have to delete VMs, use a very simple selection strategy for now
		// TODO: Possible deletion order: not yet running VMs < migrating VMs < other
		deleteCandidates := vms[0:diff]
		c.expectations.ExpectDeletions(rsKey, controller.VirtualMachineKeys(deleteCandidates))
		for i := 0; i < diff; i++ {
			go func(idx int) {
				defer wg.Done()
				deleteCandidate := vms[idx]
				err := c.clientset.VM(rs.ObjectMeta.Namespace).Delete(deleteCandidate.ObjectMeta.Name, &v1.DeleteOptions{})
				// Don't log an error if it is already deleted
				if err != nil {
					// We can't observe a delete if it was not accepted by the server
					c.expectations.DeletionObserved(rsKey, controller.VirtualMachineKey(deleteCandidate))
					c.recorder.Eventf(rs, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting virtual machine %s: %v", deleteCandidate.ObjectMeta.Name, err)
					errChan <- err
					return
				}
				c.recorder.Eventf(rs, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Deleted virtual machine: %v", deleteCandidate.ObjectMeta.UID)
			}(i)
		}

	} else if diff < 0 {
		// We have to create VMs
		c.expectations.ExpectCreations(rsKey, abs(diff))
		basename := c.getVirtualMachineBaseName(rs)
		for i := diff; i < 0; i++ {
			go func() {
				defer wg.Done()
				vm := virtv1.NewVMReferenceFromNameWithNS(rs.ObjectMeta.Namespace, "")
				vm.ObjectMeta = rs.Spec.Template.ObjectMeta
				vm.ObjectMeta.Name = ""
				vm.ObjectMeta.GenerateName = basename
				vm.Spec = rs.Spec.Template.Spec
				// TODO check if vm labels exist, and when make sure that they match. For now just override them
				vm.ObjectMeta.Labels = rs.Spec.Template.ObjectMeta.Labels
				vm.ObjectMeta.OwnerReferences = []v1.OwnerReference{OwnerRef(rs)}
				vm, err := c.clientset.VM(rs.ObjectMeta.Namespace).Create(vm)
				if err != nil {
					c.expectations.CreationObserved(rsKey)
					c.recorder.Eventf(rs, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine: %v", err)
					errChan <- err
					return
				}
				c.recorder.Eventf(rs, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Created virtual machine: %v", vm.ObjectMeta.Name)
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

// filterActiveVMs takes a list of VMs and returns all VMs which are not in a final state
func (c *VMReplicaSet) filterActiveVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return !vm.IsFinal()
	})
}

// filterReadyVMs takes a list of VMs and returns all VMs which are in ready state.
func (c *VMReplicaSet) filterReadyVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return vm.IsReady()
	})
}

func filter(vms []*virtv1.VirtualMachine, f func(vm *virtv1.VirtualMachine) bool) []*virtv1.VirtualMachine {
	filtered := []*virtv1.VirtualMachine{}
	for _, vm := range vms {
		if f(vm) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// listVMsFromNamespace takes a namespace and returns all VMs from the VM cache which run in this namespace
func (c *VMReplicaSet) listVMsFromNamespace(namespace string) ([]*virtv1.VirtualMachine, error) {
	objs, err := c.vmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vms := []*virtv1.VirtualMachine{}
	for _, obj := range objs {
		vms = append(vms, obj.(*virtv1.VirtualMachine))
	}
	return vms, nil
}

// listControllerFromNamespace takes a namespace and returns all VMReplicaSets from the ReplicaSet cache which run in this namespace
func (c *VMReplicaSet) listControllerFromNamespace(namespace string) ([]*virtv1.VirtualMachineReplicaSet, error) {
	objs, err := c.vmRSInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	replicaSets := []*virtv1.VirtualMachineReplicaSet{}
	for _, obj := range objs {
		rs := obj.(*virtv1.VirtualMachineReplicaSet)
		replicaSets = append(replicaSets, rs)
	}
	return replicaSets, nil
}

// getMatchingController returns the first VMReplicaSet which matches the labels of the VM from the listener cache.
// If there are no matching controllers, a NotFound error is returned.
func (c *VMReplicaSet) getMatchingControllers(vm *virtv1.VirtualMachine) (rss []*virtv1.VirtualMachineReplicaSet) {
	logger := log.Log
	controllers, err := c.listControllerFromNamespace(vm.ObjectMeta.Namespace)
	if err != nil {
		return nil
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, rs := range controllers {
		selector, err := v1.LabelSelectorAsSelector(rs.Spec.Selector)
		if err != nil {
			logger.Object(rs).Reason(err).Error("Failed to parse label selector from replicaset.")
			continue
		}

		if selector.Matches(labels.Set(vm.ObjectMeta.Labels)) {
			rss = append(rss, rs)
		}

	}
	return rss
}

// When a vm is created, enqueue the replica set that manages it and update its expectations.
func (c *VMReplicaSet) addVirtualMachine(obj interface{}) {
	vm := obj.(*virtv1.VirtualMachine)

	if vm.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new vm shows up in a state that
		// is already pending deletion. Prevent the vm from being a creation observation.
		c.deleteVirtualMachine(vm)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := controller.GetControllerOf(vm); controllerRef != nil {
		rs := c.resolveControllerRef(vm.Namespace, controllerRef)
		if rs == nil {
			return
		}
		rsKey, err := controller.KeyFunc(rs)
		if err != nil {
			return
		}
		log.Log.V(4).Object(vm).Infof("VirtualMachine created")
		c.expectations.CreationObserved(rsKey)
		c.enqueueReplicaSet(rs)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching ReplicaSets and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	rss := c.getMatchingControllers(vm)
	if len(rss) == 0 {
		return
	}
	log.Log.V(4).Object(vm).Infof("Orphan VirtualMachine created")
	for _, rs := range rss {
		c.enqueueReplicaSet(rs)
	}
}

// When a vm is updated, figure out what replica set/s manage it and wake them
// up. If the labels of the vm have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.VirtualMachine types.
func (c *VMReplicaSet) updateVirtualMachine(old, cur interface{}) {
	curVM := cur.(*virtv1.VirtualMachine)
	oldVM := old.(*virtv1.VirtualMachine)
	if curVM.ResourceVersion == oldVM.ResourceVersion {
		// Periodic resync will send update events for all known vms.
		// Two different versions of the same vm will always have different RVs.
		return
	}

	labelChanged := !reflect.DeepEqual(curVM.Labels, oldVM.Labels)
	if curVM.DeletionTimestamp != nil {
		// when a vm is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
		// and after such time has passed, the virt-handler actually deletes it from the store. We receive an update
		// for modification of the deletion timestamp and expect an rs to create more replicas asap, not wait
		// until the virt-handler actually deletes the vm. This is different from the Phase of a vm changing, because
		// an rs never initiates a phase change, and so is never asleep waiting for the same.
		c.deleteVirtualMachine(curVM)
		if labelChanged {
			// we don't need to check the oldVM.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deleteVirtualMachine(oldVM)
		}
		return
	}

	curControllerRef := controller.GetControllerOf(curVM)
	oldControllerRef := controller.GetControllerOf(oldVM)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if rs := c.resolveControllerRef(oldVM.Namespace, oldControllerRef); rs != nil {
			c.enqueueReplicaSet(rs)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		rs := c.resolveControllerRef(curVM.Namespace, curControllerRef)
		if rs == nil {
			return
		}
		log.Log.V(4).Object(curVM).Infof("VirtualMachine updated")
		c.enqueueReplicaSet(rs)
		// TODO: MinReadySeconds in the VM will generate an Available condition to be added in
		// Update once we support the available conect on the rs
		return
	}

	// Otherwise, it's an orphan. If anything changed, sync matching controllers
	// to see if anyone wants to adopt it now.
	if labelChanged || controllerRefChanged {
		rss := c.getMatchingControllers(curVM)
		if len(rss) == 0 {
			return
		}
		log.Log.V(4).Object(curVM).Infof("Orphan VirtualMachine updated")
		for _, rs := range rss {
			c.enqueueReplicaSet(rs)
		}
	}
}

// When a vm is deleted, enqueue the replica set that manages the vm and update its expectations.
// obj could be an *v1.VirtualMachine, or a DeletionFinalStateUnknown marker item.
func (c *VMReplicaSet) deleteVirtualMachine(obj interface{}) {
	vm, ok := obj.(*virtv1.VirtualMachine)

	// When a delete is dropped, the relist will notice a vm in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the vm
	// changed labels the new ReplicaSet will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error("Failed to process delete notification")
			return
		}
		vm, ok = tombstone.Obj.(*virtv1.VirtualMachine)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vm %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}

	controllerRef := controller.GetControllerOf(vm)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	rs := c.resolveControllerRef(vm.Namespace, controllerRef)
	if rs == nil {
		return
	}
	rsKey, err := controller.KeyFunc(rs)
	if err != nil {
		return
	}
	c.expectations.DeletionObserved(rsKey, controller.VirtualMachineKey(vm))
	c.enqueueReplicaSet(rs)
}

func (c *VMReplicaSet) addReplicaSet(obj interface{}) {
	c.enqueueReplicaSet(obj)
}

func (c *VMReplicaSet) deleteReplicaSet(obj interface{}) {
	c.enqueueReplicaSet(obj)
}

func (c *VMReplicaSet) updateReplicaSet(old, curr interface{}) {
	c.enqueueReplicaSet(curr)
}

func (c *VMReplicaSet) enqueueReplicaSet(obj interface{}) {
	logger := log.Log
	rs := obj.(*virtv1.VirtualMachineReplicaSet)
	key, err := controller.KeyFunc(rs)
	if err != nil {
		logger.Object(rs).Reason(err).Error("Failed to extract rsKey from replicaset.")
	}
	c.Queue.Add(key)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

//limit
func limit(x int, burstReplicas uint) int {
	replicas := int(burstReplicas)
	if x <= 0 {
		return max(x, -replicas)
	}
	return min(x, replicas)
}

func (c *VMReplicaSet) hasCondition(rs *virtv1.VirtualMachineReplicaSet, cond virtv1.VMReplicaSetConditionType) bool {
	for _, c := range rs.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (c *VMReplicaSet) removeCondition(rs *virtv1.VirtualMachineReplicaSet, cond virtv1.VMReplicaSetConditionType) {
	var conds []virtv1.VMReplicaSetCondition
	for _, c := range rs.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	rs.Status.Conditions = conds
}

func (c *VMReplicaSet) updateStatus(rs *virtv1.VirtualMachineReplicaSet, vms []*virtv1.VirtualMachine, scaleErr error) error {

	diff := c.calcDiff(rs, vms)

	readyReplicas := int32(len(c.filterReadyVMs(vms)))

	// check if we have reached the equilibrium
	statesMatch := int32(len(vms)) == rs.Status.Replicas && readyReplicas == rs.Status.ReadyReplicas

	// check if we need to update because of appeared or disappeard errors
	errorsMatch := (scaleErr != nil) == c.hasCondition(rs, virtv1.VMReplicaSetReplicaFailure)

	// check if we need to update because pause was modified
	pausedMatch := rs.Spec.Paused == c.hasCondition(rs, virtv1.VMReplicaSetReplicaPaused)

	// in case the replica count matches and the scaleErr and the error condition equal, don't update
	if statesMatch && errorsMatch && pausedMatch {
		return nil
	}

	rs.Status.Replicas = int32(len(vms))
	rs.Status.ReadyReplicas = readyReplicas

	// Add/Remove Paused condition
	c.checkPaused(rs)

	// Add/Remove Failure condition if necessary
	c.checkFailure(rs, diff, scaleErr)

	_, err := c.clientset.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)

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

func (c *VMReplicaSet) calcDiff(rs *virtv1.VirtualMachineReplicaSet, vms []*virtv1.VirtualMachine) int {
	// TODO default this on the aggregated api server
	wantedReplicas := int32(1)
	if rs.Spec.Replicas != nil {
		wantedReplicas = *rs.Spec.Replicas
	}

	return len(vms) - int(wantedReplicas)
}

func (c *VMReplicaSet) getVirtualMachineBaseName(replicaset *virtv1.VirtualMachineReplicaSet) string {

	// TODO defaulting should make sure that the right field is set, instead of doing this
	if len(replicaset.Spec.Template.ObjectMeta.Name) > 0 {
		return replicaset.Spec.Template.ObjectMeta.Name
	}
	if len(replicaset.Spec.Template.ObjectMeta.GenerateName) > 0 {
		return replicaset.Spec.Template.ObjectMeta.GenerateName
	}
	return replicaset.ObjectMeta.Name
}

func (c *VMReplicaSet) checkPaused(rs *virtv1.VirtualMachineReplicaSet) {

	if rs.Spec.Paused == true && !c.hasCondition(rs, virtv1.VMReplicaSetReplicaPaused) {

		rs.Status.Conditions = append(rs.Status.Conditions, virtv1.VMReplicaSetCondition{
			Type:               virtv1.VMReplicaSetReplicaPaused,
			Reason:             "Paused",
			Message:            "Controller got paused",
			LastTransitionTime: v1.Now(),
			Status:             k8score.ConditionTrue,
		})
	} else if rs.Spec.Paused == false && c.hasCondition(rs, virtv1.VMReplicaSetReplicaPaused) {
		c.removeCondition(rs, virtv1.VMReplicaSetReplicaPaused)
	}
}

func (c *VMReplicaSet) checkFailure(rs *virtv1.VirtualMachineReplicaSet, diff int, scaleErr error) {
	if scaleErr != nil && !c.hasCondition(rs, virtv1.VMReplicaSetReplicaFailure) {
		var reason string
		if diff < 0 {
			reason = "FailedCreate"
		} else {
			reason = "FailedDelete"
		}

		rs.Status.Conditions = append(rs.Status.Conditions, virtv1.VMReplicaSetCondition{
			Type:               virtv1.VMReplicaSetReplicaFailure,
			Reason:             reason,
			Message:            scaleErr.Error(),
			LastTransitionTime: v1.Now(),
			Status:             k8score.ConditionTrue,
		})

	} else if scaleErr == nil && c.hasCondition(rs, virtv1.VMReplicaSetReplicaFailure) {
		c.removeCondition(rs, virtv1.VMReplicaSetReplicaFailure)
	}
}

func OwnerRef(rs *virtv1.VirtualMachineReplicaSet) v1.OwnerReference {
	t := true
	gvk := virtv1.VMReplicaSetGroupVersionKind
	return v1.OwnerReference{
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
func (c *VMReplicaSet) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachineReplicaSet {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VMReplicaSetGroupVersionKind.Kind {
		return nil
	}
	rs, exists, err := c.vmRSInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if rs.(*virtv1.VirtualMachineReplicaSet).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return rs.(*virtv1.VirtualMachineReplicaSet)
}
