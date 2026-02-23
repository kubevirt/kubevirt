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
	"path"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/log"

	backupv1 "kubevirt.io/api/backup/v1alpha1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	backupsBasePath         = "/exports"
	vmBackupReadyReason     = "VirtualMachineBackupReady"
	vmBackupNotReadyMessage = "VMBackup is not progressing"

	backupServicePort = 9090
)

type VMBackupSource struct {
	vmBackup *backupv1.VirtualMachineBackup
	caCert   string
}

func NewVMBackupSource(vmBackup *backupv1.VirtualMachineBackup, caCert string) *VMBackupSource {
	return &VMBackupSource{
		vmBackup: vmBackup,
		caCert:   caCert,
	}
}

func (s *VMBackupSource) IsSourceAvailable() bool {
	return s.hasCondition(backupv1.ConditionProgressing)
}

func (s *VMBackupSource) HasContent() bool {
	return s.vmBackup.Status != nil && len(s.vmBackup.Status.IncludedVolumes) > 0
}

func (s *VMBackupSource) SourceCondition() exportv1.Condition {
	if s.vmBackup == nil {
		return newReadyCondition(corev1.ConditionFalse, vmBackupReadyReason, "VMBackup doesn't exist yet")
	}
	if s.vmBackup.Status != nil && s.HasContent() {
		cond := s.condition(backupv1.ConditionProgressing)
		if cond == nil {
			return newReadyCondition(corev1.ConditionFalse, vmBackupReadyReason, "backup progressing condition not found")
		}
		if cond.Status == corev1.ConditionFalse {
			return newReadyCondition(corev1.ConditionFalse, vmBackupReadyReason, cond.Message)
		}
		return newReadyCondition(corev1.ConditionTrue, vmBackupReadyReason, cond.Message)
	}
	return newReadyCondition(corev1.ConditionFalse, vmBackupReadyReason, vmBackupNotReadyMessage)
}

func (s *VMBackupSource) ReadyCondition() exportv1.Condition {
	if !s.IsSourceAvailable() || !s.HasContent() {
		return newReadyCondition(corev1.ConditionFalse, initializingReason, "")
	}
	return exportv1.Condition{}
}

func (s *VMBackupSource) ServicePorts() []corev1.ServicePort {
	return []corev1.ServicePort{exportPort()}
}

func (s *VMBackupSource) ConfigurePod(pod *corev1.Pod) {
	for index, volume := range s.vmBackup.Status.IncludedVolumes {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  fmt.Sprintf("BACKUP%d_BACKUP_PATH", index),
			Value: volume.VolumeName,
		}, corev1.EnvVar{
			Name:  fmt.Sprintf("BACKUP%d_DATA_URI", index),
			Value: backupDataURI(volume.VolumeName),
		}, corev1.EnvVar{
			Name:  fmt.Sprintf("BACKUP%d_MAP_URI", index),
			Value: backupMapURI(volume.VolumeName),
		})
	}
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "BACKUP_CACERT",
		Value: s.caCert,
	})
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "BACKUP_UID",
		Value: string(s.vmBackup.UID),
	})
	if s.vmBackup.Status != nil {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "BACKUP_TYPE",
			Value: string(s.vmBackup.Status.Type),
		})
		if s.vmBackup.Status.CheckpointName != nil {
			pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
				Name:  "BACKUP_CHECKPOINT",
				Value: *s.vmBackup.Status.CheckpointName,
			})
		}
	}
}

func (s *VMBackupSource) ConfigureExportLink(exportLink *exportv1.VirtualMachineExportLink, paths *ServerPaths, vmExport *exportv1.VirtualMachineExport, pod *corev1.Pod, hostAndBase, scheme string) {
	if pod == nil {
		return
	}

	if s.vmBackup.Status == nil || !s.HasContent() {
		return
	}

	for _, volume := range s.vmBackup.Status.IncludedVolumes {
		backupInfo := paths.GetBackupInfo(volume.VolumeName)
		if backupInfo == nil {
			log.Log.Warningf("Backup %s not found in paths", volume.VolumeName)
			continue
		}

		eb := exportv1.VirtualMachineExportBackup{
			Name: volume.VolumeName,
		}

		if backupInfo.DataURI != "" {
			eb.Endpoints = append(eb.Endpoints, exportv1.VirtualMachineExportBackupEndpoint{
				Endpoint: exportv1.Data,
				Url:      scheme + path.Join(hostAndBase, backupInfo.DataURI),
			})
		}

		if backupInfo.MapURI != "" {
			eb.Endpoints = append(eb.Endpoints, exportv1.VirtualMachineExportBackupEndpoint{
				Endpoint: exportv1.Map,
				Url:      scheme + path.Join(hostAndBase, backupInfo.MapURI),
			})
		}

		if len(eb.Endpoints) == 0 {
			log.Log.Warningf("No endpoints found for backup %s", volume.VolumeName)
			continue
		}

		exportLink.Backups = append(exportLink.Backups, eb)
	}
}

