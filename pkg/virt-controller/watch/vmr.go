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

	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

const (
	VirtualMachineRestoredSnapshotReason = "VirtualMachineRestored"
	VirtualMachineNoSnapshotReason = "VirtualMachineHasNoSnapshot"
	VirtualMachineNoVMSnapshotReason = "VirtualMachineHasNoVM"
)

func NewVMRestoreController(
	vmInformer cache.SharedIndexInformer,
	vmsInformer cache.SharedIndexInformer,
	vmrInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *VMRestoreController {

	c := &VMRestoreController{
		Queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmInformer:   vmInformer,
		vmsInformer:  vmsInformer,
		vmrInformer:  vmrInformer,
		recorder:     recorder,
		clientset:    clientset,
		expectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmrInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVMRestore,
		DeleteFunc: c.deleteVMRestore,
		UpdateFunc: c.updateVMRestore,
	})

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.updateVirtualMachine,
	})

	return c
}

type VMRestoreController struct {
	clientset    kubecli.KubevirtClient
	Queue        workqueue.RateLimitingInterface
	vmInformer   cache.SharedIndexInformer
	vmsInformer  cache.SharedIndexInformer
	vmrInformer  cache.SharedIndexInformer
	recorder     record.EventRecorder
	expectations *controller.UIDTrackingControllerExpectations
}

func (c *VMRestoreController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachineRestore controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmsInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VirtualMachineRestore controller.")
}

func (c *VMRestoreController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMRestoreController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachineRestore %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineRestore %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *VMRestoreController) execute(key string) error {
	obj, exists, err := c.vmrInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	vmr := obj.(*virtv1.VirtualMachineRestore)

	var snapshotConditions []virtv1.VirtualMachineSnapshotCondition

	logger := log.Log.Object(vmr)
	logger.Info("Started processing the restore")

	// get relevant VMs from the cache
	obj, exists, err = c.vmInformer.GetStore().GetByKey(vmr.Namespace + "/" + vmr.Name)
	var vm *virtv1.VirtualMachine
	if !exists || err != nil {
		logger.Reason(err).Errorf("Failed to fetch VirtualMachine %s", vmr.Namespace + "/" + vmr.Name)

	} else {
		vm = obj.(*virtv1.VirtualMachine)
		logger.Infof("found VM: %s", vm.Name)
	}

	// get relevant VMSs from the cache
	obj, exists, err = c.vmsInformer.GetStore().GetByKey(vmr.Namespace + "/" + vmr.Spec.VirtualMachineSnapshot)
	var vms *virtv1.VirtualMachineSnapshot
	if !exists || err != nil{
		// nothing we need to do. It should always be possible to re-create this type of controller
		logger.Reason(err).Errorf("Failed to fetch VirtualMachineSnapshot %s", vmr.Namespace + "/" + vmr.Spec.VirtualMachineSnapshot)
	} else {
		vms = obj.(*virtv1.VirtualMachineSnapshot)
		logger.Infof("found VMS: %s", vms.Name)
	}

	// check whether it can do snapshot
	doRestore, reasonCondition := shouldDoRestore(vmr, vms, vm)
	if reasonCondition != nil {
		snapshotConditions = append(snapshotConditions, reasonCondition...)
	}

	var restoredOn *metav1.Time
	var restored, updatedStatus bool

	logger.Infof("Doing restore: %t", doRestore)

	if doRestore {
		restored, err = c.doRestore(vmr, vms, vm)
		if err != nil {
			log.Log.Object(vm).Errorf("Cannot restore VM: %s", err.Error())
		}
		if restored {
			logger.Infof("Restored VM: %s", vm.Name)
			restoredTime := metav1.Now()
			restoredOn = &restoredTime
		}
	}

	vmr, updatedStatus = updateVMRestoreStatus(vmr, restoredOn, snapshotConditions)

	if updatedStatus {
		// update the snapshot in cluster
		err = c.restUpdateVirtualMachineRestore(vmr)
		if err != nil {
			logger.Errorf("Cannot update the VirtualMachineRestore in ETCD: %s", err.Error())
		}
	}

	return nil
}

func (c *VMRestoreController) addVMRestore(obj interface{}) {
	c.enqueueVMRestore(obj)
}

func (c *VMRestoreController) deleteVMRestore(obj interface{}) {
	c.enqueueVMRestore(obj)
}

func (c *VMRestoreController) updateVMRestore(old, curr interface{}) {
	c.enqueueVMRestore(curr)
}

func (c *VMRestoreController) updateVirtualMachine(old, cur interface{}) {
	curVMI := cur.(*virtv1.VirtualMachine)
	oldVMI := old.(*virtv1.VirtualMachine)
	if curVMI.ResourceVersion == oldVMI.ResourceVersion {
		// Periodic resync will send update events for all known vmis.
		// Two different versions of the same vmi will always have different RVs.
		return
	}

	// list all controller VirtualMachineSnapshots by this VirtualMachine
	obj, exists, err := c.vmrInformer.GetStore().GetByKey(curVMI.Namespace + "/" + curVMI.Name)
	if err != nil {
		log.Log.Reason(err).Error(err.Error())
		return
	}
	if !exists {
		log.Log.Errorf("Cannot react to VM update, VirtualMachineRestore %s not found", curVMI.Name)
		return
	}
	vmr := obj.(*virtv1.VirtualMachineRestore)

	log.Log.Infof("Enqueuing VirtualMachineSnapshot: %s", vmr.Name)
	c.enqueueVMRestore(vmr)
}

