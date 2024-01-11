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
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/trace"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/pkg/util/status"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	fetchingRunStrategyErrFmt = "Error fetching RunStrategy: %v"
	fetchingVMKeyErrFmt       = "Error fetching vmKey: %v"
	startingVMIFailureFmt     = "Failure while starting VMI: %v"

	revisionPrefixStart    = "start"
	revisionPrefixLastSeen = "last-seen"
)

type CloneAuthFunc func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error)

// Repeating info / error messages
const (
	stoppingVmMsg                         = "Stopping VM"
	startingVmMsg                         = "Starting VM"
	failedExtractVmkeyFromVmErrMsg        = "Failed to extract vmKey from VirtualMachine."
	failedCreateCRforVmErrMsg             = "Failed to create controller revision for VirtualMachine."
	failedProcessDeleteNotificationErrMsg = "Failed to process delete notification"
	failureDeletingVmiErrFormat           = "Failure attempting to delete VMI: %v"
	failedMemoryDump                      = "Memory dump failed"
	failedCleanupRestartRequired          = "Failed to delete RestartRequired condition or last-seen controller revisions"
	failedGetLastSeenCRforVmErrMsg        = "Failed to get last-seen controller revision"
	failedCreateLastSeenCRforVmErrMsg     = "Failed to create last-seen controller revision"

	// UnauthorizedDataVolumeCreateReason is added in an event when the DataVolume
	// ServiceAccount doesn't have permission to create a DataVolume
	UnauthorizedDataVolumeCreateReason = "UnauthorizedDataVolumeCreate"
	// FailedDataVolumeCreateReason is added in an event when posting a dynamically
	// generated dataVolume to the cluster fails.
	FailedDataVolumeCreateReason = "FailedDataVolumeCreate"
	// SuccessfulDataVolumeCreateReason is added in an event when a dynamically generated
	// dataVolume is successfully created
	SuccessfulDataVolumeCreateReason = "SuccessfulDataVolumeCreate"
	// SourcePVCNotAvailabe is added in an event when the source PVC of a valid
	// clone Datavolume doesn't exist
	SourcePVCNotAvailabe = "SourcePVCNotAvailabe"
)

const (
	HotPlugVolumeErrorReason           = "HotPlugVolumeError"
	HotPlugCPUErrorReason              = "HotPlugCPUError"
	MemoryDumpErrorReason              = "MemoryDumpError"
	FailedUpdateErrorReason            = "FailedUpdateError"
	FailedCreateReason                 = "FailedCreate"
	VMIFailedDeleteReason              = "FailedDelete"
	HotPlugNetworkInterfaceErrorReason = "HotPlugNetworkInterfaceError"
	AffinityChangeErrorReason          = "AffinityChangeError"
	HotPlugMemoryErrorReason           = "HotPlugMemoryError"
)

const defaultMaxCrashLoopBackoffDelaySeconds = 300

func NewVMController(vmiInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer,
	dataSourceInformer cache.SharedIndexInformer,
	namespaceStore cache.Store,
	pvcInformer cache.SharedIndexInformer,
	crInformer cache.SharedIndexInformer,
	podInformer cache.SharedIndexInformer,
	instancetypeMethods instancetype.Methods,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig) (*VMController, error) {

	c := &VMController{
		Queue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-vm"),
		vmiInformer:            vmiInformer,
		vmInformer:             vmInformer,
		dataVolumeInformer:     dataVolumeInformer,
		dataSourceInformer:     dataSourceInformer,
		namespaceStore:         namespaceStore,
		pvcInformer:            pvcInformer,
		crInformer:             crInformer,
		podInformer:            podInformer,
		instancetypeMethods:    instancetypeMethods,
		recorder:               recorder,
		clientset:              clientset,
		expectations:           controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		dataVolumeExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		cloneAuthFunc: func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error) {
			response, err := dv.AuthorizeSA(requestNamespace, requestName, proxy, saNamespace, saName)
			return response.Allowed, response.Reason, err
		},
		statusUpdater: status.NewVMStatusUpdater(clientset),
		clusterConfig: clusterConfig,
	}

	_, err := c.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})
	if err != nil {
		return nil, err
	}

	_, err = c.dataVolumeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addDataVolume,
		DeleteFunc: c.deleteDataVolume,
		UpdateFunc: c.updateDataVolume,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

type authProxy struct {
	client             kubecli.KubevirtClient
	dataSourceInformer cache.SharedIndexInformer
	namespaceStore     cache.Store
}

func (p *authProxy) CreateSar(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
	return p.client.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), sar, v1.CreateOptions{})
}

func (p *authProxy) GetNamespace(name string) (*k8score.Namespace, error) {
	obj, exists, err := p.namespaceStore.GetByKey(name)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("namespace %s does not exist", name)
	}

	ns := obj.(*k8score.Namespace).DeepCopy()
	return ns, nil
}

func (p *authProxy) GetDataSource(namespace, name string) (*cdiv1.DataSource, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	obj, exists, err := p.dataSourceInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("dataSource %s does not exist", key)
	}

	ds := obj.(*cdiv1.DataSource).DeepCopy()
	return ds, nil
}

type VMController struct {
	clientset              kubecli.KubevirtClient
	Queue                  workqueue.RateLimitingInterface
	vmiInformer            cache.SharedIndexInformer
	vmInformer             cache.SharedIndexInformer
	dataVolumeInformer     cache.SharedIndexInformer
	dataSourceInformer     cache.SharedIndexInformer
	namespaceStore         cache.Store
	pvcInformer            cache.SharedIndexInformer
	crInformer             cache.SharedIndexInformer
	podInformer            cache.SharedIndexInformer
	instancetypeMethods    instancetype.Methods
	recorder               record.EventRecorder
	expectations           *controller.UIDTrackingControllerExpectations
	dataVolumeExpectations *controller.UIDTrackingControllerExpectations
	cloneAuthFunc          CloneAuthFunc
	statusUpdater          *status.VMStatusUpdater
	clusterConfig          *virtconfig.ClusterConfig
}

func (c *VMController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachine controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.vmiInformer.HasSynced, c.vmInformer.HasSynced, c.dataVolumeInformer.HasSynced, c.podInformer.HasSynced)

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

var virtControllerVMWorkQueueTracer = &traceUtils.Tracer{Threshold: time.Second}

func (c *VMController) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	virtControllerVMWorkQueueTracer.StartTrace(key.(string), "virt-controller VM workqueue", trace.Field{Key: "Workqueue Key", Value: key})
	defer virtControllerVMWorkQueueTracer.StopTrace(key.(string))

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
		_, err = c.clientset.VirtualMachine(vm.Namespace).Update(context.Background(), vm)

		if err != nil {
			logger.Reason(err).Error("Updating api version annotations failed")
		}

		return err
	}

	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return err
	}

	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing VirtualMachines (see kubernetes/kubernetes#42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (v1.Object, error) {
		fresh, err := c.clientset.VirtualMachine(vm.ObjectMeta.Namespace).Get(context.Background(), vm.ObjectMeta.Name, &v1.GetOptions{})
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

		vmi, err = cm.ClaimVirtualMachineInstanceByName(vmi)
		if err != nil {
			return err
		}
	}

	dataVolumes, err := storagetypes.ListDataVolumesFromTemplates(vm.Namespace, vm.Spec.DataVolumeTemplates, c.dataVolumeInformer)
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

	var syncErr syncError

	vm, syncErr, err = c.sync(vm, vmi, key, dataVolumes)
	if err != nil {
		return err
	}

	if syncErr != nil {
		logger.Reason(syncErr).Error("Reconciling the VirtualMachine failed.")
	}

	err = c.updateStatus(vm, vmi, syncErr, logger)
	if err != nil {
		logger.Reason(err).Error("Updating the VirtualMachine status failed.")
		return err
	}

	return syncErr
}

func (c *VMController) handleCloneDataVolume(vm *virtv1.VirtualMachine, dv *cdiv1.DataVolume) error {
	if dv.Spec.SourceRef != nil {
		return fmt.Errorf("DataVolume sourceRef not supported")
	}

	if dv.Spec.Source == nil {
		return nil
	}

	// For consistency with other k8s objects, we allow creating clone DataVolumes even when the source PVC doesn't exist.
	// This means that a VirtualMachine can be successfully created with volumes that may remain unpopulated until the source PVC is created.
	// For this reason, we check if the source PVC exists and, if not, we trigger an event to let users know of this behavior.
	if dv.Spec.Source.PVC != nil {
		// TODO: a lot of CDI knowledge, maybe an API to check if source exists?
		pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(dv.Spec.Source.PVC.Namespace, dv.Spec.Source.PVC.Name, c.pvcInformer)
		if err != nil {
			return err
		}
		if pvc == nil {
			c.recorder.Eventf(vm, k8score.EventTypeWarning, SourcePVCNotAvailabe, "Source PVC %s not available: Target PVC %s will remain unpopulated until source is created", dv.Spec.Source.PVC.Name, dv.Name)
		}
	}

	if err := c.authorizeDataVolume(vm, dv); err != nil {
		c.recorder.Eventf(vm, k8score.EventTypeWarning, UnauthorizedDataVolumeCreateReason, "Not authorized to create DataVolume %s: %v", dv.Name, err)
		return fmt.Errorf("not authorized to create DataVolume: %v", err)
	}

	return nil
}

func (c *VMController) authorizeDataVolume(vm *virtv1.VirtualMachine, dataVolume *cdiv1.DataVolume) error {
	serviceAccountName := "default"
	for _, vol := range vm.Spec.Template.Spec.Volumes {
		if vol.ServiceAccount != nil {
			serviceAccountName = vol.ServiceAccount.ServiceAccountName
		}
	}

	proxy := &authProxy{client: c.clientset, dataSourceInformer: c.dataSourceInformer, namespaceStore: c.namespaceStore}
	allowed, reason, err := c.cloneAuthFunc(dataVolume, vm.Namespace, dataVolume.Name, proxy, vm.Namespace, serviceAccountName)
	if err != nil && err != cdiv1.ErrNoTokenOkay {
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
			// Don't create DV if PVC already exists
			pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vm.Namespace, template.Name, c.pvcInformer)
			if err != nil {
				return false, err
			}
			if pvc != nil {
				continue
			}

			// ready = false because encountered DataVolume that is not created yet
			ready = false
			newDataVolume, err := watchutil.CreateDataVolumeManifest(c.clientset, template, vm)
			if err != nil {
				return ready, fmt.Errorf("unable to create DataVolume manifest: %v", err)
			}

			// We validate requirements that are exclusive to clone DataVolumes
			if err = c.handleCloneDataVolume(vm, newDataVolume); err != nil {
				return ready, err
			}

			c.dataVolumeExpectations.ExpectCreations(vmKey, 1)
			curDataVolume, err = c.clientset.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), newDataVolume, v1.CreateOptions{})
			if err != nil {
				c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDataVolumeCreateReason, "Error creating DataVolume %s: %v", newDataVolume.Name, err)
				c.dataVolumeExpectations.CreationObserved(vmKey)
				return ready, fmt.Errorf("failed to create DataVolume: %v", err)
			}
			c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDataVolumeCreateReason, "Created DataVolume %s", curDataVolume.Name)
		} else if curDataVolume.Status.Phase != cdiv1.Succeeded &&
			curDataVolume.Status.Phase != cdiv1.WaitForFirstConsumer &&
			curDataVolume.Status.Phase != cdiv1.PendingPopulation {
			// ready = false because encountered DataVolume that is not populated yet
			ready = false
			if curDataVolume.Status.Phase == cdiv1.Failed {
				c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDataVolumeImportReason, "DataVolume %s failed to import disk image", curDataVolume.Name)
			}
		}
	}
	return ready, nil
}

