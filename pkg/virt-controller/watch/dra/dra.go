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

package dra

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/google/go-cmp/cmp"
	k8sv1 "k8s.io/api/core/v1"
	resourcev1beta2 "k8s.io/api/resource/v1beta2"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/trace"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	drautil "kubevirt.io/kubevirt/pkg/dra"
	traceUtils "kubevirt.io/kubevirt/pkg/util/trace"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	deleteNotifFailed        = "Failed to process delete notification"
	tombstoneGetObjectErrFmt = "couldn't get object from tombstone %+v"

	indexByNodeName              = "byNodeName"
	PCIAddressDeviceAttributeKey = "resource.kubernetes.io/pcieRoot"
	MDevUUIDDeviceAttributeKey   = "resource.kubernetes.io/mDevUUID"
)

type DeviceInfo struct {
	VMISpecClaimName   string
	VMISpecRequestName string
	*v1.DeviceStatusInfo
}

type DRAStatusController struct {
	clusterConfig *virtconfig.ClusterConfig
	recorder      record.EventRecorder
	clientset     kubecli.KubevirtClient

	podIndexer           cache.Indexer
	resourceSliceIndexer cache.Indexer
	vmiIndexer           cache.Store
	resourceClaimIndexer cache.Store

	queue workqueue.TypedRateLimitingInterface[string]

	hasSynced func() bool
}

func NewDRAStatusController(
	clusterConfig *virtconfig.ClusterConfig,
	vmiInformer,
	podInformer,
	resourceClaimInformer,
	resourceSliceInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	clientset kubecli.KubevirtClient) (*DRAStatusController, error) {
	c := &DRAStatusController{
		clusterConfig: clusterConfig,
		recorder:      recorder,
		clientset:     clientset,

		podIndexer:           podInformer.GetIndexer(),
		vmiIndexer:           vmiInformer.GetStore(),
		resourceClaimIndexer: resourceClaimInformer.GetStore(),
		resourceSliceIndexer: resourceSliceInformer.GetIndexer(),

		queue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "dra-status-controller"},
		),
	}

	c.hasSynced = func() bool {
		return vmiInformer.HasSynced() && podInformer.HasSynced() &&
			resourceClaimInformer.HasSynced() && resourceSliceInformer.HasSynced()
	}

	_, err := vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addVirtualMachineInstance,
		DeleteFunc: c.deleteVirtualMachineInstance,
		UpdateFunc: c.updateVirtualMachineInstance,
	})
	if err != nil {
		return nil, err
	}

	_, err = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		DeleteFunc: c.deletePod,
		UpdateFunc: c.updatePod,
	})
	if err != nil {
		return nil, err
	}

	err = c.resourceSliceIndexer.AddIndexers(map[string]cache.IndexFunc{
		indexByNodeName: indexResourceSliceByNodeName,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *DRAStatusController) enqueueVirtualMachine(obj interface{}) {
	vmi := obj.(*v1.VirtualMachineInstance)
	logger := log.Log.Object(vmi)
	if vmi.Status.Phase == v1.Running {
		logger.V(6).Infof("skipping enqueing vmi to dra status controller queue")
		return
	}

	key, err := controller.KeyFunc(vmi)
	if err != nil {
		logger.Object(vmi).Reason(err).Error("Failed to extract key from VirtualMachineInstance.")
		return
	}
	c.queue.Add(key)
}

func (c *DRAStatusController) addVirtualMachineInstance(obj interface{}) {
	c.enqueueVirtualMachine(obj)
}

func (c *DRAStatusController) updateVirtualMachineInstance(_, curr interface{}) {
	c.enqueueVirtualMachine(curr)
}

func (c *DRAStatusController) deleteVirtualMachineInstance(obj interface{}) {
	vmi, ok := obj.(*v1.VirtualMachineInstance)

	// When a delete is dropped, the relist will notice a vmi in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf(tombstoneGetObjectErrFmt, obj)).Error(deleteNotifFailed)
			return
		}
		vmi, ok = tombstone.Obj.(*v1.VirtualMachineInstance)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a vmi %#v", obj)).Error(deleteNotifFailed)
			return
		}
	}
	c.enqueueVirtualMachine(vmi)
}

