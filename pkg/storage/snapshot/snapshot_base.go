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
	"sync"
	"time"

	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	corev1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/status"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
)

const (
	unexpectedResourceFmt  = "unexpected resource %+v"
	failedKeyFromObjectFmt = "failed to get key from object: %v, %v"
	enqueuedForSyncFmt     = "enqueued %q for sync"
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

// VMSnapshotController is resonsible for snapshotting VMs
type VMSnapshotController struct {
	Client kubecli.KubevirtClient

	VMSnapshotInformer        cache.SharedIndexInformer
	VMSnapshotContentInformer cache.SharedIndexInformer
	VMInformer                cache.SharedIndexInformer
	VMIInformer               cache.SharedIndexInformer
	StorageClassInformer      cache.SharedIndexInformer
	PVCInformer               cache.SharedIndexInformer
	CRDInformer               cache.SharedIndexInformer
	PodInformer               cache.SharedIndexInformer
	DVInformer                cache.SharedIndexInformer
	CRInformer                cache.SharedIndexInformer

	Recorder record.EventRecorder

	ResyncPeriod time.Duration

	vmSnapshotQueue        workqueue.RateLimitingInterface
	vmSnapshotContentQueue workqueue.RateLimitingInterface
	crdQueue               workqueue.RateLimitingInterface
	vmSnapshotStatusQueue  workqueue.RateLimitingInterface
	vmQueue                workqueue.RateLimitingInterface

	dynamicInformerMap map[string]*dynamicInformer
	eventHandlerMap    map[string]cache.ResourceEventHandlerFuncs

	vmStatusUpdater *status.VMStatusUpdater
}

var supportedCRDVersions = []string{"v1"}

// Init initializes the snapshot controller
func (ctrl *VMSnapshotController) Init() error {
	ctrl.vmSnapshotQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-snapshot-vmsnapshot")
	ctrl.vmSnapshotContentQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-snapshot-vmsnapshotcontent")
	ctrl.crdQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-snapshot-crd")
	ctrl.vmSnapshotStatusQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-snapshot-vmsnashotstatus")
	ctrl.vmQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "virt-controller-snapshot-vm")

	ctrl.dynamicInformerMap = map[string]*dynamicInformer{
		volumeSnapshotCRD:      {informerFunc: controller.VolumeSnapshotInformer},
		volumeSnapshotClassCRD: {informerFunc: controller.VolumeSnapshotClassInformer},
	}

	ctrl.eventHandlerMap = map[string]cache.ResourceEventHandlerFuncs{
		volumeSnapshotCRD: {
			AddFunc:    ctrl.handleVolumeSnapshot,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVolumeSnapshot(newObj) },
			DeleteFunc: ctrl.handleVolumeSnapshot,
		},
		volumeSnapshotClassCRD: {
			AddFunc:    ctrl.handleVolumeSnapshotClass,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVolumeSnapshotClass(newObj) },
			DeleteFunc: ctrl.handleVolumeSnapshotClass,
		},
	}

	_, err := ctrl.VMSnapshotInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshot,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshot(newObj) },
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	_, err = ctrl.VMSnapshotContentInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshotContent,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshotContent(newObj) },
			DeleteFunc: ctrl.handleVMSnapshotContent,
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	_, err = ctrl.VMInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVM,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVM(newObj) },
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	_, err = ctrl.VMIInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMI,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMI(newObj) },
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	_, err = ctrl.CRDInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleCRD,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleCRD(newObj) },
			DeleteFunc: ctrl.handleCRD,
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	_, err = ctrl.DVInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleDV,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleDV(newObj) },
			DeleteFunc: ctrl.handleDV,
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	_, err = ctrl.PVCInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
			DeleteFunc: ctrl.handlePVC,
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	ctrl.vmStatusUpdater = status.NewVMStatusUpdater(ctrl.Client)
	return nil
}

