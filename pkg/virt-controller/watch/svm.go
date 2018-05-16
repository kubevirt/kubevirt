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

	"github.com/pborman/uuid"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	k8score "k8s.io/api/core/v1"

	"fmt"

	"reflect"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

func NewSVMController(vmInformer cache.SharedIndexInformer, vmSVMInformer cache.SharedIndexInformer, recorder record.EventRecorder, clientset kubecli.KubevirtClient) *SVMController {

	c := &SVMController{
		Queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmInformer:    vmInformer,
		vmSVMInformer: vmSVMInformer,
		recorder:      recorder,
		clientset:     clientset,
		expectations:  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmSVMInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addSvm,
		DeleteFunc: c.deleteSvm,
		UpdateFunc: c.updateSvm,
	})

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	return c
}

type SVMController struct {
	clientset     kubecli.KubevirtClient
	Queue         workqueue.RateLimitingInterface
	vmInformer    cache.SharedIndexInformer
	vmSVMInformer cache.SharedIndexInformer
	recorder      record.EventRecorder
	expectations  *controller.UIDTrackingControllerExpectations
}

func (c *SVMController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting StatefulVirtualMachine controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmSVMInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping StatefulVirtualMachine controller.")
}

func (c *SVMController) runWorker() {
	for c.Execute() {
	}
}

