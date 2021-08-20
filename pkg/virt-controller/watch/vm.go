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
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"time"

	"kubevirt.io/kubevirt/pkg/util/migrations"

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
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	cdiclone "kubevirt.io/containerized-data-importer/pkg/clone"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/status"
	typesutil "kubevirt.io/kubevirt/pkg/util/types"
)

type CloneAuthFunc func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error)

// Repeating info / error messages
const (
	stoppingVmMsg                         = "Stopping VM"
	startingVmMsg                         = "Starting VM"
	failedExtractVmkeyFromVmErrMsg        = "Failed to extract vmKey from VirtualMachine."
	failedProcessDeleteNotificationErrMsg = "Failed to process delete notification"
	failureDeletingVmiErrFormat           = "Failure attempting to delete VMI: %v"
)

const defaultMaxCrashLoopBackoffDelaySeconds = 300

func NewVMController(vmiInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) *VMController {

	proxy := &sarProxy{client: clientset}

	c := &VMController{
		Queue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-vm"),
		vmiInformer:            vmiInformer,
		vmInformer:             vmInformer,
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

	c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})

	c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
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
	return p.client.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), sar, v1.CreateOptions{})
}

type VMController struct {
	clientset              kubecli.KubevirtClient
	Queue                  workqueue.RateLimitingInterface
	vmiInformer            cache.SharedIndexInformer
	vmInformer             cache.SharedIndexInformer
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
	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced, c.vmInformer.HasSynced, c.dataVolumeInformer.HasSynced)

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

	obj, exists, err := c.vmInformer.GetStore().GetByKey(key)
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
		runStrategy, err := vm.RunStrategy()
		if err != nil {
			return err
		}

		dataVolumesReady, err := c.handleDataVolumes(vm, dataVolumes)
		if err != nil {
			createErr = err
		} else if dataVolumesReady || runStrategy == virtv1.RunStrategyHalted {
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

	if newDataVolume.Spec.PriorityClassName == "" && vm.Spec.Template.Spec.PriorityClassName != "" {
		newDataVolume.Spec.PriorityClassName = vm.Spec.Template.Spec.PriorityClassName
	}
	return newDataVolume
}

func (c *VMController) authorizeDataVolume(vm *virtv1.VirtualMachine, dataVolume *cdiv1.DataVolume) error {
	cloneSource, err := typesutil.GetCloneSource(context.TODO(), c.clientset, vm, &dataVolume.Spec)
	if cloneSource == nil || err != nil {
		return err
	}

	serviceAccount := "default"
	for _, vol := range vm.Spec.Template.Spec.Volumes {
		if vol.ServiceAccount != nil {
			serviceAccount = vol.ServiceAccount.ServiceAccountName
		}
	}

	allowed, reason, err := c.cloneAuthFunc(cloneSource.Namespace, cloneSource.Name, vm.Namespace, serviceAccount)
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
			curDataVolume, err = c.clientset.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), newDataVolume, v1.CreateOptions{})
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

// areDataVolumesReady determines whether all DataVolumes specified for a VM
// have been successfully provisioned, and are ready for consumption.
// Note that DataVolumes in WaitForFirstConsumer phase are not regarded as ready.
func (c *VMController) areDataVolumesReady(vm *virtv1.VirtualMachine) bool {
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.DataVolume == nil {
			continue
		}

		dvKey := fmt.Sprintf("%s/%s", vm.Namespace, volume.DataVolume.Name)
		dvObj, exists, err := c.dataVolumeInformer.GetStore().GetByKey(dvKey)
		if err != nil {
			log.Log.Object(vm).Errorf("Error fetching DataVolume %s: %v", dvKey, err)
			return false
		}
		if !exists {
			return false
		}

		dv := dvObj.(*cdiv1.DataVolume)
		if dv.Status.Phase != cdiv1.Succeeded {
			return false
		}
	}

	return true
}

