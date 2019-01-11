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
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/pborman/uuid"
	k8score "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

// TODO remove the dataVolume deletion retry logic once CDI fixes this issue.
// We're working around the fact that DataVolumes aren't eventually consistent by
// attempting to delete/recreate the datavolumes when they fail to import
// https://github.com/kubevirt/containerized-data-importer/issues/400
const (
	dataVolumeDeleteAfterTimestampAnno = "kubevirt.io/delete-after-timestamp"
	dataVolumeDeleteJitterSeconds      = 100
)

func NewVMController(vmiInformer cache.SharedIndexInformer,
	vmiVMInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *VMController {

	c := &VMController{
		Queue:                  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:            vmiInformer,
		vmiVMInformer:          vmiVMInformer,
		dataVolumeInformer:     dataVolumeInformer,
		recorder:               recorder,
		clientset:              clientset,
		expectations:           controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		dataVolumeExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
	}

	c.vmiVMInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVm,
		DeleteFunc: c.deleteVm,
		UpdateFunc: c.updateVm,
	})

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	c.dataVolumeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDataVolume,
		DeleteFunc: c.deleteDataVolume,
		UpdateFunc: c.updateDataVolume,
	})

	return c
}

type VMController struct {
	clientset              kubecli.KubevirtClient
	Queue                  workqueue.RateLimitingInterface
	vmiInformer            cache.SharedIndexInformer
	vmiVMInformer          cache.SharedIndexInformer
	dataVolumeInformer     cache.SharedIndexInformer
	recorder               record.EventRecorder
	expectations           *controller.UIDTrackingControllerExpectations
	dataVolumeExpectations *controller.UIDTrackingControllerExpectations
}

func (c *VMController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachine controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced, c.vmiVMInformer.HasSynced, c.dataVolumeInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VirtualMachine controller.")
}

func (c *VMController) runWorker() {
	for c.Execute() {
	}
}