func (c *DRAStatusController) addPod(obj interface{}) {
	pod := obj.(*k8sv1.Pod)

	if pod.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new pod shows up in a state that
		// is already pending deletion. Prevent the pod from being a creation observation.
		c.deletePod(pod)
		return
	}

	controllerRef := metav1.GetControllerOf(pod)
	vmi := c.resolveControllerRef(pod.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	c.enqueueVirtualMachine(vmi)
}

func (c *DRAStatusController) deletePod(obj interface{}) {
	pod, ok := obj.(*k8sv1.Pod)

	// When a delete is dropped, the relist will notice a pod in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the pod
	// changed labels the new vmi will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf(tombstoneGetObjectErrFmt, obj)).Error(deleteNotifFailed)
			return
		}
		pod, ok = tombstone.Obj.(*k8sv1.Pod)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not a pod %#v", obj)).Error(deleteNotifFailed)
			return
		}
	}

	controllerRef := metav1.GetControllerOf(pod)
	vmi := c.resolveControllerRef(pod.Namespace, controllerRef)
	if vmi == nil {
		return
	}
	c.enqueueVirtualMachine(vmi)
}

func (c *DRAStatusController) updatePod(old interface{}, cur interface{}) {
	curPod := cur.(*k8sv1.Pod)
	oldPod := old.(*k8sv1.Pod)
	if curPod.DeletionTimestamp != nil {
		labelChanged := !equality.Semantic.DeepEqual(curPod.Labels, oldPod.Labels)
		// having a pod marked for deletion is enough to count as a deletion expectation
		c.deletePod(curPod)
		if labelChanged {
			// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
			c.deletePod(oldPod)
		}
		return
	}
	if curPod.Status.Phase == k8sv1.PodRunning || curPod.Status.Phase == k8sv1.PodFailed ||
		curPod.Status.Phase == k8sv1.PodSucceeded {
		return
	}

	curControllerRef := metav1.GetControllerOf(curPod)
	oldControllerRef := metav1.GetControllerOf(oldPod)
	controllerRefChanged := !equality.Semantic.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged {
		// The ControllerRef was changed. Sync the old controller, if any.
		if vmi := c.resolveControllerRef(oldPod.Namespace, oldControllerRef); vmi != nil {
			c.enqueueVirtualMachine(vmi)
		}
	}

	vmi := c.resolveControllerRef(curPod.Namespace, curControllerRef)
	if vmi == nil {
		return
	}
	c.enqueueVirtualMachine(vmi)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (c *DRAStatusController) resolveControllerRef(namespace string, controllerRef *metav1.OwnerReference) *v1.VirtualMachineInstance {
	if controllerRef != nil && controllerRef.Kind == "Pod" {
		// This could be an attachment pod, look up the pod, and check if it is owned by a VMI.
		obj, exists, err := c.podIndexer.GetByKey(namespace + "/" + controllerRef.Name)
		if err != nil {
			return nil
		}
		if !exists {
			return nil
		}
		pod, _ := obj.(*k8sv1.Pod)
		controllerRef = metav1.GetControllerOf(pod)
	}
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it is nil or the wrong Kind.
	if controllerRef == nil || controllerRef.Kind != v1.VirtualMachineInstanceGroupVersionKind.Kind {
		return nil
	}
	vmi, exists, err := c.vmiIndexer.GetByKey(namespace + "/" + controllerRef.Name)
	if err != nil {
		return nil
	}
	if !exists {
		return nil
	}

	if vmi.(*v1.VirtualMachineInstance).UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return vmi.(*v1.VirtualMachineInstance)
}

func (c *DRAStatusController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting DRA Status controller")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.hasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping DRA Status controller")
}

func (c *DRAStatusController) runWorker() {
	for c.Execute() {
	}
}

var draStatusControllerWorkQueueTracer = &traceUtils.Tracer{Threshold: time.Second}