func removeMemoryDumpVolumeFromVMISpec(vmiSpec *virtv1.VirtualMachineInstanceSpec, claimName string) *virtv1.VirtualMachineInstanceSpec {
	newVolumesList := []virtv1.Volume{}
	for _, volume := range vmiSpec.Volumes {
		if volume.Name != claimName {
			newVolumesList = append(newVolumesList, volume)
		}
	}
	vmiSpec.Volumes = newVolumesList
	return vmiSpec
}

func applyMemoryDumpVolumeRequestOnVMISpec(vmiSpec *virtv1.VirtualMachineInstanceSpec, claimName string) *virtv1.VirtualMachineInstanceSpec {
	for _, volume := range vmiSpec.Volumes {
		if volume.Name == claimName {
			return vmiSpec
		}
	}

	memoryDumpVol := &virtv1.MemoryDumpVolumeSource{
		PersistentVolumeClaimVolumeSource: virtv1.PersistentVolumeClaimVolumeSource{
			PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
			Hotpluggable: true,
		},
	}

	newVolume := virtv1.Volume{
		Name: claimName,
	}
	newVolume.VolumeSource.MemoryDump = memoryDumpVol

	vmiSpec.Volumes = append(vmiSpec.Volumes, newVolume)

	return vmiSpec
}

func (c *VMController) generateVMIMemoryDumpVolumePatch(vmi *virtv1.VirtualMachineInstance, request *virtv1.VirtualMachineMemoryDumpRequest, addVolume bool) error {
	patchVerb := "add"
	if len(vmi.Spec.Volumes) > 0 {
		patchVerb = "replace"
	}

	foundRemoveVol := false
	for _, volume := range vmi.Spec.Volumes {
		if request.ClaimName == volume.Name {
			if addVolume {
				return fmt.Errorf("Unable to add volume [%s] because it already exists", volume.Name)
			} else {
				foundRemoveVol = true
			}
		}
	}

	if !foundRemoveVol && !addVolume {
		return fmt.Errorf("Unable to remove volume [%s] because it does not exist", request.ClaimName)
	}

	vmiCopy := vmi.DeepCopy()
	if addVolume {
		vmiCopy.Spec = *applyMemoryDumpVolumeRequestOnVMISpec(&vmiCopy.Spec, request.ClaimName)
	} else {
		vmiCopy.Spec = *removeMemoryDumpVolumeFromVMISpec(&vmiCopy.Spec, request.ClaimName)
	}

	oldJson, err := json.Marshal(vmi.Spec.Volumes)
	if err != nil {
		return err
	}

	newJson, err := json.Marshal(vmiCopy.Spec.Volumes)
	if err != nil {
		return err
	}

	test := fmt.Sprintf(`{ "op": "test", "path": "/spec/volumes", "value": %s}`, string(oldJson))
	update := fmt.Sprintf(`{ "op": "%s", "path": "/spec/volumes", "value": %s}`, patchVerb, string(newJson))
	patch := fmt.Sprintf("[%s, %s]", test, update)

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &v1.PatchOptions{})
	return err
}

func needUpdatePVCMemoryDumpAnnotation(pvc *k8score.PersistentVolumeClaim, request *virtv1.VirtualMachineMemoryDumpRequest) bool {
	if pvc.GetAnnotations() == nil {
		return true
	}
	annotation, hasAnnotation := pvc.Annotations[virtv1.PVCMemoryDumpAnnotation]
	return !hasAnnotation || (request.Phase == virtv1.MemoryDumpUnmounting && annotation != *request.FileName) || (request.Phase == virtv1.MemoryDumpFailed && annotation != failedMemoryDump)
}

func (c *VMController) updatePVCMemoryDumpAnnotation(vm *virtv1.VirtualMachine) error {
	request := vm.Status.MemoryDumpRequest
	pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vm.Namespace, request.ClaimName, c.pvcInformer)
	if err != nil {
		log.Log.Object(vm).Errorf("Error getting PersistentVolumeClaim to update memory dump annotation: %v", err)
		return err
	}
	if pvc == nil {
		log.Log.Object(vm).Errorf("Error getting PersistentVolumeClaim to update memory dump annotation: %v", err)
		return fmt.Errorf("Error when trying to update memory dump annotation, pvc %s not found", request.ClaimName)
	}

	if needUpdatePVCMemoryDumpAnnotation(pvc, request) {
		if pvc.GetAnnotations() == nil {
			pvc.SetAnnotations(make(map[string]string))
		}
		if request.Phase == virtv1.MemoryDumpUnmounting {
			pvc.Annotations[virtv1.PVCMemoryDumpAnnotation] = *request.FileName
		} else if request.Phase == virtv1.MemoryDumpFailed {
			pvc.Annotations[virtv1.PVCMemoryDumpAnnotation] = failedMemoryDump
		}
		if _, err = c.clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(context.Background(), pvc, v1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (c *VMController) VMICPUsPatch(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	test := fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/cpu/sockets", "value": %s}`, strconv.FormatUint(uint64(vmi.Spec.Domain.CPU.Sockets), 10))
	update := fmt.Sprintf(`{ "op": "replace", "path": "/spec/domain/cpu/sockets", "value": %s}`, strconv.FormatUint(uint64(vm.Spec.Template.Spec.Domain.CPU.Sockets), 10))
	patch := fmt.Sprintf("[%s, %s]", test, update)

	_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &v1.PatchOptions{})

	return err
}

func (c *VMController) handleCPUChangeRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	if vm.Spec.Template.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU == nil {
		return nil
	}

	if vm.Spec.Template.Spec.Domain.CPU.Sockets == vmi.Spec.Domain.CPU.Sockets {
		return nil
	}

	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	if vmiConditions.HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceVCPUChange, k8score.ConditionTrue) {
		return fmt.Errorf("another CPU hotplug is in progress")
	}

	if migrations.IsMigrating(vmi) {
		return fmt.Errorf("CPU hotplug is not allowed while VMI is migrating")
	}

	// If the following is true, MaxSockets was calculated, not manually specified (or the validation webhook would have rejected the change).
	// Since we're here, we can also assume MaxSockets was not changed in the VM spec since last boot.
	// Therefore, bumping Sockets to a value higher than MaxSockets is fine, it just requires a reboot.
	if vm.Spec.Template.Spec.Domain.CPU.Sockets > vmi.Spec.Domain.CPU.MaxSockets {
		vmConditions := controller.NewVirtualMachineConditionManager()
		vmConditions.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
			Type:               virtv1.VirtualMachineRestartRequired,
			LastTransitionTime: v1.Now(),
			Status:             k8score.ConditionTrue,
			Message:            "CPU sockets updated in template spec to a value higher than what's available",
		})
		return nil
	}

	if err := c.VMICPUsPatch(vm, vmi); err != nil {
		log.Log.Object(vmi).Errorf("unable to patch vmi to add cpu topology status: %v", err)
		return err
	}

	return nil
}

func (c *VMController) VMNodeSelectorPatch(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	var ops []string

	if vm.Spec.Template.Spec.NodeSelector != nil {
		vmNodeSelector := make(map[string]string)
		// copy the node selector map
		for k, v := range vm.Spec.Template.Spec.NodeSelector {
			vmNodeSelector[k] = v
		}
		vmNodeSelectorJson, err := json.Marshal(vmNodeSelector)
		if err != nil {
			return err
		}

		if vmi.Spec.NodeSelector == nil {
			ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/spec/nodeSelector", "value": %s }`, string(vmNodeSelectorJson)))
		} else {
			currentVMINodeSelector, err := json.Marshal(vmi.Spec.NodeSelector)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/spec/nodeSelector", "value": %s }`, string(currentVMINodeSelector)))
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec/nodeSelector", "value": %s }`, string(vmNodeSelectorJson)))
		}

	} else {
		ops = append(ops, fmt.Sprintf(`{ "op": "remove", "path": "/spec/nodeSelector" }`))
	}
	generatedPatch := controller.GeneratePatchBytes(ops)

	_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, generatedPatch, &v1.PatchOptions{})
	return err
}

func (c *VMController) VMIAffinityPatch(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	var ops []string

	if vm.Spec.Template.Spec.Affinity != nil {
		vmAffinity := vm.Spec.Template.Spec.Affinity.DeepCopy()
		vmAffinityJson, err := json.Marshal(vmAffinity)
		if err != nil {
			return err
		}
		if vmi.Spec.Affinity == nil {
			ops = append(ops, fmt.Sprintf(`{ "op": "add", "path": "/spec/affinity", "value": %s }`, string(vmAffinityJson)))
		} else {
			currentVMIAffinity, err := json.Marshal(vmi.Spec.Affinity)
			if err != nil {
				return err
			}
			ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/spec/affinity", "value": %s }`, string(currentVMIAffinity)))
			ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/spec/affinity", "value": %s }`, string(vmAffinityJson)))
		}

	} else {
		ops = append(ops, fmt.Sprintf(`{ "op": "remove", "path": "/spec/affinity" }`))
	}

	_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops), &v1.PatchOptions{})
	return err
}

func (c *VMController) handleAffinityChangeRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	hasNodeSelectorChanged := !equality.Semantic.DeepEqual(vm.Spec.Template.Spec.NodeSelector, vmi.Spec.NodeSelector)
	hasNodeAffinityChanged := !equality.Semantic.DeepEqual(vm.Spec.Template.Spec.Affinity, vmi.Spec.Affinity)

	if migrations.IsMigrating(vmi) && (hasNodeSelectorChanged || hasNodeAffinityChanged) {
		return fmt.Errorf("Node affinity should not be changed during VMI migration")
	}

	if hasNodeAffinityChanged {
		if err := c.VMIAffinityPatch(vm, vmi); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to update node affinity: %v", err)
			return err
		}
	}

	if hasNodeSelectorChanged {
		if err := c.VMNodeSelectorPatch(vm, vmi); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to update node selector: %v", err)
			return err
		}
	}
	return nil
}