func (c *VMController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}
	defer c.Queue.Done(key)
	if err := c.execute(key.(string)); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachine %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachine %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *VMController) execute(key string) error {

	obj, exists, err := c.vmiVMInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	VM := obj.(*virtv1.VirtualMachine)

	logger := log.Log.Object(VM)

	logger.V(4).Info("Started processing VM")

	//TODO default vm if necessary, the aggregated apiserver will do that in the future
	if VM.Spec.Template == nil {
		logger.Error("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	needsSync := c.expectations.SatisfiedExpectations(key) && c.dataVolumeExpectations.SatisfiedExpectations(key)

	vmKey, err := controller.KeyFunc(VM)
	if err != nil {
		return err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (v1.Object, error) {
		fresh, err := c.clientset.VirtualMachine(VM.ObjectMeta.Namespace).Get(VM.ObjectMeta.Name, &v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != VM.ObjectMeta.UID {
			return nil, fmt.Errorf("original VirtualMachine %v/%v is gone: got uid %v, wanted %v", VM.Namespace, VM.Name, fresh.UID, VM.UID)
		}
		return fresh, nil
	})
	cm := controller.NewVirtualMachineControllerRefManager(
		controller.RealVirtualMachineControl{
			Clientset: c.clientset,
		}, VM, nil, virtv1.VirtualMachineGroupVersionKind, canAdoptFunc)

	var vmi *virtv1.VirtualMachineInstance
	vmiObj, exist, err := c.vmiInformer.GetStore().GetByKey(vmKey)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch vmi for namespace from cache.")
		return err
	}
	if !exist {
		logger.V(4).Infof("VirtualMachineInstance not found in cache %s", key)
		vmi = nil
	} else {
		vmi = vmiObj.(*virtv1.VirtualMachineInstance)

		vmi, err = cm.ClaimVirtualMachineByName(vmi)
		if err != nil {
			return err
		}
	}

	dataVolumes, err := c.listDataVolumesForVM(VM)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch dataVolumes for namespace from cache.")
		return err
	}

	if len(dataVolumes) != 0 {
		dataVolumes, err = cm.ClaimMatchedDataVolumes(dataVolumes)
		if err != nil {
			return err
		}
	}

	var createErr error

	// Scale up or down, if all expected creates and deletes were report by the listener
	if needsSync && VM.ObjectMeta.DeletionTimestamp == nil {

		dataVolumesReady, err := c.handleDataVolumes(VM, dataVolumes)
		if err != nil {
			createErr = err
		} else if dataVolumesReady == true {
			createErr = c.startStop(VM, vmi)
		} else {
			log.Log.Object(VM).V(3).Infof("Waiting on DataVolumes to be ready. %d datavolumes found", len(dataVolumes))
		}
	}

	// If the controller is going to be deleted and the orphan finalizer is the next one, release the VMIs. Don't update the status
	// TODO: Workaround for https://github.com/kubernetes/kubernetes/issues/56348, remove it once it is fixed
	if VM.ObjectMeta.DeletionTimestamp != nil && controller.HasFinalizer(VM, v1.FinalizerOrphanDependents) {
		err = c.orphan(cm, vmi)
		if err != nil {
			return err
		}
		return c.orphanDataVolumes(cm, dataVolumes)
	}

	if createErr != nil {
		logger.Reason(err).Error("Creating the VirtualMachine failed.")
	}

	err = c.updateStatus(VM.DeepCopy(), vmi, createErr)
	if err != nil {
		logger.Reason(err).Error("Updating the VirtualMachine status failed.")
		return err
	}

	return createErr
}

func (c *VMController) listDataVolumesForVM(vm *virtv1.VirtualMachine) ([]*cdiv1.DataVolume, error) {

	var dataVolumes []*cdiv1.DataVolume

	if len(vm.Spec.DataVolumeTemplates) == 0 {
		return dataVolumes, nil
	}

	for _, template := range vm.Spec.DataVolumeTemplates {
		// get DataVolume from cache for each templated dataVolume
		obj, exists, err := c.dataVolumeInformer.GetStore().GetByKey(fmt.Sprintf("%s/%s", vm.Namespace, template.Name))

		if err != nil {
			return dataVolumes, err
		} else if !exists {
			continue
		}

		dataVolumes = append(dataVolumes, obj.(*cdiv1.DataVolume))
	}
	return dataVolumes, nil
}

// orphan removes the owner reference of all VMIs which are owned by the controller instance.
// Workaround for https://github.com/kubernetes/kubernetes/issues/56348 to make no-cascading deletes possible
// We don't have to remove the finalizer. This part of the gc is not affected by the mentioned bug
// TODO +pkotas unify with replicasets. This function can be the same
func (c *VMController) orphan(cm *controller.VirtualMachineControllerRefManager, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil {
		return nil
	}

	err := cm.ReleaseVirtualMachine(vmi)
	if err != nil {
		return err
	}
	return nil
}

func (c *VMController) orphanDataVolumes(cm *controller.VirtualMachineControllerRefManager, dataVolumes []*cdiv1.DataVolume) error {

	if len(dataVolumes) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(dataVolumes))
	wg.Add(len(dataVolumes))

	for _, dataVolume := range dataVolumes {
		go func(dataVolume *cdiv1.DataVolume) {
			defer wg.Done()
			err := cm.ReleaseDataVolume(dataVolume)
			if err != nil {
				errChan <- err
			}
		}(dataVolume)
	}
	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
	}
	return nil
}

func createDataVolumeManifest(dataVolume *cdiv1.DataVolume, vm *virtv1.VirtualMachine) *cdiv1.DataVolume {

	newDataVolume := dataVolume.DeepCopy()

	labels := map[string]string{}
	annotations := map[string]string{}

	labels[virtv1.CreatedByLabel] = string(vm.UID)

	for k, v := range dataVolume.Labels {
		annotations[k] = v
	}
	for k, v := range dataVolume.Labels {
		labels[k] = v
	}
	newDataVolume.ObjectMeta.Labels = labels
	newDataVolume.ObjectMeta.Annotations = annotations

	newDataVolume.ObjectMeta.OwnerReferences = []v1.OwnerReference{
		*v1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}
	return newDataVolume
}