func (s *VMBackupSource) UpdateStatus(vmExport *exportv1.VirtualMachineExport, pod *corev1.Pod, svc *corev1.Service) (time.Duration, error) {
	var requeue time.Duration
	if !s.IsSourceAvailable() {
		log.Log.V(4).Infof("Source is not available %s, requeuing", s.SourceCondition().Message)
		requeue = requeueTime
	}

	vmExport.Status.Conditions = updateCondition(vmExport.Status.Conditions, s.SourceCondition())
	return requeue, nil
}

func (s *VMBackupSource) hasCondition(condType backupv1.ConditionType) bool {
	if s.vmBackup.Status == nil {
		return false
	}
	cond := s.condition(condType)
	return cond != nil && cond.Status == corev1.ConditionTrue
}

func (s *VMBackupSource) condition(condType backupv1.ConditionType) *backupv1.Condition {
	if s.vmBackup.Status == nil {
		return nil
	}
	for _, cond := range s.vmBackup.Status.Conditions {
		if cond.Type == condType {
			return cond.DeepCopy()
		}
	}
	return nil
}

func (ctrl *VMExportController) handleVMBackup(obj any) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	if backup, ok := obj.(*backupv1.VirtualMachineBackup); ok {
		backupKey, _ := cache.MetaNamespaceKeyFunc(backup)
		keys, err := ctrl.VMExportInformer.GetIndexer().IndexKeys("virtualmachinebackup", backupKey)
		if err != nil {
			utilruntime.HandleError(err)
			return
		}

		for _, key := range keys {
			log.Log.V(3).Infof("Adding VMExport due to backup %s", backupKey)
			ctrl.vmExportQueue.Add(key)
		}
	}
}

func (ctrl *VMExportController) isSourceBackup(source *exportv1.VirtualMachineExportSpec) bool {
	return source != nil && (source.Source.APIGroup == nil || *source.Source.APIGroup == backupv1.SchemeGroupVersion.Group) && source.Source.Kind == "VirtualMachineBackup"
}

func (ctrl *VMExportController) getBackup(namespace, name string) (*backupv1.VirtualMachineBackup, bool, error) {
	key := controller.NamespacedKey(namespace, name)
	obj, exists, err := ctrl.VMBackupInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return obj.(*backupv1.VirtualMachineBackup).DeepCopy(), true, nil
}

func (ctrl *VMExportController) getVMBackupFromExport(vmExport *exportv1.VirtualMachineExport) (*backupv1.VirtualMachineBackup, error) {
	vmBackup, exists, err := ctrl.getBackup(vmExport.Namespace, vmExport.Spec.Source.Name)
	if err != nil {
		return nil, fmt.Errorf("error fetching backup %s/%s: %w", vmExport.Namespace, vmExport.Spec.Source.Name, err)
	}
	if !exists {
		return nil, fmt.Errorf("VirtualMachineBackup not found: %s/%s", vmExport.Namespace, vmExport.Spec.Source.Name)
	}
	return vmBackup, nil
}

func (ctrl *VMExportController) backupCA() (string, error) {
	key := controller.NamespacedKey(ctrl.KubevirtNamespace, components.KubeVirtBackupCASecretName)
	obj, exists, err := ctrl.BackupCAConfigMapInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return "", err

	}
	cm := obj.(*corev1.ConfigMap).DeepCopy()
	bundle := cm.Data[caBundle]
	return strings.TrimSpace(bundle), nil
}

func backupMapURI(volumeName string) string {
	return path.Join(backupsBasePath, volumeName, "map")
}

func backupDataURI(volumeName string) string {
	return path.Join(backupsBasePath, volumeName, "data")
}
