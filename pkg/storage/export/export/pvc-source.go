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

type PVCSource struct {
	sourceVolumes *sourceVolumes
}

func NewPVCSource(sourceVolumes *sourceVolumes) *PVCSource {
	return &PVCSource{
		sourceVolumes: sourceVolumes,
	}
}

func (s *PVCSource) IsSourceAvailable() bool {
	return s.sourceVolumes.isSourceAvailable()
}

func (s *PVCSource) HasContent() bool {
	return s.sourceVolumes.hasContent()
}

func (s *PVCSource) SourceCondition() exportv1.Condition {
	return s.sourceVolumes.sourceCondition
}

func (s *PVCSource) ReadyCondition() exportv1.Condition {
	return s.sourceVolumes.readyCondition
}

func (s *PVCSource) ServicePorts() []corev1.ServicePort {
	return []corev1.ServicePort{exportPort()}
}

func (s *PVCSource) ConfigurePod(pod *corev1.Pod) {
	s.sourceVolumes.configurePodVolumes(pod)
}

func (s *PVCSource) ConfigureExportLink(exportLink *exportv1.VirtualMachineExportLink, paths *ServerPaths, vmExport *exportv1.VirtualMachineExport, pod *corev1.Pod, hostAndBase, scheme string) {
	s.sourceVolumes.populateLink(exportLink, paths, pod, hostAndBase, scheme, defaultVolumeNamer)
}

func (s *PVCSource) UpdateStatus(vmExport *exportv1.VirtualMachineExport, pod *corev1.Pod, svc *corev1.Service) (time.Duration, error) {
	var requeue time.Duration
	if !s.IsSourceAvailable() && s.HasContent() {
		log.Log.V(4).Infof("Source is not available %s, requeuing", s.SourceCondition().Message)
		requeue = requeueTime
	}

	vmExport.Status.Conditions = updateCondition(vmExport.Status.Conditions, s.SourceCondition())
	return requeue, nil
}

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

func (ctrl *VMExportController) isSourceAvailablePVC(vmExport *exportv1.VirtualMachineExport, pvc *corev1.PersistentVolumeClaim) (*sourceVolumes, error) {
	sourceVolumes := &sourceVolumes{
		volumes:         ctrl.pvcsToSourceVolumes(pvc),
		inUse:           false,
		isPopulated:     false,
		readyCondition:  newReadyCondition(corev1.ConditionFalse, initializingReason, ""),
		sourceCondition: exportv1.Condition{},
	}

	isPopulated, err := ctrl.isPVCPopulated(pvc)
	if err != nil {
		return nil, err
	}
	sourceVolumes.isPopulated = isPopulated

	if isPopulated {
		inUse, err := ctrl.isPVCInUse(vmExport, pvc)
		if err != nil {
			return nil, err
		}
		sourceVolumes.inUse = inUse
		if inUse {
			sourceVolumes.readyCondition = newReadyCondition(corev1.ConditionFalse, inUseReason,
				fmt.Sprintf("PersistentVolumeClaim %s/%s is in use", pvc.Namespace, pvc.Name))
		}
	} else {
		sourceVolumes.readyCondition = newReadyCondition(corev1.ConditionFalse, pvcPendingReason,
			fmt.Sprintf("PersistentVolumeClaim %s/%s is not populated", pvc.Namespace, pvc.Name))
	}

	sourceVolumes.sourceCondition = ctrl.pvcConditionFromPVC([]*corev1.PersistentVolumeClaim{pvc})

	return sourceVolumes, nil
}

func (ctrl *VMExportController) getPVCFromSourcePVC(vmExport *exportv1.VirtualMachineExport) (*sourceVolumes, error) {
	pvc, pvcExists, err := ctrl.getPvc(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return nil, err
	}
	if !pvcExists {
		return &sourceVolumes{
			volumes:         nil,
			inUse:           false,
			isPopulated:     false,
			readyCondition:  newReadyCondition(corev1.ConditionFalse, initializingReason, ""),
			sourceCondition: newPvcCondition(corev1.ConditionFalse, pvcNotFoundReason, fmt.Sprintf("PersistentVolumeClaim %s/%s not found", vmExport.Namespace, vmExport.Spec.Source.Name)),
		}, nil
	}

	return ctrl.isSourceAvailablePVC(vmExport, pvc)
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