func (c *VMController) handleDataVolumes(vm *virtv1.VirtualMachine, dataVolumes []*cdiv1.DataVolume) (bool, error) {
	ready := true
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return ready, err
	}
	for _, template := range vm.Spec.DataVolumeTemplates {
		var curDataVolume *cdiv1.DataVolume
		exists := false
		for _, curDataVolume = range dataVolumes {
			if curDataVolume.Name == template.Name {
				exists = true
				break
			}
		}
		if !exists {
			// ready = false because encountered DataVolume that is not created yet
			ready = false
			newDataVolume := createDataVolumeManifest(&template, vm)

			c.dataVolumeExpectations.ExpectCreations(vmKey, 1)
			curDataVolume, err = c.clientset.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Create(newDataVolume)
			if err != nil {
				c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDataVolumeCreateReason, "Error creating DataVolume %s: %v", newDataVolume.Name, err)
				c.dataVolumeExpectations.CreationObserved(vmKey)
				return ready, fmt.Errorf("Failed to create DataVolume: %v", err)
			}
			c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDataVolumeCreateReason, "Created DataVolume %s", curDataVolume.Name)
		} else if curDataVolume.Status.Phase != cdiv1.Succeeded {
			// ready = false because encountered DataVolume that is not populated yet
			ready = false
			if curDataVolume.Status.Phase == cdiv1.Failed {

				c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDataVolumeImportReason, "DataVolume %s failed to import disk image", curDataVolume.Name)

				now := time.Now().UTC().Unix()
				deleteAfterStr, ok := curDataVolume.Annotations[dataVolumeDeleteAfterTimestampAnno]
				var deleteAfterTimestamp int64
				if ok {
					deleteAfterTimestamp, err = strconv.ParseInt(deleteAfterStr, 10, 64)
					if err != nil {
						// No reason to return an error here
						// we just clobber the annotation with a valid value later if it can't be parsed as an int.
						deleteAfterTimestamp = 0
						log.Log.Errorf("failed to parse deletion timestamp for datavolume %s/%s: %v", curDataVolume.Namespace, curDataVolume.Name, err)
					}
				}

				if deleteAfterTimestamp == 0 {
					dataVolumeCopy := curDataVolume.DeepCopy()
					deleteAfterTimestamp := now + int64(rand.Intn(dataVolumeDeleteJitterSeconds)+10)
					dataVolumeCopy.Annotations[dataVolumeDeleteAfterTimestampAnno] = strconv.FormatInt(deleteAfterTimestamp, 10)
					_, err := c.clientset.CdiClient().CdiV1alpha1().DataVolumes(dataVolumeCopy.Namespace).Update(dataVolumeCopy)
					if err != nil {
						return ready, err
					}
				}

				if curDataVolume.DeletionTimestamp == nil {
					if deleteAfterTimestamp >= now {
						// By deleting the failed DataVolume,
						// a new DataVolume will be created to take it's place.
						c.dataVolumeExpectations.ExpectDeletions(vmKey, []string{controller.DataVolumeKey(curDataVolume)})
						err := c.clientset.CdiClient().CdiV1alpha1().DataVolumes(curDataVolume.Namespace).Delete(curDataVolume.Name, &v1.DeleteOptions{})
						if err != nil {
							c.dataVolumeExpectations.DeletionObserved(vmKey, controller.DataVolumeKey(curDataVolume))
							return ready, err
						}
					} else {
						timeLeft := now - deleteAfterTimestamp
						c.Queue.AddAfter(vmKey, time.Duration(timeLeft)*time.Second)
					}
				}
			}
		}
	}
	return ready, nil
}

func (c *VMController) startStop(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	log.Log.Object(vm).V(4).Infof("Start the VirtualMachineInstance: %t", vm.Spec.Running)

	if vm.Spec.Running == true {
		if vmi != nil {
			if vmi.IsFinal() {
				// The VirtualMachineInstance can fail od be finished. The job of this controller
				// is keep the VirtualMachineInstance running, therefore it restarts it.
				// restarting VirtualMachineInstance by stopping it and letting it start in next step
				err := c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Error("Cannot restart VirtualMachineInstance, the VirtualMachineInstance cannot be deleted.")
					return err
				}
				// return to let the controller pick up the expected deletion
			}
			// VirtualMachineInstance is OK no need to do anything
			return nil
		}

		err := c.startVMI(vm)
		return err
	}

	if vm.Spec.Running == false {
		log.Log.Object(vm).V(4).Info("It is false delete")
		if vmi == nil {
			// vmi should not run and is not running
			return nil
		}
		err := c.stopVMI(vm, vmi)
		return err
	}

	return nil
}

