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
 * Copyright The KubeVirt Authors.
 *
 */

package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/liveupdate/memory"

	netadmitter "kubevirt.io/kubevirt/pkg/network/admitter"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	netvmliveupdate "kubevirt.io/kubevirt/pkg/network/vmliveupdate"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"

	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/storage/memorydump"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	volumemig "kubevirt.io/kubevirt/pkg/virt-controller/watch/volume-migration"
)

const (
	fetchingRunStrategyErrFmt = "Error fetching RunStrategy: %v"
	fetchingVMKeyErrFmt       = "Error fetching vmKey: %v"
	startingVMIFailureFmt     = "Failure while starting VMI: %v"
)

type CloneAuthFunc func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error)

// Repeating info / error messages
const (
	stoppingVmMsg                             = "Stopping VM"
	startingVmMsg                             = "Starting VM"
	failedExtractVmkeyFromVmErrMsg            = "Failed to extract vmKey from VirtualMachine."
	failedCreateCRforVmErrMsg                 = "Failed to create controller revision for VirtualMachine."
	failedProcessDeleteNotificationErrMsg     = "Failed to process delete notification"
	failureDeletingVmiErrFormat               = "Failure attempting to delete VMI: %v"
	failedCleanupRestartRequired              = "Failed to delete RestartRequired condition or last-seen controller revisions"
	failedManualRecoveryRequiredCondSetErrMsg = "cannot start the VM since it has the manual recovery required condtion set"

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
	hotplugVolumeErrorReason     = "HotPlugVolumeError"
	hotplugCPUErrorReason        = "HotPlugCPUError"
	failedUpdateErrorReason      = "FailedUpdateError"
	failedCreateReason           = "FailedCreate"
	vmiFailedDeleteReason        = "FailedDelete"
	affinityChangeErrorReason    = "AffinityChangeError"
	hotplugMemoryErrorReason     = "HotPlugMemoryError"
	volumesUpdateErrorReason     = "VolumesUpdateError"
	tolerationsChangeErrorReason = "TolerationsChangeError"
)

const defaultMaxCrashLoopBackoffDelaySeconds = 300

func NewController(vmiInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	dataVolumeInformer cache.SharedIndexInformer,
	dataSourceInformer cache.SharedIndexInformer,
	namespaceStore cache.Store,
	pvcInformer cache.SharedIndexInformer,
	crInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
	netSynchronizer synchronizer,
	firmwareSynchronizer synchronizer,
	instancetypeController instancetypeHandler,
) (*Controller, error) {

	c := &Controller{
		Queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-vm"},
		),
		vmiIndexer:             vmiInformer.GetIndexer(),
		vmIndexer:              vmInformer.GetIndexer(),
		dataVolumeStore:        dataVolumeInformer.GetStore(),
		dataSourceStore:        dataSourceInformer.GetStore(),
		namespaceStore:         namespaceStore,
		pvcStore:               pvcInformer.GetStore(),
		crIndexer:              crInformer.GetIndexer(),
		instancetypeController: instancetypeController,
		recorder:               recorder,
		clientset:              clientset,
		expectations:           controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		dataVolumeExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		cloneAuthFunc: func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error) {
			response, err := dv.AuthorizeSA(requestNamespace, requestName, proxy, saNamespace, saName)
			return response.Allowed, response.Reason, err
		},
		clusterConfig:        clusterConfig,
		netSynchronizer:      netSynchronizer,
		firmwareSynchronizer: firmwareSynchronizer,
	}

	c.hasSynced = func() bool {
		return vmiInformer.HasSynced() && vmInformer.HasSynced() &&
			dataVolumeInformer.HasSynced() && dataSourceInformer.HasSynced() &&
			pvcInformer.HasSynced() && crInformer.HasSynced()
	}

	_, err := vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachine,
		DeleteFunc: c.deleteVirtualMachine,
		UpdateFunc: c.updateVirtualMachine,
	})
	if err != nil {
		return nil, err
	}

	_, err = vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})
	if err != nil {
		return nil, err
	}

	_, err = dataVolumeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	client          kubecli.KubevirtClient
	dataSourceStore cache.Store
	namespaceStore  cache.Store
}

func (p *authProxy) CreateSar(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
	return p.client.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
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
	obj, exists, err := p.dataSourceStore.GetByKey(key)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("dataSource %s does not exist", key)
	}

	ds := obj.(*cdiv1.DataSource).DeepCopy()
	return ds, nil
}

type synchronizer interface {
	Sync(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
}

type instancetypeHandler interface {
	synchronizer
	ApplyToVM(*virtv1.VirtualMachine) error
	ApplyToVMI(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error
	ApplyDevicePreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error
}

type Controller struct {
	clientset              kubecli.KubevirtClient
	Queue                  workqueue.TypedRateLimitingInterface[string]
	vmiIndexer             cache.Indexer
	vmIndexer              cache.Indexer
	dataVolumeStore        cache.Store
	dataSourceStore        cache.Store
	namespaceStore         cache.Store
	pvcStore               cache.Store
	crIndexer              cache.Indexer
	instancetypeController instancetypeHandler
	recorder               record.EventRecorder
	expectations           *controller.UIDTrackingControllerExpectations
	dataVolumeExpectations *controller.UIDTrackingControllerExpectations
	cloneAuthFunc          CloneAuthFunc
	clusterConfig          *virtconfig.ClusterConfig
	hasSynced              func() bool

	netSynchronizer      synchronizer
	firmwareSynchronizer synchronizer
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.Queue.ShutDown()
	log.Log.Info("Starting VirtualMachine controller.")

	// Wait for cache sync before we start the controller
	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping VirtualMachine controller.")
}

func (c *Controller) runWorker() {
	for c.Execute() {
	}
}

func (c *Controller) satisfiedExpectations(key string) bool {
	return c.expectations.SatisfiedExpectations(key) && c.dataVolumeExpectations.SatisfiedExpectations(key)
}

var virtControllerVMWorkQueueTracer = &traceUtils.Tracer{Threshold: time.Second}

func (c *Controller) Execute() bool {
	key, quit := c.Queue.Get()
	if quit {
		return false
	}

	virtControllerVMWorkQueueTracer.StartTrace(key, "virt-controller VM workqueue", trace.Field{Key: "Workqueue Key", Value: key})
	defer virtControllerVMWorkQueueTracer.StopTrace(key)

	defer c.Queue.Done(key)

	if err := c.execute(key); err != nil {
		log.Log.Reason(err).Infof("re-enqueuing VirtualMachine %v", key)
		c.Queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachine %v", key)
		c.Queue.Forget(key)
	}
	return true
}

func (c *Controller) execute(key string) error {
	obj, exists, err := c.vmIndexer.GetByKey(key)
	if err != nil {
		return nil
	}
	if !exists {
		// nothing we need to do. It should always be possible to re-create this type of controller
		c.expectations.DeleteExpectations(key)
		return nil
	}
	originalVM := obj.(*virtv1.VirtualMachine)
	vm := originalVM.DeepCopy()

	logger := log.Log.Object(vm)

	logger.V(4).Info("Started processing vm")

	// this must be first step in execution. Writing the object
	// when api version changes ensures our api stored version is updated.
	if !controller.ObservedLatestApiVersionAnnotation(vm) {
		controller.SetLatestApiVersionAnnotation(vm)
		_, err = c.clientset.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})

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
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := c.clientset.VirtualMachine(vm.ObjectMeta.Namespace).Get(context.Background(), vm.ObjectMeta.Name, metav1.GetOptions{})
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
	vmiObj, exist, err := c.vmiIndexer.GetByKey(vmKey)
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

	dataVolumes, err := storagetypes.ListDataVolumesFromTemplates(vm.Namespace, vm.Spec.DataVolumeTemplates, c.dataVolumeStore)
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

	var syncErr common.SyncError

	vm, vmi, syncErr, err = c.sync(vm, vmi, key)
	if err != nil {
		return err
	}

	if syncErr != nil {
		logger.Reason(syncErr).Error("Reconciling the VirtualMachine failed.")
	}

	err = c.updateStatus(vm, originalVM, vmi, syncErr, logger)
	if err != nil {
		logger.Reason(err).Error("Updating the VirtualMachine status failed.")
		return err
	}

	return syncErr
}

func (c *Controller) handleCloneDataVolume(vm *virtv1.VirtualMachine, dv *cdiv1.DataVolume) error {
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
		pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(dv.Spec.Source.PVC.Namespace, dv.Spec.Source.PVC.Name, c.pvcStore)
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

func (c *Controller) authorizeDataVolume(vm *virtv1.VirtualMachine, dataVolume *cdiv1.DataVolume) error {
	serviceAccountName := "default"
	for _, vol := range vm.Spec.Template.Spec.Volumes {
		if vol.ServiceAccount != nil {
			serviceAccountName = vol.ServiceAccount.ServiceAccountName
		}
	}

	proxy := &authProxy{client: c.clientset, dataSourceStore: c.dataSourceStore, namespaceStore: c.namespaceStore}
	allowed, reason, err := c.cloneAuthFunc(dataVolume, vm.Namespace, dataVolume.Name, proxy, vm.Namespace, serviceAccountName)
	if err != nil && err != cdiv1.ErrNoTokenOkay {
		return err
	}

	if !allowed {
		return fmt.Errorf(reason)
	}

	return nil
}

func (c *Controller) handleDataVolumes(vm *virtv1.VirtualMachine) (bool, error) {
	ready := true
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		return ready, err
	}
	for _, template := range vm.Spec.DataVolumeTemplates {
		curDataVolume, err := storagetypes.GetDataVolumeFromCache(vm.Namespace, template.Name, c.dataVolumeStore)
		if err != nil {
			return false, err
		}
		if curDataVolume == nil {
			// Don't create DV if PVC already exists
			pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vm.Namespace, template.Name, c.pvcStore)
			if err != nil {
				return false, err
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
			curDataVolume, err = c.clientset.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), newDataVolume, metav1.CreateOptions{})
			if err != nil {
				c.dataVolumeExpectations.CreationObserved(vmKey)
				if pvc != nil && strings.Contains(err.Error(), "already exists") {
					// If the PVC already exists, we can ignore the error and continue
					// probably old version of CDI
					log.Log.Object(vm).Reason(err).Warning("Appear to be running a version of CDI that does not support claim adoption annotation")
					continue
				}
				c.recorder.Eventf(vm, k8score.EventTypeWarning, FailedDataVolumeCreateReason, "Error creating DataVolume %s: %v", newDataVolume.Name, err)
				return ready, fmt.Errorf("failed to create DataVolume: %v", err)
			}
			c.recorder.Eventf(vm, k8score.EventTypeNormal, SuccessfulDataVolumeCreateReason, "Created DataVolume %s", curDataVolume.Name)
		} else {
			switch curDataVolume.Status.Phase {
			case cdiv1.Succeeded, cdiv1.WaitForFirstConsumer, cdiv1.PendingPopulation:
				continue
			case cdiv1.Failed:
				c.recorder.Eventf(vm, k8score.EventTypeWarning, controller.FailedDataVolumeImportReason, "DataVolume %s failed to import disk image", curDataVolume.Name)
			case cdiv1.Pending:
				if err := storagetypes.HasDataVolumeExceededQuotaError(curDataVolume); err != nil {
					c.recorder.Eventf(vm, k8score.EventTypeWarning, controller.FailedDataVolumeImportReason, "DataVolume %s exceeds quota limits", curDataVolume.Name)
					return false, err
				}
			}
			// ready = false because encountered DataVolume that is not populated yet
			ready = false
		}
	}
	return ready, nil
}

