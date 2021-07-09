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

package snapshot

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

// VMRestoreController is resonsible for restoring VMs
type VMRestoreController struct {
	Client kubecli.KubevirtClient

	VMRestoreInformer         cache.SharedIndexInformer
	VMSnapshotInformer        cache.SharedIndexInformer
	VMSnapshotContentInformer cache.SharedIndexInformer
	VMInformer                cache.SharedIndexInformer
	VMIInformer               cache.SharedIndexInformer
	DataVolumeInformer        cache.SharedIndexInformer
	PVCInformer               cache.SharedIndexInformer
	StorageClassInformer      cache.SharedIndexInformer

	Recorder record.EventRecorder

	vmRestoreQueue workqueue.RateLimitingInterface
}

// Init initializes the restore controller
func (ctrl *VMRestoreController) Init() {
	ctrl.vmRestoreQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "restore-controller-vmrestore")

	ctrl.VMRestoreInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMRestore,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMRestore(newObj) },
		},
	)

	ctrl.PVCInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
		},
	)

	ctrl.VMInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVM,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVM(newObj) },
		},
	)
}

// Run the controller
func (ctrl *VMRestoreController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.vmRestoreQueue.ShutDown()

	log.Log.Info("Starting restore controller.")
	defer log.Log.Info("Shutting down restore controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.VMRestoreInformer.HasSynced,
		ctrl.VMSnapshotInformer.HasSynced,
		ctrl.VMSnapshotContentInformer.HasSynced,
		ctrl.VMInformer.HasSynced,
		ctrl.VMIInformer.HasSynced,
		ctrl.DataVolumeInformer.HasSynced,
		ctrl.PVCInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.vmRestoreWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (ctrl *VMRestoreController) vmRestoreWorker() {
	for ctrl.processVMRestoreWorkItem() {
	}
}

func (ctrl *VMRestoreController) processVMRestoreWorkItem() bool {
	return processWorkItem(ctrl.vmRestoreQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vmRestore worker processing key [%s]", key)

		storeObj, exists, err := ctrl.VMRestoreInformer.GetStore().GetByKey(key)
		if !exists || err != nil {
			return 0, err
		}

		vmRestore, ok := storeObj.(*snapshotv1.VirtualMachineRestore)
		if !ok {
			return 0, fmt.Errorf("unexpected resource %+v", storeObj)
		}

		return ctrl.updateVMRestore(vmRestore.DeepCopy())
	})
}

func (ctrl *VMRestoreController) handleVMRestore(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmRestore, ok := obj.(*snapshotv1.VirtualMachineRestore); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(vmRestore)
		if err != nil {
			log.Log.Errorf("failed to get key from object: %v, %v", err, vmRestore)
			return
		}

		log.Log.V(3).Infof("enqueued %q for sync", objName)
		ctrl.vmRestoreQueue.Add(objName)
	}
}

func (ctrl *VMRestoreController) handlePVC(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		restoreName, ok := pvc.Annotations[pvcRestoreAnnotation]
		if !ok {
			return
		}

		objName := cacheKeyFunc(pvc.Namespace, restoreName)

		log.Log.V(3).Infof("Handling PVC %s/%s, Restore %s", pvc.Namespace, pvc.Name, objName)
		ctrl.vmRestoreQueue.Add(objName)
	}
}

func (ctrl *VMRestoreController) handleVM(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vm, ok := obj.(*kubevirtv1.VirtualMachine); ok {
		keys, err := ctrl.VMRestoreInformer.GetIndexer().IndexKeys("vm", vm.Name)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, k := range keys {
			ctrl.vmRestoreQueue.Add(k)
		}
	}
}