// Run the controller
func (ctrl *VMSnapshotController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.vmSnapshotQueue.ShutDown()
	defer ctrl.vmSnapshotContentQueue.ShutDown()
	defer ctrl.crdQueue.ShutDown()
	defer ctrl.vmSnapshotStatusQueue.ShutDown()
	defer ctrl.vmQueue.ShutDown()

	log.Log.Info("Starting snapshot controller.")
	defer log.Log.Info("Shutting down snapshot controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.VMSnapshotInformer.HasSynced,
		ctrl.VMSnapshotContentInformer.HasSynced,
		ctrl.VMInformer.HasSynced,
		ctrl.VMIInformer.HasSynced,
		ctrl.CRDInformer.HasSynced,
		ctrl.PodInformer.HasSynced,
		ctrl.PVCInformer.HasSynced,
		ctrl.DVInformer.HasSynced,
		ctrl.StorageClassInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.crdWorker, time.Second, stopCh)
	}

	log.Log.Infof("CRD queue length: %d", ctrl.crdQueue.Len())

	for ql := ctrl.crdQueue.Len(); ql > 0; ql = ctrl.crdQueue.Len() {
		log.Log.Infof("Waiting for empty CRD queue, currently: %d", ql)
		time.Sleep(2 * time.Second)
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.vmSnapshotWorker, time.Second, stopCh)
		go wait.Until(ctrl.vmSnapshotContentWorker, time.Second, stopCh)
		go wait.Until(ctrl.vmSnapshotStatusWorker, time.Second, stopCh)
		go wait.Until(ctrl.vmWorker, time.Second, stopCh)
	}

	<-stopCh

	for crd := range ctrl.dynamicInformerMap {
		if _, err := ctrl.deleteDynamicInformer(crd); err != nil {
			log.Log.Warningf("failed to delete %s informer: %v", crd, err)
		}
	}

	return nil
}

func (ctrl *VMSnapshotController) vmSnapshotWorker() {
	for ctrl.processVMSnapshotWorkItem() {
	}
}

func (ctrl *VMSnapshotController) vmSnapshotContentWorker() {
	for ctrl.processVMSnapshotContentWorkItem() {
	}
}

func (ctrl *VMSnapshotController) crdWorker() {
	for ctrl.processCRDWorkItem() {
	}
}

func (ctrl *VMSnapshotController) vmSnapshotStatusWorker() {
	for ctrl.processVMSnapshotStatusWorkItem() {
	}
}

func (ctrl *VMSnapshotController) vmWorker() {
	for ctrl.processVMWorkItem() {
	}
}

func (ctrl *VMSnapshotController) processVMSnapshotWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.vmSnapshotQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vmSnapshot worker processing key [%s]", key)

		storeObj, exists, err := ctrl.VMSnapshotInformer.GetStore().GetByKey(key)
		if !exists || err != nil {
			return 0, err
		}

		vmSnapshot, ok := storeObj.(*snapshotv1.VirtualMachineSnapshot)
		if !ok {
			return 0, fmt.Errorf(unexpectedResourceFmt, storeObj)
		}

		return ctrl.updateVMSnapshot(vmSnapshot.DeepCopy())
	})
}

func (ctrl *VMSnapshotController) processVMSnapshotContentWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.vmSnapshotContentQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vmSnapshotContent worker processing key [%s]", key)

		storeObj, exists, err := ctrl.VMSnapshotContentInformer.GetStore().GetByKey(key)
		if !exists || err != nil {
			return 0, err
		}

		vmSnapshotContent, ok := storeObj.(*snapshotv1.VirtualMachineSnapshotContent)
		if !ok {
			return 0, fmt.Errorf(unexpectedResourceFmt, storeObj)
		}

		return ctrl.updateVMSnapshotContent(vmSnapshotContent.DeepCopy())
	})
}

func (ctrl *VMSnapshotController) processCRDWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.crdQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("CRD worker processing key [%s]", key)

		storeObj, exists, err := ctrl.CRDInformer.GetStore().GetByKey(key)
		if err != nil {
			return 0, err
		}

		if !exists {
			_, name, err := cache.SplitMetaNamespaceKey(key)
			if err != nil {
				return 0, err
			}

			return ctrl.deleteDynamicInformer(name)
		}

		crd, ok := storeObj.(*extv1.CustomResourceDefinition)
		if !ok {
			return 0, fmt.Errorf(unexpectedResourceFmt, storeObj)
		}

		if crd.DeletionTimestamp != nil {
			return ctrl.deleteDynamicInformer(crd.Name)
		}

		return ctrl.ensureDynamicInformer(crd.Name)
	})
}