func (c *SVMController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing StatefulVirtualMachine %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed StatefulVirtualMachine %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *SVMController) execute(key string) error {

	obj, exists, err := c.vmSVMInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	SVM := obj.(*virtv1.StatefulVirtualMachine)

	logger := log.Log.Object(SVM)

	logger.Info("Started processing SVM")

	//TODO default rs if necessary, the aggregated apiserver will do that in the future
	if SVM.Spec.Template == nil {
		logger.Error("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	needsSync := c.expectations.SatisfiedExpectations(key)

	svmKey, err := controller.KeyFunc(SVM)
	if err != nil {
		return err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (v1.Object, error) {
		fresh, err := c.clientset.StatefulVirtualMachine(SVM.ObjectMeta.Namespace).Get(SVM.ObjectMeta.Name, &v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != SVM.ObjectMeta.UID {
			return nil, fmt.Errorf("original StatefulVirtualMachine %v/%v is gone: got uid %v, wanted %v", SVM.Namespace, SVM.Name, fresh.UID, SVM.UID)
		}
		return fresh, nil
	})
	cm := controller.NewVirtualMachineControllerRefManager(controller.RealVirtualMachineControl{Clientset: c.clientset}, SVM, nil, virtv1.StatefulVirtualMachineGroupVersionKind, canAdoptFunc)

	var vm *virtv1.VirtualMachine
	vmObj, exist, err := c.vmInformer.GetStore().GetByKey(svmKey)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch vm for namespace from cache.")
		return err
	}
	if !exist {
		logger.V(4).Infof("VM not found in cache %s", key)
		vm = nil
	} else {
		vm = vmObj.(*virtv1.VirtualMachine)

		vm, err = cm.ClaimVirtualMachineByName(vm)
		if err != nil {
			return err
		}
	}

	var createErr, vmError error

	// Scale up or down, if all expected creates and deletes were report by the listener
	if needsSync && SVM.ObjectMeta.DeletionTimestamp == nil {
		logger.Infof("Creating or the VM: %t", SVM.Spec.Running)
		createErr = c.startStop(SVM, vm)
	}

	// If the controller is going to be deleted and the orphan finalizer is the next one, release the VMs. Don't update the status
	// TODO: Workaround for https://github.com/kubernetes/kubernetes/issues/56348, remove it once it is fixed
	if SVM.ObjectMeta.DeletionTimestamp != nil && controller.HasFinalizer(SVM, v1.FinalizerOrphanDependents) {
		return c.orphan(cm, vm)
	}

	if createErr != nil {
		logger.Reason(err).Error("Scaling the StatefulVirtualMachine failed.")
	}

	err = c.updateStatus(SVM.DeepCopy(), vm, createErr, vmError)
	if err != nil {
		logger.Reason(err).Error("Updating the StatefulVirtualMachine status failed.")
	}

	return err
}

// orphan removes the owner reference of all VMs which are owned by the controller instance.
// Workaround for https://github.com/kubernetes/kubernetes/issues/56348 to make no-cascading deletes possible
// We don't have to remove the finalizer. This part of the gc is not affected by the mentioned bug
// TODO +pkotas unify with replicasets. This function can be the same
func (c *SVMController) orphan(cm *controller.VirtualMachineControllerRefManager, vm *virtv1.VirtualMachine) error {
	if vm == nil {
		return nil
	}

	errChan := make(chan error, 1)

	go func(vm *virtv1.VirtualMachine) {
		err := cm.ReleaseVirtualMachine(vm)
		if err != nil {
			errChan <- err
		}
	}(vm)

	select {
	case err := <-errChan:
		return err
	default:
	}
	return nil
}

func (c *SVMController) startStop(svm *virtv1.StatefulVirtualMachine, vm *virtv1.VirtualMachine) error {
	log.Log.Object(svm).V(4).Infof("Start the VM: %t", svm.Spec.Running)

	if svm.Spec.Running == true {
		if vm != nil {
			if vm.IsFinal() {
				// The VM can fail od be finished. The job of this controller
				// is keep the VM running, therefore it restarts it.
				// restarting VM by stopping it and letting it start in next step
				err := c.stopVM(svm, vm)
				if err != nil {
					log.Log.Object(svm).Error("Cannot restart VM, the VM cannot be deleted.")
					return err
				}
				// return to let the controller pick up the expected deletion
			}
			// VM is OK no need to do anything
			return nil
		}

		err := c.startVM(svm)
		return err
	}

	if svm.Spec.Running == false {
		log.Log.Object(svm).V(4).Info("It is false delete")
		if vm == nil {
			log.Log.Info("vm is nil")
			// vm should not run and is not running
			return nil
		}
		err := c.stopVM(svm, vm)
		return err
	}

	return nil
}

func (c *SVMController) startVM(svm *virtv1.StatefulVirtualMachine) error {
	// TODO add check for existence
	svmKey, err := controller.KeyFunc(svm)
	if err != nil {
		log.Log.Object(svm).Reason(err).Error("Failed to extract svmKey from StatefulVirtualMachine.")
		return nil
	}

	// start it
	vm := c.setupVMFromSVM(svm)

	c.expectations.ExpectCreations(svmKey, 1)
	vm, err = c.clientset.VM(svm.ObjectMeta.Namespace).Create(vm)
	if err != nil {
		log.Log.Object(svm).Infof("Failed to create VM: %s/%s", vm.Namespace, vm.Name)
		c.expectations.CreationObserved(svmKey)
		c.recorder.Eventf(svm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine: %v", err)
		return err
	}
	c.recorder.Eventf(svm, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Created virtual machine: %v", vm.ObjectMeta.Name)

	return nil
}

func (c *SVMController) stopVM(svm *virtv1.StatefulVirtualMachine, vm *virtv1.VirtualMachine) error {
	if vm == nil {
		// nothing to do
		return nil
	}

	svmKey, err := controller.KeyFunc(svm)
	if err != nil {
		log.Log.Object(svm).Reason(err).Error("Failed to extract svmKey from StatefulVirtualMachine.")
		return nil
	}

	// stop it
	c.expectations.ExpectDeletions(svmKey, []string{controller.VirtualMachineKey(vm)})
	err = c.clientset.VM(svm.ObjectMeta.Namespace).Delete(vm.ObjectMeta.Name, &v1.DeleteOptions{})

	// Don't log an error if it is already deleted
	if err != nil {
		// We can't observe a delete if it was not accepted by the server
		c.expectations.DeletionObserved(svmKey, controller.VirtualMachineKey(vm))
		c.recorder.Eventf(svm, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting virtual machine %s: %v", vm.ObjectMeta.Name, err)
		return err
	}

	c.recorder.Eventf(svm, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Deleted virtual machine: %v", vm.ObjectMeta.UID)
	log.Log.Object(svm).Info("Dispatching delete event")

	return nil
}

// setupVMfromSVM creates a VirtualMachine object from one StatefulVirtualMachine object.
func (c *SVMController) setupVMFromSVM(svm *virtv1.StatefulVirtualMachine) *virtv1.VirtualMachine {
	basename := c.getVirtualMachineBaseName(svm)

	vm := virtv1.NewVMReferenceFromNameWithNS(svm.ObjectMeta.Namespace, "")
	vm.ObjectMeta = svm.Spec.Template.ObjectMeta
	vm.ObjectMeta.Name = basename
	vm.ObjectMeta.GenerateName = basename
	vm.Spec = svm.Spec.Template.Spec

	setupStableFirmwareUUID(svm, vm)

	t := true
	// TODO check if vm labels exist, and when make sure that they match. For now just override them
	vm.ObjectMeta.Labels = svm.Spec.Template.ObjectMeta.Labels
	vm.ObjectMeta.OwnerReferences = []v1.OwnerReference{{
		APIVersion:         virtv1.StatefulVirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               virtv1.StatefulVirtualMachineGroupVersionKind.Kind,
		Name:               svm.ObjectMeta.Name,
		UID:                svm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}

	return vm
}

// no special meaning, randomly generated on my box.
// TODO: do we want to use another constants? see examples in RFC4122
const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.Parse(magicUUID)

// setStableUUID makes sure the VM being started has a a 'stable' UUID.
// The UUID is 'stable' if doesn't change across reboots.
func setupStableFirmwareUUID(svm *virtv1.StatefulVirtualMachine, vm *virtv1.VirtualMachine) {

	logger := log.Log.Object(svm)

	if vm.Spec.Domain.Firmware == nil {
		vm.Spec.Domain.Firmware = &virtv1.Firmware{}
	}

	existingUUID := vm.Spec.Domain.Firmware.UUID
	if existingUUID != "" {
		logger.Debugf("Using existing UUID '%s'", existingUUID)
		return
	}

	vm.Spec.Domain.Firmware.UUID = types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(vm.ObjectMeta.Name)).String())
	logger.Infof("Setting stabile UUID '%s' (was '%s')", vm.Spec.Domain.Firmware.UUID, existingUUID)
}

// filterActiveVMs takes a list of VMs and returns all VMs which are not in a final state
// TODO +pkotas unify with replicaset this code is the same without dependency
func (c *SVMController) filterActiveVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return !vm.IsFinal()
	})
}

// filterReadyVMs takes a list of VMs and returns all VMs which are in ready state.
// TODO +pkotas unify with replicaset this code is the same
func (c *SVMController) filterReadyVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return vm.IsReady()
	})
}