func (c *VMController) handleMemoryDumpRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vm.Status.MemoryDumpRequest == nil {
		return nil
	}

	vmiVolumeMap := make(map[string]virtv1.Volume)
	if vmi != nil {
		for _, volume := range vmi.Spec.Volumes {
			vmiVolumeMap[volume.Name] = volume
		}
	}
	switch vm.Status.MemoryDumpRequest.Phase {
	case virtv1.MemoryDumpAssociating:
		if vmi == nil || vmi.DeletionTimestamp != nil || !vmi.IsRunning() {
			return nil
		}
		// When in state associating we want to add the memory dump pvc
		// as a volume in the vm and in the vmi to trigger the mount
		// to virt launcher and the memory dump
		vm.Spec.Template.Spec = *applyMemoryDumpVolumeRequestOnVMISpec(&vm.Spec.Template.Spec, vm.Status.MemoryDumpRequest.ClaimName)
		if _, exists := vmiVolumeMap[vm.Status.MemoryDumpRequest.ClaimName]; exists {
			return nil
		}
		if err := c.generateVMIMemoryDumpVolumePatch(vmi, vm.Status.MemoryDumpRequest, true); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to add memory dump volume: %v", err)
			return err
		}
	case virtv1.MemoryDumpUnmounting, virtv1.MemoryDumpFailed:
		if err := c.updatePVCMemoryDumpAnnotation(vm); err != nil {
			return err
		}
		// Check if the memory dump is in the vmi list of volumes,
		// if it still there remove it to make it unmount from virt launcher
		if _, exists := vmiVolumeMap[vm.Status.MemoryDumpRequest.ClaimName]; !exists {
			return nil
		}

		if err := c.generateVMIMemoryDumpVolumePatch(vmi, vm.Status.MemoryDumpRequest, false); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to remove memory dump volume: %v", err)
			return err
		}
	case virtv1.MemoryDumpDissociating:
		// Check if the memory dump is in the vmi list of volumes,
		// if it still there remove it to make it unmount from virt launcher
		if _, exists := vmiVolumeMap[vm.Status.MemoryDumpRequest.ClaimName]; exists {
			if err := c.generateVMIMemoryDumpVolumePatch(vmi, vm.Status.MemoryDumpRequest, false); err != nil {
				log.Log.Object(vmi).Errorf("unable to patch vmi to remove memory dump volume: %v", err)
				return err
			}
		}

		vm.Spec.Template.Spec = *removeMemoryDumpVolumeFromVMISpec(&vm.Spec.Template.Spec, vm.Status.MemoryDumpRequest.ClaimName)
	}

	return nil
}

func (c *VMController) handleVolumeRequests(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if len(vm.Status.VolumeRequests) == 0 {
		return nil
	}

	vmiVolumeMap := make(map[string]virtv1.Volume)
	if vmi != nil {
		for _, volume := range vmi.Spec.Volumes {
			vmiVolumeMap[volume.Name] = volume
		}
	}

	for i, request := range vm.Status.VolumeRequests {
		vm.Spec.Template.Spec = *controller.ApplyVolumeRequestOnVMISpec(&vm.Spec.Template.Spec, &vm.Status.VolumeRequests[i])

		if vmi == nil || vmi.DeletionTimestamp != nil {
			continue
		}

		if request.AddVolumeOptions != nil {
			if _, exists := vmiVolumeMap[request.AddVolumeOptions.Name]; exists {
				continue
			}

			if err := c.clientset.VirtualMachineInstance(vmi.Namespace).AddVolume(context.Background(), vmi.Name, request.AddVolumeOptions); err != nil {
				return err
			}
		} else if request.RemoveVolumeOptions != nil {
			if _, exists := vmiVolumeMap[request.RemoveVolumeOptions.Name]; !exists {
				continue
			}

			if err := c.clientset.VirtualMachineInstance(vmi.Namespace).RemoveVolume(context.Background(), vmi.Name, request.RemoveVolumeOptions); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *VMController) addStartRequest(vm *virtv1.VirtualMachine) error {
	addRequest := []virtv1.VirtualMachineStateChangeRequest{{Action: virtv1.StartRequest}}
	req, err := json.Marshal(addRequest)
	if err != nil {
		return err
	}
	patch := fmt.Sprintf(`{ "status":{ "stateChangeRequests":%s } }`, string(req))
	err = c.statusUpdater.PatchStatus(vm, types.MergePatchType, []byte(patch), &v1.PatchOptions{})
	if err != nil {
		return err
	}
	vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, addRequest[0])

	return nil
}

func (c *VMController) startStop(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) syncError {
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return &syncErrorImpl{fmt.Errorf(fetchingRunStrategyErrFmt, err), FailedCreateReason}
	}
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Errorf(fetchingVMKeyErrFmt, err)
		return &syncErrorImpl{err, FailedCreateReason}
	}
	log.Log.Object(vm).V(4).Infof("VirtualMachine RunStrategy: %s", runStrategy)

	switch runStrategy {
	case virtv1.RunStrategyAlways:
		// For this RunStrategy, a VMI should always be running. If a StateChangeRequest
		// asks to stop a VMI, a new one must be immediately re-started.
		if vmi != nil {
			var forceRestart bool
			if forceRestart = hasStopRequestForVMI(vm, vmi); forceRestart {
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
					return &syncErrorImpl{fmt.Errorf(failureDeletingVmiErrFormat, err), VMIFailedDeleteReason}
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
			return &syncErrorImpl{fmt.Errorf(startingVMIFailureFmt, err), FailedCreateReason}
		}
		return nil

	case virtv1.RunStrategyRerunOnFailure:
		// For this RunStrategy, a VMI should only be restarted if it failed.
		// If a VMI enters the Succeeded phase, it should not be restarted.
		if vmi != nil {
			forceStop := hasStopRequestForVMI(vm, vmi)
			if forceStop {
				log.Log.Object(vm).Infof("processing stop request for VMI with phase %s and VM runStrategy: %s", vmi.Status.Phase, runStrategy)
			}
			vmiFailed := vmi.Status.Phase == virtv1.Failed
			vmiSucceeded := vmi.Status.Phase == virtv1.Succeeded

			if vmi.DeletionTimestamp == nil && (forceStop || vmiFailed || vmiSucceeded) {
				// For RerunOnFailure, this controller should only restart the VirtualMachineInstance if it failed.
				log.Log.Object(vm).Infof("%s with VMI in phase %s and VM runStrategy: %s", stoppingVmMsg, vmi.Status.Phase, runStrategy)
				err := c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return &syncErrorImpl{fmt.Errorf(failureDeletingVmiErrFormat, err), VMIFailedDeleteReason}
				}

				if vmiFailed {
					err = c.addStartRequest(vm)
					if err != nil {
						return &syncErrorImpl{fmt.Errorf("failed to patch VM with start action: %v", err), VMIFailedDeleteReason}
					}
				}
			}
			// return to let the controller pick up the expected deletion
			return nil
		}

		if !hasStartRequest(vm) {
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
			return &syncErrorImpl{fmt.Errorf(startingVMIFailureFmt, err), FailedCreateReason}
		}
		return nil

	case virtv1.RunStrategyManual:
		// For this RunStrategy, VMI's will be started/stopped/restarted using api endpoints only
		if vmi != nil {
			log.Log.Object(vm).V(4).Info("VMI exists")

			if forceStop := hasStopRequestForVMI(vm, vmi); forceStop {
				log.Log.Object(vm).Infof("%s with VMI in phase %s due to stop request and VM runStrategy: %s", vmi.Status.Phase, stoppingVmMsg, runStrategy)
				err := c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return &syncErrorImpl{fmt.Errorf(failureDeletingVmiErrFormat, err), VMIFailedDeleteReason}
				}
				// return to let the controller pick up the expected deletion
				return nil
			}
		} else {
			if hasStartRequest(vm) {
				log.Log.Object(vm).Infof("%s due to start request and runStrategy: %s", startingVmMsg, runStrategy)

				err := c.startVMI(vm)
				if err != nil {
					return &syncErrorImpl{fmt.Errorf(startingVMIFailureFmt, err), FailedCreateReason}
				}
			}
		}
		return nil

	case virtv1.RunStrategyHalted:
		// For this runStrategy, no VMI should be running under any circumstances.
		// Set RunStrategyAlways/running = true if VM has StartRequest(start paused case).
		if vmi == nil {
			if hasStartRequest(vm) {
				vmCopy := vm.DeepCopy()
				runStrategy := virtv1.RunStrategyAlways
				running := true

				if vmCopy.Spec.RunStrategy != nil {
					vmCopy.Spec.RunStrategy = &runStrategy
				} else {
					vmCopy.Spec.Running = &running
				}
				_, err := c.clientset.VirtualMachine(vmCopy.Namespace).Update(context.Background(), vmCopy)
				return &syncErrorImpl{fmt.Errorf(startingVMIFailureFmt, err), FailedCreateReason}
			}
			return nil
		}
		log.Log.Object(vm).Infof("%s with VMI in phase %s due to runStrategy: %s", stoppingVmMsg, vmi.Status.Phase, runStrategy)
		if err := c.stopVMI(vm, vmi); err != nil {
			return &syncErrorImpl{fmt.Errorf(failureDeletingVmiErrFormat, err), VMIFailedDeleteReason}
		}
		return nil
	case virtv1.RunStrategyOnce:
		if vmi == nil {
			log.Log.Object(vm).Infof("%s due to start request and runStrategy: %s", startingVmMsg, runStrategy)

			err := c.startVMI(vm)
			if err != nil {
				return &syncErrorImpl{fmt.Errorf(startingVMIFailureFmt, err), FailedCreateReason}
			}
		}

		return nil
	default:
		return &syncErrorImpl{fmt.Errorf("unknown runstrategy: %s", runStrategy), FailedCreateReason}
	}
}

// isVMIStartExpected determines whether a VMI is expected to be started for this VM.
func (c *VMController) isVMIStartExpected(vm *virtv1.VirtualMachine) bool {
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Errorf(fetchingVMKeyErrFmt, err)
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
		log.Log.Object(vm).Errorf(fetchingVMKeyErrFmt, err)
		return false
	}

	expectations, exists, _ := c.expectations.GetExpectations(vmKey)
	if !exists || expectations == nil {
		return false
	}

	_, dels := expectations.GetExpectations()
	return dels > 0
}

// isSetToStart determines whether a VM is configured to be started (running).
func isSetToStart(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		log.Log.Object(vm).Errorf(fetchingRunStrategyErrFmt, err)
		return false
	}

	switch runStrategy {
	case virtv1.RunStrategyAlways:
		return true
	case virtv1.RunStrategyHalted:
		return false
	case virtv1.RunStrategyManual:
		if vmi != nil {
			return !hasStopRequestForVMI(vm, vmi)
		}
		return hasStartRequest(vm)
	case virtv1.RunStrategyRerunOnFailure:
		if vmi != nil {
			return vmi.Status.Phase != virtv1.Succeeded
		}
		return true
	case virtv1.RunStrategyOnce:
		if vmi == nil {
			return true
		}
		return false
	default:
		// Shouldn't ever be here, but...
		return false
	}
}

func (c *VMController) cleanupRestartRequired(vm *virtv1.VirtualMachine) error {
	vmConditionManager := controller.NewVirtualMachineConditionManager()
	if vmConditionManager.HasCondition(vm, virtv1.VirtualMachineRestartRequired) {
		vmConditionManager.RemoveCondition(vm, virtv1.VirtualMachineRestartRequired)
	}

	return c.deleteVMRevisions(vm, revisionPrefixLastSeen)
}

