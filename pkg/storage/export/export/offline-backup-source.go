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

	corev1 "k8s.io/api/core/v1"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	exportv1 "kubevirt.io/api/export/v1"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	offlineStateMountPath  = "/backup-state"
	offlineSocketDir       = "/sockets"
	offlineTargetMountPath = "/backup-target"
	offlineSocketVolume    = "backup-sockets"
	offlineStateVolume     = "backup-state"
	offlineTargetVolume    = "backup-target"

	backupTimeFormat = "2006-01-02_15-04-05"
)

// OfflineVMBackupSource serves an offline (stopped VM) incremental backup by
// mounting the VM's data PVCs and the backend-storage state PVC read-write and
// running qemu-nbd against the persisted QCOW2 overlays inside the export pod.
type OfflineVMBackupSource struct {
	*VMBackupSource
	vmName         string
	dataPVCs       map[string]*corev1.PersistentVolumeClaim
	statePVC       *corev1.PersistentVolumeClaim
	targetPVC      *corev1.PersistentVolumeClaim
	baseCheckpoint string
}

func NewOfflineVMBackupSource(vmBackup *backupv1.VirtualMachineBackup, vmName string, dataPVCs map[string]*corev1.PersistentVolumeClaim, statePVC, targetPVC *corev1.PersistentVolumeClaim, baseCheckpoint string) *OfflineVMBackupSource {
	return &OfflineVMBackupSource{
		VMBackupSource: NewVMBackupSource(vmBackup, "", ""),
		vmName:         vmName,
		dataPVCs:       dataPVCs,
		statePVC:       statePVC,
		targetPVC:      targetPVC,
		baseCheckpoint: baseCheckpoint,
	}
}

func (s *OfflineVMBackupSource) isPush() bool {
	return s.vmBackup.Spec.Mode != nil && *s.vmBackup.Spec.Mode == backupv1.PushMode
}

func (s *OfflineVMBackupSource) targetDir() string {
	backupTime := s.vmBackup.CreationTimestamp.UTC().Format(backupTimeFormat)
	return fmt.Sprintf("%s/%s/%s-%s", offlineTargetMountPath, s.vmName, s.vmBackup.Name, backupTime)
}

func (s *OfflineVMBackupSource) ConfigurePod(pod *corev1.Pod) {
	container := &pod.Spec.Containers[0]

	mountPVC(pod, container, offlineStateVolume, s.statePVC, offlineStateMountPath)

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name:         offlineSocketVolume,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	})
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      offlineSocketVolume,
		MountPath: offlineSocketDir,
	})

	container.Env = append(container.Env,
		corev1.EnvVar{Name: "OFFLINE_BACKUP", Value: "true"},
		corev1.EnvVar{Name: "BACKUP_STATE_PATH", Value: offlineStateMountPath},
		corev1.EnvVar{Name: "SOCKET_DIR", Value: offlineSocketDir},
		corev1.EnvVar{Name: "BACKUP_TYPE", Value: string(s.vmBackup.Status.Type)},
	)
	if s.vmBackup.Status.CheckpointName != nil {
		container.Env = append(container.Env, corev1.EnvVar{Name: "BACKUP_CHECKPOINT", Value: *s.vmBackup.Status.CheckpointName})
	}
	if s.baseCheckpoint != "" {
		container.Env = append(container.Env, corev1.EnvVar{Name: "BACKUP_BASE_CHECKPOINT", Value: s.baseCheckpoint})
	}
	container.Env = append(container.Env, corev1.EnvVar{Name: "BACKUP_UID", Value: string(s.vmBackup.UID)})

	if s.isPush() && s.targetPVC != nil {
		mountPVC(pod, container, offlineTargetVolume, s.targetPVC, offlineTargetMountPath)
		container.Env = append(container.Env,
			corev1.EnvVar{Name: "BACKUP_MODE", Value: string(backupv1.PushMode)},
			corev1.EnvVar{Name: "BACKUP_TARGET_DIR", Value: s.targetDir()},
			corev1.EnvVar{Name: "BACKUP_NAME", Value: s.vmBackup.Name},
		)
	}

	for index, volume := range s.vmBackup.Status.IncludedVolumes {
		pvc := s.dataPVCs[volume.VolumeName]
		if pvc == nil {
			continue
		}
		diskPath := offlineDataMountPath(pvc)
		mountPVC(pod, container, getExportPodVolumeName(pvc), pvc, diskPath)
		container.Env = append(container.Env,
			corev1.EnvVar{Name: fmt.Sprintf("BACKUP%d_BACKUP_PATH", index), Value: volume.VolumeName},
			corev1.EnvVar{Name: fmt.Sprintf("BACKUP%d_DATA_URI", index), Value: backupDataURI(volume.VolumeName)},
			corev1.EnvVar{Name: fmt.Sprintf("BACKUP%d_MAP_URI", index), Value: backupMapURI(volume.VolumeName)},
			corev1.EnvVar{Name: fmt.Sprintf("BACKUP%d_DISK_PATH", index), Value: diskPath},
		)
	}
}

