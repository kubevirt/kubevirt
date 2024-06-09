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

package controller

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const (
	// BurstReplicas is the maximum amount of requests in a row for CRUD operations on resources by controllers,
	// to avoid unintentional DoS
	BurstReplicas uint = 250
)

// Reasons for vmi events
const (
	// FailedCreatePodReason is added in an event and in a vmi controller condition
	// when a pod for a vmi controller failed to be created.
	FailedCreatePodReason = "FailedCreate"
	// SuccessfulCreatePodReason is added in an event when a pod for a vmi controller
	// is successfully created.
	SuccessfulCreatePodReason = "SuccessfulCreate"
	// FailedDeletePodReason is added in an event and in a vmi controller condition
	// when a pod for a vmi controller failed to be deleted.
	FailedDeletePodReason = "FailedDelete"
	// SuccessfulDeletePodReason is added in an event when a pod for a vmi controller
	// is successfully deleted.
	SuccessfulDeletePodReason = "SuccessfulDelete"
	// FailedHandOverPodReason is added in an event and in a vmi controller condition
	// when transferring the pod ownership from the controller to virt-hander fails.
	FailedHandOverPodReason = "FailedHandOver"
	// FailedBackendStorageCreateReason is added when the creation of the backend storage PVC fails.
	FailedBackendStorageCreateReason = "FailedBackendStorageCreate"
	// FailedBackendStorageProbeReason is added when probing the backend storage PVC fails.
	FailedBackendStorageProbeReason = "FailedBackendStorageProbe"
	// BackendStorageNotReadyReason is added when the backend storage PVC is pending.
	BackendStorageNotReadyReason = "BackendStorageNotReady"
	// SuccessfulHandOverPodReason is added in an event
	// when the pod ownership transfer from the controller to virt-hander succeeds.
	SuccessfulHandOverPodReason = "SuccessfulHandOver"
	// FailedDataVolumeImportReason is added in an event when a dynamically generated
	// dataVolume reaches the failed status phase.
	FailedDataVolumeImportReason = "FailedDataVolumeImport"
	// FailedGuaranteePodResourcesReason is added in an event and in a vmi controller condition
	// when a pod has been created without a Guaranteed resources.
	FailedGuaranteePodResourcesReason = "FailedGuaranteeResources"
	// FailedGatherhingClusterTopologyHints is added if the cluster topology hints can't be collected for a VMI by virt-controller
	FailedGatherhingClusterTopologyHints = "FailedGatherhingClusterTopologyHints"
	// FailedPvcNotFoundReason is added in an event
	// when a PVC for a volume was not found.
	FailedPvcNotFoundReason = "FailedPvcNotFound"
	// SuccessfulMigrationReason is added when a migration attempt completes successfully
	SuccessfulMigrationReason = "SuccessfulMigration"
	// FailedMigrationReason is added when a migration attempt fails
	FailedMigrationReason = "FailedMigration"
	// SuccessfulAbortMigrationReason is added when an attempt to abort migration completes successfully
	SuccessfulAbortMigrationReason = "SuccessfulAbortMigration"
	// MigrationTargetPodUnschedulable is added a migration target pod enters Unschedulable phase
	MigrationTargetPodUnschedulable = "migrationTargetPodUnschedulable"
	// FailedAbortMigrationReason is added when an attempt to abort migration fails
	FailedAbortMigrationReason = "FailedAbortMigration"
	// MissingAttachmentPodReason is set when we have a hotplugged volume, but the attachment pod is missing
	MissingAttachmentPodReason = "MissingAttachmentPod"
	// PVCNotReadyReason is set when the PVC is not ready to be hot plugged.
	PVCNotReadyReason = "PVCNotReady"
	// FailedHotplugSyncReason is set when a hotplug specific failure occurs during sync
	FailedHotplugSyncReason = "FailedHotplugSync"
	// ErrImagePullReason is set when an error has occured while pulling an image for a containerDisk VM volume.
	ErrImagePullReason = "ErrImagePull"
	// ImagePullBackOffReason is set when an error has occured while pulling an image for a containerDisk VM volume,
	// and that kubelet is backing off before retrying.
	ImagePullBackOffReason = "ImagePullBackOff"
	// NoSuitableNodesForHostModelMigration is set when a VMI with host-model CPU mode tries to migrate but no node
	// is suitable for migration (since CPU model / required features are not supported)
	NoSuitableNodesForHostModelMigration = "NoSuitableNodesForHostModelMigration"
	// FailedPodPatchReason is set when a pod patch error occurs during sync
	FailedPodPatchReason = "FailedPodPatch"
	// MigrationBackoffReason is set when an error has occured while migrating
	// and virt-controller is backing off before retrying.
	MigrationBackoffReason = "MigrationBackoff"
)

