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
	"sync"
	"time"

	"github.com/pborman/uuid"
	authv1 "k8s.io/api/authorization/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	cdiclone "kubevirt.io/containerized-data-importer/pkg/clone"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/status"
)

type CloneAuthFunc func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error)

// Repeating info / error messages
const (
	stoppingVmiMsg                        = "Stopping VMI"
	startingVmiMsg                        = "Starting VMI"
	failedExtractVmkeyFromVmErrMsg        = "Failed to extract vmKey from VirtualMachine."
	failedProcessDeleteNotificationErrMsg = "Failed to process delete notification"
	failureDeletingVmiErrFormat           = "Failure attempting to delete VMI: %v"
)

func NewVMController(vmiInformer cache.SharedIndexInformer,
	vmiVMInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *VMController {

	proxy := &sarProxy{client: clientset}

	c := &VMController{
		Queue:                  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmiInformer:            vmiInformer,
		vmiVMInformer:          vmiVMInformer,
		dataVolumeInformer:     dataVolumeInformer,
		pvcInformer:            pvcInformer,
		recorder:               recorder,
		clientset:              clientset,
		expectations:           controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		dataVolumeExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		cloneAuthFunc: func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
			return cdiclone.CanServiceAccountClonePVC(proxy, pvcNamespace, pvcName, saNamespace, saName)
		},
		statusUpdater: status.NewVMStatusUpdater(clientset),
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

type sarProxy struct {
	client kubecli.KubevirtClient
}

func (p *sarProxy) Create(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
	return p.client.AuthorizationV1().SubjectAccessReviews().Create(sar)
}

type VMController struct {
	clientset              kubecli.KubevirtClient
	Queue                  workqueue.RateLimitingInterface
	vmiInformer            cache.SharedIndexInformer
	vmiVMInformer          cache.SharedIndexInformer
	dataVolumeInformer     cache.SharedIndexInformer
	pvcInformer            cache.SharedIndexInformer
	recorder               record.EventRecorder
	expectations           *controller.UIDTrackingControllerExpectations
	dataVolumeExpectations *controller.UIDTrackingControllerExpectations
	cloneAuthFunc          CloneAuthFunc
	statusUpdater          *status.VMStatusUpdater
}

func (c *VMController) Run(threadiness int, stopCh <-chan struct{}) {
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

func (c *VMController) needsSync(key string) bool {
	return c.expectations.SatisfiedExpectations(key) && c.dataVolumeExpectations.SatisfiedExpectations(key)
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
	vm := obj.(*virtv1.VirtualMachine)

	logger := log.Log.Object(vm)

	logger.V(4).Info("Started processing vm")

	// this must be first step in execution. Writing the object
	// when api version changes ensures our api stored version is updated.
	if !controller.ObservedLatestApiVersionAnnotation(vm) {
		vm := vm.DeepCopy()
		controller.SetLatestApiVersionAnnotation(vm)
		_, err = c.clientset.VirtualMachine(vm.Namespace).Update(vm)

		if err != nil {
			logger.Reason(err).Error("Updating api version annotations failed")
		}

		return err
	}

	//TODO default vm if necessary, the aggregated apiserver will do that in the future
	if vm.Spec.Template == nil {
		logger.Error("Invalid controller spec, will not re-enqueue.")
		return nil
	}

	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (v1.Object, error) {
		fresh, err := c.clientset.VirtualMachine(vm.ObjectMeta.Namespace).Get(vm.ObjectMeta.Name, &v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.ObjectMeta.UID != vm.ObjectMeta.UID {
			return nil, fmt.Errorf("original VirtualMachine %v/%v is gone: got uid %v, wanted %v", vm.Namespace, vm.Name, fresh.UID, vm.UID)
		}
		return fresh, nil
	})
	cm := controller.NewVirtualMachineControllerRefManager(
		controller.RealVirtualMachineControl{
			Clientset: c.clientset,
		}, vm, nil, virtv1.VirtualMachineGroupVersionKind, canAdoptFunc)

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

	dataVolumes, err := c.listDataVolumesForVM(vm)
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
	if c.needsSync(key) && vm.ObjectMeta.DeletionTimestamp == nil {

		dataVolumesReady, err := c.handleDataVolumes(vm, dataVolumes)
		if err != nil {
			createErr = err
		} else if dataVolumesReady == true {
			createErr = c.startStop(vm, vmi)
		} else {
			log.Log.Object(vm).V(3).Infof("Waiting on DataVolumes to be ready. %d datavolumes found", len(dataVolumes))
		}

		// Must check needsSync again here because a VMI can be created or
		// deleted in the startStop function which impacts how we process
		// hotplugged volumes
		if c.needsSync(key) && createErr == nil {

			createErr = c.handleVolumeRequests(vm, vmi)
		}
	}

	// If the controller is going to be deleted and the orphan finalizer is the next one, release the VMIs. Don't update the status
	// TODO: Workaround for https://github.com/kubernetes/kubernetes/issues/56348, remove it once it is fixed
	if vm.ObjectMeta.DeletionTimestamp != nil && controller.HasFinalizer(vm, v1.FinalizerOrphanDependents) {
		err = c.orphan(cm, vmi)
		if err != nil {
			return err
		}
		return c.orphanDataVolumes(cm, dataVolumes)
	}

	if createErr != nil {
		logger.Reason(err).Error("Creating the VirtualMachine failed.")
	}

	err = c.updateStatus(vm, vmi, createErr)
	if err != nil {
		logger.Reason(err).Error("Updating the VirtualMachine status failed.")
		return err
	}

	if createErr != nil {
		return createErr
	}

	return nil
}

// Handles VM rename requests
// First return value is a boolean indicating if the controller should retry the request
func (c *VMController) handleVMRenameRequest(vm *virtv1.VirtualMachine, newName string) (bool, error) {
	err := c.clientset.VirtualMachine(vm.Namespace).Delete(newName, &v1.DeleteOptions{})

	if err != nil && !errors.IsNotFound(err) {
		// VM existence could not be determined, retry
		return true, err
	}

	// Create the copy of this VM with the new name
	newVM := vm.DeepCopy()

	newVM.ResourceVersion = ""
	newVM.Name = newName

	// Update the VM label if it exists
	if newVM.Labels != nil {
		_, hasVMLabel := newVM.Labels[virtv1.VirtualMachineLabel]

		if hasVMLabel {
			newVM.Labels[virtv1.VirtualMachineLabel] = newName
		}
	}

	// Update the VMI spec VM label if it exists
	if newVM.Spec.Template.ObjectMeta.Labels != nil {
		_, hasVMLabel := newVM.Spec.Template.ObjectMeta.Labels[virtv1.VirtualMachineLabel]

		if hasVMLabel {
			newVM.Spec.Template.ObjectMeta.Labels[virtv1.VirtualMachineLabel] = newName
		}
	}

	// Clear VM status
	newVM.Status = virtv1.VirtualMachineStatus{}

	// Add a condition to the new VM to tell the user it was renamed
	newVM.Status.Conditions = []virtv1.VirtualMachineCondition{
		{
			Type:    virtv1.RenameConditionType,
			Status:  k8score.ConditionTrue,
			Reason:  vm.Name,
			Message: fmt.Sprintf("This VM was renamed, the old name was %s", vm.Name),
		},
	}

	// Attempt creation of the new VM
	_, err = c.clientset.VirtualMachine(vm.Namespace).Create(newVM)

	if err != nil {
		return true, err
	}

	// Delete this VM because a copy of it with the desired new name was created
	err = c.clientset.VirtualMachine(vm.Namespace).Delete(vm.Name, &v1.DeleteOptions{})

	if err != nil {
		return true, err
	}

	return false, nil
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

func createDataVolumeManifest(dataVolumeTemplate *virtv1.DataVolumeTemplateSpec, vm *virtv1.VirtualMachine) *cdiv1.DataVolume {

	newDataVolume := &cdiv1.DataVolume{}

	newDataVolume.Spec = *dataVolumeTemplate.Spec.DeepCopy()
	newDataVolume.ObjectMeta = *dataVolumeTemplate.ObjectMeta.DeepCopy()

	labels := map[string]string{}
	annotations := map[string]string{}

	labels[virtv1.CreatedByLabel] = string(vm.UID)

	for k, v := range dataVolumeTemplate.Annotations {
		annotations[k] = v
	}
	for k, v := range dataVolumeTemplate.Labels {
		labels[k] = v
	}
	newDataVolume.ObjectMeta.Labels = labels
	newDataVolume.ObjectMeta.Annotations = annotations

	newDataVolume.ObjectMeta.OwnerReferences = []v1.OwnerReference{
		*v1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}
	return newDataVolume
}

func (c *VMController) authorizeDataVolume(vm *virtv1.VirtualMachine, dataVolume *cdiv1.DataVolume) error {
	if dataVolume.Spec.Source.PVC == nil {
		return nil
	}

	pvcNamespace := dataVolume.Spec.Source.PVC.Namespace
	if pvcNamespace == "" {
		pvcNamespace = vm.Namespace
	}

	pvcName := dataVolume.Spec.Source.PVC.Name

	serviceAccount := "default"
	for _, vol := range vm.Spec.Template.Spec.Volumes {
		if vol.ServiceAccount != nil {
			serviceAccount = vol.ServiceAccount.ServiceAccountName
		}
	}

	allowed, reason, err := c.cloneAuthFunc(pvcNamespace, pvcName, vm.Namespace, serviceAccount)
	if err != nil {
		return err
	}

	if !allowed {
		return fmt.Errorf(reason)
	}

	return nil
}

func (c *VMController) handleDataVolumes(vm *virtv1.VirtualMachine, dataVolumes []*cdiv1.DataVolume) (bool, error) {
	ready := true
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return ready, err
	}
	for i, template := range vm.Spec.DataVolumeTemplates {
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
			newDataVolume := createDataVolumeManifest(&vm.Spec.DataVolumeTemplates[i], vm)

			if err = c.authorizeDataVolume(vm, newDataVolume); err != nil {
				c.recorder.Eventf(vm, k8score.EventTypeWarning, UnauthorizedDataVolumeCreateReason, "Not authorized to create DataVolume %s: %v", newDataVolume.Name, err)
				return ready, fmt.Errorf("Not authorized to create DataVolume: %v", err)
			}

			c.dataVolumeExpectations.ExpectCreations(vmKey, 1)
			curDataVolume, err = c.clientset.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Create(newDataVolume)
			if err != nil {
				c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDataVolumeCreateReason, "Error creating DataVolume %s: %v", newDataVolume.Name, err)
				c.dataVolumeExpectations.CreationObserved(vmKey)
				return ready, fmt.Errorf("Failed to create DataVolume: %v", err)
			}
			c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDataVolumeCreateReason, "Created DataVolume %s", curDataVolume.Name)
		} else if curDataVolume.Status.Phase != cdiv1.Succeeded && curDataVolume.Status.Phase != cdiv1.WaitForFirstConsumer {
			// ready = false because encountered DataVolume that is not populated yet
			ready = false
			if curDataVolume.Status.Phase == cdiv1.Failed {
				c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDataVolumeImportReason, "DataVolume %s failed to import disk image", curDataVolume.Name)
			}
		}
	}
	return ready, nil
}