func (c *VMController) startVMI(vm *virtv1.VirtualMachine) error {
	// TODO add check for existence
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Failed to extract vmKey from VirtualMachine.")
		return nil
	}

	// start it
	vmi := c.setupVMIFromVM(vm)

	c.expectations.ExpectCreations(vmKey, 1)
	vmi, err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Create(vmi)
	if err != nil {
		log.Log.Object(vm).Infof("Failed to create VirtualMachineInstance: %s/%s", vmi.Namespace, vmi.Name)
		c.expectations.CreationObserved(vmKey)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine: %v", err)
		return err
	}
	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Created virtual machine: %v", vmi.ObjectMeta.Name)

	return nil
}

func (c *VMController) stopVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		// nothing to do
		return nil
	}

	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Failed to extract vmKey from VirtualMachine.")
		return nil
	}

	// stop it
	c.expectations.ExpectDeletions(vmKey, []string{controller.VirtualMachineKey(vmi)})
	err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Delete(vmi.ObjectMeta.Name, &v1.DeleteOptions{})

	// Don't log an error if it is already deleted
	if err != nil {
		// We can't observe a delete if it was not accepted by the server
		c.expectations.DeletionObserved(vmKey, controller.VirtualMachineKey(vmi))
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting virtual machine %s: %v", vmi.ObjectMeta.Name, err)
		return err
	}

	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Deleted virtual machine: %v", vmi.ObjectMeta.UID)
	log.Log.Object(vm).Info("Dispatching delete event")

	return nil
}

// setupVMIfromVM creates a VirtualMachineInstance object from one VirtualMachine object.
func (c *VMController) setupVMIFromVM(vm *virtv1.VirtualMachine) *virtv1.VirtualMachineInstance {
	basename := c.getVirtualMachineBaseName(vm)

	vmi := virtv1.NewVMIReferenceFromNameWithNS(vm.ObjectMeta.Namespace, "")
	vmi.ObjectMeta = vm.Spec.Template.ObjectMeta
	vmi.ObjectMeta.Name = basename
	vmi.ObjectMeta.GenerateName = basename
	vmi.Spec = vm.Spec.Template.Spec

	setupStableFirmwareUUID(vm, vmi)

	// TODO check if vmi labels exist, and when make sure that they match. For now just override them
	vmi.ObjectMeta.Labels = vm.Spec.Template.ObjectMeta.Labels
	vmi.ObjectMeta.OwnerReferences = []v1.OwnerReference{
		*v1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}

	return vmi
}

// no special meaning, randomly generated on my box.
// TODO: do we want to use another constants? see examples in RFC4122
const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.Parse(magicUUID)

// setStableUUID makes sure the VirtualMachineInstance being started has a a 'stable' UUID.
// The UUID is 'stable' if doesn't change across reboots.
func setupStableFirmwareUUID(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {

	logger := log.Log.Object(vm)

	if vmi.Spec.Domain.Firmware == nil {
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{}
	}

	existingUUID := vmi.Spec.Domain.Firmware.UUID
	if existingUUID != "" {
		logger.V(4).Infof("Using existing UUID '%s'", existingUUID)
		return
	}

	vmi.Spec.Domain.Firmware.UUID = types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(vmi.ObjectMeta.Name)).String())
}

// filterActiveVMIs takes a list of VMIs and returns all VMIs which are not in a final state
// TODO +pkotas unify with replicaset this code is the same without dependency
func (c *VMController) filterActiveVMIs(vmis []*virtv1.VirtualMachineInstance) []*virtv1.VirtualMachineInstance {
	return filter(vmis, func(vmi *virtv1.VirtualMachineInstance) bool {
		return !vmi.IsFinal()
	})
}

// filterReadyVMIs takes a list of VMIs and returns all VMIs which are in ready state.
// TODO +pkotas unify with replicaset this code is the same
func (c *VMController) filterReadyVMIs(vmis []*virtv1.VirtualMachineInstance) []*virtv1.VirtualMachineInstance {
	return filter(vmis, func(vmi *virtv1.VirtualMachineInstance) bool {
		return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceConditionType(k8score.PodReady), k8score.ConditionTrue)
	})
}