func (c *Controller) VMICPUsPatch(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	patchSet := patch.New(
		patch.WithTest("/spec/domain/cpu/sockets", vmi.Spec.Domain.CPU.Sockets),
		patch.WithReplace("/spec/domain/cpu/sockets", vm.Spec.Template.Spec.Domain.CPU.Sockets),
	)

	vcpusDelta := hardware.GetNumberOfVCPUs(vm.Spec.Template.Spec.Domain.CPU) - hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU)
	resourcesDelta := resource.NewMilliQuantity(vcpusDelta*int64(1000/c.clusterConfig.GetCPUAllocationRatio()), resource.DecimalSI)

	logMsg := fmt.Sprintf("hotplugging cpu to %v sockets", vm.Spec.Template.Spec.Domain.CPU.Sockets)

	if !vm.Spec.Template.Spec.Domain.Resources.Requests.Cpu().IsZero() {
		newCpuReq := vmi.Spec.Domain.Resources.Requests.Cpu().DeepCopy()
		newCpuReq.Add(*resourcesDelta)

		patchSet.AddOption(
			patch.WithTest("/spec/domain/resources/requests/cpu", vmi.Spec.Domain.Resources.Requests.Cpu().String()),
			patch.WithReplace("/spec/domain/resources/requests/cpu", newCpuReq.String()),
		)

		logMsg = fmt.Sprintf("%s, setting requests to %s", logMsg, newCpuReq.String())
	}
	if !vm.Spec.Template.Spec.Domain.Resources.Limits.Cpu().IsZero() {
		newCpuLimit := vmi.Spec.Domain.Resources.Limits.Cpu().DeepCopy()
		newCpuLimit.Add(*resourcesDelta)

		patchSet.AddOption(
			patch.WithTest("/spec/domain/resources/limits/cpu", vmi.Spec.Domain.Resources.Limits.Cpu().String()),
			patch.WithReplace("/spec/domain/resources/limits/cpu", newCpuLimit.String()),
		)

		logMsg = fmt.Sprintf("%s, setting limits to %s", logMsg, newCpuLimit.String())
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err == nil {
		log.Log.Object(vmi).Infof(logMsg)
	}

	return err
}

func (c *Controller) handleCPUChangeRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	vmCopyWithInstancetype := vm.DeepCopy()
	if err := c.instancetypeController.ApplyToVM(vmCopyWithInstancetype); err != nil {
		return err
	}

	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU == nil {
		return nil
	}

	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.CPU.Sockets == vmi.Spec.Domain.CPU.Sockets {
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
	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.CPU.Sockets > vmi.Spec.Domain.CPU.MaxSockets {
		setRestartRequired(vm, "CPU sockets updated in template spec to a value higher than what's available")
		return nil
	}

	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.CPU.Sockets < vmi.Spec.Domain.CPU.Sockets {
		setRestartRequired(vm, "Reduction of CPU socket count requires a restart")
		return nil
	}

	networkInterfaceMultiQueue := vmCopyWithInstancetype.Spec.Template.Spec.Domain.Devices.NetworkInterfaceMultiQueue
	if networkInterfaceMultiQueue != nil && *networkInterfaceMultiQueue {
		setRestartRequired(vm, "Changes to CPU sockets require a restart when NetworkInterfaceMultiQueue is enabled")
		return nil
	}

	if err := c.VMICPUsPatch(vmCopyWithInstancetype, vmi); err != nil {
		log.Log.Object(vmi).Errorf("unable to patch vmi to add cpu topology status: %v", err)
		return err
	}

	return nil
}

func (c *Controller) VMNodeSelectorPatch(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	patchset := patch.New()
	if vm.Spec.Template.Spec.NodeSelector != nil {
		vmNodeSelector := maps.Clone(vm.Spec.Template.Spec.NodeSelector)
		if vmNodeSelector == nil {
			vmNodeSelector = make(map[string]string)
		}
		if vmi.Spec.NodeSelector == nil {
			patchset.AddOption(patch.WithAdd("/spec/nodeSelector", vmNodeSelector))
		} else {
			patchset.AddOption(
				patch.WithTest("/spec/nodeSelector", vmi.Spec.NodeSelector),
				patch.WithReplace("/spec/nodeSelector", vmNodeSelector))
		}
	} else {
		patchset.AddOption(patch.WithRemove("/spec/nodeSelector"))
	}
	generatedPatch, err := patchset.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, generatedPatch, metav1.PatchOptions{})
	return err
}

func (c *Controller) VMIAffinityPatch(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	patchset := patch.New()
	if vm.Spec.Template.Spec.Affinity != nil {
		if vmi.Spec.Affinity == nil {
			patchset.AddOption(patch.WithAdd("/spec/affinity", vm.Spec.Template.Spec.Affinity))
		} else {
			patchset.AddOption(
				patch.WithTest("/spec/affinity", vmi.Spec.Affinity),
				patch.WithReplace("/spec/affinity", vm.Spec.Template.Spec.Affinity))
		}

	} else {
		patchset.AddOption(patch.WithRemove("/spec/affinity"))
	}
	generatedPatch, err := patchset.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, generatedPatch, metav1.PatchOptions{})
	return err
}

func (c *Controller) vmiTolerationsPatch(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	patchset := patch.New()

	if vm.Spec.Template.Spec.Tolerations != nil {
		if vmi.Spec.Tolerations == nil {
			patchset.AddOption(patch.WithAdd("/spec/tolerations", vm.Spec.Template.Spec.Tolerations))
		} else {
			patchset.AddOption(
				patch.WithTest("/spec/tolerations", vmi.Spec.Tolerations),
				patch.WithReplace("/spec/tolerations", vm.Spec.Template.Spec.Tolerations))
		}

	} else {
		patchset.AddOption(patch.WithRemove("/spec/tolerations"))
	}

	generatedPatch, err := patchset.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, generatedPatch, metav1.PatchOptions{})
	return err
}

func (c *Controller) handleTolerationsChangeRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	vmCopyWithInstancetype := vm.DeepCopy()
	if err := c.instancetypeController.ApplyToVM(vmCopyWithInstancetype); err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(vmCopyWithInstancetype.Spec.Template.Spec.Tolerations, vmi.Spec.Tolerations) {
		return nil
	}

	if migrations.IsMigrating(vmi) {
		return fmt.Errorf("tolerations should not be changed during VMI migration")
	}

	if err := c.vmiTolerationsPatch(vmCopyWithInstancetype, vmi); err != nil {
		log.Log.Object(vmi).Errorf("unable to patch vmi to update tolerations: %v", err)
		return err
	}

	return nil
}

func (c *Controller) handleAffinityChangeRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	vmCopyWithInstancetype := vm.DeepCopy()
	if err := c.instancetypeController.ApplyToVM(vmCopyWithInstancetype); err != nil {
		return err
	}

	hasNodeSelectorChanged := !equality.Semantic.DeepEqual(vmCopyWithInstancetype.Spec.Template.Spec.NodeSelector, vmi.Spec.NodeSelector)
	hasNodeAffinityChanged := !equality.Semantic.DeepEqual(vmCopyWithInstancetype.Spec.Template.Spec.Affinity, vmi.Spec.Affinity)

	if migrations.IsMigrating(vmi) && (hasNodeSelectorChanged || hasNodeAffinityChanged) {
		return fmt.Errorf("Node affinity should not be changed during VMI migration")
	}

	if hasNodeAffinityChanged {
		if err := c.VMIAffinityPatch(vmCopyWithInstancetype, vmi); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to update node affinity: %v", err)
			return err
		}
	}

	if hasNodeSelectorChanged {
		if err := c.VMNodeSelectorPatch(vmCopyWithInstancetype, vmi); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to update node selector: %v", err)
			return err
		}
	}
	return nil
}

