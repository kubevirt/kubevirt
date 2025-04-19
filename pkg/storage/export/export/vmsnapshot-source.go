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
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	exportv1 "kubevirt.io/api/export/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/controller"

	"kubevirt.io/kubevirt/pkg/storage/snapshot"
)

const (
	noVolumeSnapshotReason = "VMSnapshotNoVolumes"

	notAllPVCsCreated = "NotAllPVCsCreated"
	allPVCsReady      = "AllPVCsReady"
	notAllPVCsReady   = "NotAllPVCsReady"
)

func (ctrl *VMExportController) handleVMSnapshot(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if snapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot); ok {
		snapshotKey, _ := cache.MetaNamespaceKeyFunc(snapshot)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("vmsnapshot", snapshotKey)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}
		for _, key := range keys {
			log.Log.V(3).Infof("Adding VMExport due to VMSnapshot %s", snapshotKey)
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) getPVCFromSourceVMSnapshot(vmExport *exportv1.VirtualMachineExport) (*sourceVolumes, error) {
	vmSnapshot, exists, err := ctrl.getVmSnapshot(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return &sourceVolumes{}, err
	}
	if !exists {
		return &sourceVolumes{
			volumes:          nil,
			inUse:            false,
			isPopulated:      false,
			availableMessage: fmt.Sprintf("VirtualMachineSnapshot %s/%s does not exist", vmExport.Namespace, vmExport.Spec.Source.Name)}, nil
	}
	if vmSnapshot.Status != nil && vmSnapshot.Status.ReadyToUse != nil && *vmSnapshot.Status.ReadyToUse {
		pvcs, restoreableSnapshots, err := ctrl.handlePVCsForVirtualMachineSnapshot(vmExport, vmSnapshot)
		if err != nil {
			return &sourceVolumes{}, err
		}
		if len(pvcs) == restoreableSnapshots && restoreableSnapshots > 0 {
			return &sourceVolumes{
				volumes:          pvcs,
				inUse:            false,
				isPopulated:      true,
				availableMessage: ""}, nil
		}
		if restoreableSnapshots == 0 {
			return &sourceVolumes{
				volumes:          nil,
				inUse:            false,
				isPopulated:      false,
				availableMessage: fmt.Sprintf("VirtualMachineSnapshot %s/%s does not contain any volume snapshots", vmExport.Namespace, vmExport.Spec.Source.Name)}, nil
		}
		return &sourceVolumes{
			volumes:          nil,
			inUse:            false,
			isPopulated:      false,
			availableMessage: "Not all PVCs have been successfully restored"}, nil
	}
	return &sourceVolumes{
		volumes:          nil,
		inUse:            false,
		isPopulated:      false,
		availableMessage: fmt.Sprintf("VirtualMachineSnapshot %s/%s is not ready to use", vmExport.Namespace, vmExport.Spec.Source.Name)}, nil
}

func (ctrl *VMExportController) handlePVCsForVirtualMachineSnapshot(vmExport *exportv1.VirtualMachineExport, vmSnapshot *snapshotv1.VirtualMachineSnapshot) ([]*corev1.PersistentVolumeClaim, int, error) {
	var content *snapshotv1.VirtualMachineSnapshotContent
	var err error
	var pvcs []*corev1.PersistentVolumeClaim
	exists := false
	totalVolumes := 0

	if vmSnapshot.Status.VirtualMachineSnapshotContentName != nil && *vmSnapshot.Status.VirtualMachineSnapshotContentName != "" {
		content, exists, err = ctrl.getVmSnapshotContent(vmSnapshot.Namespace, *vmSnapshot.Status.VirtualMachineSnapshotContentName)
		if err != nil {
			return nil, 0, err
		}
		if exists {
			sourceVm := content.Spec.Source.VirtualMachine
			totalVolumes = len(content.Status.VolumeSnapshotStatus)

			for _, volumeBackup := range content.Spec.VolumeBackups {
				if pvc, err := ctrl.getOrCreatePVCFromSnapshot(vmExport, &volumeBackup, sourceVm); err != nil {
					return nil, 0, err
				} else {
					pvcs = append(pvcs, pvc)
				}
			}
		}
	}
	return pvcs, totalVolumes, err
}

func (ctrl *VMExportController) getOrCreatePVCFromSnapshot(vmExport *exportv1.VirtualMachineExport, volumeBackup *snapshotv1.VolumeBackup, sourceVm *snapshotv1.VirtualMachine) (*corev1.PersistentVolumeClaim, error) {
	if volumeBackup.VolumeSnapshotName == nil {
		log.Log.Errorf("VolumeSnapshot name missing %+v", volumeBackup)
		return nil, fmt.Errorf("missing VolumeSnapshot name")
	}
	restorePVCName := fmt.Sprintf("%s-%s", vmExport.Name, volumeBackup.PersistentVolumeClaim.Name)

	if pvc, exists, err := ctrl.getPvc(vmExport.Namespace, restorePVCName); err != nil {
		return nil, err
	} else if exists {
		return pvc, nil
	}

	volumeSnapshot, err := ctrl.VolumeSnapshotProvider.GetVolumeSnapshot(vmExport.Namespace, *volumeBackup.VolumeSnapshotName)
	if err != nil {
		return nil, err
	}

	// leaving source name and namespace blank because we don't care in this context
	pvc, err := snapshot.CreateRestorePVCDef(restorePVCName, volumeSnapshot, volumeBackup)
	if err != nil {
		return nil, err
	}
	if volumeBackupIsKubeVirtContent(volumeBackup, sourceVm) {
		if len(pvc.GetAnnotations()) == 0 {
			pvc.SetAnnotations(make(map[string]string))
		}
		pvc.Annotations[annContentType] = string(cdiv1.DataVolumeKubeVirt)
	}
	pvc.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         exportGVK.GroupVersion().String(),
			Kind:               exportGVK.Kind,
			Name:               vmExport.Name,
			UID:                vmExport.UID,
			Controller:         pointer.P(true),
			BlockOwnerDeletion: pointer.P(true),
		},
	})

	pvc, err = ctrl.Client.CoreV1().PersistentVolumeClaims(vmExport.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return pvc, nil
}