// listVMsFromNamespace takes a namespace and returns all VMs from the VM cache which run in this namespace
// TODO +pkotas unify this code with replicaset
func (c *SVMController) listVMsFromNamespace(namespace string) ([]*virtv1.VirtualMachine, error) {
	objs, err := c.vmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	var vms []*virtv1.VirtualMachine
	for _, obj := range objs {
		vms = append(vms, obj.(*virtv1.VirtualMachine))
	}
	return vms, nil
}

// listControllerFromNamespace takes a namespace and returns all StatefulVirtualMachines
// from the StatefulVirtualMachine cache which run in this namespace
func (c *SVMController) listControllerFromNamespace(namespace string) ([]*virtv1.StatefulVirtualMachine, error) {
	objs, err := c.vmSVMInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	var svms []*virtv1.StatefulVirtualMachine
	for _, obj := range objs {
		svm := obj.(*virtv1.StatefulVirtualMachine)
		svms = append(svms, svm)
	}
	return svms, nil
}

// getMatchingControllers returns the list of StatefulVirtualMachines which matches
// the labels of the VM from the listener cache. If there are no matching
// controllers nothing is returned
func (c *SVMController) getMatchingControllers(vm *virtv1.VirtualMachine) (svms []*virtv1.StatefulVirtualMachine) {
	controllers, err := c.listControllerFromNamespace(vm.ObjectMeta.Namespace)
	if err != nil {
		return nil
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, svm := range controllers {
		if vm.Name == svm.Name {
			svms = append(svms, svm)
		}
	}
	return svms
}

// When a vm is created, enqueue the StatefulVirtualMachine that manages it and update its expectations.
func (c *SVMController) addVirtualMachine(obj interface{}) {
	vm := obj.(*virtv1.VirtualMachine)

	log.Log.Object(vm).V(4).Info("VM added.")

	if vm.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new vm shows up in a state that
		// is already pending deletion. Prevent the vm from being a creation observation.
		c.deleteVirtualMachine(vm)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := controller.GetControllerOf(vm); controllerRef != nil {
		log.Log.Object(vm).Info("Looking for VM Ref")
		svm := c.resolveControllerRef(vm.Namespace, controllerRef)
		if svm == nil {
			log.Log.Object(vm).Errorf("Cant find the matching SVM for VM: %s", vm.Name)
			return
		}
		svmKey, err := controller.KeyFunc(svm)
		if err != nil {
			log.Log.Object(vm).Errorf("Cannot parse key of SVM: %s for VM: %s", svm.Name, vm.Name)
			return
		}
		log.Log.Object(vm).Infof("VirtualMachine created bacause %s was added.", vm.Name)
		c.expectations.CreationObserved(svmKey)
		c.enqueueSvm(svm)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching StatefulVirtualMachines and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	svms := c.getMatchingControllers(vm)
	if len(svms) == 0 {
		return
	}
	log.Log.V(4).Object(vm).Infof("Orphan VirtualMachine created")
	for _, svm := range svms {
		c.enqueueSvm(svm)
	}
}

// When a vm is updated, figure out what StatefulVirtualMachine manage it and wake them
// up. If the labels of the vm have changed we need to awaken both the old
// and new StatefulVirtualMachine. old and cur must be *v1.VirtualMachine types.
func (c *SVMController) updateVirtualMachine(old, cur interface{}) {
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
		// for modification of the deletion timestamp and expect an StatefulVirtualMachine to create newVM asap, not wait
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
			c.enqueueSvm(rs)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		rs := c.resolveControllerRef(curVM.Namespace, curControllerRef)
		if rs == nil {
			return
		}
		log.Log.V(4).Object(curVM).Infof("VirtualMachine updated")
		c.enqueueSvm(rs)
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
			c.enqueueSvm(rs)
		}
	}
}