func (c *Controller) handleVolumeRequests(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
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

func (c *Controller) handleVolumeUpdateRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil {
		return nil
	}

	// The pull policy for container disks are only set on the VMI spec and not on the VM spec.
	// In order to correctly compare the volumes set, we need to set the pull policy on the VM spec as well.
	vmCopy := vm.DeepCopy()
	volsVMI := storagetypes.GetVolumesByName(&vmi.Spec)
	for i, volume := range vmCopy.Spec.Template.Spec.Volumes {
		vmiVol, ok := volsVMI[volume.Name]
		if !ok {
			continue
		}
		if vmiVol.ContainerDisk != nil {
			vmCopy.Spec.Template.Spec.Volumes[i].ContainerDisk.ImagePullPolicy = vmiVol.ContainerDisk.ImagePullPolicy
		}
	}
	hotplugOp := false
	volsVM := storagetypes.GetVolumesByName(&vmCopy.Spec.Template.Spec)
	for _, volume := range vmi.Spec.Volumes {
		hotpluggableVol := (volume.VolumeSource.PersistentVolumeClaim != nil &&
			volume.VolumeSource.PersistentVolumeClaim.Hotpluggable) ||
			(volume.VolumeSource.DataVolume != nil && volume.VolumeSource.DataVolume.Hotpluggable)
		_, ok := volsVM[volume.Name]
		if !ok && hotpluggableVol {
			hotplugOp = true
		}
	}
	if hotplugOp {
		return nil
	}
	if equality.Semantic.DeepEqual(vmi.Spec.Volumes, vmCopy.Spec.Template.Spec.Volumes) {
		return nil
	}
	vmConditions := controller.NewVirtualMachineConditionManager()
	// Abort the volume migration if any of the previous migrated volumes
	// has changed
	if volMigAbort, err := volumemig.VolumeMigrationCancel(c.clientset, vmi, vm); volMigAbort {
		if err == nil {
			log.Log.Object(vm).Infof("Cancel volume migration")
		}
		return err
	}

	switch {
	case vm.Spec.UpdateVolumesStrategy == nil ||
		*vm.Spec.UpdateVolumesStrategy == virtv1.UpdateVolumesStrategyReplacement:
		if !vmConditions.HasCondition(vm, virtv1.VirtualMachineRestartRequired) {
			log.Log.Object(vm).Infof("Set restart required condition because of a volumes update")
			setRestartRequired(vm, "the volumes replacement is effective only after restart")
		}
	case *vm.Spec.UpdateVolumesStrategy == virtv1.UpdateVolumesStrategyMigration:
		// Validate if the update volumes can be migrated
		if err := volumemig.ValidateVolumes(vmi, vm, c.dataVolumeStore, c.pvcStore); err != nil {
			log.Log.Object(vm).Errorf("cannot migrate the VM. Volumes are invalid: %v", err)
			setRestartRequired(vm, err.Error())
			return nil
		}
		migVols, err := volumemig.GenerateMigratedVolumes(c.pvcStore, vmi, vm)
		if err != nil {
			log.Log.Object(vm).Errorf("failed to generate the migrating volumes for vm: %v", err)
			return err
		}
		if err := volumemig.ValidateVolumesUpdateMigration(vmi, vm, migVols); err != nil {
			log.Log.Object(vm).Errorf("cannot migrate the VMI: %v", err)
			setRestartRequired(vm, err.Error())
			return nil
		}
		if err := volumemig.PatchVMIStatusWithMigratedVolumes(c.clientset, migVols, vmi); err != nil {
			log.Log.Object(vm).Errorf("failed to update migrating volumes for vmi:%v", err)
			return err
		}
		log.Log.Object(vm).Infof("Updated migrating volumes in the status")
		if _, err := volumemig.PatchVMIVolumes(c.clientset, vmi, vm); err != nil {
			log.Log.Object(vm).Errorf("failed to update volumes for vmi:%v", err)
			return err
		}
		log.Log.Object(vm).Infof("Updated volumes for vmi")
		if vm.Status.VolumeUpdateState == nil {
			vm.Status.VolumeUpdateState = &virtv1.VolumeUpdateState{}
		}
		if len(migVols) > 0 {
			vm.Status.VolumeUpdateState.VolumeMigrationState = &virtv1.VolumeMigrationState{
				MigratedVolumes: migVols,
			}
		}

	default:
		return fmt.Errorf("updateVolumes strategy not recognized: %s", *vm.Spec.UpdateVolumesStrategy)
	}

	return nil
}

func (c *Controller) addStartRequest(vm *virtv1.VirtualMachine) error {
	desiredStateChangeRequests := append(vm.Status.StateChangeRequests, virtv1.VirtualMachineStateChangeRequest{Action: virtv1.StartRequest})
	patchSet := patch.New()
	patchSet.AddOption(patch.WithAdd("/status/stateChangeRequests", desiredStateChangeRequests))
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	patchedVM, err := c.clientset.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	vm.Status = patchedVM.Status
	return nil
}

func (c *Controller) syncRunStrategy(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, runStrategy virtv1.VirtualMachineRunStrategy) (*virtv1.VirtualMachine, common.SyncError) {
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Errorf(fetchingVMKeyErrFmt, err)
		return vm, common.NewSyncError(err, failedCreateReason)
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
				vm, err = c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return vm, common.NewSyncError(fmt.Errorf(failureDeletingVmiErrFormat, err), vmiFailedDeleteReason)
				}
				// return to let the controller pick up the expected deletion
			}
			// VirtualMachineInstance is OK no need to do anything
			return vm, nil
		}

		timeLeft := startFailureBackoffTimeLeft(vm)
		if timeLeft > 0 {
			log.Log.Object(vm).Infof("Delaying start of VM %s with 'runStrategy: %s' due to start failure backoff. Waiting %d more seconds before starting.", startingVmMsg, runStrategy, timeLeft)
			c.Queue.AddAfter(vmKey, time.Duration(timeLeft)*time.Second)
			return vm, nil
		}

		log.Log.Object(vm).Infof("%s due to runStrategy: %s", startingVmMsg, runStrategy)
		vm, err = c.startVMI(vm)
		if err != nil {
			return vm, common.NewSyncError(fmt.Errorf(startingVMIFailureFmt, err), failedCreateReason)
		}
		return vm, nil

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
				vm, err = c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return vm, common.NewSyncError(fmt.Errorf(failureDeletingVmiErrFormat, err), vmiFailedDeleteReason)
				}

				if vmiFailed {
					if err := c.addStartRequest(vm); err != nil {
						return vm, common.NewSyncError(fmt.Errorf("failed to patch VM with start action: %v", err), vmiFailedDeleteReason)
					}
				}
			}
			// return to let the controller pick up the expected deletion
			return vm, nil
		}

		// when coming here from a different RunStrategy we have to start the VM
		if !hasStartRequest(vm) && vm.Status.RunStrategy == runStrategy {
			return vm, nil
		}

		timeLeft := startFailureBackoffTimeLeft(vm)
		if timeLeft > 0 {
			log.Log.Object(vm).Infof("Delaying start of VM %s with 'runStrategy: %s' due to start failure backoff. Waiting %d more seconds before starting.", startingVmMsg, runStrategy, timeLeft)
			c.Queue.AddAfter(vmKey, time.Duration(timeLeft)*time.Second)
			return vm, nil
		}

		log.Log.Object(vm).Infof("%s due to runStrategy: %s", startingVmMsg, runStrategy)
		vm, err = c.startVMI(vm)
		if err != nil {
			return vm, common.NewSyncError(fmt.Errorf(startingVMIFailureFmt, err), failedCreateReason)
		}
		return vm, nil

	case virtv1.RunStrategyManual:
		// For this RunStrategy, VMI's will be started/stopped/restarted using api endpoints only
		if vmi != nil {
			log.Log.Object(vm).V(4).Info("VMI exists")

			if forceStop := hasStopRequestForVMI(vm, vmi); forceStop {
				log.Log.Object(vm).Infof("%s with VMI in phase %s due to stop request and VM runStrategy: %s", vmi.Status.Phase, stoppingVmMsg, runStrategy)
				vm, err = c.stopVMI(vm, vmi)
				if err != nil {
					log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
					return vm, common.NewSyncError(fmt.Errorf(failureDeletingVmiErrFormat, err), vmiFailedDeleteReason)
				}
				// return to let the controller pick up the expected deletion
				return vm, nil
			}
		} else {
			if hasStartRequest(vm) {
				log.Log.Object(vm).Infof("%s due to start request and runStrategy: %s", startingVmMsg, runStrategy)

				vm, err = c.startVMI(vm)
				if err != nil {
					return vm, common.NewSyncError(fmt.Errorf(startingVMIFailureFmt, err), failedCreateReason)
				}
			}
		}
		return vm, nil

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
				_, err := c.clientset.VirtualMachine(vmCopy.Namespace).Update(context.Background(), vmCopy, metav1.UpdateOptions{})
				return vm, common.NewSyncError(fmt.Errorf(startingVMIFailureFmt, err), failedCreateReason)
			}
			return vm, nil
		}
		log.Log.Object(vm).Infof("%s with VMI in phase %s due to runStrategy: %s", stoppingVmMsg, vmi.Status.Phase, runStrategy)
		vm, err = c.stopVMI(vm, vmi)
		if err != nil {
			return vm, common.NewSyncError(fmt.Errorf(failureDeletingVmiErrFormat, err), vmiFailedDeleteReason)
		}
		return vm, nil
	case virtv1.RunStrategyOnce:
		if vmi == nil {
			log.Log.Object(vm).Infof("%s due to start request and runStrategy: %s", startingVmMsg, runStrategy)

			vm, err = c.startVMI(vm)
			if err != nil {
				return vm, common.NewSyncError(fmt.Errorf(startingVMIFailureFmt, err), failedCreateReason)
			}
		}

		return vm, nil
	default:
		return vm, common.NewSyncError(fmt.Errorf("unknown runstrategy: %s", runStrategy), failedCreateReason)
	}
}

