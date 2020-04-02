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
	"sync"
	"time"

	k8ssnapshotv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	volumeSnapshotCRD      = "volumesnapshots.snapshot.storage.k8s.io"
	volumeSnapshotClassCRD = "volumesnapshotclasses.snapshot.storage.k8s.io"
)

type informerFunc func(kubecli.KubevirtClient, time.Duration) cache.SharedIndexInformer

type dynamicInformer struct {
	stopCh   chan struct{}
	informer cache.SharedIndexInformer
	mutex    sync.Mutex

	informerFunc informerFunc
}

// SnapshotController is resonsible for snapshotting VMs
type SnapshotController struct {
	client kubecli.KubevirtClient

	vmSnapshotQueue        workqueue.RateLimitingInterface
	vmSnapshotContentQueue workqueue.RateLimitingInterface
	crdQueue               workqueue.RateLimitingInterface

	vmSnapshotInformer        cache.SharedIndexInformer
	vmSnapshotContentInformer cache.SharedIndexInformer
	vmInformer                cache.SharedIndexInformer

	storageClassInformer cache.SharedIndexInformer
	pvcInformer          cache.SharedIndexInformer
	crdInformer          cache.SharedIndexInformer

	dynamicInformerMap map[string]*dynamicInformer
	eventHandlerMap    map[string]cache.ResourceEventHandlerFuncs

	recorder record.EventRecorder

	resyncPeriod time.Duration
}

// NewSnapshotController creates a new SnapshotController
func NewSnapshotController(
	client kubecli.KubevirtClient,
	vmSnapshotInformer cache.SharedIndexInformer,
	vmSnapshotContentInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	storageClassInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	crdInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
	resyncPeriod time.Duration,
) *SnapshotController {

	ctrl := &SnapshotController{
		client:                    client,
		vmSnapshotQueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "snapshot-controller-vmsnapshot"),
		vmSnapshotContentQueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "snapshot-controller-vmsnapshotcontent"),
		crdQueue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "snapshot-controller-crd"),
		vmSnapshotInformer:        vmSnapshotInformer,
		vmSnapshotContentInformer: vmSnapshotContentInformer,
		storageClassInformer:      storageClassInformer,
		pvcInformer:               pvcInformer,
		vmInformer:                vmInformer,
		crdInformer:               crdInformer,
		recorder:                  recorder,
		resyncPeriod:              resyncPeriod,
	}

	ctrl.dynamicInformerMap = map[string]*dynamicInformer{
		volumeSnapshotCRD:      &dynamicInformer{informerFunc: controller.VolumeSnapshotInformer},
		volumeSnapshotClassCRD: &dynamicInformer{informerFunc: controller.VolumeSnapshotClassInformer},
	}

	ctrl.eventHandlerMap = map[string]cache.ResourceEventHandlerFuncs{
		volumeSnapshotCRD: cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVolumeSnapshot,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVolumeSnapshot(newObj) },
			DeleteFunc: ctrl.handleVolumeSnapshot,
		},
	}

	vmSnapshotInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshot,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshot(newObj) },
		},
		ctrl.resyncPeriod,
	)

	vmSnapshotContentInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshotContent,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshotContent(newObj) },
		},
		ctrl.resyncPeriod,
	)

	vmInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVM,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVM(newObj) },
		},
		ctrl.resyncPeriod,
	)

	crdInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleCRD,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleCRD(newObj) },
			DeleteFunc: ctrl.handleCRD,
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
	defer ctrl.crdQueue.ShutDown()

	log.Log.Info("Starting snapshot controller.")
	defer log.Log.Info("Shutting down snapshot controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.vmSnapshotInformer.HasSynced,
		ctrl.vmSnapshotContentInformer.HasSynced,
		ctrl.vmInformer.HasSynced,
		ctrl.storageClassInformer.HasSynced,
		ctrl.crdInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.vmSnapshotWorker, time.Second, stopCh)
		go wait.Until(ctrl.vmSnapshotContentWorker, time.Second, stopCh)
		go wait.Until(ctrl.crdWorker, time.Second, stopCh)
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

