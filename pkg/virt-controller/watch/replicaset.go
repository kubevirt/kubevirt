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

	"k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"

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

	// Wait for cache sync before we start the pod controller
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

	// make sure we only consider active VMs
	vms = c.filterActiveVMs(vms)

	// make sure the selector of the controller matches and the VMs match
	vms = c.filterMatchingVMs(selector, vms)

	// Scale up or down, if all expected creates and deletes were report by the listener
	var scaleErr error
	if needsSync && !rs.Spec.Paused {
		scaleErr = c.scale(rs, vms)
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

func (c *VMReplicaSet) scale(rs *virtv1.VirtualMachineReplicaSet, vms []virtv1.VirtualMachine) error {

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
				deleteCandidate := &vms[idx]
				// TODO graceful delete
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
// Note that vms which have a deletion timestamp set, are still treated as active.
// This is a difference to Pod ReplicaSets
func (c *VMReplicaSet) filterActiveVMs(vms []virtv1.VirtualMachine) []virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return !vm.IsFinal()
	})
}

// filterReadyVMs takes a list of VMs and returns all VMs which are in ready state.
func (c *VMReplicaSet) filterReadyVMs(vms []virtv1.VirtualMachine) []virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return vm.IsReady()
	})
}

// filterMatchingVMs takes a selector and a list of VMs. If the VM labels match the selector it is added to the filtered collection.
// Returns the list of all VMs which match the selector
func (c *VMReplicaSet) filterMatchingVMs(selector labels.Selector, vms []virtv1.VirtualMachine) []virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return selector.Matches(labels.Set(vm.ObjectMeta.Labels))
	})
}

func filter(vms []virtv1.VirtualMachine, f func(vm *virtv1.VirtualMachine) bool) []virtv1.VirtualMachine {
	filtered := []virtv1.VirtualMachine{}
	for _, vm := range vms {
		if f(&vm) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// listVMsFromNamespace takes a namespace and returns all VMs from the VM cache which run in this namespace
func (c *VMReplicaSet) listVMsFromNamespace(namespace string) ([]virtv1.VirtualMachine, error) {
	objs, err := c.vmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	vms := []virtv1.VirtualMachine{}
	for _, obj := range objs {
		vms = append(vms, *obj.(*virtv1.VirtualMachine))
	}
	return vms, nil
}

// listControllerFromNamespace takes a namespace and returns all VMReplicaSets from the ReplicaSet cache which run in this namespace
func (c *VMReplicaSet) listControllerFromNamespace(namespace string) ([]virtv1.VirtualMachineReplicaSet, error) {
	objs, err := c.vmRSInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	replicaSets := []virtv1.VirtualMachineReplicaSet{}
	for _, obj := range objs {
		rs := obj.(*virtv1.VirtualMachineReplicaSet)
		replicaSets = append(replicaSets, *rs)
	}
	return replicaSets, nil
}

// getMatchingController returns the first VMReplicaSet which matches the labels of the VM from the listener cache.
// If there are no matching controllers, a NotFound error is returned.
func (c *VMReplicaSet) getMatchingController(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachineReplicaSet, error) {
	logger := log.Log
	controllers, err := c.listControllerFromNamespace(vm.ObjectMeta.Namespace)
	if err != nil {
		return nil, err
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, rs := range controllers {
		selector, err := v1.LabelSelectorAsSelector(rs.Spec.Selector)
		if err != nil {
			logger.Object(&rs).Reason(err).Error("Failed to parse label selector from replicaset.")
			continue
		}

		if selector.Matches(labels.Set(vm.ObjectMeta.Labels)) {
			// The first matching rs will be returned
			return &rs, nil
		}

	}
	return nil, errors.NewNotFound(v1beta1.Resource("virtualmachinereplicaset"), "")
}

// addVirtualMachine searches for a matching VMReplicaSet, updates it's expectations and wakes it up
func (c *VMReplicaSet) addVirtualMachine(obj interface{}) {

	rsKey := c.getMatchingControllerKey(obj.(*virtv1.VirtualMachine))
	if rsKey == "" {
		return
	}

	// In case the controller is waiting for a creation, tell it that we observed one
	c.expectations.CreationObserved(rsKey)
	c.Queue.Add(rsKey)
	return
}

// deleteVirtualMachine searches for a matching VMReplicaSet, updates it's expectations and wakes it up
func (c *VMReplicaSet) deleteVirtualMachine(obj interface{}) {
	vm := obj.(*virtv1.VirtualMachine)

	rsKey := c.getMatchingControllerKey(vm)
	if rsKey == "" {
		return
	}

	// In case the controller is waiting for a deletion, tell it that we observed one
	c.expectations.DeletionObserved(rsKey, controller.VirtualMachineKey(vm))
	c.Queue.Add(rsKey)
	return
}

// deleteVirtualMachine searchs for a matching VMReplicaSet and wakes it up
func (c *VMReplicaSet) updateVirtualMachine(old, curr interface{}) {
	rsKey := c.getMatchingControllerKey(curr.(*virtv1.VirtualMachine))
	if rsKey == "" {
		return
	}

	c.Queue.Add(rsKey)
	return
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

// getMatchingControllerKey takes a VirtualMachine and returns a the key of a macthing VMReplicaSet, if one exists.
// Returns an empty string if no matching controller exists
func (c *VMReplicaSet) getMatchingControllerKey(vm *virtv1.VirtualMachine) string {
	logger := log.Log

	// Let's search for a matching controller
	rs, err := c.getMatchingController(vm)

	// If none exists, ignore
	if errors.IsNotFound(err) {
		return ""
	}

	// If an unexpected error occurred, log it and ignore
	if err != nil {
		logger.Object(vm).Reason(err).Error("Searching for matching replicasets failed.")
		return ""
	}

	// If we can't extract the key, log it and ignore
	rsKey, err := controller.KeyFunc(rs)
	if err != nil {
		logger.Object(rs).Reason(err).Error("Failed to extract rsKey from replicaset.")
		return ""
	}
	return rsKey
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

func (c *VMReplicaSet) updateStatus(rs *virtv1.VirtualMachineReplicaSet, vms []virtv1.VirtualMachine, scaleErr error) error {

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

func (c *VMReplicaSet) calcDiff(rs *virtv1.VirtualMachineReplicaSet, vms []virtv1.VirtualMachine) int {
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
