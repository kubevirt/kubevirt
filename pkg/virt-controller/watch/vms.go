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
	"fmt"
	"time"

	v12 "k8s.io/api/core/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

// Reasons for VMSnapshots events
const (
	// FailedCreateVirtualMachineSnapshotReason is added in an event and in a VMSnapshot condition
	// when snapshot fails to create due to any reason.
	FailedCreateVirtualMachineSnapshotReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineReason is added in an event and in a VMSnapshot condition
	// when snapshot is successfully created.
	SuccessfulCreateVirtualMachineSnapshotReason = "SuccessfulCreate"

	// FailedCreateVolumeSnapshotReason is added in an event and in a VMSnapshot condition
	// when volume snapshot failed to create
	FailedCreateVolumeSnapshotReason = "FailedVolumeSnapshotCreate"
	// SuccessfulCreateVolumeSnapshotReason is added in an event and in a VMSnapshot condition
	// when volume snapshot is successfully created
	SuccessfulCreateVolumeSnapshotReason = "SuccessfulVolumeSnapshotCreate"
	// FailedCreateVolumeSnapshotContentReason is added in an event and in a VMSnapshot condition
	// when volume snapshot content (result of volume snapshot) failed to create
	FailedCreateVolumeSnapshotContentReason = "FailedVolumeSnapshotCreate"
	// SuccessfulCreateVolumeSnapshotContentReason is added in an event and in a VMSnapshot condition
	// when volume snapshot content (result of volume snapshot) is successfully created
	SuccessfulCreateVolumeSnapshotContentReason = "SuccessfulVolumeSnapshotCreate"

	// FailedCreateVirtualMachineSpecSnapshotReason is added in an event and in a VMSnapshot condition
	// when VirtualMachine specification (the whole object without status) cannot be copied.
	FailedCreateVirtualMachineSpecSnapshotReason = "FailedCreate"
	// SuccessfulCreateVirtualMachineSpecReason is added in an event and in a VMSnapshot condition
	// when VirtualMachine specification (the whole object without status) cannot be copied.
	SuccessfulCreateVirtualMachineSpecSnapshotReason = "SuccessfulCreate"

	VirtualMachineRunningSnapshotReason = "VirtualMachineRunning"
)

func NewVMSnapshotController(
	vmInformer cache.SharedIndexInformer,
	vmsInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *VMSnapshotController {

	c := &VMSnapshotController{
		Queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmInformer:   vmInformer,
		vmsInformer:  vmsInformer,
		recorder:     recorder,
		clientset:    clientset,
		expectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	c.vmsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMSnapshot,
		DeleteFunc: c.deleteVMSnapshot,
		UpdateFunc: c.updateVMSnapshot,
	})

	return c
}

type VMSnapshotController struct {
	clientset    kubecli.KubevirtClient
	Queue        workqueue.RateLimitingInterface
	vmInformer   cache.SharedIndexInformer
	vmsInformer  cache.SharedIndexInformer
	recorder     record.EventRecorder
	expectations *controller.UIDTrackingControllerExpectations
}

func (c *VMSnapshotController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachineSnapshot controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmsInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VirtualMachineSnapshot controller.")
}

func (c *VMSnapshotController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMSnapshotController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineSnapshot %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineSnapshot %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *VMSnapshotController) execute(key string) error {
	obj, exists, err := c.vmsInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	vms := obj.(*virtv1.VirtualMachineSnapshot)

	logger := log.Log.Object(vms)

	logger.Info("Started processing the snapshot")

	// get all potentially interesting VMIs from the cache
	obj, exists, err = c.vmInformer.GetStore().GetByKey(vms.Namespace + "/" + vms.Spec.VirtualMachine)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		logger.Reason(err).Error("Failed to fetch VirtualMachine")
		return nil
	}
	vm := obj.(*virtv1.VirtualMachine)
	logger.Infof("found VM: %s", vm.Name)

	var snapshotConditions []virtv1.VirtualMachineSnapshotCondition

	doSnapshot, reasonCondition := shouldDoSnapshot(vms, vm)
	if reasonCondition != nil {
		snapshotConditions = append(snapshotConditions, *reasonCondition)
	}

	// set snapshot owner to VM to provide automatic deletion handling
	vms, ownerUpdated := updateVirtualMachineSnapshotOwner(vms, vm)

	var specCopied bool
	if doSnapshot {
		// copy the spec
		vms, specCopied = copyVirtualMachineToStatus(vms, vm)
	}

	vms, statusUpdated := updateStatus(vms, vm, snapshotConditions)

	if ownerUpdated || specCopied || statusUpdated {
		// update the snapshot in cluster
		err = c.restUpdateVirtualMachineSnapshot(vms)
		if err != nil {
			logger.Errorf("Cannot update the VirtualMachineSnapshot in ETCD: %s", err.Error())
		}
	}

	return nil
}