func (c *DRAStatusController) Execute() bool {
	if !c.clusterConfig.GPUsWithDRAGateEnabled() && !c.clusterConfig.HostDevicesWithDRAEnabled() {
		return false
	}
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	draStatusControllerWorkQueueTracer.StartTrace(key, "dra-status-controller VMI workqueue", trace.Field{Key: "Workqueue Key", Value: key})
	defer draStatusControllerWorkQueueTracer.StopTrace(key)

	defer c.queue.Done(key)
	err := c.execute(key)

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *DRAStatusController) execute(key string) error {
	obj, exists, err := c.vmiIndexer.GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}
	vmi := obj.(*v1.VirtualMachineInstance)
	if vmi == nil {
		return fmt.Errorf("nil vmi reference")
	}
	logger := log.Log.Object(vmi)

	if vmi.DeletionTimestamp != nil {
		// object is being deleted, do not process it
		log.Log.Info("vmi being deleted, dra status controller skipping")
		return nil
	}
	// Only consider pods which belong to this vmi
	// excluding unfinalized migration targets from this list.
	pod, err := controller.CurrentVMIPod(vmi, c.podIndexer)
	if err != nil {
		logger.Reason(err).Error("Failed to fetch pods for namespace from cache.")
		return err
	}
	if pod == nil {
		return fmt.Errorf("nil pod reference for vmi")
	}

	err = c.updateStatus(logger, vmi, pod)
	if err != nil {
		logger.Reason(err).Error("error updating status")
		c.recorder.Eventf(vmi, k8sv1.EventTypeWarning, "VMIUpdateStatusFailedForDRADevices", "error updating status: %v", err)
		return err
	}
	c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, "VMIUpdatedForDRADevices", "updated status")

	return nil
}

func (c *DRAStatusController) updateStatus(logger *log.FilteredLogger, vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	key, err := controller.KeyFunc(vmi)
	if err != nil {
		return err
	}
	defer draStatusControllerWorkQueueTracer.StepTrace(key, "updateStatus", trace.Field{Key: "VMI Name", Value: vmi.Name})

	if !isPodResourceClaimStatusFilled(logger, pod) {
		logger.Infof("waiting for pod %s/%s resource claim status to be filled", pod.Namespace, pod.Name)
		return nil
	}

	var (
		gpuStatuses        []v1.DeviceStatusInfo
		hostDeviceStatuses []v1.DeviceStatusInfo
	)

	if c.clusterConfig.GPUsWithDRAGateEnabled() {
		gpuDeviceInfo, err := getGPUDevicesFromVMISpec(vmi)
		if err != nil {
			return err
		}

		gpuStatuses, err = c.getGPUStatuses(gpuDeviceInfo, pod)
		if err != nil {
			return err
		}
	}

	if c.clusterConfig.HostDevicesWithDRAEnabled() {
		hostDeviceInfo, err := c.getHostDevicesFromVMISpec(vmi)
		if err != nil {
			return err
		}

		hostDeviceStatuses, err = c.getHostDeviceStatuses(hostDeviceInfo, pod)
		if err != nil {
			return err
		}
	}

	newDeviceStatus := &v1.DeviceStatus{}
	if gpuStatuses != nil {
		newDeviceStatus.GPUStatuses = gpuStatuses
	}
	if hostDeviceStatuses != nil {
		newDeviceStatus.HostDeviceStatuses = hostDeviceStatuses
	}

	allReconciled := true
	if c.clusterConfig.GPUsWithDRAGateEnabled() {
		allReconciled = drautil.IsAllDRAGPUsReconciled(vmi, newDeviceStatus)
	}

	if c.clusterConfig.HostDevicesWithDRAEnabled() {
		allReconciled = allReconciled && drautil.IsAllDRAHostDevicesReconciled(vmi, newDeviceStatus)
	}

	if reflect.DeepEqual(vmi.Status.DeviceStatus, newDeviceStatus) && allReconciled {
		logger.V(4).Infof("All enabled DRA devices are reconciled nothing more to do")
		return nil
	}

	logger.V(4).Infof("updating VMI device status with DRA deviceattributes")
	ps := patch.New(
		patch.WithTest("/status/deviceStatus", vmi.Status.DeviceStatus),
		patch.WithReplace("/status/deviceStatus", newDeviceStatus),
	)

	patchBytes, err := ps.GeneratePayload()
	if err != nil {
		return err
	}
	logger.V(4).Infof("patching vmi device status")
	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.TODO(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		logger.Errorf("error patching VMI: %#v, %#v", errors.ReasonForError(err), err)
		return err
	}
	logger.V(6).Infof("patching vmi status successful")
	return nil
}