func (c *VMRestoreController) enqueueVMRestore(obj interface{}) {
	logger := log.Log
	vmr := obj.(*virtv1.VirtualMachineRestore)
	key, err := controller.KeyFunc(vmr)
	if err != nil {
		logger.Object(vmr).Reason(err).Error("Failed to extract vmrKey from VirtualMachineRestore.")
	}
	c.Queue.Add(key)
}

// restUpdateVirtualMachineSnapshot call the REST API and update the VirtualMachineSnapshot in the cluster
func (c *VMRestoreController) restUpdateVirtualMachineRestore(vmr *virtv1.VirtualMachineRestore) error {
	_, err := c.clientset.VirtualMachineRestore(vmr.ObjectMeta.Namespace).Update(vmr)
	return err
}

func (c *VMRestoreController) doRestore(vmr *virtv1.VirtualMachineRestore, vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) (bool, error) {

	newVM := vms.Status.VirtualMachine
	newVM.ResourceVersion = vm.ResourceVersion

	_, err := c.clientset.VirtualMachine(vm.ObjectMeta.Namespace).Update(newVM)
	if err != nil {
		log.Log.Errorf("Cannot update VM: %s", newVM.Name)
		return false, err
	}

	return true, nil
}

func (c *VMRestoreController) updateResourceVersion(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	obj, exist, err := c.vmInformer.GetStore().GetByKey(vm.Namespace + "/" + vm.Name)
	if !exist {
		log.Log.Object(vm).Errorf("VM %s not found in cache.", vm.Name)
	}
	if err != nil {
		log.Log.Object(vm).Errorf("Cannot load current VM: %s from cache %s", vm.Name, err.Error())
		return nil, err
	}
	clusterVM := obj.(*virtv1.VirtualMachine)
	updatedVM := vm.DeepCopy()
	updatedVM.ResourceVersion = clusterVM.ResourceVersion

	return clusterVM, nil
}

func shouldDoRestore(vmr *virtv1.VirtualMachineRestore, vms *virtv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) (bool, []virtv1.VirtualMachineSnapshotCondition) {
	var snapshotConditions []virtv1.VirtualMachineSnapshotCondition
	doRestore := true

	if vmr.Status.RestoredOn != nil {
		// VirtualMachine already restored from this restore
		// configuration. Cannot perform another one.
		log.Log.Infof("VM %s is already restored", vm.Name)
		doRestore = false
	}

	if vm == nil {
		log.Log.Infof("VM %s not found", vmr.Name)
		snapshotConditions = append(snapshotConditions, virtv1.VirtualMachineSnapshotCondition{
			Type:               virtv1.VirtualMachineSnapshotFailure,
			Reason:             VirtualMachineNoVMSnapshotReason,
			Message:            "No Virtual Machine to restore",
			LastTransitionTime: v1.Now(),
			Status:             v12.ConditionTrue,
		})

		doRestore = false
	} else {
		if vm.Spec.Running == true {
			// VirtualMachine have to be shutdown to perform restore
			log.Log.Infof("VM: %s is running", vm.Name)
			snapshotConditions = append(snapshotConditions, virtv1.VirtualMachineSnapshotCondition{
				Type:               virtv1.VirtualMachineSnapshotFailure,
				Reason:             VirtualMachineRunningSnapshotReason,
				Message:            "Virtual Machine is running",
				LastTransitionTime: v1.Now(),
				Status:             v12.ConditionTrue,
			})
			doRestore = false
		}
	}

	if vms == nil {
		log.Log.Infof("VMS %s not found", vmr.Spec.VirtualMachineSnapshot)
		snapshotConditions = append(snapshotConditions, virtv1.VirtualMachineSnapshotCondition{
			Type:               virtv1.VirtualMachineSnapshotFailure,
			Reason:             VirtualMachineNoSnapshotReason,
			Message:            "No Virtual Machine Snapshot to restore",
			LastTransitionTime: v1.Now(),
			Status:             v12.ConditionTrue,
		})

		doRestore = false
	} else {
		if vms.Status.VirtualMachine == nil {
			// Nothing to restore from
			log.Log.Infof("VMS: %s does not have VM cloned", vms.Name)
			snapshotConditions = append(snapshotConditions, virtv1.VirtualMachineSnapshotCondition{
				Type:               virtv1.VirtualMachineSnapshotFailure,
				Reason:             VirtualMachineNoSnapshotReason,
				Message:            "Virtual Machine has no snapshot",
				LastTransitionTime: v1.Now(),
				Status:             v12.ConditionTrue,
			})
			doRestore = false
		}
	}

	return doRestore, snapshotConditions
}

// updateVMRestoreStatus computes the VirtualMachineSnapshot status based on the snapshot state, VirtualMachine state
// and operations in progress
// it returns new updated VirtualMachineSnapshot if status has been updated.
// it returns original VirtualMachineSnapshot if there was not change
func updateVMRestoreStatus(vmr *virtv1.VirtualMachineRestore, restoredOn *metav1.Time, conditions []virtv1.VirtualMachineSnapshotCondition) (*virtv1.VirtualMachineRestore, bool) {
	updatedVMR := vmr.DeepCopy()
	updated := false

	log.Log.Infof("Updating VirtualMachineSnapshot: %s status", vmr.Name)

	if restoredOn != nil {
		updatedVMR.Status.RestoredOn = restoredOn
		updated = true
	}

	updatedVMR.Status.Conditions = make([]virtv1.VirtualMachineSnapshotCondition, len(conditions))
	if len(conditions) > 0 {
		// overwrite conditions, in this stage the conditions are set and nothing changes
		copy(updatedVMR.Status.Conditions, conditions)
		updated = true
	}

	return updatedVMR, updated
}