// arePVCVolumesReady determines whether all PersistentVolumeClaims specified for a VM
// have been successfully provisioned, and are ready for consumption.
func (c *VMController) arePVCVolumesReady(vm *virtv1.VirtualMachine) bool {
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.PersistentVolumeClaim == nil {
			continue
		}

		pvcKey := fmt.Sprintf("%s/%s", vm.Namespace, volume.PersistentVolumeClaim.ClaimName)
		pvcObj, exists, err := c.pvcInformer.GetStore().GetByKey(pvcKey)
		if err != nil {
			log.Log.Object(vm).Errorf("Error fetching PersistentVolumeClaim %s: %v", pvcKey, err)
			return false
		}
		if !exists {
			return false
		}

		pvc := pvcObj.(*k8score.PersistentVolumeClaim)
		if pvc.Status.Phase != k8score.ClaimBound {
			return false
		}
	}

	return true
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

		if vmi == nil || vmi.DeletionTimestamp != nil {
			continue
		}

		if request.AddVolumeOptions != nil {
			if _, exists := vmiVolumeMap[request.AddVolumeOptions.Name]; exists {
				continue
			}

			if err := c.clientset.VirtualMachineInstance(vmi.Namespace).AddVolume(vmi.Name, request.AddVolumeOptions); err != nil {
				return err
			}
		} else if request.RemoveVolumeOptions != nil {
			if _, exists := vmiVolumeMap[request.RemoveVolumeOptions.Name]; !exists {
				continue
			}

			if err := c.clientset.VirtualMachineInstance(vmi.Namespace).RemoveVolume(vmi.Name, request.RemoveVolumeOptions); err != nil {
				return err
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
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Errorf("Error fetching vmKey: %v", err)
		return err
	}
	log.Log.Object(vm).V(4).Infof("VirtualMachine RunStrategy: %s", runStrategy)

	isStopRequestForVMI := func(vm *virtv1.VirtualMachine) bool {
		if len(vm.Status.StateChangeRequests) != 0 {
			stateChange := vm.Status.StateChangeRequests[0]
			if stateChange.Action == virtv1.StopRequest &&
				stateChange.UID != nil &&
				*stateChange.UID == vmi.UID {
				return true
			}
		}
		return false
	}

	switch runStrategy {
	case virtv1.RunStrategyAlways:
		// For this RunStrategy, a VMI should always be running. If a StateChangeRequest
		// asks to stop a VMI, a new one must be immediately re-started.
		if vmi != nil {
			var forceRestart bool
			if forceRestart = isStopRequestForVMI(vm); forceRestart {
				log.Log.Object(vm).Infof("processing forced restart request for VMI with phase %s and VM runStrategy: %s", vmi.Status.Phase, runStrategy)
			}

			if forceRestart || vmi.IsFinal() {
				log.Log.Object(vm).Infof("%s with VMI in phase %s and VM runStrategy: %s", stoppingVmMsg, vmi.Status.Phase, runStrategy)

				// The VirtualMachineInstance can fail or be finished. The job of this controller
				// is keep the VirtualMachineInstance running, therefore it restarts it.
				// restarting VirtualMachineInstance by stopping it and letting it start in next step
				log.Log.Object(vm).V(4).Info(stoppingVmMsg)
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

		timeLeft := startFailureBackoffTimeLeft(vm)
		if timeLeft > 0 {
			log.Log.Object(vm).Infof("Delaying start of VM %s with 'runStrategy: %s' due to start failure backoff. Waiting %d more seconds before starting.", startingVmMsg, runStrategy, timeLeft)
			c.Queue.AddAfter(vmKey, time.Duration(timeLeft)*time.Second)
			return nil
		}

		log.Log.Object(vm).Infof("%s due to runStrategy: %s", startingVmMsg, runStrategy)
		err := c.startVMI(vm)
		if err != nil {
			return err
		}
		return nil

	case virtv1.RunStrategyRerunOnFailure:
		// For this RunStrategy, a VMI should only be restarted if it failed.
		// If a VMI enters the Succeeded phase, it should not be restarted.
		if vmi != nil {
			var forceStop bool
			if forceStop = isStopRequestForVMI(vm); forceStop {
				log.Log.Object(vm).Infof("processing stop request for VMI with phase %s and VM runStrategy: %s", vmi.Status.Phase, runStrategy)

			}

			if forceStop || vmi.Status.Phase == virtv1.Failed {
				// For RerunOnFailure, this controller should only restart the VirtualMachineInstance
				// if it failed.
				log.Log.Object(vm).Infof("%s with VMI in phase %s and VM runStrategy: %s", stoppingVmMsg, vmi.Status.Phase, runStrategy)
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

		timeLeft := startFailureBackoffTimeLeft(vm)
		if timeLeft > 0 {
			log.Log.Object(vm).Infof("Delaying start of VM %s with 'runStrategy: %s' due to start failure backoff. Waiting %d more seconds before starting.", startingVmMsg, runStrategy, timeLeft)
			c.Queue.AddAfter(vmKey, time.Duration(timeLeft)*time.Second)
			return nil
		}

		log.Log.Object(vm).Infof("%s due to runStrategy: %s", startingVmMsg, runStrategy)
		err := c.startVMI(vm)
		if err != nil {
			return err
		}
		return nil

	case virtv1.RunStrategyManual:
		// For this RunStrategy, VMI's will be started/stopped/restarted using api endpoints only
		if vmi != nil {
			log.Log.Object(vm).V(4).Info("VMI exists")

			if forceStop := isStopRequestForVMI(vm); forceStop {
				log.Log.Object(vm).Infof("%s with VMI in phase %s due to stop request and VM runStrategy: %s", vmi.Status.Phase, stoppingVmMsg, runStrategy)
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
					log.Log.Object(vm).Infof("%s due to start request and runStrategy: %s", startingVmMsg, runStrategy)
					forceStart = true
				}
			}
			if forceStart {
				err := c.startVMI(vm)
				if err != nil {
					return err
				}
			}
		}
		return nil

	case virtv1.RunStrategyHalted:
		// For this runStrategy, no VMI should be running under any circumstances.
		// Set RunStrategyAlways/running = true if VM has StartRequest(start paused case).
		if vmi == nil {
			if len(vm.Status.StateChangeRequests) != 0 {
				stateChange := vm.Status.StateChangeRequests[0]
				if stateChange.Action == virtv1.StartRequest {
					vmCopy := vm.DeepCopy()
					runStrategy := virtv1.RunStrategyAlways
					running := true

					if vmCopy.Spec.RunStrategy != nil {
						vmCopy.Spec.RunStrategy = &runStrategy
					} else {
						vmCopy.Spec.Running = &running
					}
					_, err := c.clientset.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
					return err
				}
			}
			return nil
		}
		log.Log.Object(vm).Infof("%s with VMI in phase %s due to runStrategy: %s", stoppingVmMsg, vmi.Status.Phase, runStrategy)
		err := c.stopVMI(vm, vmi)
		return err
	default:
		return fmt.Errorf("unknown runstrategy: %s", runStrategy)
	}
}

// isVMIStartExpected determines whether a VMI is expected to be started for this VM.
func (c *VMController) isVMIStartExpected(vm *virtv1.VirtualMachine) bool {
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Errorf("Error fetching vmKey: %v", err)
		return false
	}

	expectations, exists, _ := c.expectations.GetExpectations(vmKey)
	if !exists || expectations == nil {
		return false
	}

	adds, _ := expectations.GetExpectations()
	return adds > 0
}

// isVMIStopExpected determines whether a VMI is expected to be stopped for this VM.
func (c *VMController) isVMIStopExpected(vm *virtv1.VirtualMachine) bool {
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Errorf("Error fetching vmKey: %v", err)
		return false
	}

	expectations, exists, _ := c.expectations.GetExpectations(vmKey)
	if !exists || expectations == nil {
		return false
	}

	_, dels := expectations.GetExpectations()
	return dels > 0
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
	// add a finalizer to ensure the VM controller has a chance to see
	// the VMI before it is deleted
	vmi.Finalizers = append(vmi.Finalizers, virtv1.VirtualMachineControllerFinalizer)

	c.expectations.ExpectCreations(vmKey, 1)
	vmi, err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Create(vmi)
	if err != nil {
		c.expectations.CreationObserved(vmKey)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine instance: %v", err)
		return err
	}
	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Started the virtual machine by creating the new virtual machine instance %v", vmi.ObjectMeta.Name)

	return nil
}

// Returns in seconds how long to wait before trying to start the VM again.
func calculateStartBackoffTime(failCount int, maxDelay int) int {
	// The algorithm is designed to work well with a dynamic maxDelay
	// if we decide to expose this as a tuning in the future.
	minInterval := 10
	delaySeconds := 0

	if failCount <= 0 {
		failCount = 1
	}

	multiplier := int(math.Pow(float64(failCount), float64(2)))
	interval := maxDelay / 30

	if interval < minInterval {
		interval = minInterval
	}

	delaySeconds = (interval * multiplier)
	randomRange := (delaySeconds / 2) + 1
	// add randomized seconds to offset multiple failing VMs from one another
	delaySeconds += rand.Intn(randomRange)

	if delaySeconds > maxDelay {
		delaySeconds = maxDelay
	}

	return delaySeconds
}

// Reports if vmi has ever hit a running state
func wasVMIInRunningPhase(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	for _, ts := range vmi.Status.PhaseTransitionTimestamps {
		if ts.Phase == virtv1.Running {
			return true
		}
	}

	return false
}

// Reports if vmi failed before ever hitting a running state
func vmiFailedEarly(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil || !vmi.IsFinal() {
		return false
	}

	if wasVMIInRunningPhase(vmi) {
		return false
	}

	return true
}

// clear start failure tracking if...
// 1. VMI exists and ever hit running phase
// 2. run strategy is not set to automatically restart failed VMIs
func shouldClearStartFailure(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {

	if wasVMIInRunningPhase(vmi) {
		return true
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		log.Log.Object(vm).Errorf("Error fetching RunStrategy: %v", err)
		return false
	}

	if runStrategy != virtv1.RunStrategyAlways && runStrategy != virtv1.RunStrategyRerunOnFailure {
		return true
	}

	return false
}

func startFailureBackoffTimeLeft(vm *virtv1.VirtualMachine) int64 {

	if vm.Status.StartFailure == nil {
		return 0
	}

	now := time.Now().UTC().Unix()
	retryAfter := vm.Status.StartFailure.RetryAfterTimestamp.Time.UTC().Unix()

	diff := retryAfter - now

	if diff > 0 {
		return diff
	}
	return 0
}

func syncStartFailureStatus(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
	if shouldClearStartFailure(vm, vmi) {
		// if a vmi associated with the vm hits a running phase, then reset the start failure counter
		vm.Status.StartFailure = nil

	} else if vmi != nil && vmiFailedEarly(vmi) {
		// if the VMI failed without ever hitting running successfully,
		// record this as a start failure so we can back off retrying
		if vm.Status.StartFailure != nil && vm.Status.StartFailure.LastFailedVMIUID == vmi.UID {
			// already counted this failure
			return
		}
		count := 1

		if vm.Status.StartFailure != nil {
			count = vm.Status.StartFailure.ConsecutiveFailCount + 1
		}

		now := v1.NewTime(time.Now())
		delaySeconds := calculateStartBackoffTime(count, defaultMaxCrashLoopBackoffDelaySeconds)
		retryAfter := v1.NewTime(now.Time.Add(time.Duration(int64(delaySeconds)) * time.Second))

		vm.Status.StartFailure = &virtv1.VirtualMachineStartFailure{
			LastFailedVMIUID:     vmi.UID,
			RetryAfterTimestamp:  &retryAfter,
			ConsecutiveFailCount: count,
		}
	}
}

// here is stop
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
	c.expectations.ExpectDeletions(vmKey, []string{controller.VirtualMachineInstanceKey(vmi)})
	err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Delete(vmi.ObjectMeta.Name, &v1.DeleteOptions{})

	// Don't log an error if it is already deleted
	if err != nil {
		// We can't observe a delete if it was not accepted by the server
		c.expectations.DeletionObserved(vmKey, controller.VirtualMachineInstanceKey(vmi))
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting virtual machine instance %s: %v", vmi.ObjectMeta.Name, err)
		return err
	}

	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Stopping the virtual machine by deleting the virtual machine instance %v", vmi.ObjectMeta.UID)

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

	if hasStartPausedRequest(vm) {
		strategy := virtv1.StartStrategyPaused
		vmi.Spec.StartStrategy = &strategy
	}

	setupStableFirmwareUUID(vm, vmi)

	// TODO check if vmi labels exist, and when make sure that they match. For now just override them
	vmi.ObjectMeta.Labels = vm.Spec.Template.ObjectMeta.Labels
	vmi.ObjectMeta.OwnerReferences = []v1.OwnerReference{
		*v1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}

	return vmi
}

func hasStartPausedRequest(vm *virtv1.VirtualMachine) bool {
	if len(vm.Status.StateChangeRequests) != 0 {
		stateChange := vm.Status.StateChangeRequests[0]
		if stateChange.Action == virtv1.StartRequest {
			paused, hasPaused := stateChange.Data[virtv1.StartRequestDataPausedKey]
			if hasPaused && paused == virtv1.StartRequestDataPausedTrue {
				return true
			}
		}
	}
	return false
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
	objs, err := c.vmInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
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
func (c *VMController) addVirtualMachineInstance(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)

	log.Log.Object(vmi).V(4).Info("VirtualMachineInstance added.")

	if vmi.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new vmi shows up in a state that
		// is already pending deletion. Prevent the vmi from being a creation observation.
		c.deleteVirtualMachineInstance(vmi)
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
func (c *VMController) updateVirtualMachineInstance(old, cur interface{}) {
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
		c.deleteVirtualMachineInstance(curVMI)
		if labelChanged {
			// we don't need to check the oldVMI.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deleteVirtualMachineInstance(oldVMI)
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

	isOrphan := !labelChanged && !controllerRefChanged
	if isOrphan {
		return
	}

	// If anything changed, sync matching controllers to see if anyone wants to adopt it now.
	vms := c.getMatchingControllers(curVMI)
	if len(vms) == 0 {
		return
	}
	log.Log.V(4).Object(curVMI).Infof("Orphan VirtualMachineInstance updated")
	for _, vm := range vms {
		c.enqueueVm(vm)
	}
}

// When a vmi is deleted, enqueue the VirtualMachine that manages the vmi and update its expectations.
// obj could be an *v1.VirtualMachineInstance, or a DeletionFinalStateUnknown marker item.
func (c *VMController) deleteVirtualMachineInstance(obj interface{}) {
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
	c.expectations.DeletionObserved(vmKey, controller.VirtualMachineInstanceKey(vmi))
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

func (c *VMController) addVirtualMachine(obj interface{}) {
	c.enqueueVm(obj)
}

func (c *VMController) deleteVirtualMachine(obj interface{}) {
	c.enqueueVm(obj)
}

func (c *VMController) updateVirtualMachine(_, curr interface{}) {
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

func (c *VMController) removeVMIFinalizer(vmi *virtv1.VirtualMachineInstance) error {
	vmiCopy := vmi.DeepCopy()
	controller.RemoveFinalizer(vmiCopy, virtv1.VirtualMachineControllerFinalizer)

	if reflect.DeepEqual(vmi.Finalizers, vmiCopy.Finalizers) {
		return nil
	}

	log.Log.V(3).Object(vmi).Infof("VMI is in a final state. Removing VM controller finalizer")

	ops := []string{}
	newFinalizers, err := json.Marshal(vmiCopy.Finalizers)
	if err != nil {
		return err
	}
	oldFinalizers, err := json.Marshal(vmi.Finalizers)
	if err != nil {
		return err
	}
	ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/finalizers", "value": %s }`, string(oldFinalizers)))
	ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/finalizers", "value": %s }`, string(newFinalizers)))
	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(vmi.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops))
	return err
}

func (c *VMController) updateStatus(vmOrig *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, createErr error) error {
	vm := vmOrig.DeepCopy()

	created := vmi != nil
	vm.Status.Created = created

	ready := false
	if created {
		ready = controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceReady, k8score.ConditionTrue)
	}
	vm.Status.Ready = ready

	clearChangeRequest := false
	if len(vm.Status.StateChangeRequests) != 0 {
		// Only consider one stateChangeRequest at a time. The second and subsequent change
		// requests have not been acted upon by this controller yet!
		stateChange := vm.Status.StateChangeRequests[0]
		switch stateChange.Action {
		case virtv1.StopRequest:
			if vmi == nil {
				// because either the VM or VMI informers can trigger processing here
				// double check the state of the cluster before taking action
				_, err := c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Get(vm.GetName(), &v1.GetOptions{})
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

	if clearChangeRequest {
		vm.Status.StateChangeRequests = vm.Status.StateChangeRequests[1:]
	}

	syncStartFailureStatus(vm, vmi)

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

	c.setPrintableStatus(vm, vmi)

	// only update if necessary
	if !reflect.DeepEqual(vm.Status, vmOrig.Status) {
		if err := c.statusUpdater.UpdateStatus(vm); err != nil {
			return err
		}
	}

	if vmi != nil && vmi.IsFinal() && len(vmi.Finalizers) > 0 {
		// Remove our finalizer off of a finalized VMI now that we've been able
		// to record any status info from the VMI onto the VM object.
		err := c.removeVMIFinalizer(vmi)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VMController) setPrintableStatus(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
	// For each status, there's a separate function that evaluates
	// whether the status is "true" for the given VM.
	//
	// Note that these statuses aren't mutually exclusive,
	// and several of them can be "true" at the same time
	// (e.g., Running && Migrating, or Paused && Terminating).
	//
	// The actual precedence of these statuses are determined by the order
	// of evaluation - first match wins.
	statuses := []struct {
		statusType virtv1.VirtualMachinePrintableStatus
		statusFunc func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool
	}{
		{virtv1.VirtualMachineStatusTerminating, c.isVirtualMachineStatusTerminating},
		{virtv1.VirtualMachineStatusStopping, c.isVirtualMachineStatusStopping},
		{virtv1.VirtualMachineStatusMigrating, c.isVirtualMachineStatusMigrating},
		{virtv1.VirtualMachineStatusPaused, c.isVirtualMachineStatusPaused},
		{virtv1.VirtualMachineStatusRunning, c.isVirtualMachineStatusRunning},
		{virtv1.VirtualMachineStatusUnschedulable, c.isVirtualMachineStatusUnschedulable},
		{virtv1.VirtualMachineStatusProvisioning, c.isVirtualMachineStatusProvisioning},
		{virtv1.VirtualMachineStatusErrImagePull, c.isVirtualMachineStatusErrImagePull},
		{virtv1.VirtualMachineStatusImagePullBackOff, c.isVirtualMachineStatusImagePullBackOff},
		{virtv1.VirtualMachineStatusStarting, c.isVirtualMachineStatusStarting},
		{virtv1.VirtualMachineStatusCrashLoopBackOff, c.isVirtualMachineStatusCrashLoopBackOff},
		{virtv1.VirtualMachineStatusStopped, c.isVirtualMachineStatusStopped},
	}

	for _, status := range statuses {
		if status.statusFunc(vm, vmi) {
			vm.Status.PrintableStatus = status.statusType
			return
		}
	}

	vm.Status.PrintableStatus = virtv1.VirtualMachineStatusUnknown
}

// isVirtualMachineStatusCrashLoopBackOff determines whether the VM status field should be set to "CrashLoop".
func (c *VMController) isVirtualMachineStatusCrashLoopBackOff(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi != nil && !vmi.IsFinal() {
		return false
	} else if c.isVMIStartExpected(vm) {
		return false
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		log.Log.Object(vm).Errorf("Error fetching RunStrategy: %v", err)
		return false
	}

	if vm.Status.StartFailure != nil &&
		vm.Status.StartFailure.ConsecutiveFailCount > 0 &&
		(runStrategy == virtv1.RunStrategyAlways || runStrategy == virtv1.RunStrategyRerunOnFailure) {
		return true
	}

	return false
}

// isVirtualMachineStatusStopped determines whether the VM status field should be set to "Stopped".
func (c *VMController) isVirtualMachineStatusStopped(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi != nil {
		return vmi.IsFinal()
	}

	return !c.isVMIStartExpected(vm)
}

// isVirtualMachineStatusStopped determines whether the VM status field should be set to "Provisioning".
func (c *VMController) isVirtualMachineStatusProvisioning(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return !c.areDataVolumesReady(vm) || !c.arePVCVolumesReady(vm)
}

// isVirtualMachineStatusStarting determines whether the VM status field should be set to "Starting".
func (c *VMController) isVirtualMachineStatusStarting(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return c.isVMIStartExpected(vm)
	}

	return vmi.IsUnprocessed() || vmi.IsScheduling() || vmi.IsScheduled()
}

// isVirtualMachineStatusRunning determines whether the VM status field should be set to "Running".
func (c *VMController) isVirtualMachineStatusRunning(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	hasPausedCondition := controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstancePaused, k8score.ConditionTrue)

	return vmi.IsRunning() && !hasPausedCondition
}

// isVirtualMachineStatusPaused determines whether the VM status field should be set to "Paused".
func (c *VMController) isVirtualMachineStatusPaused(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	hasPausedCondition := controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstancePaused, k8score.ConditionTrue)

	return vmi.IsRunning() && hasPausedCondition
}

// isVirtualMachineStatusPaused determines whether the VM status field should be set to "Stopping".
func (c *VMController) isVirtualMachineStatusStopping(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return vmi != nil && !vmi.IsFinal() &&
		(vmi.IsMarkedForDeletion() || c.isVMIStopExpected(vm))
}

// isVirtualMachineStatusPaused determines whether the VM status field should be set to "Terminating".
func (c *VMController) isVirtualMachineStatusTerminating(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return vm.ObjectMeta.DeletionTimestamp != nil
}

// isVirtualMachineStatusPaused determines whether the VM status field should be set to "Migrating".
func (c *VMController) isVirtualMachineStatusMigrating(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return vmi != nil && migrations.IsMigrating(vmi)
}

// isVirtualMachineStatusUnschedulable determines whether the VM status field should be set to "FailedUnschedulable".
func (c *VMController) isVirtualMachineStatusUnschedulable(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatusAndReason(vmi,
		virtv1.VirtualMachineInstanceConditionType(k8score.PodScheduled),
		k8score.ConditionFalse,
		k8score.PodReasonUnschedulable)
}

// isVirtualMachineStatusErrImagePull determines whether the VM status field should be set to "ErrImagePull"
func (c *VMController) isVirtualMachineStatusErrImagePull(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	syncCond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, virtv1.VirtualMachineInstanceSynchronized)
	return syncCond != nil && syncCond.Status == k8score.ConditionFalse && syncCond.Reason == ErrImagePullReason
}

// isVirtualMachineStatusImagePullBackOff determines whether the VM status field should be set to "ImagePullBackOff"
func (c *VMController) isVirtualMachineStatusImagePullBackOff(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	syncCond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, virtv1.VirtualMachineInstanceSynchronized)
	return syncCond != nil && syncCond.Status == k8score.ConditionFalse && syncCond.Reason == ImagePullBackOffReason
}

func (c *VMController) syncReadyConditionFromVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
	conditionManager := controller.NewVirtualMachineConditionManager()
	vmiReadyCond := controller.NewVirtualMachineInstanceConditionManager().
		GetCondition(vmi, virtv1.VirtualMachineInstanceReady)

	now := v1.Now()
	if vmi == nil {
		conditionManager.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
			Type:               virtv1.VirtualMachineReady,
			Status:             k8score.ConditionFalse,
			Reason:             "VMINotExists",
			Message:            "VMI does not exist",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})

	} else if vmiReadyCond == nil {
		conditionManager.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
			Type:               virtv1.VirtualMachineReady,
			Status:             k8score.ConditionFalse,
			Reason:             "VMIConditionMissing",
			Message:            "VMI is missing the Ready condition",
			LastProbeTime:      now,
			LastTransitionTime: now,
		})

	} else {
		conditionManager.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
			Type:               virtv1.VirtualMachineReady,
			Status:             vmiReadyCond.Status,
			Reason:             vmiReadyCond.Reason,
			Message:            vmiReadyCond.Message,
			LastProbeTime:      vmiReadyCond.LastProbeTime,
			LastTransitionTime: vmiReadyCond.LastTransitionTime,
		})
	}
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
	vm, exists, err := c.vmInformer.GetStore().GetByKey(namespace + "/" + controllerRef.Name)
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
