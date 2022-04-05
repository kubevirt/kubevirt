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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	exportv1 "kubevirt.io/api/export/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
)

const (
	unexpectedResourceFmt  = "unexpected resource %+v"
	failedKeyFromObjectFmt = "failed to get key from object: %v, %v"
	enqueuedForSyncFmt     = "enqueued %q for sync"

	pvcNotFoundReason  = "pvcNotFound"
	pvcBoundReason     = "pvcBound"
	pvcPendingReason   = "pvcPending"
	unknownReason      = "unknown"
	initializingReason = "initializing"
	podReadyReason     = "podReady"
	podCompletedReason = "podCompleted"
)

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

// VMExportController is resonsible for exporting VMs
type VMExportController struct {
	Client kubecli.KubevirtClient

	TemplateService services.TemplateService

	VMExportInformer cache.SharedIndexInformer
	PVCInformer      cache.SharedIndexInformer

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
	ctrl.VMExportInformer.AddIndexers(cache.Indexers{
		"pvc": func(obj interface{}) ([]string, error) {
			vmExport, isObj := obj.(*exportv1.VirtualMachineExport)
			if !isObj {
				return nil, fmt.Errorf("object of type %T is not a VirtualMachineExport", obj)
			}
			return []string{controller.NamespacedKey(vmExport.Namespace, vmExport.Spec.Source.Name)}, nil
		},
	})
	ctrl.PVCInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handlePVC,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handlePVC(newObj) },
			DeleteFunc: ctrl.handlePVC,
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
		ctrl.PVCInformer.HasSynced,
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

func (ctrl *VMExportController) handlePVC(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		key, _ := cache.MetaNamespaceKeyFunc(pvc)
		log.Log.V(3).Infof("Processing PVC %s", key)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("pvc", key)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		for _, k := range keys {
			log.Log.V(1).Infof("Found key: %s", k)
			ctrl.vmExportQueue.Add(k)
		}
	}
}

func (ctrl *VMExportController) updateVMExport(vmExport *exportv1.VirtualMachineExport) (time.Duration, error) {
	log.Log.V(1).Infof("Updating VirtualMachineExport %s/%s", vmExport.Namespace, vmExport.Name)
	var retry time.Duration

	if ctrl.isSourcePvc(&vmExport.Spec) {
		pvc, err := ctrl.getPvc(vmExport.Namespace, vmExport.Spec.Source.Name)
		if err != nil {
			return 0, err
		}
		pvcs := make([]*corev1.PersistentVolumeClaim, 0)
		if pvc != nil {
			pvcs = append(pvcs, pvc)
		} else {
			pvcs = append(pvcs, &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmExport.Spec.Source.Name,
					Namespace: vmExport.Namespace,
				},
			})
		}
		pod, err := ctrl.getOrCreateExporterPod(vmExport, pvcs)
		if err != nil {
			return 0, err
		}
		return ctrl.updateVMExportPvcStatus(vmExport, pvc, pod)
	}
	return retry, nil
}

func (ctrl *VMExportController) getOrCreateExporterPod(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim) (*corev1.Pod, error) {
	manifest := ctrl.createExporterPodManifest(vmExport, pvcs)
	log.Log.V(1).Infof("Checking if pod exist: %s/%s", manifest.Namespace, manifest.Name)
	if pod, err := ctrl.Client.CoreV1().Pods(vmExport.Namespace).Get(context.Background(), manifest.Name, metav1.GetOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Log.V(1).Errorf("error %v", err)
		return nil, err
	} else if pod != nil && err == nil {
		log.Log.V(1).Infof("Found pod %s/%s", pod.Namespace, pod.Name)
		return pod, nil
	}
	log.Log.V(1).Infof("Creating new exporter pod %s/%s", manifest.Namespace, manifest.Name)
	return ctrl.Client.CoreV1().Pods(vmExport.Namespace).Create(context.Background(), manifest, metav1.CreateOptions{})
}

func (ctrl *VMExportController) createExporterPodManifest(vmExport *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim) *corev1.Pod {
	podSpec := ctrl.TemplateService.RenderExporterManifest(vmExport)
	for i, pvc := range pvcs {
		if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock {
			podSpec.Spec.Containers[0].VolumeDevices = append(podSpec.Spec.Containers[0].VolumeDevices, corev1.VolumeDevice{
				Name:       pvc.Name,
				DevicePath: fmt.Sprintf("/dev/export-block%d", i),
			})
		} else {
			podSpec.Spec.Containers[0].VolumeMounts = append(podSpec.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      pvc.Name,
				ReadOnly:  true,
				MountPath: fmt.Sprintf("/export-fs%d", i),
			})
		}
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		})
	}
	return podSpec
}