// listVMIsFromNamespace takes a namespace and returns all VMIs from the VirtualMachineInstance cache which run in this namespace
// TODO +pkotas unify this code with replicaset
func (c *VMController) listVMIsFromNamespace(namespace string) ([]*virtv1.VirtualMachineInstance, error) {
	objs, err := c.vmiInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	var vmis []*virtv1.VirtualMachineInstance
	for _, obj := range objs {
		vmis = append(vmis, obj.(*virtv1.VirtualMachineInstance))
	}
	return vmis, nil
}

// listControllerFromNamespace takes a namespace and returns all VirtualMachines
// from the VirtualMachine cache which run in this namespace
func (c *VMController) listControllerFromNamespace(namespace string) ([]*virtv1.VirtualMachine, error) {
	objs, err := c.vmiVMInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}
	var vms []*virtv1.VirtualMachine
	for _, obj := range objs {
		vm := obj.(*virtv1.VirtualMachine)
		vms = append(vms, vm)
	}
	return vms, nil
}

// getMatchingControllers returns the list of VirtualMachines which matches
// the labels of the VirtualMachineInstance from the listener cache. If there are no matching
// controllers nothing is returned
func (c *VMController) getMatchingControllers(vmi *virtv1.VirtualMachineInstance) (vms []*virtv1.VirtualMachine) {
	controllers, err := c.listControllerFromNamespace(vmi.ObjectMeta.Namespace)
	if err != nil {
		return nil
	}

	// TODO check owner reference, if we have an existing controller which owns this one

	for _, vm := range controllers {
		if vmi.Name == vm.Name {
			vms = append(vms, vm)
		}
	}
	return vms
}

// When a vmi is created, enqueue the VirtualMachine that manages it and update its expectations.
func (c *VMController) addVirtualMachine(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)

	log.Log.Object(vmi).V(4).Info("VirtualMachineInstance added.")

	if vmi.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new vmi shows up in a state that
		// is already pending deletion. Prevent the vmi from being a creation observation.
		c.deleteVirtualMachine(vmi)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := v1.GetControllerOf(vmi); controllerRef != nil {
		log.Log.Object(vmi).V(4).Info("Looking for VirtualMachineInstance Ref")
		vm := c.resolveControllerRef(vmi.Namespace, controllerRef)
		if vm == nil {
			log.Log.Object(vmi).Errorf("Cant find the matching VM for VirtualMachineInstance: %s", vmi.Name)
			return
		}
		vmKey, err := controller.KeyFunc(vm)
		if err != nil {
			log.Log.Object(vmi).Errorf("Cannot parse key of VM: %s for VirtualMachineInstance: %s", vm.Name, vmi.Name)
			return
		}
		log.Log.Object(vmi).V(4).Infof("VirtualMachineInstance created because %s was added.", vmi.Name)
		c.expectations.CreationObserved(vmKey)
		c.enqueueVm(vm)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching VirtualMachines and sync
	// them to see if anyone wants to adopt it.
	// DO NOT observe creation because no controller should be waiting for an
	// orphan.
	vms := c.getMatchingControllers(vmi)
	if len(vms) == 0 {
		return
	}
	log.Log.V(4).Object(vmi).Infof("Orphan VirtualMachineInstance created")
	for _, vm := range vms {
		c.enqueueVm(vm)
	}
}

