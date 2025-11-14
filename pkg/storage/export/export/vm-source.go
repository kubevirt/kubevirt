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
 * Copyright The KubeVirt Authors.
 *
 */

package export

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	storageutils "kubevirt.io/kubevirt/pkg/storage/utils"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	noVolumeVMReason = "VMNoVolumes"
)

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

func (ctrl *VMExportController) handleVM(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vm, ok := obj.(*virtv1.VirtualMachine); ok {
		vmKey, _ := cache.MetaNamespaceKeyFunc(vm)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("vm", vmKey)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		for _, key := range keys {
			log.Log.V(3).Infof("Adding VMExport due to vm %s", vmKey)
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) handleVMI(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if vmi, ok := obj.(*virtv1.VirtualMachineInstance); ok {
		vm := ctrl.getVMFromVMI(vmi)
		if vm != nil {
			vmKey, _ := cache.MetaNamespaceKeyFunc(vm)
			keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("vm", vmKey)
			if err != nil {
				utilruntime.HandleError(err)
				return
			}
			for _, key := range keys {
				log.Log.V(3).Infof("Adding VMExport due to VM %s", vmKey)
				ctrl.vmExportQueue.Add(key)
			}
			return
		}
		pvcs := ctrl.getPVCsFromVMI(vmi)
		for _, pvc := range pvcs {
			log.Log.V(3).Infof("Adding VMExport due to PVC %s/%s", pvc.Namespace, pvc.Name)
			ctrl.handlePVC(pvc)
		}
	}
}

func (ctrl *VMExportController) getPVCsFromVMI(vmi *virtv1.VirtualMachineInstance) []*corev1.PersistentVolumeClaim {
	var pvcs []*corev1.PersistentVolumeClaim

	// No need to handle error when using VMI to fetch volumes
	volumes, _ := storageutils.GetVolumes(vmi, ctrl.K8sClient, storageutils.WithAllVolumes)

	for _, volume := range volumes {
		pvcName := storagetypes.PVCNameFromVirtVolume(&volume)
		if pvc := ctrl.getPVCsFromName(vmi.Namespace, pvcName); pvc != nil {
			pvcs = append(pvcs, pvc)
		}
	}
	return pvcs
}

func (ctrl *VMExportController) getOwnerVMexportKey(obj metav1.Object) string {
	ownerRef := metav1.GetControllerOf(obj)
	var key string
	if ownerRef != nil {
		if ownerRef.Kind == exportGVK.Kind && ownerRef.APIVersion == exportGVK.GroupVersion().String() {
			key = controller.NamespacedKey(obj.GetNamespace(), ownerRef.Name)
		}
	}
	return key
}

func (ctrl *VMExportController) getVMFromVMI(vmi *virtv1.VirtualMachineInstance) *virtv1.VirtualMachine {
	ownerRef := metav1.GetControllerOf(vmi)
	if ownerRef != nil {
		if ownerRef.Kind == "VirtualMachine" && ownerRef.APIVersion == virtv1.GroupVersion.String() {
			if vm, exists, err := ctrl.getVm(vmi.Namespace, ownerRef.Name); !exists || err != nil {
				log.Log.V(3).Infof("Unable to get owner VM %s/%s for VMI %s/%s", vmi.Namespace, ownerRef.Name, vmi.Namespace, vmi.Name)
			} else {
				return vm
			}
		}
	}
	return nil
}

func (ctrl *VMExportController) isSourceInUseVM(vmExport *exportv1.VirtualMachineExport) (bool, string, error) {
	vmi, exists, err := ctrl.getVmi(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return false, "", err
	}
	if exists {
		// Only if the VMI is running, the source VM is in use
		if vmi.Status.Phase != virtv1.Succeeded && vmi.Status.Phase != virtv1.Failed {
			return exists, fmt.Sprintf("Virtual Machine %s/%s is running", vmi.Namespace, vmi.Name), nil
		}
		return false, "", nil
	}
	return exists, "", nil
}

func (ctrl *VMExportController) getPVCFromSourceVM(vmExport *exportv1.VirtualMachineExport) (*sourceVolumes, error) {
	pvcs, allPopulated, err := ctrl.getPVCsFromVM(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return &sourceVolumes{}, err
	}
	log.Log.V(3).Infof("Number of volumes found for VM %s/%s, %d, allPopulated %t", vmExport.Namespace, vmExport.Spec.Source.Name, len(pvcs), allPopulated)
	if len(pvcs) > 0 && !allPopulated {
		return &sourceVolumes{
			volumes:          pvcs,
			inUse:            false,
			isPopulated:      allPopulated,
			availableMessage: fmt.Sprintf("Not all volumes in the Virtual Machine %s/%s are populated", vmExport.Namespace, vmExport.Spec.Source.Name)}, nil
	}
	inUse, availableMessage, err := ctrl.isSourceInUseVM(vmExport)
	if err != nil {
		return &sourceVolumes{}, err
	}
	return &sourceVolumes{
		volumes:          pvcs,
		inUse:            inUse,
		isPopulated:      allPopulated,
		availableMessage: availableMessage}, nil
}

func (ctrl *VMExportController) getPVCsFromVM(vmNamespace, vmName string) ([]*corev1.PersistentVolumeClaim, bool, error) {
	var pvcs []*corev1.PersistentVolumeClaim
	vm, exists, err := ctrl.getVm(vmNamespace, vmName)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	allPopulated := true

	volumes, err := storageutils.GetVolumes(vm, ctrl.K8sClient, storageutils.WithAllVolumes)
	if err != nil {
		if storageutils.IsErrNoBackendPVC(err) {
			// No backend pvc when we should have one, lets wait
			return nil, false, nil
		}
		return nil, false, err
	}

	for _, volume := range volumes {
		pvcName := storagetypes.PVCNameFromVirtVolume(&volume)
		if pvcName == "" {
			continue
		}
		pvc, exists, err := ctrl.getPvc(vmNamespace, pvcName)
		if err != nil {
			return nil, false, nil
		}
		if exists {
			populated, err := ctrl.isPVCPopulated(pvc)
			if err != nil {
				return nil, false, err
			}
			pvcs = append(pvcs, pvc)
			if !populated {
				allPopulated = false
			}
			continue
		}
		if volume.DataVolume != nil {
			// PVC has not been created yet, otherwise exist would be true. Setting allPopulated to false will
			// trigger a requeue
			log.Log.V(2).Infof("Found data volume %s but PVC does not exist yet", volume.DataVolume.Name)
			allPopulated = false
		}
	}
	return pvcs, allPopulated, nil
}

func (ctrl *VMExportController) updateVMExportVMStatus(vmExport *exportv1.VirtualMachineExport, exporterPod *corev1.Pod, service *corev1.Service, sourceVolumes *sourceVolumes) (time.Duration, error) {
	var requeue time.Duration

	vmExportCopy := vmExport.DeepCopy()
	vmExportCopy.Status.VirtualMachineName = pointer.P(vmExport.Spec.Source.Name)
	if err := ctrl.updateCommonVMExportStatusFields(vmExport, vmExportCopy, exporterPod, service, sourceVolumes, getVolumeName); err != nil {
		return requeue, err
	}
	if len(sourceVolumes.volumes) == 0 {
		vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, noVolumeVMReason, sourceVolumes.availableMessage))
		vmExportCopy.Status.Phase = exportv1.Skipped
	}
	if !sourceVolumes.isPopulated {
		requeue = requeueTime
	}
	if err := ctrl.updateVMExportStatus(vmExport, vmExportCopy); err != nil {
		return requeue, err
	}
	return requeue, nil
}

func (ctrl *VMExportController) isSourceVM(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && source.Source.APIGroup != nil && *source.Source.APIGroup == virtv1.SchemeGroupVersion.Group && source.Source.Kind == "VirtualMachine"
}

func (ctrl *VMExportController) getVm(namespace, name string) (*virtv1.VirtualMachine, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.VMInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*virtv1.VirtualMachine).DeepCopy(), true, nil
}

func (ctrl *VMExportController) getVmi(namespace, name string) (*virtv1.VirtualMachineInstance, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.VMIInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*virtv1.VirtualMachineInstance).DeepCopy(), true, nil
}