func (c *VMSnapshotController) updateVirtualMachine(old, cur interface{}) {
	curVMI := cur.(*virtv1.VirtualMachine)
	oldVMI := old.(*virtv1.VirtualMachine)
	if curVMI.ResourceVersion == oldVMI.ResourceVersion {
		// Periodic resync will send update events for all known vmis.
		// Two different versions of the same vmi will always have different RVs.
		return
	}

	// list all controller VirtualMachineSnapshots by this VirtualMachine
	vmss := c.getControlledVirtualMachineSnapshots(curVMI)

	// since VirtualMachine has been updated, enqueue the VirtualMachineSnapshots for an update
	for _, vms := range vmss {
		log.Log.Infof("Enqueuing VirtualMachineSnapshot: %s", vms.Name)
		c.enqueueVMSnapshot(vms)
	}
}

// When a VirtualMachine is deleted, enqueue the VirtualMachineSnapshot that belongs to the VirtualMachine.
// obj could be an *metav1.VirtualMachine, or a DeletionFinalStateUnknown marker item.
func (c *VMSnapshotController) deleteVirtualMachine(obj interface{}) {
	vm, ok := obj.(*virtv1.VirtualMachine)

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
		vm, ok = tombstone.Obj.(*virtv1.VirtualMachine)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vm %#v", obj)).Error("Failed to process delete notification")
			return
		}
	}

	// Get the list of snapshots belonging to the VirtualMachine from the same namespace
	objs, err := c.vmsInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vm.Namespace)
	if err != nil {
		log.Log.Reason(fmt.Errorf("cannot list VirtualMachineSnapshots for a vm %#v", obj)).Error(err.Error())
		return
	}

	// When snapshot is linked to VirtualMachine, enqueue it so it is deleted when there is no more VirtualMachine
	for _, snapshotObj := range objs {
		vms := snapshotObj.(*virtv1.VirtualMachineSnapshot)
		if vms.Spec.VirtualMachine == vm.Name {
			c.enqueueVMSnapshot(vms)
		}
	}
}

func (c *VMSnapshotController) addVMSnapshot(obj interface{}) {
	c.enqueueVMSnapshot(obj)
}

func (c *VMSnapshotController) deleteVMSnapshot(obj interface{}) {
	c.enqueueVMSnapshot(obj)
}

func (c *VMSnapshotController) updateVMSnapshot(old, curr interface{}) {
	c.enqueueVMSnapshot(curr)
}

func (c *VMSnapshotController) enqueueVMSnapshot(obj interface{}) {
	logger := log.Log
	vms := obj.(*virtv1.VirtualMachineSnapshot)
	key, err := controller.KeyFunc(vms)
	if err != nil {
		logger.Object(vms).Reason(err).Error("Failed to extract vmsKey from VirtualMachineSnapshot.")
	}
	c.Queue.Add(key)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *VMSnapshotController) getControlledVirtualMachineSnapshots(vm *virtv1.VirtualMachine) []*virtv1.VirtualMachineSnapshot {

	// Get the list of snapshots belonging to the VirtualMachine from the same namespace
	objs, err := c.vmsInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vm.Namespace)
	if err != nil {
		log.Log.Reason(fmt.Errorf("cannot list VirtualMachineSnapshots for a vm %#v", vm)).Error(err.Error())
		return nil
	}

	// When snapshot is linked to VirtualMachine, enqueue it so it is deleted when there is no more VirtualMachine
	snapshots := []*virtv1.VirtualMachineSnapshot{}
	for _, snapshotObj := range objs {
		vms := snapshotObj.(*virtv1.VirtualMachineSnapshot)
		vmsController := v1.GetControllerOf(vms)
		if vmsController != nil {
			if vmsController.UID == vm.UID {
				snapshots = append(snapshots, vms)
			}
		}
	}
	return snapshots
}