func isPodResourceClaimStatusFilled(logger *log.FilteredLogger, pod *k8sv1.Pod) bool {
	if pod.Status.ResourceClaimStatuses == nil {
		return false
	}
	if len(pod.Spec.ResourceClaims) != len(pod.Status.ResourceClaimStatuses) {
		var want, got []string
		for _, status := range pod.Status.ResourceClaimStatuses {
			if status.ResourceClaimName != nil {
				got = append(got, status.Name)
			}
		}
		for _, rc := range pod.Spec.ResourceClaims {
			want = append(want, rc.Name)
		}
		logger.V(4).Infof("do not have enough resource claim statuses to proceed further, want vs got: %v",
			cmp.Diff(want, got))
		return false
	}
	logger.V(6).Infof("all the pod resource claim statuses have been filled")
	return true
}

func getGPUDevicesFromVMISpec(vmi *v1.VirtualMachineInstance) ([]DeviceInfo, error) {
	var gpuDevices []DeviceInfo
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if !drautil.IsGPUDRA(gpu) {
			continue
		}
		gpuDevices = append(gpuDevices, DeviceInfo{
			VMISpecClaimName:   *gpu.ClaimName,
			VMISpecRequestName: *gpu.RequestName,
			DeviceStatusInfo: &v1.DeviceStatusInfo{
				Name:                      gpu.Name,
				DeviceResourceClaimStatus: nil,
			},
		})
	}
	return gpuDevices, nil
}

func (c *DRAStatusController) getGPUStatuses(gpuInfos []DeviceInfo, pod *k8sv1.Pod) ([]v1.DeviceStatusInfo, error) {
	statuses := make([]v1.DeviceStatusInfo, 0, len(gpuInfos))
	for _, info := range gpuInfos {
		st, err := c.getGPUStatus(info, pod)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, st)
	}
	return statuses, nil
}

func (c *DRAStatusController) getGPUStatus(gpuInfo DeviceInfo, pod *k8sv1.Pod) (v1.DeviceStatusInfo, error) {
	gpuStatus := v1.DeviceStatusInfo{
		Name: gpuInfo.Name,
		DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
			ResourceClaimName: getResourceClaimNameForDevice(gpuInfo.VMISpecClaimName, pod),
		},
	}

	if gpuStatus.DeviceResourceClaimStatus.ResourceClaimName == nil {
		return gpuStatus, nil
	}

	device, err := c.getAllocatedDevice(pod.Namespace, *gpuStatus.DeviceResourceClaimStatus.ResourceClaimName, gpuInfo.VMISpecRequestName)
	if err != nil {
		return gpuStatus, err
	}
	if device == nil {
		return gpuStatus, nil
	}

	gpuStatus.DeviceResourceClaimStatus.Name = &device.Device
	pciAddress, mDevUUID, err := c.getDeviceAttributes(pod.Spec.NodeName, device.Device, device.Driver)
	if err != nil {
		return gpuStatus, err
	}
	attrs := v1.DeviceAttribute{}
	if pciAddress != "" {
		attrs.PCIAddress = &pciAddress
	}
	if mDevUUID != "" {
		attrs.MDevUUID = &mDevUUID
	}
	gpuStatus.DeviceResourceClaimStatus.Attributes = &attrs

	return gpuStatus, nil
}

func getResourceClaimNameForDevice(claimName string, pod *k8sv1.Pod) *string {
	for _, rc := range pod.Status.ResourceClaimStatuses {
		if rc.Name == claimName {
			return rc.ResourceClaimName
		}
	}
	return nil
}

func (c *DRAStatusController) getAllocatedDevice(resourceClaimNamespace, resourceClaimName, requestName string) (*resourcev1beta2.DeviceRequestAllocationResult, error) {
	key := controller.NamespacedKey(resourceClaimNamespace, resourceClaimName)
	obj, exists, err := c.resourceClaimIndexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("resource claim %s does not exist", key)
	}
	resourceClaim := obj.(*resourcev1beta2.ResourceClaim)

	if resourceClaim.Status.Allocation == nil {
		return nil, nil
	}
	if resourceClaim.Status.Allocation.Devices.Results == nil {
		return nil, nil
	}

	for _, status := range resourceClaim.Status.Allocation.Devices.Results {
		if status.Request == requestName {
			return status.DeepCopy(), nil
		}
	}

	return nil, nil
}