func (ctrl *VMSnapshotController) processVMSnapshotStatusWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.vmSnapshotStatusQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vmSnapshotStatus worker processing VM [%s]", key)

		storeObj, exists, err := ctrl.VMInformer.GetStore().GetByKey(key)
		if err != nil {
			return 0, err
		}

		if exists {
			vm, ok := storeObj.(*kubevirtv1.VirtualMachine)
			if !ok {
				return 0, fmt.Errorf(unexpectedResourceFmt, storeObj)
			}

			if err = ctrl.updateVolumeSnapshotStatuses(vm); err != nil {
				return 0, err
			}
		}

		return 0, nil
	})
}

func (ctrl *VMSnapshotController) processVMWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.vmQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("vm worker processing VM [%s]", key)

		storeObj, exists, err := ctrl.VMInformer.GetStore().GetByKey(key)
		if err != nil {
			return 0, err
		}

		if exists {
			vm, ok := storeObj.(*kubevirtv1.VirtualMachine)
			if !ok {
				return 0, fmt.Errorf(unexpectedResourceFmt, storeObj)
			}

			ctrl.handleVM(vm)
		}

		return 0, nil
	})
}

func (ctrl *VMSnapshotController) handleVMSnapshot(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmSnapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(vmSnapshot)
		if err != nil {
			log.Log.Errorf(failedKeyFromObjectFmt, err, vmSnapshot)
			return
		}
		log.Log.V(3).Infof(enqueuedForSyncFmt, objName)
		ctrl.vmSnapshotQueue.Add(objName)
	}
}

func (ctrl *VMSnapshotController) handleVMSnapshotContent(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if content, ok := obj.(*snapshotv1.VirtualMachineSnapshotContent); ok {
		objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(content)
		if err != nil {
			log.Log.Errorf(failedKeyFromObjectFmt, err, content)
			return
		}

		if content.Spec.VirtualMachineSnapshotName != nil {
			k := cacheKeyFunc(content.Namespace, *content.Spec.VirtualMachineSnapshotName)
			log.Log.V(5).Infof("enqueued vmsnapshot %q for sync", k)
			ctrl.vmSnapshotQueue.Add(k)
		}

		log.Log.V(5).Infof(enqueuedForSyncFmt, objName)
		ctrl.vmSnapshotContentQueue.Add(objName)
	}
}

func (ctrl *VMSnapshotController) handleVM(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vm, ok := obj.(*kubevirtv1.VirtualMachine); ok {
		k, _ := cache.MetaNamespaceKeyFunc(vm)
		keys, err := ctrl.VMSnapshotInformer.GetIndexer().IndexKeys("vm", k)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, k := range keys {
			ctrl.vmSnapshotQueue.Add(k)
		}

		key, err := controller.KeyFunc(vm)
		if err != nil {
			log.Log.Error("Failed to extract vmKey from VirtualMachine.")
		} else {
			ctrl.vmSnapshotStatusQueue.Add(key)
		}
	}
}

func (ctrl *VMSnapshotController) handleVMI(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmi, ok := obj.(*kubevirtv1.VirtualMachineInstance); ok {
		k, _ := cache.MetaNamespaceKeyFunc(vmi)
		keys, err := ctrl.VMSnapshotInformer.GetIndexer().IndexKeys("vm", k)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, k := range keys {
			ctrl.vmSnapshotQueue.Add(k)
		}
	}
}

func (ctrl *VMSnapshotController) handleVolumeSnapshotClass(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if _, ok := obj.(*vsv1.VolumeSnapshotClass); ok {
		for _, vmKey := range ctrl.VMInformer.GetStore().ListKeys() {
			ctrl.vmQueue.Add(vmKey)
		}
	}
}

func (ctrl *VMSnapshotController) handleCRD(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if crd, ok := obj.(*extv1.CustomResourceDefinition); ok {
		_, ok = ctrl.dynamicInformerMap[crd.Name]
		if ok {
			hasSupportedVersion := false
			for _, crdVersion := range crd.Spec.Versions {
				for _, supportedVersion := range supportedCRDVersions {
					if crdVersion.Name == supportedVersion && crdVersion.Served {
						hasSupportedVersion = true
					}
				}
			}

			if !hasSupportedVersion {
				return
			}

			objName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(crd)
			if err != nil {
				log.Log.Errorf(failedKeyFromObjectFmt, err, crd)
				return
			}

			log.Log.V(3).Infof(enqueuedForSyncFmt, objName)
			ctrl.crdQueue.Add(objName)
		}
	}
}