func (ctrl *VMExportController) updateVMExporVMSnapshotStatus(vmExport *exportv1.VirtualMachineExport, exporterPod *corev1.Pod, service *corev1.Service, sourceVolumes *sourceVolumes) (time.Duration, error) {
	vmExportCopy := vmExport.DeepCopy()
	vmExportCopy.Status.VirtualMachineName = pointer.P(ctrl.getVmNameFromVmSnapshot(vmExport))

	if err := ctrl.updateCommonVMExportStatusFields(vmExport, vmExportCopy, exporterPod, service, sourceVolumes, getSnapshotVolumeName); err != nil {
		return 0, err
	}

	if err := ctrl.updateVMSnapshotExportStatusConditions(vmExportCopy, sourceVolumes.volumes, sourceVolumes.availableMessage); err != nil {
		return 0, err
	}

	if err := ctrl.updateVMExportStatus(vmExport, vmExportCopy); err != nil {
		return 0, err
	}
	return 0, nil
}

func (ctrl *VMExportController) getVmNameFromVmSnapshot(vmExport *exportv1.VirtualMachineExport) string {
	if ctrl.isSourceVMSnapshot(&vmExport.Spec) {
		if vmSnapshot, exists, err := ctrl.getVmSnapshot(vmExport.Namespace, vmExport.Spec.Source.Name); err != nil {
			log.Log.V(3).Infof("Error getting snapshot %v", err)
			return ""
		} else if exists {
			return vmSnapshot.Spec.Source.Name
		}
	}
	return ""
}

func (ctrl *VMExportController) updateVMSnapshotExportStatusConditions(vmExportCopy *exportv1.VirtualMachineExport, pvcs []*corev1.PersistentVolumeClaim, availableMessage string) error {
	vmSnapshot, exists, err := ctrl.getVmSnapshot(vmExportCopy.Namespace, vmExportCopy.Spec.Source.Name)
	if err != nil {
		return err
	}

	if !exists {
		vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, initializingReason, ""))
		return nil
	}
	if vmSnapshot.Status != nil && vmSnapshot.Status.VirtualMachineSnapshotContentName != nil && *vmSnapshot.Status.VirtualMachineSnapshotContentName != "" {
		content, exists, err := ctrl.getVmSnapshotContent(vmSnapshot.Namespace, *vmSnapshot.Status.VirtualMachineSnapshotContentName)
		if err != nil {
			return err
		}
		if exists {
			if len(content.Status.VolumeSnapshotStatus) == 0 {
				vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newVolumesCreatedCondition(corev1.ConditionFalse, noVolumeSnapshotReason, availableMessage))
				vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newReadyCondition(corev1.ConditionFalse, initializingReason, ""))
				vmExportCopy.Status.Phase = exportv1.Skipped
			} else if len(content.Status.VolumeSnapshotStatus) != len(pvcs) {
				vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newVolumesCreatedCondition(corev1.ConditionFalse, notAllPVCsCreated, availableMessage))
			} else {
				readyCount := 0
				for _, pvc := range pvcs {
					if pvc.Status.Phase == corev1.ClaimBound {
						readyCount++
					}
				}
				if readyCount == len(pvcs) {
					vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newVolumesCreatedCondition(corev1.ConditionTrue, allPVCsReady, availableMessage))
				} else {
					vmExportCopy.Status.Conditions = updateCondition(vmExportCopy.Status.Conditions, newVolumesCreatedCondition(corev1.ConditionFalse, notAllPVCsReady, "Not all PVCs are ready"))
				}
			}
		}
	}
	return nil
}

func (ctrl *VMExportController) isSourceVMSnapshot(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && source.Source.APIGroup != nil && *source.Source.APIGroup == snapshotv1.SchemeGroupVersion.Group && source.Source.Kind == "VirtualMachineSnapshot"
}

func (ctrl *VMExportController) getVmSnapshot(namespace, name string) (*snapshotv1.VirtualMachineSnapshot, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.VMSnapshotInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*snapshotv1.VirtualMachineSnapshot).DeepCopy(), true, nil
}

func (ctrl *VMExportController) getVmSnapshotContent(namespace, name string) (*snapshotv1.VirtualMachineSnapshotContent, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.VMSnapshotContentInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*snapshotv1.VirtualMachineSnapshotContent).DeepCopy(), true, nil
}

func volumeBackupIsKubeVirtContent(volumeBackup *snapshotv1.VolumeBackup, sourceVm *snapshotv1.VirtualMachine) bool {
	if sourceVm != nil && sourceVm.Spec.Template != nil {
		for _, volume := range sourceVm.Spec.Template.Spec.Volumes {
			if volume.Name == volumeBackup.VolumeName && (volume.DataVolume != nil || volume.PersistentVolumeClaim != nil) {
				return true
			}
		}
	}
	return false
}

func getSnapshotVolumeName(pvc *corev1.PersistentVolumeClaim, vmExport *exportv1.VirtualMachineExport) string {
	// When exporting snapshots, we change the name of the
	// restore PVC to match the volume name of the source VM
	return strings.TrimPrefix(pvc.Name, fmt.Sprintf("%s-", vmExport.Name))
}
