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

func NewOVMController(vmInformer cache.SharedIndexInformer, vmOVMInformer cache.SharedIndexInformer, recorder record.EventRecorder, clientset kubecli.KubevirtClient) *OVMController {

	c := &OVMController{
		Queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmInformer:    vmInformer,
		vmOVMInformer: vmOVMInformer,
		recorder:      recorder,
		clientset:     clientset,
		expectations:  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmOVMInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addOvm,
		DeleteFunc: c.deleteOvm,
		UpdateFunc: c.updateOvm,
	})

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	return c
}

type OVMController struct {
	clientset     kubecli.KubevirtClient
	Queue         workqueue.RateLimitingInterface
	vmInformer    cache.SharedIndexInformer
	vmOVMInformer cache.SharedIndexInformer
	recorder      record.EventRecorder
	expectations  *controller.UIDTrackingControllerExpectations
}

func (c *OVMController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting OfflineVirtualMachine controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmInformer.HasSynced, c.vmOVMInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping OfflineVirtualMachine controller.")
}

func (c *OVMController) runWorker() {
	for c.Execute() {
	}
}

func (c *OVMController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing OfflineVirtualMachine %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed OfflineVirtualMachine %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *OVMController) execute(key string) error {

	obj, exists, err := c.vmOVMInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	OVM := obj.(*virtv1.OfflineVirtualMachine)

	logger := log.Log.Object(OVM)

	logger.Info("Started processing OVM")

	//TODO default rs if necessary, the aggregated apiserver will do that in the future
	if OVM.Spec.Template == nil {
		logger.Error("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	needsSync := c.expectations.SatisfiedExpectations(key)

	ovmKey, err := controller.KeyFunc(OVM)
	if err != nil {
		return err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (v1.Object, error) {
		fresh, err := c.clientset.OfflineVirtualMachine(OVM.ObjectMeta.Namespace).Get(OVM.ObjectMeta.Name, &v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != OVM.ObjectMeta.UID {
			return nil, fmt.Errorf("original OfflineVirtualMachine %v/%v is gone: got uid %v, wanted %v", OVM.Namespace, OVM.Name, fresh.UID, OVM.UID)
		}
		return fresh, nil
	})
	cm := controller.NewVirtualMachineControllerRefManager(controller.RealVirtualMachineControl{Clientset: c.clientset}, OVM, nil, virtv1.OfflineVirtualMachineGroupVersionKind, canAdoptFunc)

	var vm *virtv1.VirtualMachine
	vmObj, exist, err := c.vmInformer.GetStore().GetByKey(ovmKey)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch vm for namespace from cache.")
		return err
	}
	if !exist {
		logger.Infof("VM not found in cache %s", key)
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
	if needsSync && OVM.ObjectMeta.DeletionTimestamp == nil {
		logger.Infof("Creating or the VM: %t", OVM.Spec.Running)
		createErr = c.startStop(OVM, vm)
	}

	// If the controller is going to be deleted and the orphan finalizer is the next one, release the VMs. Don't update the status
	// TODO: Workaround for https://github.com/kubernetes/kubernetes/issues/56348, remove it once it is fixed
	if OVM.ObjectMeta.DeletionTimestamp != nil && controller.HasFinalizer(OVM, v1.FinalizerOrphanDependents) {
		return c.orphan(cm, vm)
	}

	if createErr != nil {
		logger.Reason(err).Error("Scaling the OfflineVirtualMachine failed.")
	}

	err = c.updateStatus(OVM.DeepCopy(), vm, createErr, vmError)
	if err != nil {
		logger.Reason(err).Error("Updating the OfflineVirtualMachine status failed.")
	}

	return err
}

// orphan removes the owner reference of all VMs which are owned by the controller instance.
// Workaround for https://github.com/kubernetes/kubernetes/issues/56348 to make no-cascading deletes possible
// We don't have to remove the finalizer. This part of the gc is not affected by the mentioned bug
// TODO +pkotas unify with replicasets. This function can be the same
func (c *OVMController) orphan(cm *controller.VirtualMachineControllerRefManager, vm *virtv1.VirtualMachine) error {
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

func (c *OVMController) startStop(ovm *virtv1.OfflineVirtualMachine, vm *virtv1.VirtualMachine) error {
	log.Log.Object(ovm).V(4).Infof("Start the VM: %t", ovm.Spec.Running)

	if ovm.Spec.Running == true {
		if vm != nil {
			if vm.IsFinal() {
				// The VM can fail od be finished. The job of this controller
				// is keep the VM running, therefore it restarts it.
				// restarting VM by stopping it and letting it start in next step
				err := c.stopVM(ovm, vm)
				if err != nil {
					log.Log.Object(ovm).Error("Cannot restart VM, the VM cannot be deleted.")
					return err
				}
				// return to let the controller pick up the expected deletion
			}
			// VM is OK no need to do anything
			return nil
		}

		err := c.startVM(ovm)
		return err
	}

	if ovm.Spec.Running == false {
		log.Log.Object(ovm).V(4).Info("It is false delete")
		if vm == nil {
			log.Log.Info("vm is nil")
			// vm should not run and is not running
			return nil
		}
		err := c.stopVM(ovm, vm)
		return err
	}

	return nil
}

func (c *OVMController) startVM(ovm *virtv1.OfflineVirtualMachine) error {
	// TODO add check for existence
	ovmKey, err := controller.KeyFunc(ovm)
	if err != nil {
		log.Log.Object(ovm).Reason(err).Error("Failed to extract ovmKey from OfflineVirtualMachine.")
		return nil
	}

	// start it
	vm := c.setupVMFromOVM(ovm)

	c.expectations.ExpectCreations(ovmKey, 1)
	vm, err = c.clientset.VM(ovm.ObjectMeta.Namespace).Create(vm)
	if err != nil {
		log.Log.Object(ovm).Infof("Failed to create VM: %s/%s", vm.Namespace, vm.Name)
		c.expectations.CreationObserved(ovmKey)
		c.recorder.Eventf(ovm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine: %v", err)
		return err
	}
	c.recorder.Eventf(ovm, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Created virtual machine: %v", vm.ObjectMeta.Name)

	return nil
}

func (c *OVMController) stopVM(ovm *virtv1.OfflineVirtualMachine, vm *virtv1.VirtualMachine) error {
	if vm == nil {
		// nothing to do
		return nil
	}

	ovmKey, err := controller.KeyFunc(ovm)
	if err != nil {
		log.Log.Object(ovm).Reason(err).Error("Failed to extract ovmKey from OfflineVirtualMachine.")
		return nil
	}

	// stop it
	c.expectations.ExpectDeletions(ovmKey, []string{controller.VirtualMachineKey(vm)})
	err = c.clientset.VM(ovm.ObjectMeta.Namespace).Delete(vm.ObjectMeta.Name, &v1.DeleteOptions{})

	// Don't log an error if it is already deleted
	if err != nil {
		// We can't observe a delete if it was not accepted by the server
		c.expectations.DeletionObserved(ovmKey, controller.VirtualMachineKey(vm))
		c.recorder.Eventf(ovm, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting virtual machine %s: %v", vm.ObjectMeta.Name, err)
		return err
	}

	c.recorder.Eventf(ovm, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Deleted virtual machine: %v", vm.ObjectMeta.UID)
	log.Log.Object(ovm).Info("Dispatching delete event")

	return nil
}

// setupVMfromOVM creates a VirtualMachine object from one OfflineVirtualMachine object.
func (c *OVMController) setupVMFromOVM(ovm *virtv1.OfflineVirtualMachine) *virtv1.VirtualMachine {
	basename := c.getVirtualMachineBaseName(ovm)

	vm := virtv1.NewVMReferenceFromNameWithNS(ovm.ObjectMeta.Namespace, "")
	vm.ObjectMeta = ovm.Spec.Template.ObjectMeta
	vm.ObjectMeta.Name = basename
	vm.ObjectMeta.GenerateName = basename
	vm.Spec = ovm.Spec.Template.Spec

	setupStableFirmwareUUID(ovm, vm)

	t := true
	// TODO check if vm labels exist, and when make sure that they match. For now just override them
	vm.ObjectMeta.Labels = ovm.Spec.Template.ObjectMeta.Labels
	vm.ObjectMeta.OwnerReferences = []v1.OwnerReference{v1.OwnerReference{
		APIVersion:         virtv1.OfflineVirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               virtv1.OfflineVirtualMachineGroupVersionKind.Kind,
		Name:               ovm.ObjectMeta.Name,
		UID:                ovm.ObjectMeta.UID,
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
func setupStableFirmwareUUID(ovm *virtv1.OfflineVirtualMachine, vm *virtv1.VirtualMachine) {

	logger := log.Log.Object(ovm)

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
func (c *OVMController) filterActiveVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return !vm.IsFinal()
	})
}

// filterReadyVMs takes a list of VMs and returns all VMs which are in ready state.
// TODO +pkotas unify with replicaset this code is the same
func (c *OVMController) filterReadyVMs(vms []*virtv1.VirtualMachine) []*virtv1.VirtualMachine {
	return filter(vms, func(vm *virtv1.VirtualMachine) bool {
		return vm.IsReady()
	})
}

// listVMsFromNamespace takes a namespace and returns all VMs from the VM cache which run in this namespace
// TODO +pkotas unify this code with replicaset
func (c *OVMController) listVMsFromNamespace(namespace string) ([]*virtv1.VirtualMachine, error) {
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

// listControllerFromNamespace takes a namespace and returns all OfflineVirtualMachines
// from the OfflineVirtualMachine cache which run in this namespace
func (c *OVMController) listControllerFromNamespace(namespace string) ([]*virtv1.OfflineVirtualMachine, error) {
	objs, err := c.vmOVMInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	ovms := []*virtv1.OfflineVirtualMachine{}
	for _, obj := range objs {
		ovm := obj.(*virtv1.OfflineVirtualMachine)
		ovms = append(ovms, ovm)
	}
	return ovms, nil
}

// getMatchingControllers returns the list of OfflineVirtualMachines which matches
// the labels of the VM from the listener cache. If there are no matching
// controllers nothing is returned
func (c *OVMController) getMatchingControllers(vm *virtv1.VirtualMachine) (ovms []*virtv1.OfflineVirtualMachine) {
	controllers, err := c.listControllerFromNamespace(vm.ObjectMeta.Namespace)
	if err != nil {
		return nil
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, ovm := range controllers {
		if vm.Name == ovm.Name {
			ovms = append(ovms, ovm)
		}
	}
	return ovms
}

// When a vm is created, enqueue the OfflineVirtualMachine that manages it and update its expectations.
func (c *OVMController) addVirtualMachine(obj interface{}) {
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
		ovm := c.resolveControllerRef(vm.Namespace, controllerRef)
		if ovm == nil {
			log.Log.Object(vm).Errorf("Cant find the matching OVM for VM: %s", vm.Name)
			return
		}
		ovmKey, err := controller.KeyFunc(ovm)
		if err != nil {
			log.Log.Object(vm).Errorf("Cannot parse key of OVM: %s for VM: %s", ovm.Name, vm.Name)
			return
		}
		log.Log.Object(vm).Infof("VirtualMachine created bacause %s was added.", vm.Name)
		c.expectations.CreationObserved(ovmKey)
		c.enqueueOvm(ovm)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching OfflineVirtualMachines and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	ovms := c.getMatchingControllers(vm)
	if len(ovms) == 0 {
		return
	}
	log.Log.V(4).Object(vm).Infof("Orphan VirtualMachine created")
	for _, ovm := range ovms {
		c.enqueueOvm(ovm)
	}
}

// When a vm is updated, figure out what OfflineVirtualMachine manage it and wake them
// up. If the labels of the vm have changed we need to awaken both the old
// and new OfflineVirtualMachine. old and cur must be *v1.VirtualMachine types.
func (c *OVMController) updateVirtualMachine(old, cur interface{}) {
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
		// for modification of the deletion timestamp and expect an OfflineVirtualMachine to create newVM asap, not wait
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
			c.enqueueOvm(rs)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		rs := c.resolveControllerRef(curVM.Namespace, curControllerRef)
		if rs == nil {
			return
		}
		log.Log.V(4).Object(curVM).Infof("VirtualMachine updated")
		c.enqueueOvm(rs)
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
			c.enqueueOvm(rs)
		}
	}
}

// When a vm is deleted, enqueue the OfflineVirtualMachine that manages the vm and update its expectations.
// obj could be an *v1.VirtualMachine, or a DeletionFinalStateUnknown marker item.
func (c *OVMController) deleteVirtualMachine(obj interface{}) {
	vm, ok := obj.(*virtv1.VirtualMachine)

	// When a delete is dropped, the relist will notice a vm in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the vm
	// changed labels the new OfflineVirtualMachine will not be woken up till the periodic resync.
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
	ovm := c.resolveControllerRef(vm.Namespace, controllerRef)
	if ovm == nil {
		return
	}
	ovmKey, err := controller.KeyFunc(ovm)
	if err != nil {
		return
	}
	c.expectations.DeletionObserved(ovmKey, controller.VirtualMachineKey(vm))
	c.enqueueOvm(ovm)
}

func (c *OVMController) addOvm(obj interface{}) {
	c.enqueueOvm(obj)
}

func (c *OVMController) deleteOvm(obj interface{}) {
	c.enqueueOvm(obj)
}

func (c *OVMController) updateOvm(old, curr interface{}) {
	c.enqueueOvm(curr)
}

func (c *OVMController) enqueueOvm(obj interface{}) {
	logger := log.Log
	ovm := obj.(*virtv1.OfflineVirtualMachine)
	key, err := controller.KeyFunc(ovm)
	if err != nil {
		logger.Object(ovm).Reason(err).Error("Failed to extract ovmKey from OfflineVirtualMachine.")
	}
	c.Queue.Add(key)
}

func (c *OVMController) hasCondition(ovm *virtv1.OfflineVirtualMachine, cond virtv1.OfflineVirtualMachineConditionType) bool {
	for _, c := range ovm.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (c *OVMController) removeCondition(ovm *virtv1.OfflineVirtualMachine, cond virtv1.OfflineVirtualMachineConditionType) {
	var conds []virtv1.OfflineVirtualMachineCondition
	for _, c := range ovm.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	ovm.Status.Conditions = conds
}

func (c *OVMController) updateStatus(ovm *virtv1.OfflineVirtualMachine, vm *virtv1.VirtualMachine, createErr, vmError error) error {

	// Check if it is worth updating
	runningMatch := ovm.Spec.Running == c.hasCondition(ovm, virtv1.OfflineVirtualMachineRunning)
	errMatch := (createErr != nil) == c.hasCondition(ovm, virtv1.OfflineVirtualMachineFailure)

	log.Log.Object(ovm).Infof("Update: shouldRun: %t; shouldHaveErr: %t", runningMatch, errMatch)

	if errMatch && runningMatch {
		return nil
	}

	// Add/Remove Failure condition if necessary
	if !(errMatch) {
		c.processFailure(ovm, vm, createErr)
	}

	// update condition if the vm is running or not
	if !runningMatch {
		c.processRunning(ovm, vm, createErr)
	}

	_, err := c.clientset.OfflineVirtualMachine(ovm.ObjectMeta.Namespace).Update(ovm)

	return err
}

func (c *OVMController) getVirtualMachineBaseName(ovm *virtv1.OfflineVirtualMachine) string {

	// TODO defaulting should make sure that the right field is set, instead of doing this
	if len(ovm.Spec.Template.ObjectMeta.Name) > 0 {
		return ovm.Spec.Template.ObjectMeta.Name
	}
	if len(ovm.Spec.Template.ObjectMeta.GenerateName) > 0 {
		return ovm.Spec.Template.ObjectMeta.GenerateName
	}
	return ovm.ObjectMeta.Name
}

func (c *OVMController) processRunning(ovm *virtv1.OfflineVirtualMachine, vm *virtv1.VirtualMachine, createErr error) {
	log.Log.Object(ovm).Infof("Processing running status:: shouldRun: %t; noErr: %t; noVm: %t", ovm.Spec.Running, createErr != nil, vm != nil)
	if vm == nil {
		c.removeCondition(ovm, virtv1.OfflineVirtualMachineRunning)
		return
	}

	if ovm.Spec.Running && createErr == nil && !c.hasCondition(ovm, virtv1.OfflineVirtualMachineRunning) && vm.Status.Phase == virtv1.Running {
		log.Log.Object(ovm).Info("Adding running condition")
		ovm.Status.Conditions = append(ovm.Status.Conditions, virtv1.OfflineVirtualMachineCondition{
			Type:               virtv1.OfflineVirtualMachineRunning,
			Reason:             fmt.Sprintf("Created by OVM %s", ovm.ObjectMeta.Name),
			Message:            fmt.Sprintf("Created by OVM %s", ovm.ObjectMeta.Name),
			LastTransitionTime: v1.Now(),
			Status:             k8score.ConditionTrue,
		})

		return
	}

	c.removeCondition(ovm, virtv1.OfflineVirtualMachineRunning)
}

func (c *OVMController) processFailure(ovm *virtv1.OfflineVirtualMachine, vm *virtv1.VirtualMachine, createErr error) {
	reason := ""
	message := ""
	log.Log.Object(ovm).Infof("Processing failure status:: shouldRun: %t; noErr: %t; noVm: %t", ovm.Spec.Running, createErr != nil, vm != nil)

	if createErr != nil {
		if ovm.Spec.Running == true {
			reason = "FailedCreate"
		} else {
			reason = "FailedDelete"
		}
		message = createErr.Error()

		if !c.hasCondition(ovm, virtv1.OfflineVirtualMachineFailure) {
			log.Log.Object(ovm).Infof("Reason to fail: %s", reason)
			ovm.Status.Conditions = append(ovm.Status.Conditions, virtv1.OfflineVirtualMachineCondition{
				Type:               virtv1.OfflineVirtualMachineFailure,
				Reason:             reason,
				Message:            message,
				LastTransitionTime: v1.Now(),
				Status:             k8score.ConditionTrue,
			})
		}

		return
	}

	log.Log.Object(ovm).Info("Removing failure")
	c.removeCondition(ovm, virtv1.OfflineVirtualMachineFailure)
}

func OvmOwnerRef(ovm *virtv1.OfflineVirtualMachine) v1.OwnerReference {
	t := true
	gvk := virtv1.OfflineVirtualMachineGroupVersionKind
	return v1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               ovm.ObjectMeta.Name,
		UID:                ovm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *OVMController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.OfflineVirtualMachine {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.OfflineVirtualMachineGroupVersionKind.Kind {
		return nil
	}
	ovm, exists, err := c.vmOVMInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if ovm.(*virtv1.OfflineVirtualMachine).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return ovm.(*virtv1.OfflineVirtualMachine)
}
