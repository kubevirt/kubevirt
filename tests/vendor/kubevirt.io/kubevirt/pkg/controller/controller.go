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

func CurrentVMIPod(vmi *v1.VirtualMachineInstance, podInformer cache.SharedIndexInformer) (*k8sv1.Pod, error) {

	// current pod is the most recent pod created on the current VMI node
	// OR the most recent pod created if no VMI node is set.

	// Get all pods from the namespace
	objs, err := podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vmi.Namespace)
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

func VMIActivePodsCount(vmi *v1.VirtualMachineInstance, vmiPodInformer cache.SharedIndexInformer) int {

	objs, err := vmiPodInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vmi.Namespace)
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

func AttachmentPods(ownerPod *k8sv1.Pod, podInformer cache.SharedIndexInformer) ([]*k8sv1.Pod, error) {
	objs, err := podInformer.GetIndexer().ByIndex(cache.NamespaceIndex, ownerPod.Namespace)
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
