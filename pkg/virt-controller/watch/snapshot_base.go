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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package watch

import (
	"fmt"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

// SnapshotController is resonsible for snapshotting VMs
type SnapshotController struct {
	client kubecli.KubevirtClient

	vmSnapshotQueue        workqueue.RateLimitingInterface
	vmSnapshotContentQueue workqueue.RateLimitingInterface

	vmSnapshotInformer        cache.SharedIndexInformer
	vmSnapshotContentInformer cache.SharedIndexInformer
	vmInformer                cache.SharedIndexInformer

	recorder record.EventRecorder

	resyncPeriod time.Duration
}

// NewSnapshotController creates a new SnapshotController
func NewSnapshotController(
	client kubecli.KubevirtClient,
	vmSnapshotInformer cache.SharedIndexInformer,
	vmSnapshotContentInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	resyncPeriod time.Duration,
) *SnapshotController {

	ctrl := &SnapshotController{
		client:                    client,
		vmSnapshotQueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "snapshot-controller-vmsnapshot"),
		vmSnapshotContentQueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "snapshot-controller-vmsnapshotcontent"),
		vmSnapshotInformer:        vmSnapshotInformer,
		vmSnapshotContentInformer: vmSnapshotContentInformer,
		vmInformer:                vmInformer,
		recorder:                  recorder,
		resyncPeriod:              resyncPeriod,
	}

	vmSnapshotInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { ctrl.enqueueVMSnapshotWork(obj) },
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.enqueueVMSnapshotWork(newObj) },
		},
		ctrl.resyncPeriod,
	)

	vmSnapshotContentInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { ctrl.enqueueVMSnapshotContentWork(obj) },
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.enqueueVMSnapshotContentWork(newObj) },
		},
		ctrl.resyncPeriod,
	)

	vmInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { ctrl.enqueueVMSnapshotsForVM(obj) },
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.enqueueVMSnapshotsForVM(newObj) },
		},
		ctrl.resyncPeriod,
	)

	return ctrl
}

// Run the controller
func (ctrl *SnapshotController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.vmSnapshotQueue.ShutDown()
	defer ctrl.vmSnapshotContentQueue.ShutDown()

	log.Log.Info("Starting snapshot controller.")
	defer log.Log.Info("Shutting down snapshot controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.vmSnapshotInformer.HasSynced,
		ctrl.vmSnapshotContentInformer.HasSynced,
		ctrl.vmInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.vmSnapshotWorker, time.Second, stopCh)
		go wait.Until(ctrl.vmSnapshotContentWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (ctrl *SnapshotController) vmSnapshotWorker() {
	for ctrl.processVMSnapshotWorkItem() {
	}
}

func (ctrl *SnapshotController) vmSnapshotContentWorker() {
	for ctrl.processVMSnapshotContentWorkItem() {
	}
}

func (ctrl *SnapshotController) processVMSnapshotWorkItem() bool {
	return processWorkItem(ctrl.vmSnapshotQueue, func(key string) error {
		log.Log.V(3).Infof("vmSnapshot worker processing key [%s]", key)

		storeObj, exists, err := ctrl.vmSnapshotInformer.GetStore().GetByKey(key)
		if err != nil {
			return err
		}

		if exists {
			vmSnapshot, ok := storeObj.(*vmsnapshotv1alpha1.VirtualMachineSnapshot)
			if !ok {
				return fmt.Errorf("unexpected resource %+v", storeObj)
			}

			if err = ctrl.updateVMSnapshot(vmSnapshot); err != nil {
				return err
			}
		}

		return nil
	})
}

func (ctrl *SnapshotController) processVMSnapshotContentWorkItem() bool {
	return processWorkItem(ctrl.vmSnapshotContentQueue, func(key string) error {
		log.Log.V(3).Infof("vmSnapshotContent worker processing key [%s]", key)

		storeObj, exists, err := ctrl.vmSnapshotContentInformer.GetStore().GetByKey(key)
		if err != nil {
			return err
		}

		if exists {
			vmSnapshotContent, ok := storeObj.(*vmsnapshotv1alpha1.VirtualMachineSnapshotContent)
			if !ok {
				return fmt.Errorf("unexpected resource %+v", storeObj)
			}

			if err = ctrl.updateVMSnapshotContent(vmSnapshotContent); err != nil {
				return err
			}
		}

		return nil
	})
}

func processWorkItem(queue workqueue.RateLimitingInterface, handler func(string) error) bool {
	obj, shutdown := queue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		log.Log.V(3).Infof("processing key [%s]", key)

		if err := handler(key); err != nil {
			queue.AddRateLimited(key)
			return err
		}

		queue.Forget(obj)

		return nil

	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (ctrl *SnapshotController) enqueueVMSnapshotWork(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmSnapshot, ok := obj.(*vmsnapshotv1alpha1.VirtualMachineSnapshot); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(vmSnapshot)
		if err != nil {
			log.Log.Errorf("failed to get key from object: %v, %v", err, vmSnapshot)
			return
		}
		log.Log.V(3).Infof("enqueued %q for sync", objName)
		ctrl.vmSnapshotQueue.AddRateLimited(objName)
	}
}

func (ctrl *SnapshotController) enqueueVMSnapshotContentWork(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if content, ok := obj.(*vmsnapshotv1alpha1.VirtualMachineSnapshotContent); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(content)
		if err != nil {
			log.Log.Errorf("failed to get key from object: %v, %v", err, content)
			return
		}

		if content.Spec.VirtualMachineSnapshotName != nil {
			k := cacheKeyFunc(content.Namespace, *content.Spec.VirtualMachineSnapshotName)
			log.Log.V(5).Infof("enqueued vmsnapshot %q for sync", k)
			ctrl.vmSnapshotQueue.AddRateLimited(k)
		}

		log.Log.V(5).Infof("enqueued %q for sync", objName)
		ctrl.vmSnapshotContentQueue.AddRateLimited(objName)
	}
}

func (ctrl *SnapshotController) enqueueVMSnapshotsForVM(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vm, ok := obj.(*kubevirtv1.VirtualMachine); ok {
		keys, err := ctrl.vmSnapshotInformer.GetIndexer().IndexKeys("vm", vm.Name)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, k := range keys {
			ctrl.vmSnapshotQueue.AddRateLimited(k)
		}
	}
}