func (c *VMController) startVMI(vm *virtv1.VirtualMachine) error {
	// TODO add check for existence
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedExtractVmkeyFromVmErrMsg)
		return nil
	}

	err = c.cleanupRestartRequired(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedCleanupRestartRequired)
		return err
	}

	// start it
	vmi := c.setupVMIFromVM(vm)
	vmRevisionName, err := c.createVMRevision(vm, revisionPrefixStart)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedCreateCRforVmErrMsg)
		return err
	}
	vmi.Status.VirtualMachineRevisionName = vmRevisionName

	setGenerationAnnotationOnVmi(vm.Generation, vmi)

	// add a finalizer to ensure the VM controller has a chance to see
	// the VMI before it is deleted
	vmi.Finalizers = append(vmi.Finalizers, virtv1.VirtualMachineControllerFinalizer)

	// We need to apply device preferences before any new network or input devices are added. Doing so allows
	// any autoAttach preferences we might have to be applied, either enabling or disabling the attachment of these devices.
	preferenceSpec, err := c.applyDevicePreferences(vm, vmi)
	if err != nil {
		log.Log.Object(vm).Infof("Failed to apply device preferences again to VirtualMachineInstance: %s/%s", vmi.Namespace, vmi.Name)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error applying device preferences again: %v", err)
		return err
	}

	util.SetDefaultVolumeDisk(&vmi.Spec)

	autoAttachInputDevice(vmi)

	err = c.clusterConfig.SetVMISpecDefaultNetworkInterface(&vmi.Spec)
	if err != nil {
		return err
	}

	err = c.applyInstancetypeToVmi(vm, vmi, preferenceSpec)
	if err != nil {
		log.Log.Object(vm).Infof("Failed to apply instancetype to VirtualMachineInstance: %s/%s", vmi.Namespace, vmi.Name)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine instance: Failed to apply instancetype: %v", err)
		return err
	}

	c.expectations.ExpectCreations(vmKey, 1)
	vmi, err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Create(context.Background(), vmi)
	if err != nil {
		log.Log.Object(vm).Infof("Failed to create VirtualMachineInstance: %s", controller.NamespacedKey(vmi.Namespace, vmi.Name))
		c.expectations.CreationObserved(vmKey)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error creating virtual machine instance: %v", err)
		return err
	}
	log.Log.Object(vm).Infof("Started VM by creating the new virtual machine instance %s", vmi.Name)
	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulCreateVirtualMachineReason, "Started the virtual machine by creating the new virtual machine instance %v", vmi.ObjectMeta.Name)

	return nil
}

func setGenerationAnnotationOnVmi(generation int64, vmi *virtv1.VirtualMachineInstance) {
	annotations := vmi.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[virtv1.VirtualMachineGenerationAnnotation] = strconv.FormatInt(generation, 10)
	vmi.SetAnnotations(annotations)
}

func (c *VMController) patchVmGenerationAnnotationOnVmi(generation int64, vmi *virtv1.VirtualMachineInstance) error {
	origVmi := vmi.DeepCopy()

	setGenerationAnnotationOnVmi(generation, vmi)

	var ops []string
	oldAnnotations, err := json.Marshal(origVmi.Annotations)
	if err != nil {
		return err
	}
	newAnnotations, err := json.Marshal(vmi.Annotations)
	if err != nil {
		return err
	}
	ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/annotations", "value": %s }`, string(oldAnnotations)))
	ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/annotations", "value": %s }`, string(newAnnotations)))
	_, err = c.clientset.VirtualMachineInstance(origVmi.Namespace).Patch(context.Background(), origVmi.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops), &v1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

// getGenerationAnnotation will return the generation annotation on the
// vmi as an string pointer. The string pointer will be nil if the annotation is
// not found.
func getGenerationAnnotation(vmi *virtv1.VirtualMachineInstance) (i *string, err error) {
	if vmi == nil {
		return nil, errors.New("received nil pointer for vmi")
	}

	currentGenerationAnnotation, found := vmi.Annotations[virtv1.VirtualMachineGenerationAnnotation]
	if found {
		return &currentGenerationAnnotation, nil
	}

	return nil, nil
}

// getGenerationAnnotation will return the generation annotation on the
// vmi as an int64 pointer. The int64 pointer will be nil if the annotation is
// not found.
func getGenerationAnnotationAsInt(vmi *virtv1.VirtualMachineInstance, logger *log.FilteredLogger) (i *int64, err error) {
	if vmi == nil {
		return nil, errors.New("received nil pointer for vmi")
	}

	currentGenerationAnnotation, found := vmi.Annotations[virtv1.VirtualMachineGenerationAnnotation]
	if found {
		i, err := strconv.ParseInt(currentGenerationAnnotation, 10, 64)
		if err != nil {
			// If there is an error during parsing, it will be treated as if the
			// annotation does not exist since the annotation is not formatted
			// correctly. Further iterations / logic in the controller will handle
			// re-annotating this by the controller revision. Still log the error for
			// debugging, since there should never be a ParseInt error during normal
			// use.
			logger.Reason(err).Errorf("Failed to parse virtv1.VirtualMachineGenerationAnnotation as an int from vmi %v annotations", vmi.Name)
			return nil, nil
		}

		return &i, nil
	}

	return nil, nil
}

// Follows the template used in createVMRevision for the Data.Raw value
type VirtualMachineRevisionData struct {
	Spec virtv1.VirtualMachineSpec `json:"spec"`
}

// conditionallyBumpGenerationAnnotationOnVmi will check whether the
// generation annotation needs to be bumped on the VMI, and then bump that
// annotation if needed. The checks are:
// 1. If the generation has not changed, do not bump.
// 2. Only bump if the templates are the same.
//
// Note that if only the Run Strategy of the VM has changed, the generaiton
// annotation will still be bumped, since this does not affect the VMI.
func (c *VMController) conditionallyBumpGenerationAnnotationOnVmi(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vm == nil {
		return nil
	}

	// If this is an old vmi created before a controller update, then the
	// annotation may not exist. In that case, continue on as if the generation
	// annotation needs to be bumped.
	currentGeneration, err := getGenerationAnnotation(vmi)
	if err != nil {
		return err
	}
	if currentGeneration != nil && *currentGeneration == strconv.FormatInt(vm.Generation, 10) {
		return nil
	}

	currentRevision, err := c.getControllerRevision(vmi.Namespace, vmi.Status.VirtualMachineRevisionName)
	if currentRevision == nil || err != nil {
		return err
	}

	revisionSpec := &VirtualMachineRevisionData{}
	if err = json.Unmarshal(currentRevision.Data.Raw, revisionSpec); err != nil {
		return err
	}

	// If the templates are the same, we can safely bump the annotation.
	if equality.Semantic.DeepEqual(revisionSpec.Spec.Template, vm.Spec.Template) {
		if err := c.patchVmGenerationAnnotationOnVmi(vm.Generation, vmi); err != nil {
			return err
		}
	}

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
		log.Log.Object(vm).Errorf(fetchingRunStrategyErrFmt, err)
		return false
	}

	if runStrategy != virtv1.RunStrategyAlways &&
		runStrategy != virtv1.RunStrategyRerunOnFailure &&
		runStrategy != virtv1.RunStrategyOnce {
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
	// if for some reason the VM has been requested to be deleted, we want to use the
	// deletion grace period specified to the VM as the TerminationGracePeriodSeconds
	// for the VMI.
	if vm.DeletionTimestamp != nil {
		err = c.patchVMITerminationGracePeriod(vm.GetDeletionGracePeriodSeconds(), vmi)
		if err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi termination grace period: %v", err)
			return err
		}
	}
	c.expectations.ExpectDeletions(vmKey, []string{controller.VirtualMachineInstanceKey(vmi)})
	err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Delete(context.Background(), vmi.ObjectMeta.Name, &v1.DeleteOptions{})

	// Don't log an error if it is already deleted
	if err != nil {
		// We can't observe a delete if it was not accepted by the server
		c.expectations.DeletionObserved(vmKey, controller.VirtualMachineInstanceKey(vmi))
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDeleteVirtualMachineReason, "Error deleting virtual machine instance %s: %v", vmi.ObjectMeta.Name, err)
		return err
	}

	err = c.cleanupRestartRequired(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedCleanupRestartRequired)
		return nil
	}

	c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDeleteVirtualMachineReason, "Stopped the virtual machine by deleting the virtual machine instance %v", vmi.ObjectMeta.UID)
	log.Log.Object(vm).Infof("Dispatching delete event for vmi %s with phase %s", controller.NamespacedKey(vmi.Namespace, vmi.Name), vmi.Status.Phase)

	return nil
}

func vmRevisionNamePrefix(vmUID types.UID, prefix string) string {
	return fmt.Sprintf("revision-%s-vm-%s", prefix, vmUID)
}

func getVMRevisionName(vmUID types.UID, generation int64, prefix string) string {
	return fmt.Sprintf("%s-%d", vmRevisionNamePrefix(vmUID, prefix), generation)
}

func patchVMRevision(vm *virtv1.VirtualMachine) ([]byte, error) {
	vmBytes, err := json.Marshal(vm)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	err = json.Unmarshal(vmBytes, &raw)
	if err != nil {
		return nil, err
	}
	objCopy := make(map[string]interface{})
	spec := raw["spec"].(map[string]interface{})
	objCopy["spec"] = spec
	patch, err := json.Marshal(objCopy)
	return patch, err
}

func (c *VMController) deleteOlderVMRevision(vm *virtv1.VirtualMachine, prefix string) (bool, error) {
	keys, err := c.crInformer.GetIndexer().IndexKeys("vm", string(vm.UID))
	if err != nil {
		return false, err
	}

	createNotNeeded := false
	for _, key := range keys {
		if !strings.Contains(key, vmRevisionNamePrefix(vm.UID, prefix)) {
			continue
		}

		storeObj, exists, err := c.crInformer.GetStore().GetByKey(key)
		if !exists || err != nil {
			return false, err
		}
		cr, ok := storeObj.(*appsv1.ControllerRevision)
		if !ok {
			return false, fmt.Errorf("unexpected resource %+v", storeObj)
		}

		if cr.Revision == vm.ObjectMeta.Generation {
			createNotNeeded = true
			continue
		}

		err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), cr.Name, v1.DeleteOptions{})
		if err != nil {
			return false, err
		}
	}

	return createNotNeeded, nil
}

func (c *VMController) deleteVMRevisions(vm *virtv1.VirtualMachine, prefix string) error {
	keys, err := c.crInformer.GetIndexer().IndexKeys("vm", string(vm.UID))
	if err != nil {
		return err
	}

	for _, key := range keys {
		if !strings.Contains(key, vmRevisionNamePrefix(vm.UID, prefix)) {
			continue
		}

		storeObj, exists, err := c.crInformer.GetStore().GetByKey(key)
		if !exists || err != nil {
			return err
		}
		cr, ok := storeObj.(*appsv1.ControllerRevision)
		if !ok {
			return fmt.Errorf("unexpected resource %+v", storeObj)
		}

		err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), cr.Name, v1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// getControllerRevision attempts to get the controller revision by name and
// namespace. It will return (nil, nil) if the controller revision is not found.
func (c *VMController) getControllerRevision(namespace string, name string) (*appsv1.ControllerRevision, error) {
	cr, err := c.clientset.AppsV1().ControllerRevisions(namespace).Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return cr, nil
}

func (c *VMController) getVMSpecForKey(key string) (*virtv1.VirtualMachineSpec, error) {
	obj, exists, err := c.crInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("could not find key %s", key)
	}

	cr, ok := obj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("unexpected resource %+v", obj)
	}

	raw := map[string]interface{}{}
	err = json.Unmarshal(cr.Data.Raw, &raw)
	if err != nil {
		return nil, err
	}
	patch, err := json.Marshal(raw["spec"])
	if err != nil {
		return nil, err
	}
	vmSpec := virtv1.VirtualMachineSpec{}
	err = json.Unmarshal(patch, &vmSpec)
	if err != nil {
		return nil, err
	}
	return &vmSpec, nil
}

func genFromKey(key string) (int64, error) {
	items := strings.Split(key, "-")
	genString := items[len(items)-1]
	return strconv.ParseInt(genString, 10, 64)
}