func offlineDataMountPath(pvc *corev1.PersistentVolumeClaim) string {
	volumeName := getExportPodVolumeName(pvc)
	if types.IsPVCBlock(pvc.Spec.VolumeMode) {
		return fmt.Sprintf("%s/%s", blockVolumeMountPath, volumeName)
	}
	return fmt.Sprintf("%s/%s", fileSystemMountPath, volumeName)
}

func mountPVC(pod *corev1.Pod, container *corev1.Container, volumeName string, pvc *corev1.PersistentVolumeClaim, mountPath string) {
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvc.Name},
		},
	})
	if types.IsPVCBlock(pvc.Spec.VolumeMode) {
		container.VolumeDevices = append(container.VolumeDevices, corev1.VolumeDevice{
			Name:       volumeName,
			DevicePath: mountPath,
		})
		return
	}
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	})
}

func isOfflineBackup(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Status != nil && backup.Status.Offline != nil && *backup.Status.Offline
}

func (ctrl *VMExportController) getOfflineBackupSource(vmExport *exportv1.VirtualMachineExport, vmBackup *backupv1.VirtualMachineBackup) (*OfflineVMBackupSource, error) {
	vmName, err := ctrl.getBackupSourceVMName(vmBackup)
	if err != nil {
		return nil, err
	}

	statePVC := ctrl.findStatePVC(vmExport.Namespace, vmName)
	if statePVC == nil {
		return nil, fmt.Errorf("backend-storage PVC for VM %s/%s not found", vmExport.Namespace, vmName)
	}

	dataPVCs, err := ctrl.resolveOfflineDataPVCs(vmExport.Namespace, vmName, vmBackup)
	if err != nil {
		return nil, err
	}

	baseCheckpoint := ""
	if vmBackup.Status.Type == backupv1.Incremental {
		tracker, exists, err := ctrl.getBackupTracker(vmExport.Namespace, vmBackup.Spec.Source.Name)
		if err != nil {
			return nil, err
		}
		if exists && tracker.Status != nil && tracker.Status.LatestCheckpoint != nil {
			baseCheckpoint = tracker.Status.LatestCheckpoint.Name
		}
	}

	var targetPVC *corev1.PersistentVolumeClaim
	if vmBackup.Spec.Mode != nil && *vmBackup.Spec.Mode == backupv1.PushMode {
		if vmBackup.Spec.PvcName == nil {
			return nil, fmt.Errorf("push mode offline backup %s/%s has no target PVC", vmBackup.Namespace, vmBackup.Name)
		}
		pvc, exists, err := ctrl.getPvc(vmExport.Namespace, *vmBackup.Spec.PvcName)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("target PVC %s/%s not found", vmExport.Namespace, *vmBackup.Spec.PvcName)
		}
		targetPVC = pvc
	}

	return NewOfflineVMBackupSource(vmBackup, vmName, dataPVCs, statePVC, targetPVC, baseCheckpoint), nil
}

func (ctrl *VMExportController) findStatePVC(namespace, vmName string) *corev1.PersistentVolumeClaim {
	for _, obj := range ctrl.PVCInformer.GetStore().List() {
		pvc := obj.(*corev1.PersistentVolumeClaim)
		if pvc.Namespace != namespace || pvc.DeletionTimestamp != nil {
			continue
		}
		if pvc.Labels[backendstorage.PVCPrefix] == vmName {
			return pvc
		}
	}
	return nil
}

func (ctrl *VMExportController) resolveOfflineDataPVCs(namespace, vmName string, vmBackup *backupv1.VirtualMachineBackup) (map[string]*corev1.PersistentVolumeClaim, error) {
	vm, exists, err := ctrl.getVm(namespace, vmName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("VM %s/%s not found", namespace, vmName)
	}
	if vm.Spec.Template == nil {
		return nil, fmt.Errorf("VM %s/%s has no template", namespace, vmName)
	}

	included := map[string]bool{}
	for _, v := range vmBackup.Status.IncludedVolumes {
		included[v.VolumeName] = true
	}

	dataPVCs := map[string]*corev1.PersistentVolumeClaim{}
	for i := range vm.Spec.Template.Spec.Volumes {
		volume := vm.Spec.Template.Spec.Volumes[i]
		if !included[volume.Name] {
			continue
		}
		pvcName := types.PVCNameFromVirtVolume(&volume)
		if pvcName == "" {
			continue
		}
		pvc, exists, err := ctrl.getPvc(namespace, pvcName)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("data PVC %s/%s for volume %s not found", namespace, pvcName, volume.Name)
		}
		dataPVCs[volume.Name] = pvc
	}
	return dataPVCs, nil
}

func (ctrl *VMExportController) getBackupSourceVMName(vmBackup *backupv1.VirtualMachineBackup) (string, error) {
	if vmBackup.Spec.Source.Kind == backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		tracker, exists, err := ctrl.getBackupTracker(vmBackup.Namespace, vmBackup.Spec.Source.Name)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("VirtualMachineBackupTracker not found: %s/%s", vmBackup.Namespace, vmBackup.Spec.Source.Name)
		}
		return tracker.Spec.Source.Name, nil
	}
	return vmBackup.Spec.Source.Name, nil
}