func (c *VMController) handleVolumeRequests(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if len(vm.Status.VolumeRequests) == 0 {
		return nil
	}

	vmCopy := vm.DeepCopy()
	vmiVolumeMap := make(map[string]virtv1.Volume)
	if vmi != nil {
		for _, volume := range vmi.Spec.Volumes {
			vmiVolumeMap[volume.Name] = volume
		}
	}

	for i, request := range vm.Status.VolumeRequests {
		vmCopy.Spec.Template.Spec = *controller.ApplyVolumeRequestOnVMISpec(&vmCopy.Spec.Template.Spec, &vm.Status.VolumeRequests[i])

		if vmi != nil && vmi.DeletionTimestamp == nil {
			if request.AddVolumeOptions != nil {
				_, exists := vmiVolumeMap[request.AddVolumeOptions.Name]
				if !exists {
					err := c.clientset.VirtualMachineInstance(vmi.Namespace).AddVolume(vmi.Name, request.AddVolumeOptions)
					if err != nil {
						return err
					}
				}
			} else if request.RemoveVolumeOptions != nil {
				_, exists := vmiVolumeMap[request.RemoveVolumeOptions.Name]
				if exists {
					err := c.clientset.VirtualMachineInstance(vmi.Namespace).RemoveVolume(vmi.Name, request.RemoveVolumeOptions)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	if !reflect.DeepEqual(vm, vmCopy) {
		_, err := c.clientset.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VMController) startStop(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		log.Log.Object(vm).Errorf("Error fetching RunStrategy: %v", err)
		return err
	}
	log.Log.Object(vm).V(4).Infof("VirtualMachine RunStrategy: %s", runStrategy)

	switch runStrategy {
	case virtv1.RunStrategyAlways:
		// For this RunStrategy, a VMI should always be running. If a StateChangeRequest
		// asks to stop a VMI, a new one must be immediately re-started.
		if vmi != nil {
			forceRestart := false
			if len(vm.Status.StateChangeRequests) != 0 {
				stateChange := vm.Status.StateChangeRequests[0]
				if stateChange.Action == virtv1.StopRequest &&
					stateChange.UID != nil &&
					*stateChange.UID == vmi.UID {
					log.Log.Object(vm).V(4).Info("VMI should be restarted")
					forceRestart = true
				}
			}

			if forceRestart || vmi.IsFinal() {
				// The VirtualMachineInstance can fail or be finished. The job of this controller
				// is keep the VirtualMachineInstance running, therefore it restarts it.
				// restarting VirtualMachineInstance by stopping it and letting it start in next step
				log.Log.Object(vm).V(4).Info(stoppingVmiMsg)
				err := c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return err
				}
				// return to let the controller pick up the expected deletion
			}
			// VirtualMachineInstance is OK no need to do anything
			return nil
		}

		log.Log.Object(vm).V(4).Info(startingVmiMsg)
		err := c.startVMI(vm)
		if err != nil {
			return err
		}
		return nil

	case virtv1.RunStrategyRerunOnFailure:
		// For this RunStrategy, a VMI should only be restarted if it failed.
		// If a VMI enters the Succeeded phase, it should not be restarted.
		if vmi != nil {
			forceStop := false
			// If there's a stop request that matches the existing VMI's UUID
			if len(vm.Status.StateChangeRequests) != 0 {
				stateChange := vm.Status.StateChangeRequests[0]
				if stateChange.Action == virtv1.StopRequest &&
					stateChange.UID != nil &&
					*stateChange.UID == vmi.UID {
					log.Log.Object(vm).V(4).Info("VMI should be stopped")
					forceStop = true
				}
			}

			if forceStop || vmi.Status.Phase == virtv1.Failed {
				// For RerunOnFailure, this controller should only restart the VirtualMachineInstance
				// if it failed.
				log.Log.Object(vm).V(4).Info(stoppingVmiMsg)
				err := c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return err
				}
				// return to let the controller pick up the expected deletion
			}
			// VirtualMachineInstance is OK no need to do anything
			return nil
		}

		log.Log.Object(vm).V(4).Info(startingVmiMsg)
		err := c.startVMI(vm)
		if err != nil {
			return err
		}
		return nil

	case virtv1.RunStrategyManual:
		// For this RunStrategy, VMI's will be started/stopped/restarted using api endpoints only
		if vmi != nil {
			log.Log.Object(vm).V(4).Info("VMI exists")
			forceStop := false
			if len(vm.Status.StateChangeRequests) != 0 {
				stateChange := vm.Status.StateChangeRequests[0]
				if stateChange.Action == virtv1.StopRequest &&
					stateChange.UID != nil &&
					*stateChange.UID == vmi.UID {
					log.Log.Object(vm).V(4).Info("VMI should be stopped")
					forceStop = true
				}
			}
			if forceStop {
				log.Log.Object(vm).V(4).Info(stoppingVmiMsg)
				err := c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return err
				}
				// return to let the controller pick up the expected deletion
				return nil
			}
		} else {
			forceStart := false
			if len(vm.Status.StateChangeRequests) != 0 {
				stateChange := vm.Status.StateChangeRequests[0]
				if stateChange.Action == virtv1.StartRequest {
					log.Log.Object(vm).V(4).Info("VMI should be started")
					forceStart = true
				}
			}
			if forceStart {
				log.Log.Object(vm).V(4).Info(startingVmiMsg)
				err := c.startVMI(vm)
				if err != nil {
					return err
				}
			}
		}
		return nil

	case virtv1.RunStrategyHalted:
		// For this runStrategy, no VMI should be running under any circumstances.
		log.Log.Object(vm).V(4).Info("VMI should be deleted")
		if vmi == nil {
			return nil
		}
		err := c.stopVMI(vm, vmi)
		return err
	default:
		return fmt.Errorf("unknown runstrategy: %s", runStrategy)
	}
}

func (c *VMController) startVMI(vm *virtv1.VirtualMachine) error {
	// TODO add check for existence
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedExtractVmkeyFromVmErrMsg)
		return nil
	}

	// start it
	vmi := c.setupVMIFromVM(vm)

	c.expectations.ExpectCreations(vmKey, 1)
	vmi, err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Create(vmi)
	if err != nil {
		log.Log.Object(vm).Infof("Failed to create VirtualMachineInstance: %s/%s", vmi.Namespace, vmi.Name)
		c.expectations.CreationObserved(vmKey)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine instance: %v", err)
		return err
	}
	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Started the virtual machine by creating the new virtual machine instance %v", vmi.ObjectMeta.Name)

	return nil
}