func (c *VMController) getLastVMRevisionSpec(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachineSpec, error) {
	keys, err := c.crInformer.GetIndexer().IndexKeys("vm", string(vm.UID))
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}

	var highestGen int64 = 0
	var key string
	for _, k := range keys {
		if !strings.Contains(k, vmRevisionNamePrefix(vm.UID, revisionPrefixLastSeen)) {
			continue
		}
		gen, err := genFromKey(k)

		if err != nil {
			return nil, fmt.Errorf("invalid key: %s", k)
		}
		if gen > highestGen {
			if key != "" {
				log.Log.Object(vm).Warningf("expected no more than 1 revision, found at least 2")
			}
			highestGen = gen
			key = k
		}
	}

	if key == "" {
		return nil, nil
	}

	return c.getVMSpecForKey(key)
}

func (c *VMController) hasLastSeenRevision(vm *virtv1.VirtualMachine) (bool, error) {
	keys, err := c.crInformer.GetIndexer().IndexKeys("vm", string(vm.UID))
	if err != nil {
		return false, err
	}
	if len(keys) == 0 {
		return false, nil
	}

	for _, k := range keys {
		if strings.Contains(k, vmRevisionNamePrefix(vm.UID, revisionPrefixLastSeen)) {
			return true, nil
		}
	}

	return false, nil
}

func (c *VMController) createVMRevision(vm *virtv1.VirtualMachine, prefix string) (string, error) {
	vmRevisionName := getVMRevisionName(vm.UID, vm.Generation, prefix)
	createNotNeeded, err := c.deleteOlderVMRevision(vm, prefix)
	if err != nil || createNotNeeded {
		return vmRevisionName, err
	}
	patch, err := patchVMRevision(vm)
	if err != nil {
		return "", err
	}
	cr := &appsv1.ControllerRevision{
		ObjectMeta: v1.ObjectMeta{
			Name:            vmRevisionName,
			Namespace:       vm.Namespace,
			OwnerReferences: []v1.OwnerReference{*v1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
		},
		Data:     runtime.RawExtension{Raw: patch},
		Revision: vm.ObjectMeta.Generation,
	}
	_, err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, v1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return cr.Name, nil
}

func hasCompletedMemoryDump(vm *virtv1.VirtualMachine) bool {
	return vm.Status.MemoryDumpRequest != nil && vm.Status.MemoryDumpRequest.Phase != virtv1.MemoryDumpAssociating && vm.Status.MemoryDumpRequest.Phase != virtv1.MemoryDumpInProgress
}

// setupVMIfromVM creates a VirtualMachineInstance object from one VirtualMachine object.
func (c *VMController) setupVMIFromVM(vm *virtv1.VirtualMachine) *virtv1.VirtualMachineInstance {
	vmi := virtv1.NewVMIReferenceFromNameWithNS(vm.ObjectMeta.Namespace, "")
	vmi.ObjectMeta = *vm.Spec.Template.ObjectMeta.DeepCopy()
	vmi.ObjectMeta.Name = vm.ObjectMeta.Name
	vmi.ObjectMeta.GenerateName = ""
	vmi.ObjectMeta.Namespace = vm.ObjectMeta.Namespace
	vmi.Spec = *vm.Spec.Template.Spec.DeepCopy()

	if hasStartPausedRequest(vm) {
		strategy := virtv1.StartStrategyPaused
		vmi.Spec.StartStrategy = &strategy
	}

	// prevent from retriggering memory dump after shutdown if memory dump is complete
	if hasCompletedMemoryDump(vm) {
		vmi.Spec = *removeMemoryDumpVolumeFromVMISpec(&vmi.Spec, vm.Status.MemoryDumpRequest.ClaimName)
	}

	setupStableFirmwareUUID(vm, vmi)

	// TODO check if vmi labels exist, and when make sure that they match. For now just override them
	vmi.ObjectMeta.Labels = vm.Spec.Template.ObjectMeta.Labels
	vmi.ObjectMeta.OwnerReferences = []v1.OwnerReference{
		*v1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}

	VMIDefaults := &virtv1.VirtualMachineInstance{}
	webhooks.SetDefaultGuestCPUTopology(c.clusterConfig, &VMIDefaults.Spec)

	vmi.Status.CurrentCPUTopology = &virtv1.CPUTopology{
		Sockets: VMIDefaults.Spec.Domain.CPU.Sockets,
		Cores:   VMIDefaults.Spec.Domain.CPU.Cores,
		Threads: VMIDefaults.Spec.Domain.CPU.Threads,
	}

	if topology := vm.Spec.Template.Spec.Domain.CPU; topology != nil {
		if topology.Sockets != 0 {
			vmi.Status.CurrentCPUTopology.Sockets = topology.Sockets
		}
		if topology.Cores != 0 {
			vmi.Status.CurrentCPUTopology.Cores = topology.Cores
		}
		if topology.Threads != 0 {
			vmi.Status.CurrentCPUTopology.Threads = topology.Threads
		}
	}

	c.setupHotplug(vmi, VMIDefaults)

	return vmi
}

func (c *VMController) applyInstancetypeToVmi(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {

	instancetypeSpec, err := c.instancetypeMethods.FindInstancetypeSpec(vm)
	if err != nil {
		return err
	}

	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil
	}

	instancetype.AddInstancetypeNameAnnotations(vm, vmi)
	instancetype.AddPreferenceNameAnnotations(vm, vmi)

	if conflicts := c.instancetypeMethods.ApplyToVmi(k8sfield.NewPath("spec"), instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta); len(conflicts) > 0 {
		return fmt.Errorf("VMI conflicts with instancetype spec in fields: [%s]", conflicts.String())
	}

	return nil
}

func hasStartPausedRequest(vm *virtv1.VirtualMachine) bool {
	if len(vm.Status.StateChangeRequests) == 0 {
		return false
	}

	stateChange := vm.Status.StateChangeRequests[0]
	pausedValue, hasPaused := stateChange.Data[virtv1.StartRequestDataPausedKey]
	return stateChange.Action == virtv1.StartRequest &&
		hasPaused &&
		pausedValue == virtv1.StartRequestDataPausedTrue
}

func hasStartRequest(vm *virtv1.VirtualMachine) bool {
	if len(vm.Status.StateChangeRequests) == 0 {
		return false
	}

	stateChange := vm.Status.StateChangeRequests[0]
	return stateChange.Action == virtv1.StartRequest
}

func hasStopRequestForVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if len(vm.Status.StateChangeRequests) == 0 {
		return false
	}

	stateChange := vm.Status.StateChangeRequests[0]
	return stateChange.Action == virtv1.StopRequest &&
		stateChange.UID != nil &&
		*stateChange.UID == vmi.UID
}

// no special meaning, randomly generated on my box.
// TODO: do we want to use another constants? see examples in RFC4122
const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.MustParse(magicUUID)

// setStableUUID makes sure the VirtualMachineInstance being started has a 'stable' UUID.
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

func (c *VMController) setupCPUHotplug(vmi *virtv1.VirtualMachineInstance, VMIDefaults *virtv1.VirtualMachineInstance, maxRatio uint32) {
	if vmi.Spec.Domain.CPU == nil {
		vmi.Spec.Domain.CPU = &virtv1.CPU{}
	}

	if vmi.Spec.Domain.CPU.MaxSockets == 0 {
		vmi.Spec.Domain.CPU.MaxSockets = c.clusterConfig.GetMaximumCpuSockets()
	}

	if vmi.Spec.Domain.CPU.MaxSockets == 0 {
		vmi.Spec.Domain.CPU.MaxSockets = vmi.Spec.Domain.CPU.Sockets * maxRatio
	}

	if vmi.Spec.Domain.CPU.MaxSockets == 0 {
		vmi.Spec.Domain.CPU.MaxSockets = VMIDefaults.Spec.Domain.CPU.Sockets * maxRatio
	}
}

func (c *VMController) setupMemoryHotplug(vmi *virtv1.VirtualMachineInstance, maxRatio uint32) {
	if vmi.Spec.Domain.Memory == nil {
		return
	}

	if vmi.Spec.Domain.Memory.MaxGuest == nil {
		vmi.Spec.Domain.Memory.MaxGuest = c.clusterConfig.GetMaximumGuestMemory()
	}

	if vmi.Spec.Domain.Memory.MaxGuest == nil {
		vmi.Spec.Domain.Memory.MaxGuest = resource.NewQuantity(vmi.Spec.Domain.Memory.Guest.Value()*int64(maxRatio), resource.BinarySI)
	}
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

	labelChanged := !equality.Semantic.DeepEqual(curVMI.Labels, oldVMI.Labels)
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
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
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
	if controllerRef != nil {
		log.Log.Object(dataVolume).Info("Looking for DataVolume Ref")
		vm := c.resolveControllerRef(dataVolume.Namespace, controllerRef)
		if vm != nil {
			vmKey, err := controller.KeyFunc(vm)
			if err != nil {
				log.Log.Object(dataVolume).Errorf("Cannot parse key of VM: %s for DataVolume: %s", vm.Name, dataVolume.Name)
			} else {
				log.Log.Object(dataVolume).Infof("DataVolume created because %s was added.", dataVolume.Name)
				c.dataVolumeExpectations.CreationObserved(vmKey)
			}
		} else {
			log.Log.Object(dataVolume).Errorf("Cant find the matching VM for DataVolume: %s", dataVolume.Name)
		}
	}
	c.queueVMsForDataVolume(dataVolume)
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
	labelChanged := !equality.Semantic.DeepEqual(curDataVolume.Labels, oldDataVolume.Labels)
	if curDataVolume.DeletionTimestamp != nil {
		// having a DataVolume marked for deletion is enough
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
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vm := c.resolveControllerRef(oldDataVolume.Namespace, oldControllerRef); vm != nil {
			c.enqueueVm(vm)
		}
	}
	c.queueVMsForDataVolume(curDataVolume)
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
	if controllerRef := v1.GetControllerOf(dataVolume); controllerRef != nil {
		if vm := c.resolveControllerRef(dataVolume.Namespace, controllerRef); vm != nil {
			if vmKey, err := controller.KeyFunc(vm); err == nil {
				c.dataVolumeExpectations.DeletionObserved(vmKey, controller.DataVolumeKey(dataVolume))
			}
		}
	}
	c.queueVMsForDataVolume(dataVolume)
}

func (c *VMController) queueVMsForDataVolume(dataVolume *cdiv1.DataVolume) {
	var vmOwner string
	if controllerRef := v1.GetControllerOf(dataVolume); controllerRef != nil {
		if vm := c.resolveControllerRef(dataVolume.Namespace, controllerRef); vm != nil {
			vmOwner = vm.Name
			log.Log.V(4).Object(dataVolume).Infof("DataVolume updated for vm %s", vm.Name)
			c.enqueueVm(vm)
		}
	}
	// handle DataVolumes not owned by the VM but referenced in the spec
	// TODO come back when DV/PVC name may differ
	k, err := controller.KeyFunc(dataVolume)
	if err != nil {
		log.Log.Object(dataVolume).Errorf("Cannot parse key of DataVolume: %s", dataVolume.Name)
		return
	}
	for _, indexName := range []string{"dv", "pvc"} {
		objs, err := c.vmInformer.GetIndexer().ByIndex(indexName, k)
		if err != nil {
			log.Log.Object(dataVolume).Errorf("Cannot get index %s of DataVolume: %s", indexName, dataVolume.Name)
			return
		}
		for _, obj := range objs {
			vm := obj.(*virtv1.VirtualMachine)
			if vm.Name != vmOwner {
				log.Log.V(4).Object(dataVolume).Infof("DataVolume updated for vm %s", vm.Name)
				c.enqueueVm(vm)
			}
		}
	}
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
		return
	}
	c.Queue.Add(key)
}

