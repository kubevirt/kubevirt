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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
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
		return c.Get().
			Prefix("watch").
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch(context.Background())
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func HandlePanic() {
	if r := recover(); r != nil {
		log.Log.Level(log.FATAL).Log("stacktrace", debug.Stack(), "msg", r)
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

func NewResourceEventHandlerFuncsForFunc(f func(interface{})) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			f(obj)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			f(new)

		},
		DeleteFunc: func(obj interface{}) {
			f(obj)
		},
	}
}

func MigrationKey(migration *v1.VirtualMachineInstanceMigration) string {
	return fmt.Sprintf("%v/%v", migration.ObjectMeta.Namespace, migration.ObjectMeta.Name)
}

func VirtualMachineKey(vmi *v1.VirtualMachineInstance) string {
	return fmt.Sprintf("%v/%v", vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name)
}

func PodKey(pod *k8sv1.Pod) string {
	return fmt.Sprintf("%v/%v", pod.Namespace, pod.Name)
}

func DataVolumeKey(dataVolume *cdiv1.DataVolume) string {
	return fmt.Sprintf("%v/%v", dataVolume.Namespace, dataVolume.Name)
}

func VirtualMachineKeys(vmis []*v1.VirtualMachineInstance) []string {
	keys := []string{}
	for _, vmi := range vmis {
		keys = append(keys, VirtualMachineKey(vmi))
	}
	return keys
}

func PodKeys(pods []*k8sv1.Pod) []string {
	keys := []string{}
	for _, pod := range pods {
		keys = append(keys, PodKey(pod))
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
				newVolume.VolumeSource.PersistentVolumeClaim = request.AddVolumeOptions.VolumeSource.PersistentVolumeClaim
			} else if request.AddVolumeOptions.VolumeSource.DataVolume != nil {

				newVolume.VolumeSource.DataVolume = request.AddVolumeOptions.VolumeSource.DataVolume
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