func (c *VMController) stopVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		// nothing to do
		return nil
	}

	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedExtractVmkeyFromVmErrMsg)
		return nil
	}

	// stop it
	c.expectations.ExpectDeletions(vmKey, []string{controller.VirtualMachineKey(vmi)})
	err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Delete(vmi.ObjectMeta.Name, &v1.DeleteOptions{})

	// Don't log an error if it is already deleted
	if err != nil {
		// We can't observe a delete if it was not accepted by the server
		c.expectations.DeletionObserved(vmKey, controller.VirtualMachineKey(vmi))
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting virtual machine instance %s: %v", vmi.ObjectMeta.Name, err)
		return err
	}

	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Stopped the virtual machine by deleting the virtual machine instance %v", vmi.ObjectMeta.UID)
	log.Log.Object(vm).Info("Dispatching delete event")

	return nil
}

// setupVMIfromVM creates a VirtualMachineInstance object from one VirtualMachine object.
func (c *VMController) setupVMIFromVM(vm *virtv1.VirtualMachine) *virtv1.VirtualMachineInstance {

	vmi := virtv1.NewVMIReferenceFromNameWithNS(vm.ObjectMeta.Namespace, "")
	vmi.ObjectMeta = vm.Spec.Template.ObjectMeta
	vmi.ObjectMeta.Name = vm.ObjectMeta.Name
	vmi.ObjectMeta.GenerateName = ""
	vmi.ObjectMeta.Namespace = vm.ObjectMeta.Namespace
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
			// not managed by us
			log.Log.Object(vmi).V(4).Infof("Cant find the matching VM for VirtualMachineInstance: %s", vmi.Name)
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
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error(failedProcessDeleteNotificationErrMsg)
			return
		}
		vmi, ok = tombstone.Obj.(*virtv1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vmi %#v", obj)).Error(failedProcessDeleteNotificationErrMsg)
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
			log.Log.Reason(fmt.Errorf("couldn't get object from tombstone %+v", obj)).Error(failedProcessDeleteNotificationErrMsg)
			return
		}
		dataVolume, ok = tombstone.Obj.(*cdiv1.DataVolume)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a dataVolume %#v", obj)).Error(failedProcessDeleteNotificationErrMsg)
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
		logger.Object(vm).Reason(err).Error(failedExtractVmkeyFromVmErrMsg)
	}
	c.Queue.Add(key)
}