func (c *VMController) getPatchFinalizerOps(oldObj, newObj v1.Object) ([]string, error) {
	var ops []string

	oldFinalizers, err := json.Marshal(oldObj.GetFinalizers())
	if err != nil {
		return ops, err
	}

	newFinalizers, err := json.Marshal(newObj.GetFinalizers())
	if err != nil {
		return ops, err
	}

	ops = append(ops, fmt.Sprintf(`{ "op": "test", "path": "/metadata/finalizers", "value": %s }`, string(oldFinalizers)))
	ops = append(ops, fmt.Sprintf(`{ "op": "replace", "path": "/metadata/finalizers", "value": %s }`, string(newFinalizers)))
	return ops, nil
}

func (c *VMController) removeVMIFinalizer(vmi *virtv1.VirtualMachineInstance) error {
	if !controller.HasFinalizer(vmi, virtv1.VirtualMachineControllerFinalizer) {
		return nil
	}

	log.Log.V(3).Object(vmi).Infof("VMI is in a final state. Removing VM controller finalizer")
	newVmi := vmi.DeepCopy()
	controller.RemoveFinalizer(newVmi, virtv1.VirtualMachineControllerFinalizer)
	ops, err := c.getPatchFinalizerOps(vmi, newVmi)
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops), &v1.PatchOptions{})
	return err
}

func (c *VMController) removeVMFinalizer(vm *virtv1.VirtualMachine, finalizer string) (*virtv1.VirtualMachine, error) {
	if !controller.HasFinalizer(vm, finalizer) {
		return vm, nil
	}

	log.Log.V(3).Object(vm).Infof("Removing VM controller finalizer: %s", finalizer)
	newVm := vm.DeepCopy()
	controller.RemoveFinalizer(newVm, finalizer)
	ops, err := c.getPatchFinalizerOps(vm, newVm)
	if err != nil {
		return vm, err
	}

	vm, err = c.clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops), &v1.PatchOptions{})
	return vm, err
}

func (c *VMController) addVMFinalizer(vm *virtv1.VirtualMachine, finalizer string) (*virtv1.VirtualMachine, error) {
	if controller.HasFinalizer(vm, finalizer) {
		return vm, nil
	}

	log.Log.V(3).Object(vm).Infof("Adding VM controller finalizer: %s", finalizer)
	newVm := vm.DeepCopy()
	controller.AddFinalizer(newVm, finalizer)
	ops, err := c.getPatchFinalizerOps(vm, newVm)
	if err != nil {
		return vm, err
	}

	return c.clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, controller.GeneratePatchBytes(ops), &v1.PatchOptions{})
}

// parseGeneration will parse for the last value after a '-'. It is assumed the
// revision name is created with getVMRevisionName. If the name is not formatted
// correctly and the generation cannot be found, then nil will be returned.
func parseGeneration(revisionName string, logger *log.FilteredLogger) *int64 {
	idx := strings.LastIndexAny(revisionName, "-")
	if idx == -1 {
		logger.Errorf("Failed to parse generation as an int from revision %v", revisionName)
		return nil
	}

	generationStr := revisionName[idx+1:]

	generation, err := strconv.ParseInt(generationStr, 10, 64)
	if err != nil {
		logger.Reason(err).Errorf("Failed to parse generation as an int from revision %v", revisionName)
		return nil
	}

	return &generation
}

// patchVmGenerationFromControllerRevision will first fetch the generation from
// the corresponding controller revision, and then patch the vmi with the
// generation annotation. If the controller revision does not exist,
// (nil, nil) will be returned.
func (c *VMController) patchVmGenerationFromControllerRevision(vmi *virtv1.VirtualMachineInstance, logger *log.FilteredLogger) (generation *int64, err error) {
	generation = nil

	cr, err := c.getControllerRevision(vmi.Namespace, vmi.Status.VirtualMachineRevisionName)
	if err != nil || cr == nil {
		return generation, err
	}

	generation = parseGeneration(cr.Name, logger)
	if generation == nil {
		return nil, nil
	}

	if err := c.patchVmGenerationAnnotationOnVmi(*generation, vmi); err != nil {
		return generation, err
	}

	return generation, err
}

// syncGenerationInfo will update the vm.Status with the ObservedGeneration
// from the vmi and the DesiredGeneration from the vm current generation.
func (c *VMController) syncGenerationInfo(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, logger *log.FilteredLogger) error {
	if vm == nil || vmi == nil {
		return errors.New("passed nil pointer")
	}

	generation, err := getGenerationAnnotationAsInt(vmi, logger)
	if err != nil {
		return err
	}

	// If the generation annotation does not exist, the VMI could have been
	// been created before the controller was updated. In this case, check the
	// ControllerRevision on what the latest observed generation is and back-fill
	// the info onto the vmi annotation.
	if generation == nil {
		generation, err = c.patchVmGenerationFromControllerRevision(vmi, logger)
		if generation == nil || err != nil {
			return err
		}
	}

	vm.Status.ObservedGeneration = *generation
	vm.Status.DesiredGeneration = vm.Generation

	return nil
}

func (c *VMController) updateStatus(vmOrig *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, syncErr syncError, logger *log.FilteredLogger) error {
	key := controller.VirtualMachineKey(vmOrig)
	defer virtControllerVMWorkQueueTracer.StepTrace(key, "updateStatus", trace.Field{Key: "VM Name", Value: vmOrig.Name})

	vm := vmOrig.DeepCopy()

	created := vmi != nil
	vm.Status.Created = created

	ready := false
	if created {
		ready = controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceReady, k8score.ConditionTrue)
		if err := c.syncGenerationInfo(vm, vmi, logger); err != nil {
			return err
		}
	}
	vm.Status.Ready = ready

	c.trimDoneVolumeRequests(vm)
	c.updateMemoryDumpRequest(vm, vmi)

	if c.isTrimFirstChangeRequestNeeded(vm, vmi) {
		vm.Status.StateChangeRequests = vm.Status.StateChangeRequests[1:]
	}

	syncStartFailureStatus(vm, vmi)
	c.syncConditions(vm, vmi, syncErr)
	c.setPrintableStatus(vm, vmi)

	// only update if necessary
	if !equality.Semantic.DeepEqual(vm.Status, vmOrig.Status) {
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
		{virtv1.VirtualMachineStatusPvcNotFound, c.isVirtualMachineStatusPvcNotFound},
		{virtv1.VirtualMachineStatusDataVolumeError, c.isVirtualMachineStatusDataVolumeError},
		{virtv1.VirtualMachineStatusUnschedulable, c.isVirtualMachineStatusUnschedulable},
		{virtv1.VirtualMachineStatusProvisioning, c.isVirtualMachineStatusProvisioning},
		{virtv1.VirtualMachineStatusWaitingForVolumeBinding, c.isVirtualMachineStatusWaitingForVolumeBinding},
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
		log.Log.Object(vm).Errorf(fetchingRunStrategyErrFmt, err)
		return false
	}

	if vm.Status.StartFailure != nil &&
		vm.Status.StartFailure.ConsecutiveFailCount > 0 &&
		(runStrategy == virtv1.RunStrategyAlways || runStrategy == virtv1.RunStrategyRerunOnFailure || runStrategy == virtv1.RunStrategyOnce) {
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
	return storagetypes.HasDataVolumeProvisioning(vm.Namespace, vm.Spec.Template.Spec.Volumes, c.dataVolumeInformer)
}

// isVirtualMachineStatusWaitingForVolumeBinding
func (c *VMController) isVirtualMachineStatusWaitingForVolumeBinding(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if !isSetToStart(vm, vmi) {
		return false
	}

	return storagetypes.HasUnboundPVC(vm.Namespace, vm.Spec.Template.Spec.Volumes, c.pvcInformer)
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

// isVirtualMachineStatusPvcNotFound determines whether the VM status field should be set to "FailedPvcNotFound".
func (c *VMController) isVirtualMachineStatusPvcNotFound(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatusAndReason(vmi,
		virtv1.VirtualMachineInstanceSynchronized,
		k8score.ConditionFalse,
		FailedPvcNotFoundReason)
}

// isVirtualMachineStatusDataVolumeError determines whether the VM status field should be set to "DataVolumeError"
func (c *VMController) isVirtualMachineStatusDataVolumeError(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	err := storagetypes.HasDataVolumeErrors(vm.Namespace, vm.Spec.Template.Spec.Volumes, c.dataVolumeInformer)
	if err != nil {
		log.Log.Object(vm).Errorf("%v", err)
		return true
	}
	return false
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

func (c *VMController) syncConditions(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, syncErr syncError) {
	cm := controller.NewVirtualMachineConditionManager()

	// ready condition is handled differently as it persists regardless if vmi exists or not
	c.syncReadyConditionFromVMI(vm, vmi)
	c.processFailureCondition(vm, vmi, syncErr)

	// nothing to do if vmi hasn't been created yet.
	if vmi == nil {
		return
	}

	// sync VMI conditions, ignore list represents conditions that are not synced generically
	syncIgnoreMap := map[string]interface{}{
		string(virtv1.VirtualMachineReady):           nil,
		string(virtv1.VirtualMachineFailure):         nil,
		string(virtv1.VirtualMachineInitialized):     nil,
		string(virtv1.VirtualMachineRestartRequired): nil,
	}
	vmiCondMap := make(map[string]interface{})

	// generically add/update all vmi conditions
	for _, cond := range vmi.Status.Conditions {
		_, ignore := syncIgnoreMap[string(cond.Type)]
		if ignore {
			continue
		}
		vmiCondMap[string(cond.Type)] = nil
		cm.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
			Type:               virtv1.VirtualMachineConditionType(cond.Type),
			Status:             cond.Status,
			Reason:             cond.Reason,
			Message:            cond.Message,
			LastProbeTime:      cond.LastProbeTime,
			LastTransitionTime: cond.LastTransitionTime,
		})
	}

	// remove vm conditions that don't exist on vmi (excluding the ignore list)
	for _, cond := range vm.Status.Conditions {
		_, ignore := syncIgnoreMap[string(cond.Type)]
		if ignore {
			continue
		}

		_, exists := vmiCondMap[string(cond.Type)]
		if !exists {
			cm.RemoveCondition(vm, cond.Type)
		}
	}
}

func (c *VMController) processFailureCondition(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, syncErr syncError) {

	vmConditionManager := controller.NewVirtualMachineConditionManager()
	if syncErr == nil {
		if vmConditionManager.HasCondition(vm, virtv1.VirtualMachineFailure) {
			log.Log.Object(vm).V(4).Info("Removing failure")
			vmConditionManager.RemoveCondition(vm, virtv1.VirtualMachineFailure)
		}
		// nothing to do
		return
	}

	vmConditionManager.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
		Type:               virtv1.VirtualMachineFailure,
		Reason:             syncErr.Reason(),
		Message:            syncErr.Error(),
		LastTransitionTime: v1.Now(),
		Status:             k8score.ConditionTrue,
	})

	return
}