type PodCacheStore struct {
	indexer cache.Indexer
}

func NewPodCacheStore(indexer cache.Indexer) *PodCacheStore {
	return &PodCacheStore{indexer: indexer}
}

func (p *PodCacheStore) CurrentPod(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	return CurrentVMIPod(vmi, p.indexer)
}

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, kubevirtNamespace and field selector.
func NewListWatchFromClient(c cache.Getter, resource string, namespace string, fieldSelector fields.Selector, labelSelector labels.Selector) *cache.ListWatch {
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Do(context.Background()).
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.FieldSelector = fieldSelector.String()
		options.LabelSelector = labelSelector.String()
		options.Watch = true
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch(context.Background())
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func HandlePanic() {
	if r := recover(); r != nil {
		// Ignoring error - There is nothing to do, if logging fails
		_ = log.Log.Level(log.FATAL).Log("stacktrace", debug.Stack(), "msg", r)
	}
}

func NewResourceEventHandlerFuncsForWorkqueue(queue workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := KeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := KeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := KeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}
}

func MigrationKey(migration *v1.VirtualMachineInstanceMigration) string {
	return fmt.Sprintf("%v/%v", migration.ObjectMeta.Namespace, migration.ObjectMeta.Name)
}

func VirtualMachineInstanceKey(vmi *v1.VirtualMachineInstance) string {
	return fmt.Sprintf("%v/%v", vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name)
}

func VirtualMachineKey(vm *v1.VirtualMachine) string {
	return fmt.Sprintf("%v/%v", vm.ObjectMeta.Namespace, vm.ObjectMeta.Name)
}

func PodKey(pod *k8sv1.Pod) string {
	return fmt.Sprintf("%v/%v", pod.Namespace, pod.Name)
}

func DataVolumeKey(dataVolume *cdiv1.DataVolume) string {
	return fmt.Sprintf("%v/%v", dataVolume.Namespace, dataVolume.Name)
}

func VirtualMachineInstanceKeys(vmis []*v1.VirtualMachineInstance) []string {
	keys := []string{}
	for _, vmi := range vmis {
		keys = append(keys, VirtualMachineInstanceKey(vmi))
	}
	return keys
}

func VirtualMachineKeys(vms []*v1.VirtualMachine) []string {
	keys := []string{}
	for _, vm := range vms {
		keys = append(keys, VirtualMachineKey(vm))
	}
	return keys
}

func HasFinalizer(object metav1.Object, finalizer string) bool {
	for _, f := range object.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}

func RemoveFinalizer(object metav1.Object, finalizer string) {
	filtered := []string{}
	for _, f := range object.GetFinalizers() {
		if f != finalizer {
			filtered = append(filtered, f)
		}
	}
	object.SetFinalizers(filtered)
}

func AddFinalizer(object metav1.Object, finalizer string) {
	if HasFinalizer(object, finalizer) {
		return
	}
	object.SetFinalizers(append(object.GetFinalizers(), finalizer))
}

func ObservedLatestApiVersionAnnotation(object metav1.Object) bool {
	annotations := object.GetAnnotations()
	if annotations == nil {
		return false
	}

	version, ok := annotations[v1.ControllerAPILatestVersionObservedAnnotation]
	if !ok || version != v1.ApiLatestVersion {
		return false
	}
	return true
}

func SetLatestApiVersionAnnotation(object metav1.Object) {
	annotations := object.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[v1.ControllerAPILatestVersionObservedAnnotation] = v1.ApiLatestVersion
	annotations[v1.ControllerAPIStorageVersionObservedAnnotation] = v1.ApiStorageVersion
	object.SetAnnotations(annotations)
}