func (ctrl *VMExportController) updateVMExportPvcStatus(vmExport *exportv1.VirtualMachineExport, pvc *corev1.PersistentVolumeClaim, exporterPod *corev1.Pod) (time.Duration, error) {
	var retry time.Duration
	vmExportCopy := vmExport.DeepCopy()
	if vmExportCopy.Status == nil {
		vmExportCopy.Status = &exportv1.VirtualMachineExportStatus{
			Phase: exportv1.Pending,
			Conditions: []exportv1.Condition{
				newReadyCondition(corev1.ConditionFalse, initializingReason),
				newPvcCondition(corev1.ConditionFalse, unknownReason),
			},
		}
	}

	if exporterPod == nil {
		updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, initializingReason), true)
		vmExportCopy.Status.Phase = exportv1.Pending
	} else {
		if exporterPod.Status.Phase == corev1.PodRunning {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionTrue, podReadyReason), true)
			vmExportCopy.Status.Phase = exportv1.Ready
		} else if exporterPod.Status.Phase == corev1.PodSucceeded {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, podCompletedReason), true)
			vmExportCopy.Status.Phase = exportv1.Terminated
		} else if exporterPod.Status.Phase == corev1.PodPending {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, initializingReason), true)
			vmExportCopy.Status.Phase = exportv1.Pending
		} else {
			updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, unknownReason), true)
			vmExportCopy.Status.Phase = exportv1.Pending
		}
	}

	if pvc == nil {
		log.Log.V(1).Info("PVC not found, updating status to not found")
		updateCondition(vmExportCopy.Status.Conditions, newPvcCondition(corev1.ConditionFalse, pvcNotFoundReason), true)
	} else {
		updateCondition(vmExportCopy.Status.Conditions, ctrl.pvcConditionFromPVC(pvc), true)
	}

	//	updateSnapshotCondition(vmSnapshotCpy, newProgressingCondition(corev1.ConditionFalse, "Source does not exist"))
	if !equality.Semantic.DeepEqual(vmExport, vmExportCopy) {
		if _, err := ctrl.Client.VirtualMachineExport(vmExportCopy.Namespace).Update(context.Background(), vmExportCopy, metav1.UpdateOptions{}); err != nil {
			return retry, err
		}
	}
	if vmExportCopy.Status.Phase == exportv1.Pending {
		log.Log.V(1).Info("Not ready requeueing")
		retry = time.Second
	}
	return retry, nil
}

func (ctrl *VMExportController) isSourcePvc(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && source.Source.APIGroup != nil && *source.Source.APIGroup == "v1" && source.Source.Kind == "PersistentVolumeClaim"
}

func (ctrl *VMExportController) getPvc(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, err
	}
	return obj.(*corev1.PersistentVolumeClaim).DeepCopy(), nil
}

func newReadyCondition(status corev1.ConditionStatus, reason string) exportv1.Condition {
	return exportv1.Condition{
		Type:               exportv1.ConditionReady,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func newPvcCondition(status corev1.ConditionStatus, reason string) exportv1.Condition {
	return exportv1.Condition{
		Type:               exportv1.ConditionPVC,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func updateCondition(conditions []exportv1.Condition, c exportv1.Condition, includeReason bool) []exportv1.Condition {
	found := false
	for i := range conditions {
		if conditions[i].Type == c.Type {
			if conditions[i].Status != c.Status || (includeReason && conditions[i].Reason != c.Reason) {
				conditions[i] = c
			}
			found = true
			break
		}
	}

	if !found {
		conditions = append(conditions, c)
	}

	return conditions
}

func (ctrl *VMExportController) pvcConditionFromPVC(pvc *corev1.PersistentVolumeClaim) exportv1.Condition {
	cond := exportv1.Condition{
		Type:               exportv1.ConditionPVC,
		LastTransitionTime: *currentTime(),
	}
	switch pvc.Status.Phase {
	case corev1.ClaimBound:
		cond.Status = corev1.ConditionTrue
		cond.Reason = pvcBoundReason
	case corev1.ClaimPending:
		cond.Status = corev1.ConditionFalse
		cond.Reason = pvcPendingReason
	default:
		cond.Status = corev1.ConditionFalse
		cond.Reason = unknownReason
	}
	return cond
}