func (c *VMController) isTrimFirstChangeRequestNeeded(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (clearChangeRequest bool) {
	if len(vm.Status.StateChangeRequests) == 0 {
		return false
	}

	// Only consider one stateChangeRequest at a time. The second and subsequent change
	// requests have not been acted upon by this controller yet!
	stateChange := vm.Status.StateChangeRequests[0]
	switch stateChange.Action {
	case virtv1.StopRequest:
		if vmi == nil {
			// because either the VM or VMI informers can trigger processing here
			// double check the state of the cluster before taking action
			_, err := c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Get(context.Background(), vm.GetName(), &v1.GetOptions{})
			if err != nil && apiErrors.IsNotFound(err) {
				// If there's no VMI, then the VMI was stopped, and the stopRequest can be cleared
				log.Log.Object(vm).V(4).Infof("No VMI. Clearing stop request")
				return true
			}
		} else {
			if stateChange.UID == nil {
				// It never makes sense to have a request to stop a VMI that doesn't
				// have a UUID associated with it. This shouldn't be possible -- but if
				// it occurs, clear the stopRequest because it can't be acted upon
				log.Log.Object(vm).Errorf("Stop Request has no UID.")
				return true
			} else if *stateChange.UID != vmi.UID {
				// If there is a VMI, but the UID doesn't match, then it
				// must have been previously stopped, so the stopRequest can be cleared
				log.Log.Object(vm).V(4).Infof("VMI's UID doesn't match. clearing stop request")
				return true
			}
		}
	case virtv1.StartRequest:
		// If the current VMI is running, then it has been started.
		if vmi != nil && vmi.DeletionTimestamp == nil {
			log.Log.Object(vm).V(4).Infof("VMI exists. clearing start request")
			return true
		}
	}

	return false
}

func (c *VMController) updateMemoryDumpRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
	if vm.Status.MemoryDumpRequest == nil {
		return
	}

	updatedMemoryDumpReq := vm.Status.MemoryDumpRequest.DeepCopy()

	if vm.Status.MemoryDumpRequest.Remove {
		updatedMemoryDumpReq.Phase = virtv1.MemoryDumpDissociating
	}

	switch vm.Status.MemoryDumpRequest.Phase {
	case virtv1.MemoryDumpCompleted:
		// Once memory dump completed, there is no update neeeded,
		// A new update will come from the subresource API once
		// a new request will be issued
		return
	case virtv1.MemoryDumpAssociating:
		// Update Phase to InProgrees once the memory dump
		// is in the list of vm volumes
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if vm.Status.MemoryDumpRequest.ClaimName == volume.Name {
				updatedMemoryDumpReq.Phase = virtv1.MemoryDumpInProgress
				break
			}
		}
	case virtv1.MemoryDumpInProgress:
		// Update to unmounting once getting update in the vmi volume status
		// that the dump timestamp is updated
		if vmi != nil && len(vmi.Status.VolumeStatus) > 0 {
			for _, volumeStatus := range vmi.Status.VolumeStatus {
				if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName &&
					volumeStatus.MemoryDumpVolume != nil {
					if volumeStatus.MemoryDumpVolume.StartTimestamp != nil {
						updatedMemoryDumpReq.StartTimestamp = volumeStatus.MemoryDumpVolume.StartTimestamp
					}
					if volumeStatus.Phase == virtv1.MemoryDumpVolumeCompleted {
						updatedMemoryDumpReq.Phase = virtv1.MemoryDumpUnmounting
						updatedMemoryDumpReq.EndTimestamp = volumeStatus.MemoryDumpVolume.EndTimestamp
						updatedMemoryDumpReq.FileName = &volumeStatus.MemoryDumpVolume.TargetFileName
					} else if volumeStatus.Phase == virtv1.MemoryDumpVolumeFailed {
						updatedMemoryDumpReq.Phase = virtv1.MemoryDumpFailed
						updatedMemoryDumpReq.Message = volumeStatus.Message
						updatedMemoryDumpReq.EndTimestamp = volumeStatus.MemoryDumpVolume.EndTimestamp
					}
				}
			}
		}
	case virtv1.MemoryDumpUnmounting:
		// Update memory dump as completed once the memory dump has been
		// unmounted - not a part of the vmi volume status
		if vmi != nil {
			for _, volumeStatus := range vmi.Status.VolumeStatus {
				// If we found the claim name in the vmi volume status
				// then the pvc is still mounted
				if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName {
					return
				}
			}
		}
		updatedMemoryDumpReq.Phase = virtv1.MemoryDumpCompleted
	case virtv1.MemoryDumpDissociating:
		// Make sure the memory dump is not in the vmi list of volumes
		if vmi != nil {
			for _, volumeStatus := range vmi.Status.VolumeStatus {
				if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName {
					return
				}
			}
		}
		// Make sure the memory dump is not in the list of vm volumes
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if vm.Status.MemoryDumpRequest.ClaimName == volume.Name {
				return
			}
		}
		// Remove the memory dump request
		updatedMemoryDumpReq = nil
	}

	vm.Status.MemoryDumpRequest = updatedMemoryDumpReq
}

func (c *VMController) trimDoneVolumeRequests(vm *virtv1.VirtualMachine) {
	if len(vm.Status.VolumeRequests) == 0 {
		return
	}

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

// addRestartRequiredIfNeeded adds the restartRequired condition to the VM if any non-live-updatable field was changed
func (c *VMController) addRestartRequiredIfNeeded(lastSeenVMSpec *virtv1.VirtualMachineSpec, vm *virtv1.VirtualMachine) bool {
	if lastSeenVMSpec == nil {
		return false
	}
	// Ignore all the live-updatable fields by copying them over. (If the feature gate is disabled, nothing is live-updatable)
	// Note: this list needs to stay up-to-date with everything that can be live-updated
	// Note2: destroying lastSeenVMSpec here is fine, we don't need it later
	if c.clusterConfig.IsVMRolloutStrategyLiveUpdate() {
		lastSeenVMSpec.Template.Spec.Volumes = vm.Spec.Template.Spec.Volumes
		lastSeenVMSpec.Template.Spec.Domain.Devices.Disks = vm.Spec.Template.Spec.Domain.Devices.Disks
		if lastSeenVMSpec.Template.Spec.Domain.CPU != nil && vm.Spec.Template.Spec.Domain.CPU != nil {
			lastSeenVMSpec.Template.Spec.Domain.CPU.Sockets = vm.Spec.Template.Spec.Domain.CPU.Sockets
		}
		if lastSeenVMSpec.Template.Spec.Domain.Memory != nil && vm.Spec.Template.Spec.Domain.Memory != nil {
			lastSeenVMSpec.Template.Spec.Domain.Memory.Guest = vm.Spec.Template.Spec.Domain.Memory.Guest
		}
		lastSeenVMSpec.Template.Spec.NodeSelector = vm.Spec.Template.Spec.NodeSelector
		lastSeenVMSpec.Template.Spec.Affinity = vm.Spec.Template.Spec.Affinity
	}

	if !equality.Semantic.DeepEqual(lastSeenVMSpec.Template.Spec, vm.Spec.Template.Spec) {
		vmConditionManager := controller.NewVirtualMachineConditionManager()
		vmConditionManager.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
			Type:               virtv1.VirtualMachineRestartRequired,
			LastTransitionTime: v1.Now(),
			Status:             k8score.ConditionTrue,
			Message:            "a non-live-updatable field was changed in the template spec",
		})
		return true
	}

	return false
}

