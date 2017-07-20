/*
 * This file is part of the kubevirt project
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

package kubecli

import (
	"runtime/debug"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/logging"
)

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
func NewListWatchFromClient(c cache.Getter, resource string, namespace string, fieldSelector fields.Selector, labelSelector labels.Selector) *cache.ListWatch {
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, api.ParameterCodec).
			FieldsSelectorParam(fieldSelector).
			LabelsSelectorParam(labelSelector).
			Do().
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		return c.Get().
			Prefix("watch").
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, api.ParameterCodec).
			FieldsSelectorParam(fieldSelector).
			LabelsSelectorParam(labelSelector).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func HandlePanic() {
	if r := recover(); r != nil {
		logging.DefaultLogger().Critical().Log("stacktrace", debug.Stack(), "msg", r)
	}
}

func NewResourceEventHandlerFuncsForWorkqueue(queue workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
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

type ControllerDispatch interface {
	Execute( /*cache*/ cache.Store /*queue*/, workqueue.RateLimitingInterface /*key*/, interface{})
}

func NewControllerFromInformer(informer cache.SharedIndexInformer, queue workqueue.RateLimitingInterface, dispatch ControllerDispatch) *Controller {
	return &Controller{
		informer: informer,
		indexer:  informer.GetStore(),
		queue:    queue,
		dispatch: dispatch,
	}
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
	logging.DefaultLogger().Info().Msg("Starting controller.")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	logging.DefaultLogger().Info().Msg("Stopping controller.")
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