func ApplyVolumeRequestOnVMISpec(vmiSpec *v1.VirtualMachineInstanceSpec, request *v1.VirtualMachineVolumeRequest) *v1.VirtualMachineInstanceSpec {
	if request.AddVolumeOptions != nil {
		alreadyAdded := false
		for _, volume := range vmiSpec.Volumes {
			if volume.Name == request.AddVolumeOptions.Name {
				alreadyAdded = true
				break
			}
		}

		if !alreadyAdded {
			newVolume := v1.Volume{
				Name: request.AddVolumeOptions.Name,
			}

			if request.AddVolumeOptions.VolumeSource.PersistentVolumeClaim != nil {
				pvcSource := request.AddVolumeOptions.VolumeSource.PersistentVolumeClaim.DeepCopy()
				pvcSource.Hotpluggable = true
				newVolume.VolumeSource.PersistentVolumeClaim = pvcSource
			} else if request.AddVolumeOptions.VolumeSource.DataVolume != nil {
				dvSource := request.AddVolumeOptions.VolumeSource.DataVolume.DeepCopy()
				dvSource.Hotpluggable = true
				newVolume.VolumeSource.DataVolume = dvSource
			}

			vmiSpec.Volumes = append(vmiSpec.Volumes, newVolume)

			if request.AddVolumeOptions.Disk != nil {
				newDisk := request.AddVolumeOptions.Disk.DeepCopy()
				newDisk.Name = request.AddVolumeOptions.Name

				vmiSpec.Domain.Devices.Disks = append(vmiSpec.Domain.Devices.Disks, *newDisk)
			}
		}

	} else if request.RemoveVolumeOptions != nil {

		newVolumesList := []v1.Volume{}
		newDisksList := []v1.Disk{}

		for _, volume := range vmiSpec.Volumes {
			if volume.Name != request.RemoveVolumeOptions.Name {
				newVolumesList = append(newVolumesList, volume)
			}
		}

		for _, disk := range vmiSpec.Domain.Devices.Disks {
			if disk.Name != request.RemoveVolumeOptions.Name {
				newDisksList = append(newDisksList, disk)
			}
		}

		vmiSpec.Volumes = newVolumesList
		vmiSpec.Domain.Devices.Disks = newDisksList
	}

	return vmiSpec
}

func CurrentVMIPod(vmi *v1.VirtualMachineInstance, podIndexer cache.Indexer) (*k8sv1.Pod, error) {

	// current pod is the most recent pod created on the current VMI node
	// OR the most recent pod created if no VMI node is set.

	// Get all pods from the namespace
	objs, err := podIndexer.ByIndex(cache.NamespaceIndex, vmi.Namespace)
	if err != nil {
		return nil, err
	}
	pods := []*k8sv1.Pod{}
	for _, obj := range objs {
		pod := obj.(*k8sv1.Pod)
		pods = append(pods, pod)
	}

	var curPod *k8sv1.Pod = nil
	for _, pod := range pods {
		if !IsControlledBy(pod, vmi) {
			continue
		}

		if vmi.Status.NodeName != "" &&
			vmi.Status.NodeName != pod.Spec.NodeName {
			// This pod isn't scheduled to the current node.
			// This can occur during the initial migration phases when
			// a new target node is being prepared for the VMI.
			continue
		}

		if curPod == nil || curPod.CreationTimestamp.Before(&pod.CreationTimestamp) {
			curPod = pod
		}
	}

	return curPod, nil
}

func VMIActivePodsCount(vmi *v1.VirtualMachineInstance, vmiPodIndexer cache.Indexer) int {

	objs, err := vmiPodIndexer.ByIndex(cache.NamespaceIndex, vmi.Namespace)
	if err != nil {
		return 0
	}

	running := 0
	for _, obj := range objs {
		pod := obj.(*k8sv1.Pod)

		if pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed {
			// not interested in terminated pods
			continue
		} else if !IsControlledBy(pod, vmi) {
			// not interested pods not associated with the vmi
			continue
		}
		running++
	}

	return running
}

func GeneratePatchBytes(ops []string) []byte {
	return []byte(fmt.Sprintf("[%s]", strings.Join(ops, ", ")))
}

func SetVMIPhaseTransitionTimestamp(oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) {
	if oldVMI.Status.Phase != newVMI.Status.Phase {
		for _, transitionTimeStamp := range newVMI.Status.PhaseTransitionTimestamps {
			if transitionTimeStamp.Phase == newVMI.Status.Phase {
				// already exists.
				return
			}
		}

		now := metav1.NewTime(time.Now())
		newVMI.Status.PhaseTransitionTimestamps = append(newVMI.Status.PhaseTransitionTimestamps, v1.VirtualMachineInstancePhaseTransitionTimestamp{
			Phase:                    newVMI.Status.Phase,
			PhaseTransitionTimestamp: now,
		})
	}
}

func SetVMIMigrationPhaseTransitionTimestamp(oldVMIMigration *v1.VirtualMachineInstanceMigration, newVMIMigration *v1.VirtualMachineInstanceMigration) {
	if oldVMIMigration.Status.Phase != newVMIMigration.Status.Phase {
		for _, transitionTimeStamp := range newVMIMigration.Status.PhaseTransitionTimestamps {
			if transitionTimeStamp.Phase == newVMIMigration.Status.Phase {
				// already exists.
				return
			}
		}

		now := metav1.NewTime(time.Now())
		newVMIMigration.Status.PhaseTransitionTimestamps = append(newVMIMigration.Status.PhaseTransitionTimestamps, v1.VirtualMachineInstanceMigrationPhaseTransitionTimestamp{
			Phase:                    newVMIMigration.Status.Phase,
			PhaseTransitionTimestamp: now,
		})
	}
}