// isVMIStartExpected determines whether a VMI is expected to be started for this VM.
func (c *Controller) isVMIStartExpected(vm *virtv1.VirtualMachine) bool {
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
func (c *Controller) isVMIStopExpected(vm *virtv1.VirtualMachine) bool {
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

func (c *Controller) cleanupRestartRequired(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	vmConditionManager := controller.NewVirtualMachineConditionManager()
	if vmConditionManager.HasCondition(vm, virtv1.VirtualMachineRestartRequired) {
		vmConditionManager.RemoveCondition(vm, virtv1.VirtualMachineRestartRequired)
	}

	return vm, c.deleteVMRevisions(vm)
}

func (c *Controller) startVMI(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	ready, err := c.handleDataVolumes(vm)
	if err != nil {
		return vm, err
	}

	if !ready {
		log.Log.Object(vm).V(4).Info("Waiting for DataVolumes to be created, delaying start")
		return vm, nil
	}

	if controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm, virtv1.VirtualMachineManualRecoveryRequired, k8score.ConditionTrue) {
		log.Log.Object(vm).Reason(err).Error(failedManualRecoveryRequiredCondSetErrMsg)
		return vm, nil
	}

	// TODO add check for existence
	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedExtractVmkeyFromVmErrMsg)
		return vm, nil
	}

	vm, err = c.cleanupRestartRequired(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedCleanupRestartRequired)
		return vm, err
	}

	// start it
	vmi := c.setupVMIFromVM(vm)
	vmRevisionName, err := c.createVMRevision(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedCreateCRforVmErrMsg)
		return vm, err
	}
	vmi.Status.VirtualMachineRevisionName = vmRevisionName

	setGenerationAnnotationOnVmi(vm.Generation, vmi)

	// add a finalizer to ensure the VM controller has a chance to see
	// the VMI before it is deleted
	vmi.Finalizers = append(vmi.Finalizers, virtv1.VirtualMachineControllerFinalizer)

	// We need to apply device preferences before any new network or input devices are added. Doing so allows
	// any autoAttach preferences we might have to be applied, either enabling or disabling the attachment of these devices.
	if err := c.instancetypeController.ApplyDevicePreferences(vm, vmi); err != nil {
		log.Log.Object(vm).Infof("Failed to apply device preferences again to VirtualMachineInstance: %s/%s", vmi.Namespace, vmi.Name)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, common.FailedCreateVirtualMachineReason, "Error applying device preferences again: %v", err)
		return vm, err
	}

	util.SetDefaultVolumeDisk(&vmi.Spec)

	autoAttachInputDevice(vmi)

	err = netvmispec.SetDefaultNetworkInterface(c.clusterConfig, &vmi.Spec)
	if err != nil {
		return vm, err
	}

	if err = c.instancetypeController.ApplyToVMI(vm, vmi); err != nil {
		log.Log.Object(vm).Infof("Failed to apply instancetype to VirtualMachineInstance: %s/%s", vmi.Namespace, vmi.Name)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, common.FailedCreateVirtualMachineReason, "Error creating virtual machine instance: Failed to apply instancetype: %v", err)
		return vm, err
	}

	netValidator := netadmitter.NewValidator(k8sfield.NewPath("spec"), &vmi.Spec, c.clusterConfig)
	var validateErrors []error
	for _, cause := range netValidator.ValidateCreation() {
		validateErrors = append(validateErrors, errors.New(cause.String()))
	}
	if validateErr := errors.Join(validateErrors...); validateErrors != nil {
		return vm, fmt.Errorf("failed create validation: %v", validateErr)
	}

	c.expectations.ExpectCreations(vmKey, 1)
	vmi, err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
	if err != nil {
		log.Log.Object(vm).Infof("Failed to create VirtualMachineInstance: %s", controller.NamespacedKey(vmi.Namespace, vmi.Name))
		c.expectations.CreationObserved(vmKey)
		c.recorder.Eventf(vm, k8score.EventTypeWarning, common.FailedCreateVirtualMachineReason, "Error creating virtual machine instance: %v", err)
		return vm, err
	}
	log.Log.Object(vm).Infof("Started VM by creating the new virtual machine instance %s", vmi.Name)
	c.recorder.Eventf(vm, k8score.EventTypeNormal, common.SuccessfulCreateVirtualMachineReason, "Started the virtual machine by creating the new virtual machine instance %v", vmi.ObjectMeta.Name)

	return vm, nil
}

func setGenerationAnnotation(generation int64, annotations map[string]string) {
	annotations[virtv1.VirtualMachineGenerationAnnotation] = strconv.FormatInt(generation, 10)
}

func setGenerationAnnotationOnVmi(generation int64, vmi *virtv1.VirtualMachineInstance) {
	annotations := vmi.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	setGenerationAnnotation(generation, annotations)
	vmi.SetAnnotations(annotations)
}

func (c *Controller) patchVmGenerationAnnotationOnVmi(generation int64, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstance, error) {
	oldAnnotations := vmi.Annotations
	newAnnotations := map[string]string{}
	maps.Copy(newAnnotations, oldAnnotations)
	setGenerationAnnotation(generation, newAnnotations)

	patchBytes, err := patch.New(
		patch.WithTest("/metadata/annotations", oldAnnotations),
		patch.WithReplace("/metadata/annotations", newAnnotations)).GeneratePayload()
	if err != nil {
		return vmi, err
	}
	patchedVMI, err := c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return vmi, err
	}

	return patchedVMI, nil
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
func (c *Controller) conditionallyBumpGenerationAnnotationOnVmi(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstance, error) {
	if vmi == nil || vm == nil {
		return vmi, nil
	}

	// If this is an old vmi created before a controller update, then the
	// annotation may not exist. In that case, continue on as if the generation
	// annotation needs to be bumped.
	currentGeneration, err := getGenerationAnnotation(vmi)
	if err != nil {
		return vmi, err
	}
	if currentGeneration != nil && *currentGeneration == strconv.FormatInt(vm.Generation, 10) {
		return vmi, nil
	}

	currentRevision, err := c.getControllerRevision(vmi.Namespace, vmi.Status.VirtualMachineRevisionName)
	if currentRevision == nil || err != nil {
		return vmi, err
	}

	revisionSpec := &VirtualMachineRevisionData{}
	if err = json.Unmarshal(currentRevision.Data.Raw, revisionSpec); err != nil {
		return vmi, err
	}

	// If the templates are the same, we can safely bump the annotation.
	if equality.Semantic.DeepEqual(revisionSpec.Spec.Template, vm.Spec.Template) {
		patchedVMI, err := c.patchVmGenerationAnnotationOnVmi(vm.Generation, vmi)
		if err != nil {
			return vmi, err
		}
		vmi = patchedVMI
	}

	return vmi, nil
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

	delaySeconds = interval * multiplier
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

		now := metav1.NewTime(time.Now())
		delaySeconds := calculateStartBackoffTime(count, defaultMaxCrashLoopBackoffDelaySeconds)
		retryAfter := metav1.NewTime(now.Time.Add(time.Duration(int64(delaySeconds)) * time.Second))

		vm.Status.StartFailure = &virtv1.VirtualMachineStartFailure{
			LastFailedVMIUID:     vmi.UID,
			RetryAfterTimestamp:  &retryAfter,
			ConsecutiveFailCount: count,
		}
	}
}

func syncVolumeMigration(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
	if vm.Status.VolumeUpdateState == nil || vm.Status.VolumeUpdateState.VolumeMigrationState == nil {
		return
	}
	vmCond := controller.NewVirtualMachineConditionManager()
	vmiCond := controller.NewVirtualMachineInstanceConditionManager()

	// Check if the volumes have been recovered and point to the original ones
	srcMigVols := make(map[string]string)
	for _, v := range vm.Status.VolumeUpdateState.VolumeMigrationState.MigratedVolumes {
		if v.SourcePVCInfo != nil {
			srcMigVols[v.VolumeName] = v.SourcePVCInfo.ClaimName
		}
	}
	recoveredOldVMVolumes := true
	for _, v := range vm.Spec.Template.Spec.Volumes {
		name := storagetypes.PVCNameFromVirtVolume(&v)
		origName, ok := srcMigVols[v.Name]
		if !ok {
			continue
		}
		if origName != name {
			recoveredOldVMVolumes = false
		}
	}
	if recoveredOldVMVolumes || (vm.Spec.UpdateVolumesStrategy == nil || *vm.Spec.UpdateVolumesStrategy != virtv1.UpdateVolumesStrategyMigration) {
		vm.Status.VolumeUpdateState.VolumeMigrationState = nil
		// Clean-up the volume change label when the volume set has been restored
		vmCond.RemoveCondition(vm, virtv1.VirtualMachineConditionType(virtv1.VirtualMachineInstanceVolumesChange))
		vmCond.RemoveCondition(vm, virtv1.VirtualMachineManualRecoveryRequired)
		return
	}
	if vmi == nil || vmi.IsFinal() {
		if vmCond.HasConditionWithStatus(vm, virtv1.VirtualMachineConditionType(virtv1.VirtualMachineInstanceVolumesChange), k8score.ConditionTrue) {
			// Something went wrong with the VMI while the volume migration was in progress
			vmCond.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
				Type:   virtv1.VirtualMachineManualRecoveryRequired,
				Status: k8score.ConditionTrue,
				Reason: "VMI was removed or was final during the volume migration",
			})
		}
		return
	}

	// The volume migration has been cancelled
	if cond := vmiCond.GetCondition(vmi, virtv1.VirtualMachineInstanceVolumesChange); cond != nil &&
		cond.Status == k8score.ConditionFalse &&
		cond.Reason == virtv1.VirtualMachineInstanceReasonVolumesChangeCancellation {
		vm.Status.VolumeUpdateState.VolumeMigrationState = nil
	}
}

// here is stop
func (c *Controller) stopVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		// nothing to do
		return vm, nil
	}

	vmKey, err := controller.KeyFunc(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedExtractVmkeyFromVmErrMsg)
		return vm, nil
	}

	// stop it
	c.expectations.ExpectDeletions(vmKey, []string{controller.VirtualMachineInstanceKey(vmi)})
	err = c.clientset.VirtualMachineInstance(vm.ObjectMeta.Namespace).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})

	// Don't log an error if it is already deleted
	if err != nil {
		// We can't observe a delete if it was not accepted by the server
		c.expectations.DeletionObserved(vmKey, controller.VirtualMachineInstanceKey(vmi))
		c.recorder.Eventf(vm, k8score.EventTypeWarning, common.FailedDeleteVirtualMachineReason, "Error deleting virtual machine instance %s: %v", vmi.ObjectMeta.Name, err)
		return vm, err
	}

	vm, err = c.cleanupRestartRequired(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error(failedCleanupRestartRequired)
		return vm, nil
	}

	c.recorder.Eventf(vm, k8score.EventTypeNormal, common.SuccessfulDeleteVirtualMachineReason, "Stopped the virtual machine by deleting the virtual machine instance %v", vmi.ObjectMeta.UID)
	log.Log.Object(vm).Infof("Dispatching delete event for vmi %s with phase %s", controller.NamespacedKey(vmi.Namespace, vmi.Name), vmi.Status.Phase)

	return vm, nil
}

func popStateChangeRequest(vm *virtv1.VirtualMachine) {
	vm.Status.StateChangeRequests = vm.Status.StateChangeRequests[1:]
}

func vmRevisionName(vmUID types.UID) string {
	return fmt.Sprintf("revision-start-vm-%s", vmUID)
}

func getVMRevisionName(vmUID types.UID, generation int64) string {
	return fmt.Sprintf("%s-%d", vmRevisionName(vmUID), generation)
}