func (c *VMController) sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, key string, dataVolumes []*cdiv1.DataVolume) (*virtv1.VirtualMachine, syncError, error) {
	var syncErr syncError
	var err error
	var lastSeenVMSpec *virtv1.VirtualMachineSpec

	if !c.needsSync(key) {
		return vm, nil, nil
	}

	if vm.Generation > 1 {
		lastSeenVMSpec, err = c.getLastVMRevisionSpec(vm)
		if err != nil {
			return vm, nil, err
		}
	}

	conditionManager := controller.NewVirtualMachineConditionManager()
	if !conditionManager.HasCondition(vm, virtv1.VirtualMachineInitialized) {
		runStrategy, err := vm.RunStrategy()
		if err != nil {
			return vm, nil, err
		}
		if runStrategy == virtv1.RunStrategyRerunOnFailure {
			// VM just got created with RerunOnFailure runStrategy, it needs to auto-start
			err = c.addStartRequest(vm)
			if err != nil {
				return vm, nil, err
			}
		}
		vm.Status.Conditions = append(vm.Status.Conditions, virtv1.VirtualMachineCondition{
			Type:   virtv1.VirtualMachineInitialized,
			Status: k8score.ConditionTrue,
		})
	}

	if vm.DeletionTimestamp != nil {
		if vmi == nil || controller.HasFinalizer(vm, v1.FinalizerOrphanDependents) {
			vm, err = c.removeVMFinalizer(vm, virtv1.VirtualMachineControllerFinalizer)
			if err != nil {
				return vm, nil, err
			}
		} else {
			err = c.stopVMI(vm, vmi)
			if err != nil {
				log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
				return vm, &syncErrorImpl{fmt.Errorf(failureDeletingVmiErrFormat, err), VMIFailedDeleteReason}, nil
			}
		}
		return vm, nil, nil
	} else {
		vm, err = c.addVMFinalizer(vm, virtv1.VirtualMachineControllerFinalizer)
		if err != nil {
			return vm, nil, err
		}
	}

	if err := c.conditionallyBumpGenerationAnnotationOnVmi(vm, vmi); err != nil {
		return nil, nil, err
	}

	// Scale up or down, if all expected creates and deletes were report by the listener
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return vm, nil, err
	}

	// Ensure we have ControllerRevisions of any instancetype or preferences referenced by the VM
	err = c.instancetypeMethods.StoreControllerRevisions(vm)
	if err != nil {
		log.Log.Object(vm).Infof("Failed to store Instancetype ControllerRevisions for VirtualMachine: %s/%s", vm.Namespace, vm.Name)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedCreateVirtualMachineReason, "Error encountered while storing Instancetype ControllerRevisions: %v", err)
		syncErr = &syncErrorImpl{fmt.Errorf("Error encountered while storing Instancetype ControllerRevisions: %v", err), FailedCreateVirtualMachineReason}
	}

	if syncErr == nil {
		dataVolumesReady, err := c.handleDataVolumes(vm, dataVolumes)
		if err != nil {
			syncErr = &syncErrorImpl{fmt.Errorf("Error encountered while creating DataVolumes: %v", err), FailedCreateReason}
		} else if dataVolumesReady || runStrategy == virtv1.RunStrategyHalted {
			syncErr = c.startStop(vm, vmi)
		} else {
			log.Log.Object(vm).V(3).Infof("Waiting on DataVolumes to be ready. %d datavolumes found", len(dataVolumes))
		}
	}

	restartRequired := c.addRestartRequiredIfNeeded(lastSeenVMSpec, vm)

	// Must check needsSync again here because a VMI can be created or
	// deleted in the startStop function which impacts how we process
	// hotplugged volumes and interfaces
	if c.needsSync(key) && syncErr == nil {
		vmCopy := vm.DeepCopy()
		if c.clusterConfig.HotplugNetworkInterfacesEnabled() &&
			vmi != nil && vmi.DeletionTimestamp == nil {
			vmiCopy := vmi.DeepCopy()

			indexedStatusIfaces := vmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces,
				func(ifaceStatus virtv1.VirtualMachineInstanceNetworkInterface) bool { return true })

			ifaces, networks := clearDetachedInterfaces(vmCopy.Spec.Template.Spec.Domain.Devices.Interfaces, vmCopy.Spec.Template.Spec.Networks, indexedStatusIfaces)
			vmCopy.Spec.Template.Spec.Domain.Devices.Interfaces = ifaces
			vmCopy.Spec.Template.Spec.Networks = networks

			ifaces, networks = clearDetachedInterfaces(vmiCopy.Spec.Domain.Devices.Interfaces, vmiCopy.Spec.Networks, indexedStatusIfaces)
			vmiCopy.Spec.Domain.Devices.Interfaces = ifaces
			vmiCopy.Spec.Networks = networks

			if hasOrdinalIfaces, err := c.hasOrdinalNetworkInterfaces(vmi); err != nil {
				syncErr = &syncErrorImpl{fmt.Errorf("Error encountered when trying to check if VMI has interface with ordinal names (e.g.: eth1, eth2..): %v", err), HotPlugNetworkInterfaceErrorReason}
			} else {
				updatedVmiSpec := applyDynamicIfaceRequestOnVMI(vmCopy, vmiCopy, hasOrdinalIfaces)
				vmiCopy.Spec = *updatedVmiSpec
			}

			if syncErr == nil {
				if err = c.vmiInterfacesPatch(&vmiCopy.Spec, vmi); err != nil {
					syncErr = &syncErrorImpl{fmt.Errorf("Error encountered when trying to patch vmi: %v", err), FailedUpdateErrorReason}
				}
			}
		}

		err = c.handleVolumeRequests(vmCopy, vmi)
		if err != nil {
			syncErr = &syncErrorImpl{fmt.Errorf("Error encountered while handling volume hotplug requests: %v", err), HotPlugVolumeErrorReason}
		} else {
			err = c.handleMemoryDumpRequest(vmCopy, vmi)
			if err != nil {
				syncErr = &syncErrorImpl{fmt.Errorf("Error encountered while handling memory dump request: %v", err), MemoryDumpErrorReason}
			}
		}

		if c.clusterConfig.IsVMRolloutStrategyLiveUpdate() && !restartRequired && !conditionManager.HasCondition(vm, virtv1.VirtualMachineRestartRequired) {
			err = c.handleCPUChangeRequest(vmCopy, vmi)
			if err != nil {
				syncErr = &syncErrorImpl{fmt.Errorf("Error encountered while handling CPU change request: %v", err), HotPlugCPUErrorReason}
			}

			if err := c.handleAffinityChangeRequest(vmCopy, vmi); err != nil {
				syncErr = &syncErrorImpl{fmt.Errorf("Error encountered while handling node affinity change request: %v", err), AffinityChangeErrorReason}
			}

			if err := c.handleMemoryHotplugRequest(vmCopy, vmi); err != nil {
				syncErr = &syncErrorImpl{
					err:    fmt.Errorf("error encountered while handling memory hotplug requests: %v", err),
					reason: HotPlugMemoryErrorReason,
				}
			}
		}

		if syncErr == nil {
			if !equality.Semantic.DeepEqual(vm, vmCopy) {
				vm, err = c.clientset.VirtualMachine(vmCopy.Namespace).Update(context.Background(), vmCopy)
				if err != nil {
					syncErr = &syncErrorImpl{fmt.Errorf("Error encountered when trying to update vm according to add volume and/or memory dump requests: %v", err), FailedUpdateErrorReason}
				}
			}
		}
	}

	// Adding last-seen VM revision only once for now, since we don't yet have a revert process.
	// Remove the hasLastSeenRevision() check to keep the last-seen CR up-to-date.
	hasIt, err := c.hasLastSeenRevision(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedGetLastSeenCRforVmErrMsg)
		return vm, syncErr, err
	}
	if !hasIt {
		_, err = c.createVMRevision(vm, revisionPrefixLastSeen)
	}
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedCreateLastSeenCRforVmErrMsg)
		return vm, syncErr, err
	}

	virtControllerVMWorkQueueTracer.StepTrace(key, "sync", trace.Field{Key: "VM Name", Value: vm.Name})

	return vm, syncErr, nil
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

func autoAttachInputDevice(vmi *virtv1.VirtualMachineInstance) {
	autoAttachInput := vmi.Spec.Domain.Devices.AutoattachInputDevice
	// Default to False if nil and return, otherwise return if input devices are already present
	if autoAttachInput == nil || *autoAttachInput == false || len(vmi.Spec.Domain.Devices.Inputs) > 0 {
		return
	}
	// Only add the device with an alias here. Preferences for the bus and type might
	// be applied later and if not the VMI mutation webhook will apply defaults for both.
	vmi.Spec.Domain.Devices.Inputs = append(
		vmi.Spec.Domain.Devices.Inputs,
		virtv1.Input{
			Name: "default-0",
		},
	)
}

func (c *VMController) applyDevicePreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
	if vm.Spec.Preference != nil {
		preferenceSpec, err := c.instancetypeMethods.FindPreferenceSpec(vm)
		if err != nil {
			return nil, err
		}
		instancetype.ApplyDevicePreferences(preferenceSpec, &vmi.Spec)

		return preferenceSpec, nil
	}
	return nil, nil
}

func (c *VMController) hasOrdinalNetworkInterfaces(vmi *virtv1.VirtualMachineInstance) (bool, error) {
	pod, err := controller.CurrentVMIPod(vmi, c.podInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to fetch pod from cache.")
		return false, err
	}
	if pod == nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to find VMI pod in cache.")
		return false, err
	}
	hasOrdinalIfaces := namescheme.PodHasOrdinalInterfaceName(services.NonDefaultMultusNetworksIndexedByIfaceName(pod))
	return hasOrdinalIfaces, nil
}

func (c *VMController) handleMemoryHotplugRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	if vm.Spec.Template.Spec.Domain.Memory == nil || vmi.Spec.Domain.Memory == nil {
		return nil
	}

	guestMemory := vmi.Spec.Domain.Memory.Guest

	if vmi.Status.Memory == nil ||
		vmi.Status.Memory.GuestCurrent == nil ||
		vm.Spec.Template.Spec.Domain.Memory.Guest.Equal(*guestMemory) {
		return nil
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()
	if conditionManager.HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstanceMemoryChange, k8score.ConditionTrue) {
		return fmt.Errorf("another memory hotplug is in progress")
	}

	if migrations.IsMigrating(vmi) {
		return fmt.Errorf("memory hotplug is not allowed while VMI is migrating")
	}

	// If the following is true, MaxGuest was calculated, not manually specified (or the validation webhook would have rejected the change).
	// Since we're here, we can also assume MaxGuest was not changed in the VM spec since last boot.
	// Therefore, bumping Guest to a value higher than MaxGuest is fine, it just requires a reboot.
	if vm.Spec.Template.Spec.Domain.Memory.Guest != nil && vmi.Spec.Domain.Memory.MaxGuest != nil &&
		vm.Spec.Template.Spec.Domain.Memory.Guest.Cmp(*vmi.Spec.Domain.Memory.MaxGuest) == 1 {
		vmConditions := controller.NewVirtualMachineConditionManager()
		vmConditions.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
			Type:               virtv1.VirtualMachineRestartRequired,
			LastTransitionTime: v1.Now(),
			Status:             k8score.ConditionTrue,
			Message:            "memory updated in template spec to a value higher than what's available",
		})
		return nil
	}

	newMemoryReq := vm.Spec.Template.Spec.Domain.Memory.Guest.DeepCopy()
	newMemoryReq.Sub(*vmi.Status.Memory.GuestCurrent)
	newMemoryReq.Add(*vmi.Spec.Domain.Resources.Requests.Memory())

	guestTest := fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/memory/guest", "value": "%s"}`, vmi.Spec.Domain.Memory.Guest.String())
	updateGuest := fmt.Sprintf(`{ "op": "replace", "path": "/spec/domain/memory/guest", "value": "%s"}`, vm.Spec.Template.Spec.Domain.Memory.Guest.String())
	MemoryReqTest := fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/resources/requests/memory", "value": "%s"}`, vmi.Spec.Domain.Resources.Requests.Memory().String())
	updateMemoryReq := fmt.Sprintf(`{ "op": "replace", "path": "/spec/domain/resources/requests/memory", "value": "%s"}`, newMemoryReq.String())
	patch := fmt.Sprintf(`[%s, %s, %s, %s]`, guestTest, updateGuest, MemoryReqTest, updateMemoryReq)

	_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &v1.PatchOptions{})
	if err != nil {
		return err
	}

	log.Log.Object(vmi).V(4).Infof("hotplugging memory to %s", vm.Spec.Template.Spec.Domain.Memory.Guest.String())

	return nil
}

func (c *VMController) vmiInterfacesPatch(newVmiSpec *virtv1.VirtualMachineInstanceSpec, vmi *virtv1.VirtualMachineInstance) error {
	if equality.Semantic.DeepEqual(vmi.Spec.Domain.Devices.Interfaces, newVmiSpec.Domain.Devices.Interfaces) {
		return nil
	}

	oldIfacesJSON, err := json.Marshal(vmi.Spec.Domain.Devices.Interfaces)
	if err != nil {
		return err
	}

	newIfacesJSON, err := json.Marshal(newVmiSpec.Domain.Devices.Interfaces)
	if err != nil {
		return err
	}

	oldNetworksJSON, err := json.Marshal(vmi.Spec.Networks)
	if err != nil {
		return err
	}

	newNetworksJSON, err := json.Marshal(newVmiSpec.Networks)
	if err != nil {
		return err
	}

	const verb = "add"
	testNetworks := fmt.Sprintf(`{ "op": "test", "path": "/spec/networks", "value": %s}`, string(oldNetworksJSON))
	updateNetworks := fmt.Sprintf(`{ "op": %q, "path": "/spec/networks", "value": %s}`, verb, string(newNetworksJSON))

	testInterfaces := fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/devices/interfaces", "value": %s}`, string(oldIfacesJSON))
	updateInterfaces := fmt.Sprintf(`{ "op": %q, "path": "/spec/domain/devices/interfaces", "value": %s}`, verb, string(newIfacesJSON))

	patch := fmt.Sprintf("[%s, %s, %s, %s]", testNetworks, testInterfaces, updateNetworks, updateInterfaces)

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, []byte(patch), &v1.PatchOptions{})

	return err
}

func (c *VMController) setupHotplug(vmi, VMIDefaults *virtv1.VirtualMachineInstance) {
	if !c.clusterConfig.IsVMRolloutStrategyLiveUpdate() {
		return
	}

	maxRatio := c.clusterConfig.GetMaxHotplugRatio()

	c.setupCPUHotplug(vmi, VMIDefaults, maxRatio)
	c.setupMemoryHotplug(vmi, maxRatio)
}

func (c *VMController) patchVMITerminationGracePeriod(gracePeriod *int64, vmi *virtv1.VirtualMachineInstance) error {
	if gracePeriod == nil {
		return nil
	}
	patch := fmt.Sprintf(`{"spec":{"terminationGracePeriodSeconds": %d }}`, *gracePeriod)
	_, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.MergePatchType, []byte(patch), &v1.PatchOptions{})
	return err
}