// When a vm is deleted, enqueue the StatefulVirtualMachine that manages the vm and update its expectations.
// obj could be an *v1.VirtualMachine, or a DeletionFinalStateUnknown marker item.
func (c *SVMController) deleteVirtualMachine(obj interface{}) {
	vm, ok := obj.(*virtv1.VirtualMachine)

	// When a delete is dropped, the relist will notice a vm in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the vm
	// changed labels the new StatefulVirtualMachine will not be woken up till the periodic resync.
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
	svm := c.resolveControllerRef(vm.Namespace, controllerRef)
	if svm == nil {
		return
	}
	svmKey, err := controller.KeyFunc(svm)
	if err != nil {
		return
	}
	c.expectations.DeletionObserved(svmKey, controller.VirtualMachineKey(vm))
	c.enqueueSvm(svm)
}

func (c *SVMController) addSvm(obj interface{}) {
	c.enqueueSvm(obj)
}

func (c *SVMController) deleteSvm(obj interface{}) {
	c.enqueueSvm(obj)
}

func (c *SVMController) updateSvm(old, curr interface{}) {
	c.enqueueSvm(curr)
}

func (c *SVMController) enqueueSvm(obj interface{}) {
	logger := log.Log
	svm := obj.(*virtv1.StatefulVirtualMachine)
	key, err := controller.KeyFunc(svm)
	if err != nil {
		logger.Object(svm).Reason(err).Error("Failed to extract svmKey from StatefulVirtualMachine.")
	}
	c.Queue.Add(key)
}

