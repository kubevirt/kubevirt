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

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
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
			Do().
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
			Watch()
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