func patchVMRevision(vm *virtv1.VirtualMachine) ([]byte, error) {
	vmCopy := vm.DeepCopy()
	if revision.HasControllerRevisionRef(vmCopy.Status.InstancetypeRef) {
		vmCopy.Spec.Instancetype.RevisionName = vmCopy.Status.InstancetypeRef.ControllerRevisionRef.Name
	}
	if revision.HasControllerRevisionRef(vm.Status.PreferenceRef) {
		vmCopy.Spec.Preference.RevisionName = vm.Status.PreferenceRef.ControllerRevisionRef.Name
	}
	vmBytes, err := json.Marshal(vmCopy)
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

func (c *Controller) deleteOlderVMRevision(vm *virtv1.VirtualMachine) (bool, error) {
	keys, err := c.crIndexer.IndexKeys("vm", string(vm.UID))
	if err != nil {
		return false, err
	}

	createNotNeeded := false
	for _, key := range keys {
		if !strings.Contains(key, vmRevisionName(vm.UID)) {
			continue
		}

		storeObj, exists, err := c.crIndexer.GetByKey(key)
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
		err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), cr.Name, metav1.DeleteOptions{})
		if err != nil {
			return false, err
		}
	}

	return createNotNeeded, nil
}

func (c *Controller) deleteVMRevisions(vm *virtv1.VirtualMachine) error {
	keys, err := c.crIndexer.IndexKeys("vm", string(vm.UID))
	if err != nil {
		return err
	}

	for _, key := range keys {
		if !strings.Contains(key, vmRevisionName(vm.UID)) {
			continue
		}

		storeObj, exists, err := c.crIndexer.GetByKey(key)
		if !exists || err != nil {
			return err
		}
		cr, ok := storeObj.(*appsv1.ControllerRevision)
		if !ok {
			return fmt.Errorf("unexpected resource %+v", storeObj)
		}

		err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), cr.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// getControllerRevision attempts to get the controller revision by name and
// namespace. It will return (nil, nil) if the controller revision is not found.
func (c *Controller) getControllerRevision(namespace string, name string) (*appsv1.ControllerRevision, error) {
	cr, err := c.clientset.AppsV1().ControllerRevisions(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return cr, nil
}

func (c *Controller) getVMSpecForKey(key string) (*virtv1.VirtualMachineSpec, error) {
	obj, exists, err := c.crIndexer.GetByKey(key)
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

	revisionData := VirtualMachineRevisionData{}
	err = json.Unmarshal(cr.Data.Raw, &revisionData)
	if err != nil {
		return nil, err
	}

	return &revisionData.Spec, nil
}

func genFromKey(key string) (int64, error) {
	items := strings.Split(key, "-")
	genString := items[len(items)-1]
	return strconv.ParseInt(genString, 10, 64)
}

func (c *Controller) getLastVMRevisionSpec(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachineSpec, error) {
	keys, err := c.crIndexer.IndexKeys("vm", string(vm.UID))
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}

	var highestGen int64 = 0
	var key string
	for _, k := range keys {
		if !strings.Contains(k, vmRevisionName(vm.UID)) {
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

func (c *Controller) createVMRevision(vm *virtv1.VirtualMachine) (string, error) {
	vmRevisionName := getVMRevisionName(vm.UID, vm.Generation)
	createNotNeeded, err := c.deleteOlderVMRevision(vm)
	if err != nil || createNotNeeded {
		return vmRevisionName, err
	}
	patch, err := patchVMRevision(vm)
	if err != nil {
		return "", err
	}
	cr := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            vmRevisionName,
			Namespace:       vm.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
		},
		Data:     runtime.RawExtension{Raw: patch},
		Revision: vm.ObjectMeta.Generation,
	}
	_, err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return cr.Name, nil
}

// setupVMIfromVM creates a VirtualMachineInstance object from one VirtualMachine object.
func (c *Controller) setupVMIFromVM(vm *virtv1.VirtualMachine) *virtv1.VirtualMachineInstance {
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
	if memorydump.HasCompleted(vm) {
		vmi.Spec = *memorydump.RemoveMemoryDumpVolumeFromVMISpec(&vmi.Spec, vm.Status.MemoryDumpRequest.ClaimName)
	}

	setupStableFirmwareUUID(vm, vmi)

	// TODO check if vmi labels exist, and when make sure that they match. For now just override them
	vmi.ObjectMeta.Labels = vm.Spec.Template.ObjectMeta.Labels
	vmi.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}

	return vmi
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

	vmi.Spec.Domain.Firmware.UUID = CalculateLegacyUUID(vmi.Name)
}

// listControllerFromNamespace takes a namespace and returns all VirtualMachines
// from the VirtualMachine cache which run in this namespace
func (c *Controller) listControllerFromNamespace(namespace string) ([]*virtv1.VirtualMachine, error) {
	objs, err := c.vmIndexer.ByIndex(cache.NamespaceIndex, namespace)
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
func (c *Controller) getMatchingControllers(vmi *virtv1.VirtualMachineInstance) (vms []*virtv1.VirtualMachine) {
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
func (c *Controller) addVirtualMachineInstance(obj interface{}) {
	vmi := obj.(*virtv1.VirtualMachineInstance)

	log.Log.Object(vmi).V(4).Info("VirtualMachineInstance added.")

	if vmi.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new vmi shows up in a state that
		// is already pending deletion. Prevent the vmi from being a creation observation.
		c.deleteVirtualMachineInstance(vmi)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(vmi); controllerRef != nil {
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
func (c *Controller) updateVirtualMachineInstance(old, cur interface{}) {
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

	curControllerRef := metav1.GetControllerOf(curVMI)
	oldControllerRef := metav1.GetControllerOf(oldVMI)
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
func (c *Controller) deleteVirtualMachineInstance(obj interface{}) {
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

	controllerRef := metav1.GetControllerOf(vmi)
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

func (c *Controller) addDataVolume(obj interface{}) {
	dataVolume := obj.(*cdiv1.DataVolume)
	if dataVolume.DeletionTimestamp != nil {
		c.deleteDataVolume(dataVolume)
		return
	}
	controllerRef := metav1.GetControllerOf(dataVolume)
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
func (c *Controller) updateDataVolume(old, cur interface{}) {
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
	curControllerRef := metav1.GetControllerOf(curDataVolume)
	oldControllerRef := metav1.GetControllerOf(oldDataVolume)
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vm := c.resolveControllerRef(oldDataVolume.Namespace, oldControllerRef); vm != nil {
			c.enqueueVm(vm)
		}
	}
	c.queueVMsForDataVolume(curDataVolume)
}

func (c *Controller) deleteDataVolume(obj interface{}) {
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
	if controllerRef := metav1.GetControllerOf(dataVolume); controllerRef != nil {
		if vm := c.resolveControllerRef(dataVolume.Namespace, controllerRef); vm != nil {
			if vmKey, err := controller.KeyFunc(vm); err == nil {
				c.dataVolumeExpectations.DeletionObserved(vmKey, controller.DataVolumeKey(dataVolume))
			}
		}
	}
	c.queueVMsForDataVolume(dataVolume)
}

func (c *Controller) queueVMsForDataVolume(dataVolume *cdiv1.DataVolume) {
	var vmOwner string
	if controllerRef := metav1.GetControllerOf(dataVolume); controllerRef != nil {
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
		objs, err := c.vmIndexer.ByIndex(indexName, k)
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

func (c *Controller) addVirtualMachine(obj interface{}) {
	c.enqueueVm(obj)
}

func (c *Controller) deleteVirtualMachine(obj interface{}) {
	c.enqueueVm(obj)
}

func (c *Controller) updateVirtualMachine(_, curr interface{}) {
	c.enqueueVm(curr)
}

func (c *Controller) enqueueVm(obj interface{}) {
	logger := log.Log
	vm := obj.(*virtv1.VirtualMachine)
	key, err := controller.KeyFunc(vm)
	if err != nil {
		logger.Object(vm).Reason(err).Error(failedExtractVmkeyFromVmErrMsg)
		return
	}
	c.Queue.Add(key)
}

func (c *Controller) getPatchFinalizerOps(oldFinalizers, newFinalizers []string) ([]byte, error) {
	return patch.New(
		patch.WithTest("/metadata/finalizers", oldFinalizers),
		patch.WithReplace("/metadata/finalizers", newFinalizers)).
		GeneratePayload()
}

func (c *Controller) removeVMIFinalizer(vmi *virtv1.VirtualMachineInstance) error {
	if !controller.HasFinalizer(vmi, virtv1.VirtualMachineControllerFinalizer) {
		return nil
	}

	log.Log.V(3).Object(vmi).Infof("VMI is in a final state. Removing VM controller finalizer")

	newFinalizers := []string{}

	for _, fin := range vmi.Finalizers {
		if fin != virtv1.VirtualMachineControllerFinalizer {
			newFinalizers = append(newFinalizers, fin)
		}
	}

	patch, err := c.getPatchFinalizerOps(vmi.Finalizers, newFinalizers)
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	return err
}

func (c *Controller) removeVMFinalizer(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	if !controller.HasFinalizer(vm, virtv1.VirtualMachineControllerFinalizer) {
		return vm, nil
	}

	log.Log.V(3).Object(vm).Infof("Removing VM controller finalizer: %s", virtv1.VirtualMachineControllerFinalizer)

	newFinalizers := []string{}

	for _, fin := range vm.Finalizers {
		if fin != virtv1.VirtualMachineControllerFinalizer {
			newFinalizers = append(newFinalizers, fin)
		}
	}

	patch, err := c.getPatchFinalizerOps(vm.Finalizers, newFinalizers)
	if err != nil {
		return vm, err
	}

	vm, err = c.clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	return vm, err
}

func (c *Controller) addVMFinalizer(vm *virtv1.VirtualMachine) (*virtv1.VirtualMachine, error) {
	if controller.HasFinalizer(vm, virtv1.VirtualMachineControllerFinalizer) {
		return vm, nil
	}

	log.Log.V(3).Object(vm).Infof("Adding VM controller finalizer: %s", virtv1.VirtualMachineControllerFinalizer)

	newFinalizers := make([]string, len(vm.Finalizers))
	copy(newFinalizers, vm.Finalizers)
	newFinalizers = append(newFinalizers, virtv1.VirtualMachineControllerFinalizer)

	patch, err := c.getPatchFinalizerOps(vm.Finalizers, newFinalizers)
	if err != nil {
		return vm, err
	}

	return c.clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
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
func (c *Controller) patchVmGenerationFromControllerRevision(vmi *virtv1.VirtualMachineInstance, logger *log.FilteredLogger) (*virtv1.VirtualMachineInstance, *int64, error) {

	cr, err := c.getControllerRevision(vmi.Namespace, vmi.Status.VirtualMachineRevisionName)
	if err != nil || cr == nil {
		return vmi, nil, err
	}

	generation := parseGeneration(cr.Name, logger)
	if generation == nil {
		return vmi, nil, nil
	}

	vmi, err = c.patchVmGenerationAnnotationOnVmi(*generation, vmi)
	if err != nil {
		return vmi, generation, err
	}

	return vmi, generation, err
}

// syncGenerationInfo will update the vm.Status with the ObservedGeneration
// from the vmi and the DesiredGeneration from the vm current generation.
func (c *Controller) syncGenerationInfo(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, logger *log.FilteredLogger) (*virtv1.VirtualMachineInstance, error) {
	if vm == nil || vmi == nil {
		return vmi, errors.New("passed nil pointer")
	}

	generation, err := getGenerationAnnotationAsInt(vmi, logger)
	if err != nil {
		return vmi, err
	}

	// If the generation annotation does not exist, the VMI could have been
	// been created before the controller was updated. In this case, check the
	// ControllerRevision on what the latest observed generation is and back-fill
	// the info onto the vmi annotation.
	if generation == nil {
		var patchedVMI *virtv1.VirtualMachineInstance
		patchedVMI, generation, err = c.patchVmGenerationFromControllerRevision(vmi, logger)
		if generation == nil || err != nil {
			return vmi, err
		}
		vmi = patchedVMI
	}

	vm.Status.ObservedGeneration = *generation
	vm.Status.DesiredGeneration = vm.Generation

	return vmi, nil
}

func (c *Controller) updateStatus(vm, vmOrig *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, syncErr common.SyncError, logger *log.FilteredLogger) error {
	key := controller.VirtualMachineKey(vmOrig)
	defer virtControllerVMWorkQueueTracer.StepTrace(key, "updateStatus", trace.Field{Key: "VM Name", Value: vmOrig.Name})

	created := vmi != nil
	vm.Status.Created = created

	ready := false
	if created {
		ready = controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceReady, k8score.ConditionTrue)
		var err error
		vmi, err = c.syncGenerationInfo(vm, vmi, logger)
		if err != nil {
			return err
		}
	}
	vm.Status.Ready = ready

	runStrategy, _ := vmOrig.RunStrategy()
	// sync for the first time only when the VMI gets created
	// so that we can tell if the VM got started at least once
	if vm.Status.RunStrategy != "" || vm.Status.Created {
		vm.Status.RunStrategy = runStrategy
	}

	c.trimDoneVolumeRequests(vm)
	memorydump.UpdateRequest(vm, vmi)

	if c.isTrimFirstChangeRequestNeeded(vm, vmi) {
		popStateChangeRequest(vm)
	}

	syncStartFailureStatus(vm, vmi)
	// On a successful migration, the volume change condition is removed and we need to detect the removal before the synchronization of the VMI
	// condition to the VM
	syncVolumeMigration(vm, vmi)
	syncConditions(vm, vmi, syncErr)
	c.setPrintableStatus(vm, vmi)

	// only update if necessary
	if !equality.Semantic.DeepEqual(vm.Status, vmOrig.Status) {
		if _, err := c.clientset.VirtualMachine(vm.Namespace).UpdateStatus(context.Background(), vm, v1.UpdateOptions{}); err != nil {
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

func (c *Controller) setPrintableStatus(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
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
func (c *Controller) isVirtualMachineStatusCrashLoopBackOff(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
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
func (c *Controller) isVirtualMachineStatusStopped(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi != nil {
		return vmi.IsFinal()
	}

	return !c.isVMIStartExpected(vm)
}

// isVirtualMachineStatusStopped determines whether the VM status field should be set to "Provisioning".
func (c *Controller) isVirtualMachineStatusProvisioning(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return storagetypes.HasDataVolumeProvisioning(vm.Namespace, vm.Spec.Template.Spec.Volumes, c.dataVolumeStore)
}

// isVirtualMachineStatusWaitingForVolumeBinding
func (c *Controller) isVirtualMachineStatusWaitingForVolumeBinding(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if !isSetToStart(vm, vmi) {
		return false
	}

	return storagetypes.HasUnboundPVC(vm.Namespace, vm.Spec.Template.Spec.Volumes, c.pvcStore)
}

// isVirtualMachineStatusStarting determines whether the VM status field should be set to "Starting".
func (c *Controller) isVirtualMachineStatusStarting(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return c.isVMIStartExpected(vm)
	}

	return vmi.IsUnprocessed() || vmi.IsScheduling() || vmi.IsScheduled()
}

// isVirtualMachineStatusRunning determines whether the VM status field should be set to "Running".
func (c *Controller) isVirtualMachineStatusRunning(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	hasPausedCondition := controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstancePaused, k8score.ConditionTrue)

	return vmi.IsRunning() && !hasPausedCondition
}

// isVirtualMachineStatusPaused determines whether the VM status field should be set to "Paused".
func (c *Controller) isVirtualMachineStatusPaused(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}

	hasPausedCondition := controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstancePaused, k8score.ConditionTrue)

	return vmi.IsRunning() && hasPausedCondition
}

// isVirtualMachineStatusStopping determines whether the VM status field should be set to "Stopping".
func (c *Controller) isVirtualMachineStatusStopping(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return vmi != nil && !vmi.IsFinal() &&
		(vmi.IsMarkedForDeletion() || c.isVMIStopExpected(vm))
}

// isVirtualMachineStatusTerminating determines whether the VM status field should be set to "Terminating".
func (c *Controller) isVirtualMachineStatusTerminating(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return vm.ObjectMeta.DeletionTimestamp != nil
}

// isVirtualMachineStatusMigrating determines whether the VM status field should be set to "Migrating".
func (c *Controller) isVirtualMachineStatusMigrating(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return vmi != nil && migrations.IsMigrating(vmi)
}

// isVirtualMachineStatusUnschedulable determines whether the VM status field should be set to "FailedUnschedulable".
func (c *Controller) isVirtualMachineStatusUnschedulable(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatusAndReason(vmi,
		virtv1.VirtualMachineInstanceConditionType(k8score.PodScheduled),
		k8score.ConditionFalse,
		k8score.PodReasonUnschedulable)
}

// isVirtualMachineStatusErrImagePull determines whether the VM status field should be set to "ErrImagePull"
func (c *Controller) isVirtualMachineStatusErrImagePull(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	syncCond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, virtv1.VirtualMachineInstanceSynchronized)
	return syncCond != nil && syncCond.Status == k8score.ConditionFalse && syncCond.Reason == controller.ErrImagePullReason
}

// isVirtualMachineStatusImagePullBackOff determines whether the VM status field should be set to "ImagePullBackOff"
func (c *Controller) isVirtualMachineStatusImagePullBackOff(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	syncCond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, virtv1.VirtualMachineInstanceSynchronized)
	return syncCond != nil && syncCond.Status == k8score.ConditionFalse && syncCond.Reason == controller.ImagePullBackOffReason
}

// isVirtualMachineStatusPvcNotFound determines whether the VM status field should be set to "FailedPvcNotFound".
func (c *Controller) isVirtualMachineStatusPvcNotFound(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatusAndReason(vmi,
		virtv1.VirtualMachineInstanceSynchronized,
		k8score.ConditionFalse,
		controller.FailedPvcNotFoundReason)
}

// isVirtualMachineStatusDataVolumeError determines whether the VM status field should be set to "DataVolumeError"
func (c *Controller) isVirtualMachineStatusDataVolumeError(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	err := storagetypes.HasDataVolumeErrors(vm.Namespace, vm.Spec.Template.Spec.Volumes, c.dataVolumeStore)
	if err != nil {
		log.Log.Object(vm).Errorf("%v", err)
		return true
	}
	return false
}

func syncReadyConditionFromVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) {
	conditionManager := controller.NewVirtualMachineConditionManager()
	vmiReadyCond := controller.NewVirtualMachineInstanceConditionManager().
		GetCondition(vmi, virtv1.VirtualMachineInstanceReady)

	now := metav1.Now()
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

func syncConditions(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, syncErr common.SyncError) {
	cm := controller.NewVirtualMachineConditionManager()

	// ready condition is handled differently as it persists regardless if vmi exists or not
	syncReadyConditionFromVMI(vm, vmi)
	processFailureCondition(vm, syncErr)

	// nothing to do if vmi hasn't been created yet.
	if vmi == nil {
		return
	}

	// sync VMI conditions, ignore list represents conditions that are not synced generically
	syncIgnoreMap := map[string]interface{}{
		string(virtv1.VirtualMachineReady):           nil,
		string(virtv1.VirtualMachineFailure):         nil,
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

func processFailureCondition(vm *virtv1.VirtualMachine, syncErr common.SyncError) {

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
		LastTransitionTime: metav1.Now(),
		Status:             k8score.ConditionTrue,
	})
}

func (c *Controller) isTrimFirstChangeRequestNeeded(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (clearChangeRequest bool) {
	if len(vm.Status.StateChangeRequests) == 0 {
		return false
	}

	// Only consider one stateChangeRequest at a time. The second and subsequent change
	// requests have not been acted upon by this controller yet!
	stateChange := vm.Status.StateChangeRequests[0]
	switch stateChange.Action {
	case virtv1.StopRequest:
		if vmi == nil {
			// If there's no VMI, then the VMI was stopped, and the stopRequest can be cleared
			log.Log.Object(vm).V(4).Infof("No VMI. Clearing stop request")
			return true
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
		// Update VMI as the runStrategy might have started/stopped the VM.
		// Example: if the runStrategy is `RerunOnFailure` and the VMI just failed
		// `syncRunStrategy()` will delete the VMI object and enqueue a StartRequest.
		// If we do not update `vmi` by asking the API Server this function could
		// erroneously trim the just added StartRequest because it would see a running
		// vmi with no DeletionTimestamp
		if vmi != nil && vmi.DeletionTimestamp == nil && !vmi.IsFinal() {
			log.Log.Object(vm).V(4).Infof("VMI exists. clearing start request")
			return true
		}
	}

	return false
}

func (c *Controller) trimDoneVolumeRequests(vm *virtv1.VirtualMachine) {
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

func validLiveUpdateVolumes(oldVMSpec *virtv1.VirtualMachineSpec, vm *virtv1.VirtualMachine) bool {
	oldVols := storagetypes.GetVolumesByName(&oldVMSpec.Template.Spec)
	// Evaluate if any volume has changed or has been added
	for _, v := range vm.Spec.Template.Spec.Volumes {
		oldVol, okOld := oldVols[v.Name]
		switch {
		// Changes for hotlpugged volumes are valid
		case storagetypes.IsHotplugVolume(&v):
			delete(oldVols, v.Name)
		// The volume has been freshly added
		case !okOld:
			return false
		// if the update strategy is migration the PVC/DV could have
		// changed
		case (v.VolumeSource.PersistentVolumeClaim != nil || v.VolumeSource.DataVolume != nil) &&
			vm.Spec.UpdateVolumesStrategy != nil &&
			*vm.Spec.UpdateVolumesStrategy == virtv1.UpdateVolumesStrategyMigration:
			delete(oldVols, v.Name)
		// The volume has changed
		case !equality.Semantic.DeepEqual(*oldVol, v):
			return false
		default:
			delete(oldVols, v.Name)
		}
	}
	// Evaluate if any volumes were removed and they were hotplugged volumes
	for _, v := range oldVols {
		if !storagetypes.IsHotplugVolume(v) {
			return false
		}
	}

	return true
}

func validLiveUpdateDisks(oldVMSpec *virtv1.VirtualMachineSpec, vm *virtv1.VirtualMachine) bool {
	oldDisks := storagetypes.GetDisksByName(&oldVMSpec.Template.Spec)
	oldVols := storagetypes.GetVolumesByName(&oldVMSpec.Template.Spec)
	vols := storagetypes.GetVolumesByName(&vm.Spec.Template.Spec)
	// Evaluate if any disk has changed or has been added
	for _, newDisk := range vm.Spec.Template.Spec.Domain.Devices.Disks {
		v := vols[newDisk.Name]
		oldDisk, okOld := oldDisks[newDisk.Name]
		switch {
		// Changes for disks associated to a hotpluggable volume are valid
		case storagetypes.IsHotplugVolume(v):
			delete(oldDisks, v.Name)
		// The disk has been freshly added
		case !okOld:
			return false
		// The disk has changed
		case !equality.Semantic.DeepEqual(*oldDisk, newDisk):
			return false
		default:
			delete(oldDisks, v.Name)
		}
	}
	// Evaluate if any disks were removed and they were hotplugged volumes
	for _, d := range oldDisks {
		v := oldVols[d.Name]
		if !storagetypes.IsHotplugVolume(v) {
			return false
		}
	}

	return true
}

func setRestartRequired(vm *virtv1.VirtualMachine, message string) {
	vmConditions := controller.NewVirtualMachineConditionManager()
	vmConditions.UpdateCondition(vm, &virtv1.VirtualMachineCondition{
		Type:               virtv1.VirtualMachineRestartRequired,
		LastTransitionTime: metav1.Now(),
		Status:             k8score.ConditionTrue,
		Message:            message,
	})
}

// addRestartRequiredIfNeeded adds the restartRequired condition to the VM if any non-live-updatable field was changed
func (c *Controller) addRestartRequiredIfNeeded(lastSeenVMSpec *virtv1.VirtualMachineSpec, vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) bool {
	if lastSeenVMSpec == nil {
		return false
	}

	// Expand any instance types and preferences associated with lastSeenVMSpec or the current VM before working out if things are live-updatable
	currentVM := vm.DeepCopy()
	if err := c.instancetypeController.ApplyToVM(currentVM); err != nil {
		return false
	}
	lastSeenVM := &virtv1.VirtualMachine{
		// We need the namespace to be populated here for the lookup and application of instance types to work below
		ObjectMeta: currentVM.DeepCopy().ObjectMeta,
		Spec:       *lastSeenVMSpec.DeepCopy(),
	}
	if err := c.instancetypeController.ApplyToVM(lastSeenVM); err != nil {
		return false
	}

	// Ignore all the live-updatable fields by copying them over. (If the feature gate is disabled, nothing is live-updatable)
	// Note: this list needs to stay up-to-date with everything that can be live-updated
	// Note2: destroying lastSeenVM here is fine, we don't need it later
	if c.clusterConfig.IsVMRolloutStrategyLiveUpdate() {
		if validLiveUpdateVolumes(&lastSeenVM.Spec, currentVM) {
			lastSeenVM.Spec.Template.Spec.Volumes = currentVM.Spec.Template.Spec.Volumes
		}
		if validLiveUpdateDisks(&lastSeenVM.Spec, currentVM) {
			lastSeenVM.Spec.Template.Spec.Domain.Devices.Disks = currentVM.Spec.Template.Spec.Domain.Devices.Disks
		}
		if lastSeenVM.Spec.Template.Spec.Domain.CPU != nil && currentVM.Spec.Template.Spec.Domain.CPU != nil {
			lastSeenVM.Spec.Template.Spec.Domain.CPU.Sockets = currentVM.Spec.Template.Spec.Domain.CPU.Sockets
		}

		if currentVM.Spec.Template.Spec.Domain.Memory != nil && currentVM.Spec.Template.Spec.Domain.Memory.Guest != nil {
			if lastSeenVM.Spec.Template.Spec.Domain.Memory == nil {
				lastSeenVM.Spec.Template.Spec.Domain.Memory = &virtv1.Memory{}
			}
			lastSeenVM.Spec.Template.Spec.Domain.Memory.Guest = currentVM.Spec.Template.Spec.Domain.Memory.Guest
		}

		lastSeenVM.Spec.Template.Spec.NodeSelector = currentVM.Spec.Template.Spec.NodeSelector
		lastSeenVM.Spec.Template.Spec.Affinity = currentVM.Spec.Template.Spec.Affinity
		lastSeenVM.Spec.Template.Spec.Tolerations = currentVM.Spec.Template.Spec.Tolerations
	} else {
		// In the case live-updates aren't enable the volume set of the VM can be still changed by volume hotplugging.
		// For imperative volume hotplug, first the VM status with the request AND the VMI spec are updated, then in the
		// next iteration, the VM spec is updated as well. Here, we're in this iteration where the currentVM has for the first
		// time the updated hotplugged volumes. Hence, we can compare the current VM volumes and disks with the ones belonging
		// to the VMI.
		// In case of a declarative update, the flow is the opposite, first we update the VM spec and then the VMI. Therefore, if
		// the change was declarative, then the VMI would still not have the update.
		if equality.Semantic.DeepEqual(currentVM.Spec.Template.Spec.Volumes, vmi.Spec.Volumes) &&
			equality.Semantic.DeepEqual(currentVM.Spec.Template.Spec.Domain.Devices.Disks, vmi.Spec.Domain.Devices.Disks) {
			lastSeenVM.Spec.Template.Spec.Volumes = currentVM.Spec.Template.Spec.Volumes
			lastSeenVM.Spec.Template.Spec.Domain.Devices.Disks = currentVM.Spec.Template.Spec.Domain.Devices.Disks
		}
	}

	if !netvmliveupdate.IsRestartRequired(currentVM, vmi) {
		lastSeenVM.Spec.Template.Spec.Domain.Devices.Interfaces = currentVM.Spec.Template.Spec.Domain.Devices.Interfaces
		lastSeenVM.Spec.Template.Spec.Networks = currentVM.Spec.Template.Spec.Networks
	}

	if !equality.Semantic.DeepEqual(lastSeenVM.Spec.Template.Spec, currentVM.Spec.Template.Spec) {
		setRestartRequired(vm, "a non-live-updatable field was changed in the template spec")
		return true
	}

	return false
}

func (c *Controller) sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, key string) (*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance, common.SyncError, error) {

	defer virtControllerVMWorkQueueTracer.StepTrace(key, "sync", trace.Field{Key: "VM Name", Value: vm.Name})

	var (
		syncErr     common.SyncError
		err         error
		startVMSpec *virtv1.VirtualMachineSpec
	)

	if !c.satisfiedExpectations(key) {
		return vm, vmi, nil, nil
	}

	if vmi != nil {
		startVMSpec, err = c.getLastVMRevisionSpec(vm)
		if err != nil {
			return vm, vmi, nil, err
		}
	}

	if vm.DeletionTimestamp != nil {
		if vmi == nil || controller.HasFinalizer(vm, metav1.FinalizerOrphanDependents) {
			vm, err = c.removeVMFinalizer(vm)
			if err != nil {
				return vm, vmi, nil, err
			}
		} else {
			vm, err = c.stopVMI(vm, vmi)
			if err != nil {
				log.Log.Object(vm).Errorf(failureDeletingVmiErrFormat, err)
				return vm, vmi, common.NewSyncError(fmt.Errorf(failureDeletingVmiErrFormat, err), vmiFailedDeleteReason), nil
			}
		}
		return vm, vmi, nil, nil
	} else {
		vm, err = c.addVMFinalizer(vm)
		if err != nil {
			return vm, vmi, nil, err
		}
	}

	vmi, err = c.conditionallyBumpGenerationAnnotationOnVmi(vm, vmi)
	if err != nil {
		return nil, vmi, nil, err
	}

	// Scale up or down, if all expected creates and deletes were report by the listener
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return vm, vmi, common.NewSyncError(fmt.Errorf(fetchingRunStrategyErrFmt, err), failedCreateReason), err
	}

	// FIXME(lyarwood): Move alongside netSynchronizer
	syncedVM, err := c.instancetypeController.Sync(vm, vmi)
	if err != nil {
		return vm, vmi, handleSynchronizerErr(err), nil
	}
	if !equality.Semantic.DeepEqual(vm.Spec, syncedVM.Spec) {
		return syncedVM, vmi, nil, nil
	}

	vm.ObjectMeta = syncedVM.ObjectMeta
	vm.Spec = syncedVM.Spec

	// eventually, would like the condition to be `== "true"`, but for now we need to support legacy behavior by default
	if vm.Annotations[virtv1.ImmediateDataVolumeCreation] != "false" {
		dataVolumesReady, err := c.handleDataVolumes(vm)
		if err != nil {
			return vm, vmi, common.NewSyncError(fmt.Errorf("Error encountered while creating DataVolumes: %v", err), failedCreateReason), nil
		}

		// not sure why we allow to proceed when halted but preserving legacy behavior
		if !dataVolumesReady && runStrategy != virtv1.RunStrategyHalted {
			log.Log.Object(vm).V(3).Info("Waiting on DataVolumes to be ready.")
			return vm, vmi, nil, nil
		}
	}

	vm, syncErr = c.syncRunStrategy(vm, vmi, runStrategy)
	if syncErr != nil {
		return vm, vmi, syncErr, nil
	}

	restartRequired := c.addRestartRequiredIfNeeded(startVMSpec, vm, vmi)

	// Must check satisfiedExpectations again here because a VMI can be created or
	// deleted in the startStop function which impacts how we process
	// hotplugged volumes and interfaces
	if !c.satisfiedExpectations(key) {
		return vm, vmi, nil, nil
	}

	vmCopy := vm.DeepCopy()

	if c.netSynchronizer != nil {
		syncedVM, err := c.netSynchronizer.Sync(vmCopy, vmi)
		if err != nil {
			return vm, vmi, handleSynchronizerErr(err), nil
		}
		vmCopy.ObjectMeta = syncedVM.ObjectMeta
		vmCopy.Spec = syncedVM.Spec
	}

	if c.firmwareSynchronizer != nil {
		syncedVM, err := c.firmwareSynchronizer.Sync(vmCopy, vmi)
		if err != nil {
			return vm, vmi, handleSynchronizerErr(err), nil
		}
		vmCopy.ObjectMeta = syncedVM.ObjectMeta
		vmCopy.Spec = syncedVM.Spec
	}

	if err := c.handleVolumeRequests(vmCopy, vmi); err != nil {
		return vm, vmi, common.NewSyncError(fmt.Errorf("Error encountered while handling volume hotplug requests: %v", err), hotplugVolumeErrorReason), nil
	}

	if err := memorydump.HandleRequest(c.clientset, vmCopy, vmi, c.pvcStore); err != nil {
		return vm, vmi, common.NewSyncError(fmt.Errorf("Error encountered while handling memory dump request: %v", err), memorydump.ErrorReason), nil
	}

	conditionManager := controller.NewVirtualMachineConditionManager()
	if c.clusterConfig.IsVMRolloutStrategyLiveUpdate() && !restartRequired && !conditionManager.HasCondition(vm, virtv1.VirtualMachineRestartRequired) {
		if err := c.handleCPUChangeRequest(vmCopy, vmi); err != nil {
			return vm, vmi, common.NewSyncError(fmt.Errorf("Error encountered while handling CPU change request: %v", err), hotplugCPUErrorReason), nil
		}

		if err := c.handleAffinityChangeRequest(vmCopy, vmi); err != nil {
			return vm, vmi, common.NewSyncError(fmt.Errorf("Error encountered while handling node affinity change request: %v", err), affinityChangeErrorReason), nil
		}

		if err := c.handleTolerationsChangeRequest(vmCopy, vmi); err != nil {
			return vm, vmi, common.NewSyncError(fmt.Errorf("Error encountered while handling tolerations change request: %v", err), tolerationsChangeErrorReason), nil
		}

		if err := c.handleMemoryHotplugRequest(vmCopy, vmi); err != nil {
			return vm, vmi, common.NewSyncError(fmt.Errorf("error encountered while handling memory hotplug requests: %v", err), hotplugMemoryErrorReason), nil
		}

		if err := c.handleVolumeUpdateRequest(vmCopy, vmi); err != nil {
			return vm, vmi, common.NewSyncError(fmt.Errorf("error encountered while handling volumes update requests: %v", err), volumesUpdateErrorReason), nil
		}
	}

	if !equality.Semantic.DeepEqual(vm.Spec, vmCopy.Spec) || !equality.Semantic.DeepEqual(vm.ObjectMeta, vmCopy.ObjectMeta) {
		updatedVm, err := c.clientset.VirtualMachine(vmCopy.Namespace).Update(context.Background(), vmCopy, metav1.UpdateOptions{})
		if err != nil {
			return vm, vmi, common.NewSyncError(fmt.Errorf("Error encountered when trying to update vm according to add volume and/or memory dump requests: %v", err), failedUpdateErrorReason), nil
		}
		vm = updatedVm
	} else {
		vm = vmCopy
	}

	return vm, vmi, nil, nil
}

func handleSynchronizerErr(err error) common.SyncError {
	if err == nil {
		return nil
	}
	var errWithReason common.SyncError
	if errors.As(err, &errWithReason) {
		return errWithReason
	}
	return common.NewSyncError(fmt.Errorf("unsupported error: %v", err), "UnsupportedSyncError")
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *Controller) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) *virtv1.VirtualMachine {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != virtv1.VirtualMachineGroupVersionKind.Kind {
		return nil
	}
	vm, exists, err := c.vmIndexer.GetByKey(controller.NamespacedKey(namespace, controllerRef.Name))
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
	if autoAttachInput == nil || !*autoAttachInput || len(vmi.Spec.Domain.Devices.Inputs) > 0 {
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

func (c *Controller) handleMemoryHotplugRequest(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil {
		return nil
	}

	vmCopyWithInstancetype := vm.DeepCopy()
	if err := c.instancetypeController.ApplyToVM(vmCopyWithInstancetype); err != nil {
		return err
	}

	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory == nil ||
		vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest == nil ||
		vmi.Spec.Domain.Memory == nil ||
		vmi.Spec.Domain.Memory.Guest == nil ||
		vmi.Status.Memory == nil ||
		vmi.Status.Memory.GuestCurrent == nil {
		return nil
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()

	if conditionManager.HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceMemoryChange, k8score.ConditionFalse) {
		setRestartRequired(vm, "memory updated in template spec. Memory-hotplug failed and is not available for this VM configuration")
		return nil
	}

	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest.Equal(*vmi.Spec.Domain.Memory.Guest) {
		return nil
	}

	if !vmi.IsMigratable() {
		setRestartRequired(vm, "memory updated in template spec. Memory-hotplug is only available for migratable VMs")
		return nil
	}

	if vmi.Spec.Domain.Memory.MaxGuest == nil {
		setRestartRequired(vm, "memory updated in template spec. Memory-hotplug is not available for this VM configuration")
		return nil
	}

	if conditionManager.HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstanceMemoryChange, k8score.ConditionTrue) {
		return fmt.Errorf("another memory hotplug is in progress")
	}

	if migrations.IsMigrating(vmi) {
		return fmt.Errorf("memory hotplug is not allowed while VMI is migrating")
	}

	if err := memory.ValidateLiveUpdateMemory(&vmCopyWithInstancetype.Spec.Template.Spec, vmi.Spec.Domain.Memory.MaxGuest); err != nil {
		setRestartRequired(vm, fmt.Sprintf("memory hotplug not supported, %s", err.Error()))
		return nil
	}

	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest != nil && vmi.Status.Memory.GuestAtBoot != nil &&
		vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest.Cmp(*vmi.Status.Memory.GuestAtBoot) == -1 {
		setRestartRequired(vm, "memory updated in template spec to a value lower than what the VM started with")
		return nil
	}

	// If the following is true, MaxGuest was calculated, not manually specified (or the validation webhook would have rejected the change).
	// Since we're here, we can also assume MaxGuest was not changed in the VM spec since last boot.
	// Therefore, bumping Guest to a value higher than MaxGuest is fine, it just requires a reboot.
	if vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest != nil && vmi.Spec.Domain.Memory.MaxGuest != nil &&
		vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest.Cmp(*vmi.Spec.Domain.Memory.MaxGuest) == 1 {
		setRestartRequired(vm, "memory updated in template spec to a value higher than what's available")
		return nil
	}

	memoryDelta := resource.NewQuantity(vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest.Value()-vmi.Status.Memory.GuestCurrent.Value(), resource.BinarySI)

	newMemoryReq := vmi.Spec.Domain.Resources.Requests.Memory().DeepCopy()
	newMemoryReq.Add(*memoryDelta)

	// checking if the new memory req are at least equal to the memory being requested in the handleMemoryHotplugRequest
	// this is necessary as weirdness can arise after hot-unplugs as not all memory is guaranteed to be released when doing hot-unplug.
	if newMemoryReq.Cmp(*vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest) == -1 {
		newMemoryReq = *vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest
		// adjusting memoryDelta too for the new limits computation (if required)
		memoryDelta = resource.NewQuantity(vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest.Value()-newMemoryReq.Value(), resource.BinarySI)
	}

	patchSet := patch.New(
		patch.WithTest("/spec/domain/memory/guest", vmi.Spec.Domain.Memory.Guest.String()),
		patch.WithReplace("/spec/domain/memory/guest", vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest.String()),
		patch.WithTest("/spec/domain/resources/requests/memory", vmi.Spec.Domain.Resources.Requests.Memory().String()),
		patch.WithReplace("/spec/domain/resources/requests/memory", newMemoryReq.String()),
	)

	logMsg := fmt.Sprintf("hotplugging memory to %s, setting requests to %s", vmCopyWithInstancetype.Spec.Template.Spec.Domain.Memory.Guest.String(), newMemoryReq.String())

	if !vmCopyWithInstancetype.Spec.Template.Spec.Domain.Resources.Limits.Memory().IsZero() {
		newMemoryLimit := vmi.Spec.Domain.Resources.Limits.Memory().DeepCopy()
		newMemoryLimit.Add(*memoryDelta)

		patchSet.AddOption(
			patch.WithTest("/spec/domain/resources/limits/memory", vmi.Spec.Domain.Resources.Limits.Memory().String()),
			patch.WithReplace("/spec/domain/resources/limits/memory", newMemoryLimit.String()),
		)

		logMsg = fmt.Sprintf("%s, setting limits to %s", logMsg, newMemoryLimit.String())
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		return err
	}

	log.Log.Object(vmi).Infof(logMsg)

	return nil
}