func (c *SVMController) hasCondition(svm *virtv1.StatefulVirtualMachine, cond virtv1.StatefulVirtualMachineConditionType) bool {
	for _, c := range svm.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (c *SVMController) removeCondition(svm *virtv1.StatefulVirtualMachine, cond virtv1.StatefulVirtualMachineConditionType) {
	var conds []virtv1.StatefulVirtualMachineCondition
	for _, c := range svm.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	svm.Status.Conditions = conds
}

func (c *SVMController) updateStatus(svm *virtv1.StatefulVirtualMachine, vm *virtv1.VirtualMachine, createErr, vmError error) error {

	// Check if it is worth updating
	errMatch := (createErr != nil) == c.hasCondition(svm, virtv1.StatefulVirtualMachineFailure)
	created := vm != nil
	createdMatch := created == svm.Status.Created

	ready := false
	if created {
		ready = vm.IsReady()
	}
	readyMatch := ready == svm.Status.Ready

	if errMatch && createdMatch && readyMatch {
		return nil
	}

	// Set created and ready flags
	svm.Status.Created = created
	svm.Status.Ready = ready

	// Add/Remove Failure condition if necessary
	if !(errMatch) {
		c.processFailure(svm, vm, createErr)
	}

	_, err := c.clientset.StatefulVirtualMachine(svm.ObjectMeta.Namespace).Update(svm)

	return err
}

func (c *SVMController) getVirtualMachineBaseName(svm *virtv1.StatefulVirtualMachine) string {

	// TODO defaulting should make sure that the right field is set, instead of doing this
	if len(svm.Spec.Template.ObjectMeta.Name) > 0 {
		return svm.Spec.Template.ObjectMeta.Name
	}
	if len(svm.Spec.Template.ObjectMeta.GenerateName) > 0 {
		return svm.Spec.Template.ObjectMeta.GenerateName
	}
	return svm.ObjectMeta.Name
}

func (c *SVMController) processFailure(svm *virtv1.StatefulVirtualMachine, vm *virtv1.VirtualMachine, createErr error) {
	reason := ""
	message := ""
	log.Log.Object(svm).Infof("Processing failure status:: shouldRun: %t; noErr: %t; noVm: %t", svm.Spec.Running, createErr != nil, vm != nil)

	if createErr != nil {
		if svm.Spec.Running == true {
			reason = "FailedCreate"
		} else {
			reason = "FailedDelete"
		}
		message = createErr.Error()

		if !c.hasCondition(svm, virtv1.StatefulVirtualMachineFailure) {
			log.Log.Object(svm).Infof("Reason to fail: %s", reason)
			svm.Status.Conditions = append(svm.Status.Conditions, virtv1.StatefulVirtualMachineCondition{
				Type:               virtv1.StatefulVirtualMachineFailure,
				Reason:             reason,
				Message:            message,
				LastTransitionTime: v1.Now(),
				Status:             k8score.ConditionTrue,
			})
		}

		return
	}

	log.Log.Object(svm).Info("Removing failure")
	c.removeCondition(svm, virtv1.StatefulVirtualMachineFailure)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *SVMController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.StatefulVirtualMachine {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.StatefulVirtualMachineGroupVersionKind.Kind {
		return nil
	}
	svm, exists, err := c.vmSVMInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if svm.(*virtv1.StatefulVirtualMachine).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return svm.(*virtv1.StatefulVirtualMachine)
}