func (ctrl *SnapshotController) crdWorker() {
	for ctrl.processCRDWorkItem() {
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

func (ctrl *SnapshotController) processCRDWorkItem() bool {
	return processWorkItem(ctrl.crdQueue, func(key string) error {
		log.Log.V(3).Infof("CRD worker processing key [%s]", key)

		storeObj, exists, err := ctrl.crdInformer.GetStore().GetByKey(key)
		if err != nil {
			return err
		}

		if !exists {
			_, name, err := cache.SplitMetaNamespaceKey(key)
			if err != nil {
				return err
			}

			return ctrl.deleteDynamicInformer(name)
		}

		crd, ok := storeObj.(*extv1beta1.CustomResourceDefinition)
		if !ok {
			return fmt.Errorf("unexpected resource %+v", storeObj)
		}

		if crd.DeletionTimestamp != nil {
			return ctrl.deleteDynamicInformer(crd.Name)
		}

		return ctrl.ensureDynamicInformer(crd.Name)
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

func (ctrl *SnapshotController) handleVMSnapshot(obj interface{}) {
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
		ctrl.vmSnapshotQueue.Add(objName)
	}
}

func (ctrl *SnapshotController) handleVMSnapshotContent(obj interface{}) {
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
			ctrl.vmSnapshotQueue.Add(k)
		}

		log.Log.V(5).Infof("enqueued %q for sync", objName)
		ctrl.vmSnapshotContentQueue.Add(objName)
	}
}

func (ctrl *SnapshotController) handleVM(obj interface{}) {
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
			ctrl.vmSnapshotQueue.Add(k)
		}
	}
}

func (ctrl *SnapshotController) handleCRD(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if crd, ok := obj.(*extv1beta1.CustomResourceDefinition); ok {
		_, ok = ctrl.dynamicInformerMap[crd.Name]
		if ok {
			objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(crd)
			if err != nil {
				log.Log.Errorf("failed to get key from object: %v, %v", err, crd)
				return
			}

			log.Log.V(3).Infof("enqueued %q for sync", objName)
			ctrl.crdQueue.Add(objName)
		}
	}
}

func (ctrl *SnapshotController) handleVolumeSnapshot(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if volumeSnapshot, ok := obj.(*k8ssnapshotv1beta1.VolumeSnapshot); ok {
		keys, err := ctrl.vmSnapshotContentInformer.GetIndexer().IndexKeys("volumeSnapshot", volumeSnapshot.Name)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, k := range keys {
			ctrl.vmSnapshotContentQueue.Add(k)
		}
	}
}

func (ctrl *SnapshotController) getVolumeSnapshot(namespace, name string) (*k8ssnapshotv1beta1.VolumeSnapshot, error) {
	di := ctrl.dynamicInformerMap[volumeSnapshotCRD]
	di.mutex.Lock()
	defer di.mutex.Unlock()

	if di.informer == nil {
		return nil, nil
	}

	key := fmt.Sprintf("%s/%s", namespace, name)
	obj, exists, err := di.informer.GetStore().GetByKey(key)
	if !exists || err != nil {
		return nil, err
	}

	return obj.(*k8ssnapshotv1beta1.VolumeSnapshot), nil
}

func (ctrl *SnapshotController) getVolumeSnapshotClasses() []k8ssnapshotv1beta1.VolumeSnapshotClass {
	di := ctrl.dynamicInformerMap[volumeSnapshotClassCRD]
	di.mutex.Lock()
	defer di.mutex.Unlock()

	if di.informer == nil {
		return nil
	}

	var vscs []k8ssnapshotv1beta1.VolumeSnapshotClass
	objs := di.informer.GetStore().List()
	for _, obj := range objs {
		vsc := obj.(*k8ssnapshotv1beta1.VolumeSnapshotClass)
		vscs = append(vscs, *vsc)
	}

	return vscs
}

func (ctrl *SnapshotController) ensureDynamicInformer(name string) error {
	di, ok := ctrl.dynamicInformerMap[name]
	if !ok {
		return fmt.Errorf("unexpected CRD %s", name)
	}

	di.mutex.Lock()
	defer di.mutex.Unlock()
	if di.informer != nil {
		return nil
	}

	di.stopCh = make(chan struct{})
	di.informer = di.informerFunc(ctrl.client, ctrl.resyncPeriod)
	handlerFuncs, ok := ctrl.eventHandlerMap[name]
	if ok {
		di.informer.AddEventHandlerWithResyncPeriod(handlerFuncs, ctrl.resyncPeriod)
	}

	go di.informer.Run(di.stopCh)
	cache.WaitForCacheSync(di.stopCh, di.informer.HasSynced)

	log.Log.Infof("Successfully created informer for %q", name)

	return nil
}

func (ctrl *SnapshotController) deleteDynamicInformer(name string) error {
	di, ok := ctrl.dynamicInformerMap[name]
	if !ok {
		return fmt.Errorf("unexpected CRD %s", name)
	}

	di.mutex.Lock()
	defer di.mutex.Unlock()
	if di.informer == nil {
		return nil
	}

	close(di.stopCh)
	di.stopCh = nil
	di.informer = nil

	log.Log.Infof("Successfully deleted informer for %q", name)

	return nil
}
