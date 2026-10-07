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

package cbt

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	failedExportCreate             = "failed to create backup export: %w"
	exportExistsWithDifferentOwner = "VMExport %s already exists but is not owned by backup %s"
	defaultPullModeDurationTTL     = 2 * time.Hour
	backupTTLExpiredMsg            = "pull mode backup TTL has expired"
)

func isPullMode(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Spec.Mode != nil && *backup.Spec.Mode == backupv1.PullMode
}

func getPullBackupTTL(backup *backupv1.VirtualMachineBackup) *metav1.Duration {
	ttl := &metav1.Duration{Duration: defaultPullModeDurationTTL}
	if backup.Spec.TTLDuration != nil {
		ttl = backup.Spec.TTLDuration
	}
	return ttl
}

func getPullBackupRemainingTTL(backup *backupv1.VirtualMachineBackup) *metav1.Duration {
	totalTTL := getPullBackupTTL(backup)
	creationTime := backup.CreationTimestamp.Time

	if creationTime.IsZero() {
		return totalTTL
	}

	elapsed := time.Since(creationTime)
	remaining := totalTTL.Duration - elapsed

	if remaining <= 0 {
		return &metav1.Duration{Duration: 0}
	}

	return &metav1.Duration{Duration: remaining}
}

func isPullBackupTTLExpired(backup *backupv1.VirtualMachineBackup) bool {
	ttl := getPullBackupTTL(backup)
	return time.Since(backup.CreationTimestamp.Time) >= ttl.Duration
}

func (ctrl *VMBackupController) handlePullMode(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) error {
	if isPullBackupTTLExpired(backup) {
		return ctrl.handlePullModeTTLExpiry(backup, vmi)
	}

	vmExport, err := ctrl.getOrCreateBackupExport(backup)
	if err != nil || vmExport == nil {
		return err
	}

	return ctrl.populateExportLinks(backup, vmExport)
}

func (ctrl *VMBackupController) getOrCreateBackupExport(backup *backupv1.VirtualMachineBackup) (*exportv1.VirtualMachineExport, error) {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
	obj, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		return nil, fmt.Errorf("error getting VMExport from store: %w", err)
	}
	if exists {
		vmExport := obj.(*exportv1.VirtualMachineExport)
		if !metav1.IsControlledBy(vmExport, backup) {
			return nil, fmt.Errorf(exportExistsWithDifferentOwner, vmExport.Name, backup.Name)
		}
		return vmExport, nil
	}

	return nil, ctrl.createBackupExport(backup)
}

func (ctrl *VMBackupController) createBackupExport(backup *backupv1.VirtualMachineBackup) error {
	vmExport := &exportv1.VirtualMachineExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(backup, backupv1.SchemeGroupVersion.WithKind(backupv1.VirtualMachineBackupGroupVersionKind.Kind)),
			},
		},
		Spec: exportv1.VirtualMachineExportSpec{
			TokenSecretRef: &backup.Spec.TokenSecretRef,
			TTLDuration:    getPullBackupRemainingTTL(backup),
			Source: corev1.TypedLocalObjectReference{
				APIGroup: pointer.P(backupv1.VirtualMachineBackupGroupVersionKind.Group),
				Kind:     backupv1.VirtualMachineBackupGroupVersionKind.Kind,
				Name:     backup.Name,
			},
		},
	}

	_, err := ctrl.client.VirtualMachineExport(backup.Namespace).Create(context.Background(), vmExport, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf(failedExportCreate, err)
	}

	setPreparingExport(backup)
	backup.Status.Links = nil
	return nil
}

func (ctrl *VMBackupController) populateExportLinks(backup *backupv1.VirtualMachineBackup, vmExport *exportv1.VirtualMachineExport) error {
	if vmExport.Status == nil || vmExport.Status.Phase != exportv1.Ready {
		return nil
	}

	if len(backup.Status.IncludedVolumes) == 0 {
		return nil
	}

	backupLinks, err := buildBackupLinks(vmExport.Status.Links)
	if err != nil {
		return err
	}

	backup.Status.Links = backupLinks
	setExportReady(backup)

	return nil
}

func buildBackupLinks(links *exportv1.VirtualMachineExportLinks) (*backupv1.BackupLinks, error) {
	if links == nil || (links.Internal == nil && links.External == nil) {
		return nil, fmt.Errorf("associated export ready but has no backup links")
	}

	result := &backupv1.BackupLinks{}
	if links.Internal != nil {
		if links.Internal.Cert == "" {
			return nil, fmt.Errorf("associated export ready but internal link has no cert exposed")
		}
		result.Internal = toBackupLink(links.Internal)
	}
	if links.External != nil {
		result.External = toBackupLink(links.External)
	}

	return result, nil
}

func toBackupLink(link *exportv1.VirtualMachineExportLink) *backupv1.BackupLink {
	bl := &backupv1.BackupLink{Cert: link.Cert}
	for _, b := range link.Backups {
		vl := backupv1.BackupVolumeLink{VolumeName: b.Name}
		for _, ep := range b.Endpoints {
			switch ep.Endpoint {
			case exportv1.Data:
				vl.DataEndpoint = ep.Url
			case exportv1.Map:
				vl.MapEndpoint = ep.Url
			}
		}
		bl.Volumes = append(bl.Volumes, vl)
	}
	return bl
}

func (ctrl *VMBackupController) handlePullModeTTLExpiry(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) error {
	if hasVMIBackupStatus(vmi) && !vmi.Status.ChangedBlockTracking.BackupStatus.Completed {
		if err := ctrl.handleAbort(backup, vmi); err != nil {
			return err
		}
		ctrl.setAborting(backup, fmt.Sprintf("%s: %s", backupTTLExpiredMsg, backupAborting))
	}
	return nil
}

func (ctrl *VMBackupController) cleanupBackupExport(backup *backupv1.VirtualMachineBackup) error {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
	_, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		return fmt.Errorf("error getting VMExport from store during TTL expiry: %w", err)
	}
	if exists {
		if err := ctrl.client.VirtualMachineExport(backup.Namespace).Delete(context.Background(), backup.Name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete VMExport during TTL expiry: %w", err)
		}
	}
	return nil
}

func setPreparingExport(backup *backupv1.VirtualMachineBackup) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue,
		Reason: backupv1.ReasonPreparingExport, Message: backupPreparingVMExport,
	})
}

func setExportReady(backup *backupv1.VirtualMachineBackup) {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue,
		Reason: backupv1.ReasonExportReady, Message: backupExportReady,
	})
}