func (c *VMSnapshotController) hasCondition(rs *virtv1.VirtualMachineInstanceReplicaSet, cond virtv1.VirtualMachineInstanceReplicaSetConditionType) bool {
	for _, c := range rs.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (c *VMSnapshotController) removeCondition(rs *virtv1.VirtualMachineInstanceReplicaSet, cond virtv1.VirtualMachineInstanceReplicaSetConditionType) {
	var conds []virtv1.VirtualMachineInstanceReplicaSetCondition
	for _, c := range rs.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	rs.Status.Conditions = conds
}

// restUpdateVirtualMachineSnapshot call the REST API and update the VirtualMachineSnapshot in the cluster
func (c *VMSnapshotController) restUpdateVirtualMachineSnapshot(vms *virtv1.VirtualMachineSnapshot) error {
	_, err := c.clientset.VirtualMachineSnapshot(vms.ObjectMeta.Namespace).Update(vms)
	return err
}

// cleanVirtualMachine copies the virtual machine and cleans the status
func cleanVirtualMachine(vm *virtv1.VirtualMachine) *virtv1.VirtualMachine {
	cleanVM := vm.DeepCopy()
	cleanVM.Status = virtv1.VirtualMachineStatus{}
	return cleanVM
}

// copyVirtualMachineToStatus created VirtualMachine spec snapshot by copying it to the status field in VirtualMachineSnapshot object
// The copy is done only once, if there already is copy present, nothing happens.
// New VirtualMachineSnapshot object is returned when copy is done, original VirtualMachineSnapshot is returned, when nothing happened.
// Also returned bool flag indicate whether copy was performed.
func copyVirtualMachineToStatus(vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) (*virtv1.VirtualMachineSnapshot, bool) {
	if vms.Status.VirtualMachine != nil {
		// already copied the VirtualMachine, nothing to do
		return vms, false
	}

	updatedVMS := vms.DeepCopy()
	updatedVMS.Status.VirtualMachine = cleanVirtualMachine(vm)

	return updatedVMS, true
}

// updateStatus computes the VirtualMachineSnapshot status based on the snapshot state, VirtualMachine state
// and operations in progress
// it returns new updated VirtualMachineSnapshot if status has been updated.
// it returns original VirtualMachineSnapshot if there was not change
func updateStatus(vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine, conditions []virtv1.VirtualMachineSnapshotCondition) (*virtv1.VirtualMachineSnapshot, bool) {

	if conditions == nil {
		return vms, false
	}

	updatedVMS := vms.DeepCopy()

	// overwrite conditions, in this stage the conditions are set and nothing changes
	updatedVMS.Status.Conditions = make([]virtv1.VirtualMachineSnapshotCondition, len(conditions))
	copy(updatedVMS.Status.Conditions, conditions)

	return updatedVMS, true
}

// shouldDoSnapshot looks at the configurtion of VirtualMachineSnapshot and state of the VirtualMachine and decide
// whether to perform snapshot
// it return true/false to signal whether to perform snapshot
// it returns VirtualMachineSnapshotCondition with reason why not to do snapshot
func shouldDoSnapshot(vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) (bool, *virtv1.VirtualMachineSnapshotCondition) {

	if vms.Status.VirtualMachine != nil {
		// vm is already copied, do nothing
		return false, nil
	}

	if vm.Spec.Running && vm.Status.Ready {
		// currently only offline VirtualMachineSnapshot is supported
		condition := virtv1.VirtualMachineSnapshotCondition{
			Type:               virtv1.VirtualMachineSnapshotFailure,
			Reason:             VirtualMachineRunningSnapshotReason,
			Message:            "Snapshot can be done only for offline VirtualMachine",
			LastTransitionTime: v1.Now(),
			Status:             v12.ConditionTrue,
		}
		return false, &condition
	}

	// no objections, do snapshot
	return true, nil
}

// updateVirtualMachineSnapshotOwner set VirtualMachine as controller for the VirtualMachineSnapshot
// it will enable automatic deletion when VirtualMachine is deleted.
// Returns new VirtualMachineSnapshot when owner had been added. Original VirtualMachineSnapshot is returned when nothing changed.
// Bool flag signals whether update has been done
func updateVirtualMachineSnapshotOwner(vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) (*virtv1.VirtualMachineSnapshot, bool) {
	controller := v1.GetControllerOf(vms)
	if controller != nil {
		return vms, false
	}

	updatedVMS := vms.DeepCopy()
	t := true
	gvk := virtv1.VirtualMachineSnapshotGroupVersionKind
	updatedVMS.OwnerReferences = []v1.OwnerReference{v1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       vm.ObjectMeta.Name,
		UID:        vm.ObjectMeta.UID,
		Controller: &t,
	},
	}

	return updatedVMS, true
}