// getDeviceAttributes returns the pciAddress and mdevUUID of the device. It will return both if found, otherwise it will return empty strings
func (c *DRAStatusController) getDeviceAttributes(nodeName string, deviceName, driverName string) (string, string, error) {
	resourceSlices, err := c.resourceSliceIndexer.ByIndex(indexByNodeName, nodeName)
	if err != nil {
		return "", "", err
	}
	if len(resourceSlices) == 0 {
		return "", "", fmt.Errorf("no resource slice objects found in cache")
	}

	pciAddress := ""
	mdevUUID := ""
	for _, obj := range resourceSlices {
		rs := obj.(*resourcev1beta2.ResourceSlice)
		if rs.Spec.Driver == driverName {
			for _, device := range rs.Spec.Devices {
				if device.Name == deviceName {
					for key, value := range device.Attributes {
						if string(key) == PCIAddressDeviceAttributeKey {
							pciAddress = *value.StringValue
						} else if string(key) == MDevUUIDDeviceAttributeKey {
							mdevUUID = *value.StringValue
						}
					}
					if pciAddress == "" && mdevUUID == "" {
						return "", "", fmt.Errorf("neither pciAddress nor mdevUUIDa attribute found for device %s", deviceName)
					}
					return pciAddress, mdevUUID, nil
				}
			}
		}
	}
	return pciAddress, mdevUUID, nil
}

func indexResourceSliceByNodeName(obj interface{}) ([]string, error) {
	rs, ok := obj.(*resourcev1beta2.ResourceSlice)
	if !ok {
		return nil, nil
	}
	if rs.Spec.NodeName == nil {
		return nil, nil
	}
	return []string{*rs.Spec.NodeName}, nil
}

func (c *DRAStatusController) getHostDevicesFromVMISpec(vmi *v1.VirtualMachineInstance) ([]DeviceInfo, error) {
	var hostDevices []DeviceInfo
	for _, hostDevice := range vmi.Spec.Domain.Devices.HostDevices {
		if !drautil.IsHostDeviceDRA(hostDevice) {
			continue
		}
		hostDevices = append(hostDevices, DeviceInfo{
			VMISpecClaimName:   *hostDevice.ClaimRequest.ClaimName,
			VMISpecRequestName: *hostDevice.ClaimRequest.RequestName,
			DeviceStatusInfo: &v1.DeviceStatusInfo{
				Name:                      hostDevice.Name,
				DeviceResourceClaimStatus: nil,
			},
		})
	}
	return hostDevices, nil
}

func (c *DRAStatusController) getHostDeviceStatuses(hostDeviceInfos []DeviceInfo, pod *k8sv1.Pod) ([]v1.DeviceStatusInfo, error) {
	statuses := make([]v1.DeviceStatusInfo, 0, len(hostDeviceInfos))
	for _, info := range hostDeviceInfos {
		st, err := c.getHostDeviceStatus(info, pod)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, st)
	}
	return statuses, nil
}

func (c *DRAStatusController) getHostDeviceStatus(hostDeviceInfo DeviceInfo, pod *k8sv1.Pod) (v1.DeviceStatusInfo, error) {
	hostDeviceStatus := v1.DeviceStatusInfo{
		Name: hostDeviceInfo.Name,
		DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
			ResourceClaimName: getResourceClaimNameForDevice(hostDeviceInfo.VMISpecClaimName, pod),
		},
	}

	if hostDeviceStatus.DeviceResourceClaimStatus.ResourceClaimName == nil {
		return hostDeviceStatus, nil
	}

	device, err := c.getAllocatedDevice(pod.Namespace, *hostDeviceStatus.DeviceResourceClaimStatus.ResourceClaimName, hostDeviceInfo.VMISpecRequestName)
	if err != nil {
		return hostDeviceStatus, err
	}
	if device == nil {
		return hostDeviceStatus, nil
	}

	hostDeviceStatus.DeviceResourceClaimStatus.Name = &device.Device
	pciAddress, mDevUUID, err := c.getDeviceAttributes(pod.Spec.NodeName, device.Device, device.Driver)
	if err != nil {
		return hostDeviceStatus, err
	}
	attrs := v1.DeviceAttribute{}
	if pciAddress != "" {
		attrs.PCIAddress = &pciAddress
	}
	if mDevUUID != "" {
		attrs.MDevUUID = &mDevUUID
	}
	hostDeviceStatus.DeviceResourceClaimStatus.Attributes = &attrs

	return hostDeviceStatus, nil
}