// When a vmi is updated, figure out what VirtualMachine manage it and wake them
// up. If the labels of the vmi have changed we need to awaken both the old
// and new VirtualMachine. old and cur must be *v1.VirtualMachineInstance types.
func (c *VMController) updateVirtualMachine(old, cur interface{}) {
	curVMI := cur.(*virtv1.VirtualMachineInstance)
	oldVMI := old.(*virtv1.VirtualMachineInstance)
	if curVMI.ResourceVersion == oldVMI.ResourceVersion {
		// Periodic resync will send update events for all known vmis.
		// Two different versions of the same vmi will always have different RVs.
		return
	}

	labelChanged := !reflect.DeepEqual(curVMI.Labels, oldVMI.Labels)
	if curVMI.DeletionTimestamp != nil {
		// when a vmi is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
		// and after such time has passed, the virt-handler actually deletes it from the store. We receive an update
		// for modification of the deletion timestamp and expect an VirtualMachine to create newVMI asap, not wait
		// until the virt-handler actually deletes the vmi. This is different from the Phase of a vmi changing, because
		// an rs never initiates a phase change, and so is never asleep waiting for the same.
		c.deleteVirtualMachine(curVMI)
		if labelChanged {
			// we don't need to check the oldVMI.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deleteVirtualMachine(oldVMI)
		}
		return
	}

	curControllerRef := v1.GetControllerOf(curVMI)
	oldControllerRef := v1.GetControllerOf(oldVMI)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vm := c.resolveControllerRef(oldVMI.Namespace, oldControllerRef); vm != nil {
			c.enqueueVm(vm)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		vm := c.resolveControllerRef(curVMI.Namespace, curControllerRef)
		if vm == nil {
			return
		}
		log.Log.V(4).Object(curVMI).Infof("VirtualMachineInstance updated")
		c.enqueueVm(vm)
		// TODO: MinReadySeconds in the VirtualMachineInstance will generate an Available condition to be added in
		// Update once we support the available conect on the rs
		return
	}

	// Otherwise, it's an orphan. If anything changed, sync matching controllers
	// to see if anyone wants to adopt it now.
	if labelChanged || controllerRefChanged {
		vms := c.getMatchingControllers(curVMI)
		if len(vms) == 0 {
			return
		}
		log.Log.V(4).Object(curVMI).Infof("Orphan VirtualMachineInstance updated")
		for _, vm := range vms {
			c.enqueueVm(vm)
		}
	}
}

// When a vmi is deleted, enqueue the VirtualMachine that manages the vmi and update its expectations.
// obj could be an *v1.VirtualMachineInstance, or a DeletionFinalStateUnknown marker item.
func (c *VMController) deleteVirtualMachine(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a vmi in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the vmi
	// changed labels the new VirtualMachine will not be woken up till the periodic resync.
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

	controllerRef := v1.GetControllerOf(vmi)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	vm := c.resolveControllerRef(vmi.Namespace, controllerRef)
	if vm == nil {
		return
	}
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return
	}
	c.expectations.DeletionObserved(vmKey, controller.VirtualMachineKey(vmi))
	c.enqueueVm(vm)
}

func (c *VMController) addDataVolume(obj interface{}) {
	dataVolume := obj.(*cdiv1.DataVolume)
	if dataVolume.DeletionTimestamp != nil {
		c.deleteDataVolume(dataVolume)
		return
	}
	controllerRef := v1.GetControllerOf(dataVolume)
	if controllerRef == nil {
		return
	}
	log.Log.Object(dataVolume).Info("Looking for DataVolume Ref")
	vm := c.resolveControllerRef(dataVolume.Namespace, controllerRef)
	if vm == nil {
		log.Log.Object(dataVolume).Errorf("Cant find the matching VM for DataVolume: %s", dataVolume.Name)
		return
	}
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(dataVolume).Errorf("Cannot parse key of VM: %s for DataVolume: %s", vm.Name, dataVolume.Name)
		return
	}
	log.Log.Object(dataVolume).Infof("DataVolume created because %s was added.", dataVolume.Name)
	c.dataVolumeExpectations.CreationObserved(vmKey)
	c.enqueueVm(vm)
}
func (c *VMController) updateDataVolume(old, cur interface{}) {
	curDataVolume := cur.(*cdiv1.DataVolume)
	oldDataVolume := old.(*cdiv1.DataVolume)
	if curDataVolume.ResourceVersion == oldDataVolume.ResourceVersion {
		// Periodic resync will send update events for all known DataVolumes.
		// Two different versions of the same dataVolume will always
		// have different RVs.
		return
	}
	labelChanged := !reflect.DeepEqual(curDataVolume.Labels, oldDataVolume.Labels)
	if curDataVolume.DeletionTimestamp != nil {
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
	curControllerRef := v1.GetControllerOf(curDataVolume)
	oldControllerRef := v1.GetControllerOf(oldDataVolume)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vm := c.resolveControllerRef(oldDataVolume.Namespace, oldControllerRef); vm != nil {
			c.enqueueVm(vm)
		}
	}
	if curControllerRef == nil {
		return
	}
	vm := c.resolveControllerRef(curDataVolume.Namespace, curControllerRef)
	if vm == nil {
		return
	}
	log.Log.V(4).Object(curDataVolume).Infof("DataVolume updated")
	c.enqueueVm(vm)
}