func (c *VMController) updateStatus(vmOrig *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, createErr error) error {
	vm := vmOrig.DeepCopy()

	created := vmi != nil
	vm.Status.Created = created

	ready := false
	if created {
		ready = controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceConditionType(k8score.PodReady), k8score.ConditionTrue)
	}
	vm.Status.Ready = ready

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		log.Log.Object(vm).Errorf("Error getting RunStrategy: %v", err)
	}
	clearChangeRequest := false
	vmRenamedAndDeleted := false
	if len(vm.Status.StateChangeRequests) != 0 {
		// Only consider one stateChangeRequest at a time. The second and subsequent change
		// requests have not been acted upon by this controller yet!
		stateChange := vm.Status.StateChangeRequests[0]
		switch stateChange.Action {
		case virtv1.StopRequest:
			if vmi == nil {
				// because either the VM or VMI informers can trigger processing here
				// double check the state of the cluster before taking action
				_, err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Get(vm.GetName(), &v1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					// If there's no VMI, then the VMI was stopped, and the stopRequest can be cleared
					log.Log.Object(vm).V(4).Infof("No VMI. Clearing stop request")
					clearChangeRequest = true
				}
			} else {
				if stateChange.UID == nil {
					// It never makes sense to have a request to stop a VMI that doesn't
					// have a UUID associated with it. This shouldn't be possible -- but if
					// it occurs, clear the stopRequest because it can't be acted upon
					log.Log.Object(vm).Errorf("Stop Request has no UID.")
					clearChangeRequest = true
				} else if *stateChange.UID != vmi.UID {
					// If there is a VMI, but the UID doesn't match, then it
					// must have been previously stopped, so the stopRequest can be cleared
					log.Log.Object(vm).V(4).Infof("VMI's UID doesn't match. clearing stop request")
					clearChangeRequest = true
				}
			}
		case virtv1.StartRequest:
			// If the current VMI is running, then it has been started.
			if vmi != nil {
				log.Log.Object(vm).V(4).Infof("VMI exists. clearing start request")
				clearChangeRequest = true
			}
			// It never makes sense to start a VM with RunStrategy Halted -- This shouldn't be
			// possible -- but if it occurs, clear the request, because it can't be acted upon.
			if runStrategy == virtv1.RunStrategyHalted {
				log.Log.Object(vm).Errorf("Start request shouldn't be honored for RunStrategyHalted.")
				clearChangeRequest = true
			}
		case virtv1.RenameRequest:
			newName, hasNewName := stateChange.Data["newName"]

			if !hasNewName {
				log.Log.Object(vm).V(4).Errorf("Rename request is missing 'newName' field")
				clearChangeRequest = true
			} else {
				retry, err := c.handleVMRenameRequest(vm, newName)

				if err != nil {
					log.Log.Object(vm).V(4).
						Errorf("Rename request for vm %s failed: %v", vm.Name, err)
				} else {
					vmRenamedAndDeleted = !retry
				}
			}
		}
	}

	if len(vm.Status.VolumeRequests) > 0 {
		volumeMap := make(map[string]virtv1.Volume)
		diskMap := make(map[string]virtv1.Disk)

		for _, volume := range vm.Spec.Template.Spec.Volumes {
			volumeMap[volume.Name] = volume
		}
		for _, disk := range vm.Spec.Template.Spec.Domain.Devices.Disks {
			diskMap[disk.Name] = disk
		}

		tmpVolRequests := vm.Status.VolumeRequests[:0]
		for _, request := range vm.Status.VolumeRequests {

			var added bool
			var volName string

			removeRequest := false

			if request.AddVolumeOptions != nil {
				volName = request.AddVolumeOptions.Name
				added = true
			} else if request.RemoveVolumeOptions != nil {
				volName = request.RemoveVolumeOptions.Name
				added = false
			}

			_, volExists := volumeMap[volName]
			_, diskExists := diskMap[volName]

			if added && volExists && diskExists {
				removeRequest = true
			} else if !added && !volExists && !diskExists {
				removeRequest = true
			}

			if !removeRequest {
				tmpVolRequests = append(tmpVolRequests, request)
			}
		}
		vm.Status.VolumeRequests = tmpVolRequests
	}

	if vmRenamedAndDeleted {
		return nil
	}

	if clearChangeRequest {
		vm.Status.StateChangeRequests = vm.Status.StateChangeRequests[1:]
	}

	c.syncReadyConditionFromVMI(vm, vmi)

	// Add/Remove Failure condition if necessary
	vmCondManager := controller.NewVirtualMachineConditionManager()
	errMatch := (createErr != nil) == vmCondManager.HasCondition(vm, virtv1.VirtualMachineFailure)
	if !(errMatch) {
		c.processFailure(vm, vmi, createErr)
	}

	// Add/Remove Paused condition (VMI paused by user)
	vmiCondManager := controller.NewVirtualMachineInstanceConditionManager()
	if vmiCondManager.HasCondition(vmi, virtv1.VirtualMachineInstancePaused) {
		if !vmCondManager.HasCondition(vm, virtv1.VirtualMachinePaused) {
			log.Log.Object(vm).V(3).Info("Adding paused condition")
			now := v1.NewTime(time.Now())
			vm.Status.Conditions = append(vm.Status.Conditions, virtv1.VirtualMachineCondition{
				Type:               virtv1.VirtualMachinePaused,
				Status:             k8score.ConditionTrue,
				LastProbeTime:      now,
				LastTransitionTime: now,
				Reason:             "PausedByUser",
				Message:            "VMI was paused by user",
			})
		}
	} else if vmCondManager.HasCondition(vm, virtv1.VirtualMachinePaused) {
		log.Log.Object(vm).V(3).Info("Removing paused condition")
		vmCondManager.RemoveCondition(vm, virtv1.VirtualMachinePaused)
	}

	// only update if necessary
	err = nil
	if !reflect.DeepEqual(vm.Status, vmOrig.Status) {
		err = c.statusUpdater.UpdateStatus(vm)
	}

	return err
}