func SetSourcePod(migration *v1.VirtualMachineInstanceMigration, vmi *v1.VirtualMachineInstance, podIndexer cache.Indexer) {
	if migration.Status.Phase != v1.MigrationPending {
		return
	}
	sourcePod, err := CurrentVMIPod(vmi, podIndexer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Warning("migration source pod not found")
	}
	if sourcePod != nil {
		if migration.Status.MigrationState == nil {
			migration.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{}
		}
		migration.Status.MigrationState.SourcePod = sourcePod.Name
	}

}

func VMIHasHotplugVolumes(vmi *v1.VirtualMachineInstance) bool {
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume != nil {
			return true
		}
	}
	for _, volume := range vmi.Spec.Volumes {
		if volume.DataVolume != nil && volume.DataVolume.Hotpluggable {
			return true
		}
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.Hotpluggable {
			return true
		}
	}
	return false
}

func vmiHasCondition(vmi *v1.VirtualMachineInstance, conditionType v1.VirtualMachineInstanceConditionType) bool {
	vmiConditionManager := NewVirtualMachineInstanceConditionManager()
	return vmiConditionManager.HasCondition(vmi, conditionType)
}

func VMIHasHotplugCPU(vmi *v1.VirtualMachineInstance) bool {
	return vmiHasCondition(vmi, v1.VirtualMachineInstanceVCPUChange)
}

func VMIHasHotplugMemory(vmi *v1.VirtualMachineInstance) bool {
	return vmiHasCondition(vmi, v1.VirtualMachineInstanceMemoryChange)
}

func AttachmentPods(ownerPod *k8sv1.Pod, podIndexer cache.Indexer) ([]*k8sv1.Pod, error) {
	objs, err := podIndexer.ByIndex(cache.NamespaceIndex, ownerPod.Namespace)
	if err != nil {
		return nil, err
	}
	attachmentPods := []*k8sv1.Pod{}
	for _, obj := range objs {
		pod := obj.(*k8sv1.Pod)
		ownerRef := GetControllerOf(pod)
		if ownerRef == nil || ownerRef.UID != ownerPod.UID {
			continue
		}
		attachmentPods = append(attachmentPods, pod)
	}
	return attachmentPods, nil
}

// IsPodReady treats the pod as ready to be handed over to virt-handler, as soon as all pods except
// the compute pod are ready.
func IsPodReady(pod *k8sv1.Pod) bool {
	if IsPodDownOrGoingDown(pod) {
		return false
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		// The compute container potentially holds a readiness probe for the VMI. Therefore
		// don't wait for the compute container to become ready (the VMI later on will trigger the change to ready)
		// and only check that the container started
		if containerStatus.Name == "compute" {
			if containerStatus.State.Running == nil {
				return false
			}
		} else if containerStatus.Name == "istio-proxy" {
			// When using istio the istio-proxy container will not be ready
			// until there is a service pointing to this pod.
			// We need to start the VM anyway
			if containerStatus.State.Running == nil {
				return false
			}

		} else if containerStatus.Ready == false {
			return false
		}
	}

	return pod.Status.Phase == k8sv1.PodRunning
}

func IsPodDownOrGoingDown(pod *k8sv1.Pod) bool {
	return PodIsDown(pod) || isComputeContainerDown(pod) || pod.DeletionTimestamp != nil
}

func IsPodFailedOrGoingDown(pod *k8sv1.Pod) bool {
	return isPodFailed(pod) || isComputeContainerFailed(pod) || pod.DeletionTimestamp != nil
}

func isComputeContainerDown(pod *k8sv1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "compute" {
			return containerStatus.State.Terminated != nil
		}
	}
	return false
}

func isComputeContainerFailed(pod *k8sv1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == "compute" {
			return containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0
		}
	}
	return false
}

func PodIsDown(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodSucceeded || pod.Status.Phase == k8sv1.PodFailed
}

func isPodFailed(pod *k8sv1.Pod) bool {
	return pod.Status.Phase == k8sv1.PodFailed
}

func PodExists(pod *k8sv1.Pod) bool {
	return pod != nil
}

func GetHotplugVolumes(vmi *v1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod) []*v1.Volume {
	hotplugVolumes := make([]*v1.Volume, 0)
	podVolumes := virtlauncherPod.Spec.Volumes
	vmiVolumes := vmi.Spec.Volumes

	podVolumeMap := make(map[string]k8sv1.Volume)
	for _, podVolume := range podVolumes {
		podVolumeMap[podVolume.Name] = podVolume
	}
	for _, vmiVolume := range vmiVolumes {
		if _, ok := podVolumeMap[vmiVolume.Name]; !ok && (vmiVolume.DataVolume != nil || vmiVolume.PersistentVolumeClaim != nil || vmiVolume.MemoryDump != nil) {
			hotplugVolumes = append(hotplugVolumes, vmiVolume.DeepCopy())
		}
	}
	return hotplugVolumes
}
