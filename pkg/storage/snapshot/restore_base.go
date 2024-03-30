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

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/util/status"
	virtcontroller "kubevirt.io/kubevirt/pkg/virt-controller"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
)

// VMRestoreController is responsible for restoring VMs
type VMRestoreController struct {
	virtcontroller.Controller
	VolumeSnapshotProvider VolumeSnapshotProvider
	Recorder               record.EventRecorder
	vmStatusUpdater        *status.VMStatusUpdater
}

func NewVMRestoreController(client kubecli.KubevirtClient,
	vsProvider VolumeSnapshotProvider,
	recorder record.EventRecorder,
	vmRestoreInformer,
	vmSnapshotInformer,
	vmSnapshotContentInformer,
	vmInformer,
	vmiInformer,
	dvInformer,
	pvcInformer,
	scInformer,
	controllerRevisionInformer cache.SharedIndexInformer,
) (*VMRestoreController, error) {
	ctrl := virtcontroller.NewController(
		workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-restore-vmrestore"),
		client,
	)
	ctrl.SetInformer(virtcontroller.KeyVMRestore, vmRestoreInformer)
	ctrl.SetInformer(virtcontroller.KeyVMSnapshot, vmSnapshotInformer)
	ctrl.SetInformer(virtcontroller.KeyVMSnapshotContent, vmSnapshotContentInformer)
	ctrl.SetInformer(virtcontroller.KeyVM, vmInformer)
	ctrl.SetInformer(virtcontroller.KeyVMI, vmiInformer)
	ctrl.SetInformer(virtcontroller.KeyDV, dvInformer)
	ctrl.SetInformer(virtcontroller.KeyPVC, pvcInformer)
	ctrl.SetInformer(virtcontroller.KeySC, scInformer)
	ctrl.SetInformer(virtcontroller.KeyControllerRevision, controllerRevisionInformer)

	c := VMRestoreController{
		Controller:             *ctrl,
		Recorder:               recorder,
		VolumeSnapshotProvider: vsProvider,
		vmStatusUpdater:        status.NewVMStatusUpdater(client),
	}

	err := c.init()
	return &c, err
}

// init initializes the restore controller
func (ctrl *VMRestoreController) init() error {
	_, err := ctrl.Informer(virtcontroller.KeyVMRestore).AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMRestore,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMRestore(newObj) },
		},
	)

	if err != nil {
		return err
	}

	_, err = ctrl.Informer(virtcontroller.KeyDV).AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleDataVolume,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleDataVolume(newObj) },
		},
	)
	if err != nil {
		return err
	}

	_, err = ctrl.Informer(virtcontroller.KeyPVC).AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
		},
	)
	if err != nil {
		return err
	}

	_, err = ctrl.Informer(virtcontroller.KeyVM).AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVM,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVM(newObj) },
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// Run the controller
func (ctrl *VMRestoreController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.Queue().ShutDown()

	log.Log.Info("Starting restore controller.")
	defer log.Log.Info("Shutting down restore controller.")

	if !ctrl.WaitForCacheSync(stopCh) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.vmRestoreWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (ctrl *VMRestoreController) vmRestoreWorker() {
	for ctrl.Execute() {
	}
}

func (ctrl *VMRestoreController) Execute() bool {
	return watchutil.ProcessWorkItem(ctrl.Queue(), func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vmRestore worker processing key [%s]", key)

		storeObj, exists, err := ctrl.Informer(virtcontroller.KeyVMRestore).GetStore().GetByKey(key)
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
		ctrl.Queue().Add(objName)
	}
}

func (ctrl *VMRestoreController) handleDataVolume(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if dv, ok := obj.(*v1beta1.DataVolume); ok {
		restoreName, ok := dv.Annotations[RestoreNameAnnotation]
		if !ok {
			return
		}

		objName := cacheKeyFunc(dv.Namespace, restoreName)

		log.Log.V(3).Infof("Handling DV %s/%s, Restore %s", dv.Namespace, dv.Name, objName)
		ctrl.Queue().Add(objName)
	}
}

func (ctrl *VMRestoreController) handlePVC(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		restoreName, ok := pvc.Annotations[RestoreNameAnnotation]
		if !ok {
			return
		}

		objName := cacheKeyFunc(pvc.Namespace, restoreName)

		log.Log.V(3).Infof("Handling PVC %s/%s, Restore %s", pvc.Namespace, pvc.Name, objName)
		ctrl.Queue().Add(objName)
	}
}

func (ctrl *VMRestoreController) handleVM(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vm, ok := obj.(*kubevirtv1.VirtualMachine); ok {
		k, _ := cache.MetaNamespaceKeyFunc(vm)
		keys, err := ctrl.Informer(virtcontroller.KeyVMRestore).GetIndexer().IndexKeys("vm", k)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, k := range keys {
			ctrl.Queue().Add(k)
		}
	}
}