func (ctrl *VMSnapshotController) handleVolumeSnapshot(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if volumeSnapshot, ok := obj.(*vsv1.VolumeSnapshot); ok {
		k, _ := cache.MetaNamespaceKeyFunc(volumeSnapshot)
		keys, err := ctrl.VMSnapshotContentInformer.GetIndexer().IndexKeys("volumeSnapshot", k)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, k := range keys {
			ctrl.vmSnapshotContentQueue.Add(k)
		}
	}
}

func (ctrl *VMSnapshotController) handleDV(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if dv, ok := obj.(*cdiv1.DataVolume); ok {
		key, _ := cache.MetaNamespaceKeyFunc(dv)
		log.Log.V(3).Infof("Processing DV %s", key)
		// TODO come back when DV/PVC name may differ
		for _, idx := range []string{"dv", "pvc"} {
			keys, err := ctrl.VMInformer.GetIndexer().IndexKeys(idx, key)
			if err != nil {
				utilruntime.HandleError(err)
				return
			}
			for _, k := range keys {
				ctrl.vmSnapshotStatusQueue.Add(k)
			}
		}
	}
}

func (ctrl *VMSnapshotController) handlePVC(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		key, _ := cache.MetaNamespaceKeyFunc(pvc)
		log.Log.V(3).Infof("Processing PVC %s", key)
		keys, err := ctrl.VMInformer.GetIndexer().IndexKeys("pvc", key)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		for _, k := range keys {
			ctrl.vmSnapshotStatusQueue.Add(k)
		}
	}
}

func (ctrl *VMSnapshotController) getVolumeSnapshotClasses() []vsv1.VolumeSnapshotClass {
	di := ctrl.dynamicInformerMap[volumeSnapshotClassCRD]
	di.mutex.Lock()
	defer di.mutex.Unlock()

	if di.informer == nil {
		return nil
	}

	var vscs []vsv1.VolumeSnapshotClass
	objs := di.informer.GetStore().List()
	for _, obj := range objs {
		vsc := obj.(*vsv1.VolumeSnapshotClass).DeepCopy()
		vscs = append(vscs, *vsc)
	}

	return vscs
}

func (ctrl *VMSnapshotController) ensureDynamicInformer(name string) (time.Duration, error) {
	di, ok := ctrl.dynamicInformerMap[name]
	if !ok {
		return 0, fmt.Errorf("unexpected CRD %s", name)
	}

	di.mutex.Lock()
	defer di.mutex.Unlock()
	if di.informer != nil {
		return 0, nil
	}

	di.stopCh = make(chan struct{})
	di.informer = di.informerFunc(ctrl.Client, ctrl.ResyncPeriod)
	handlerFuncs, ok := ctrl.eventHandlerMap[name]
	if ok {
		di.informer.AddEventHandlerWithResyncPeriod(handlerFuncs, ctrl.ResyncPeriod)
	}

	go di.informer.Run(di.stopCh)
	cache.WaitForCacheSync(di.stopCh, di.informer.HasSynced)

	log.Log.Infof("Successfully created informer for %q", name)

	return 0, nil
}

func (ctrl *VMSnapshotController) deleteDynamicInformer(name string) (time.Duration, error) {
	di, ok := ctrl.dynamicInformerMap[name]
	if !ok {
		return 0, fmt.Errorf("unexpected CRD %s", name)
	}

	di.mutex.Lock()
	defer di.mutex.Unlock()
	if di.informer == nil {
		return 0, nil
	}

	close(di.stopCh)
	di.stopCh = nil
	di.informer = nil

	log.Log.Infof("Successfully deleted informer for %q", name)

	return 0, nil
}

type VolumeSnapshotProvider interface {
	GetVolumeSnapshot(string, string) (*vsv1.VolumeSnapshot, error)
}

func (ctrl *VMSnapshotController) GetVolumeSnapshot(namespace, name string) (*vsv1.VolumeSnapshot, error) {
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

	return obj.(*vsv1.VolumeSnapshot).DeepCopy(), nil
}