func (c *VMController) syncReadyConditionFromVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
	vmReadyCond := controller.NewVirtualMachineConditionManager().
		GetCondition(vm, virtv1.VirtualMachineReady)
	vmiReadyCond := controller.NewVirtualMachineInstanceConditionManager().
		GetCondition(vmi, virtv1.VirtualMachineInstanceConditionType(k8score.PodReady))

	if vmReadyCond == nil && vmiReadyCond != nil {
		log.Log.Object(vm).V(4).Info("Adding ready condition")
		newCond := virtv1.VirtualMachineCondition{Type: virtv1.VirtualMachineReady}
		copyConditionDetails(vmiReadyCond, &newCond)
		vm.Status.Conditions = append(vm.Status.Conditions, newCond)
	} else if vmReadyCond != nil && vmiReadyCond != nil {
		log.Log.Object(vm).V(4).Info("Updating ready condition")
		copyConditionDetails(vmiReadyCond, vmReadyCond)
	} else if vmReadyCond != nil && vmiReadyCond == nil {
		log.Log.Object(vm).V(4).Info("Removing ready condition")
		controller.NewVirtualMachineConditionManager().RemoveCondition(vm, virtv1.VirtualMachineReady)
	}
}

func copyConditionDetails(source *virtv1.VirtualMachineInstanceCondition, dest *virtv1.VirtualMachineCondition) {
	dest.Status = source.Status
	dest.LastProbeTime = source.LastProbeTime
	dest.LastTransitionTime = source.LastTransitionTime
	dest.Reason = source.Reason
	dest.Message = source.Message
}

func (c *VMController) processFailure(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, createErr error) {
	reason := ""
	message := ""
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		log.Log.Object(vm).Errorf("Error fetching RunStrategy: %v", err)
	}
	log.Log.Object(vm).V(4).Infof("Processing failure status:: runStrategy: %s; noErr: %t; noVm: %t", runStrategy, createErr != nil, vmi != nil)

	vmConditionManager := controller.NewVirtualMachineConditionManager()
	if createErr != nil {
		if (vm.Spec.Running != nil && *vm.Spec.Running == true) || (vm.Spec.RunStrategy != nil && *vm.Spec.RunStrategy != virtv1.RunStrategyHalted) {
			reason = "FailedCreate"
		} else {
			reason = "FailedDelete"
		}
		message = createErr.Error()

		if !vmConditionManager.HasCondition(vm, virtv1.VirtualMachineFailure) {
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
	vmConditionManager.RemoveCondition(vm, virtv1.VirtualMachineFailure)
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
