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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package export

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	exportv1 "kubevirt.io/api/export/v1alpha1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
)

const (
	unexpectedResourceFmt  = "unexpected resource %+v"
	failedKeyFromObjectFmt = "failed to get key from object: %v, %v"
	enqueuedForSyncFmt     = "enqueued %q for sync"
)

// VMExportController is resonsible for exporting VMs
type VMExportController struct {
	Client kubecli.KubevirtClient

	VMExportInformer cache.SharedIndexInformer

	Recorder record.EventRecorder

	ResyncPeriod time.Duration

	vmExportQueue workqueue.RateLimitingInterface
}

// Init initializes the export controller
func (ctrl *VMExportController) Init() {
	ctrl.vmExportQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-export-vmexport")

	ctrl.VMExportInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMExport,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMExport(newObj) },
		},
		ctrl.ResyncPeriod,
	)
}

// Run the controller
func (ctrl *VMExportController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.vmExportQueue.ShutDown()

	log.Log.Info("Starting export controller.")
	defer log.Log.Info("Shutting down export controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.VMExportInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.vmExportWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (ctrl *VMExportController) vmExportWorker() {
	for ctrl.processVMExportWorkItem() {
	}
}

func (ctrl *VMExportController) processVMExportWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.vmExportQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vmExport worker processing key [%s]", key)

		storeObj, exists, err := ctrl.VMExportInformer.GetStore().GetByKey(key)
		if !exists || err != nil {
			return 0, err
		}

		vmExport, ok := storeObj.(*exportv1.VirtualMachineExport)
		if !ok {
			return 0, fmt.Errorf(unexpectedResourceFmt, storeObj)
		}

		return ctrl.updateVMExport(vmExport.DeepCopy())
	})
}

func (ctrl *VMExportController) handleVMExport(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmExport, ok := obj.(*exportv1.VirtualMachineExport); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(vmExport)
		if err != nil {
			log.Log.Errorf(failedKeyFromObjectFmt, err, vmExport)
			return
		}
		log.Log.V(3).Infof(enqueuedForSyncFmt, objName)
		ctrl.vmExportQueue.Add(objName)
	}
}

func (ctrl *VMExportController) updateVMExport(vmExport *exportv1.VirtualMachineExport) (time.Duration, error) {
	log.Log.V(1).Infof("Updating VirtualMachineExport %s/%s", vmExport.Namespace, vmExport.Name)
	var retry time.Duration

	if err := ctrl.updateVMExportStatus(vmExport); err != nil {
		return 0, err
	}
	return retry, nil
}

func (ctrl *VMExportController) updateVMExportStatus(vmExport *exportv1.VirtualMachineExport) error {
	vmExportCopy := vmExport.DeepCopy()
	if vmExportCopy.Status == nil {
		vmExportCopy.Status = &exportv1.VirtualMachineExportStatus{
			Phase: exportv1.Pending,
		}
	}

	//	updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Source does not exist"))
	if !equality.Semantic.DeepEqual(vmExport, vmExportCopy) {
		if _, err := ctrl.Client.VirtualMachineExport(vmExportCopy.Namespace).Update(context.Background(), vmExportCopy, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}
