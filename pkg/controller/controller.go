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
	"runtime/debug"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"fmt"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

const (
	// BurstReplicas is the maximum amount of requests in a row for CRUD operations on resources by controllers,
	// to avoid unintentional DoS
	BurstReplicas uint = 250
)

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
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
		log.Log.Level(log.CRITICAL).Log("stacktrace", debug.Stack(), "msg", r)
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

type Controller struct {
	indexer  cache.Store
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	dispatch ControllerDispatch
}

func NewController(lw cache.ListerWatcher, queue workqueue.RateLimitingInterface, objType runtime.Object, dispatch ControllerDispatch) (cache.Store, *Controller) {

	indexer, informer := cache.NewIndexerInformer(lw, objType, 0, NewResourceEventHandlerFuncsForWorkqueue(queue), cache.Indexers{})
	return NewControllerFromInformer(indexer, informer, queue, dispatch)
}

type ControllerDispatch interface {
	Execute( /*cache*/ cache.Store /*queue*/, workqueue.RateLimitingInterface /*key*/, interface{})
}

func NewControllerFromInformer(indexer cache.Store, informer cache.Controller, queue workqueue.RateLimitingInterface, dispatch ControllerDispatch) (cache.Store, *Controller) {
	c := &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
		dispatch: dispatch,
	}
	return indexer, c
}

type ControllerFunc func(cache.Store, workqueue.RateLimitingInterface, interface{})

func (c *Controller) callControllerFn(s cache.Store, w workqueue.RateLimitingInterface) bool {
	quit := !Dequeue(s, w, c.dispatch)
	return quit
}

func Dequeue(s cache.Store, w workqueue.RateLimitingInterface, dispatch ControllerDispatch) bool {
	key, quit := w.Get()
	if quit {
		return false
	} else {
		defer w.Done(key)
		dispatch.Execute(s, w, key)
		return true
	}
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting controller.")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping controller.")
}

func (c *Controller) StartInformer(stopCh chan struct{}) {
	go c.informer.Run(stopCh)
}

func (c *Controller) WaitForSync(stopCh chan struct{}) {
	cache.WaitForCacheSync(stopCh, c.informer.HasSynced)
}

func (c *Controller) runWorker() {
	for c.callControllerFn(c.indexer, c.queue) {
	}
}

// Shut down the embedded queue. After the shutdown was issued, all items already in the queue will be processed but no
// new items will be accepted. It is possible to wait via #WaitUntilDone() until the last item was processed.
func (c *Controller) ShutDownQueue() {
	c.queue.ShutDown()
}

func VirtualMachineKey(vm *v1.VirtualMachine) string {
	return fmt.Sprintf("%v/%v", vm.ObjectMeta.Namespace, vm.ObjectMeta.Name)
}

func VirtualMachineKeys(vms []v1.VirtualMachine) []string {
	keys := []string{}
	for _, vm := range vms {
		keys = append(keys, VirtualMachineKey(&vm))
	}
	return keys
}