func (c *VMController) deleteDataVolume(obj interface{}) {
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
	controllerRef := v1.GetControllerOf(dataVolume)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	vm := c.resolveControllerRef(dataVolume.Namespace, controllerRef)
	if vm == nil {
		return
	}
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return
	}
	c.dataVolumeExpectations.DeletionObserved(vmKey, controller.DataVolumeKey(dataVolume))
	c.enqueueVm(vm)
}

func (c *VMController) addVm(obj interface{}) {
	c.enqueueVm(obj)
}

func (c *VMController) deleteVm(obj interface{}) {
	c.enqueueVm(obj)
}

func (c *VMController) updateVm(old, curr interface{}) {
	c.enqueueVm(curr)
}

func (c *VMController) enqueueVm(obj interface{}) {
	logger := log.Log
	vm := obj.(*virtv1.VirtualMachine)
	key, err := controller.KeyFunc(vm)
	if err != nil {
		logger.Object(vm).Reason(err).Error("Failed to extract vmKey from VirtualMachine.")
	}
	c.Queue.Add(key)
}

func (c *VMController) hasCondition(vm *virtv1.VirtualMachine, cond virtv1.VirtualMachineConditionType) bool {
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (c *VMController) removeCondition(vm *virtv1.VirtualMachine, cond virtv1.VirtualMachineConditionType) {
	var conds []virtv1.VirtualMachineCondition
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	vm.Status.Conditions = conds
}

func (c *VMController) updateStatus(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, createErr error) error {

	// Check if it is worth updating
	errMatch := (createErr != nil) == c.hasCondition(vm, virtv1.VirtualMachineFailure)
	created := vmi != nil
	createdMatch := created == vm.Status.Created

	ready := false
	if created {
		ready = controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceConditionType(k8score.PodReady), k8score.ConditionTrue)
	}
	readyMatch := ready == vm.Status.Ready

	if errMatch && createdMatch && readyMatch {
		return nil
	}

	// Set created and ready flags
	vm.Status.Created = created
	vm.Status.Ready = ready

	// Add/Remove Failure condition if necessary
	if !(errMatch) {
		c.processFailure(vm, vmi, createErr)
	}

	_, err := c.clientset.VirtualMachine(vm.ObjectMeta.Namespace).Update(vm)

	return err
}

func (c *VMController) getVirtualMachineBaseName(vm *virtv1.VirtualMachine) string {

	// TODO defaulting should make sure that the right field is set, instead of doing this
	if len(vm.Spec.Template.ObjectMeta.Name) > 0 {
		return vm.Spec.Template.ObjectMeta.Name
	}
	if len(vm.Spec.Template.ObjectMeta.GenerateName) > 0 {
		return vm.Spec.Template.ObjectMeta.GenerateName
	}
	return vm.ObjectMeta.Name
}

func (c *VMController) processFailure(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, createErr error) {
	reason := ""
	message := ""
	log.Log.Object(vm).V(4).Infof("Processing failure status:: shouldRun: %t; noErr: %t; noVm: %t", vm.Spec.Running, createErr != nil, vmi != nil)

	if createErr != nil {
		if vm.Spec.Running == true {
			reason = "FailedCreate"
		} else {
			reason = "FailedDelete"
		}
		message = createErr.Error()

		if !c.hasCondition(vm, virtv1.VirtualMachineFailure) {
			log.Log.Object(vm).Infof("Reason to fail: %s", reason)
			vm.Status.Conditions = append(vm.Status.Conditions, virtv1.VirtualMachineCondition{
				Type:               virtv1.VirtualMachineFailure,
				Reason:             reason,
				Message:            message,
				LastTransitionTime: v1.Now(),
				Status:             k8score.ConditionTrue,
			})
		}

		return
	}

	log.Log.Object(vm).V(4).Info("Removing failure")
	c.removeCondition(vm, virtv1.VirtualMachineFailure)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *VMController) resolveControllerRef(namespace string, controllerRef *v1.OwnerReference) *virtv1.VirtualMachine {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return nil
	}
	vm, exists, err := c.vmiVMInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if vm.(*virtv1.VirtualMachine).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return vm.(*virtv1.VirtualMachine)
}
