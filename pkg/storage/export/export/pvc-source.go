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
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
)

func (ctrl *VMExportController) handlePVC(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		pvcKey, _ := cache.MetaNamespaceKeyFunc(pvc)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("pvc", pvcKey)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, key := range keys {
			log.Log.V(3).Infof("Adding VMExport due to pvc %s", pvcKey)
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) isSourcePvc(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && (source.Source.APIGroup == nil || *source.Source.APIGroup == corev1.SchemeGroupVersion.Group) && source.Source.Kind == "PersistentVolumeClaim"
}

func (ctrl *VMExportController) getPvc(namespace, name string) (*corev1.PersistentVolumeClaim, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*corev1.PersistentVolumeClaim).DeepCopy(), true, nil
}

func (ctrl *VMExportController) isSourceAvailablePVC(vmExport *exportv1.VirtualMachineExport, pvc *corev1.PersistentVolumeClaim) (bool, bool, string, error) {
	availableMessage := ""
	isPopulated, err := ctrl.isPVCPopulated(pvc)
	inUse := false
	if err != nil {
		return false, false, "", err
	}
	if isPopulated {
		inUse, err = ctrl.isPVCInUse(vmExport, pvc)
		if err != nil {
			return false, false, "", err
		}
		if inUse {
			availableMessage = fmt.Sprintf("pvc %s/%s is in use", pvc.Namespace, pvc.Name)
		}
	} else {
		availableMessage = fmt.Sprintf("pvc %s/%s is not populated", pvc.Namespace, pvc.Name)
	}
	return isPopulated, inUse, availableMessage, nil
}

func (ctrl *VMExportController) getPVCFromSourcePVC(vmExport *exportv1.VirtualMachineExport) (*sourceVolumes, error) {
	pvc, pvcExists, err := ctrl.getPvc(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return &sourceVolumes{}, err
	}
	if !pvcExists {
		return &sourceVolumes{
			volumes:          nil,
			inUse:            false,
			isPopulated:      false,
			availableMessage: fmt.Sprintf("pvc %s/%s not found", vmExport.Namespace, vmExport.Spec.Source.Name)}, nil
	}

	isPopulated, inUse, availableMessage, err := ctrl.isSourceAvailablePVC(vmExport, pvc)
	if err != nil {
		return &sourceVolumes{}, err
	}
	return &sourceVolumes{
		volumes:          []*corev1.PersistentVolumeClaim{pvc},
		inUse:            inUse,
		isPopulated:      isPopulated,
		availableMessage: availableMessage}, nil
}

func (ctrl *VMExportController) isPVCInUse(vmExport *exportv1.VirtualMachineExport, pvc *corev1.PersistentVolumeClaim) (bool, error) {
	if pvc == nil {
		return false, nil
	}
	pvcSet := sets.NewString(pvc.Name)
	if usedPods, err := watchutil.PodsUsingPVCs(ctrl.PodInformer, pvc.Namespace, pvcSet); err != nil {
		return false, err
	} else {
		for _, pod := range usedPods {
			if !metav1.IsControlledBy(&pod, vmExport) {
				return true, nil
			}
		}
		return false, nil
	}
}

func (ctrl *VMExportController) updateVMExportPvcStatus(vmExport *exportv1.VirtualMachineExport, exporterPod *corev1.Pod, service *corev1.Service, sourceVolumes *sourceVolumes) (time.Duration, error) {
	var requeue time.Duration

	if !sourceVolumes.isSourceAvailable() && len(sourceVolumes.volumes) > 0 {
		log.Log.V(4).Infof("Source is not available %s, requeuing", sourceVolumes.availableMessage)
		requeue = requeueTime
	}

	vmExportCopy := vmExport.DeepCopy()

	if err := ctrl.updateCommonVMExportStatusFields(vmExport, vmExportCopy, exporterPod, service, sourceVolumes, getVolumeName); err != nil {
		return requeue, err
	}

	if len(sourceVolumes.volumes) == 0 {
		log.Log.V(3).Info("PVC(s) not found, updating status to not found")
		updateCondition(vmExportCopy.Status.Conditions, newPvcCondition(corev1.ConditionFalse, pvcNotFoundReason, sourceVolumes.availableMessage))
	} else {
		updateCondition(vmExportCopy.Status.Conditions, ctrl.pvcConditionFromPVC(sourceVolumes.volumes))
	}

	if err := ctrl.updateVMExportStatus(vmExport, vmExportCopy); err != nil {
		return requeue, err
	}
	return requeue, nil
}
