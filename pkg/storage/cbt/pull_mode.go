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
	"crypto/ecdsa"
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
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	failedExportCreate             = "failed to create backup export: %w"
	backupTTLExpiredMsg            = "pull mode backup TTL has expired"
	exportExistsWithDifferentOwner = "VMExport %s already exists but is not owned by backup %s"
	defaultPullModeDurationTTL     = 2 * time.Hour
)

func isPushMode(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Spec.Mode == nil || *backup.Spec.Mode == backupv1.PushMode
}

func isPullMode(backup *backupv1.VirtualMachineBackup) bool {
	return backup.Spec.Mode != nil && *backup.Spec.Mode == backupv1.PullMode
}

func isBackupExportInitialized(backup *backupv1.VirtualMachineBackup) bool {
	return meta.IsStatusConditionTrue(backupConditions(backup), string(backupv1.ConditionExportInitiated))
}

func isBackupExportReady(backup *backupv1.VirtualMachineBackup) bool {
	return meta.IsStatusConditionTrue(backupConditions(backup), string(backupv1.ConditionExportReady))
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

func (ctrl *VMBackupController) handlePullMode(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	if isPullBackupTTLExpired(backup) {
		return ctrl.handlePullModeTTLExpiry(backup, vmi)
	}

	if !isBackupExportInitialized(backup) {
		if syncInfo := ctrl.handlePrepareBackupExport(backup, vmi); syncInfo != nil {
			return syncInfo
		}
	}

	if !isBackupExportReady(backup) {
		if syncInfo := ctrl.waitForBackupExportReady(backup, vmi); syncInfo != nil {
			return syncInfo
		}
	}

	if isBackupExportReady(backup) {
		if syncInfo := ctrl.validateExportHealth(backup); syncInfo != nil {
			return syncInfo
		}
	}

	return nil
}

func (ctrl *VMBackupController) handlePrepareBackupExport(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	syncInfo, vmExport := ctrl.getOrCreateBackupExport(vmi, backup)
	if syncInfo != nil {
		return syncInfo
	}
	if vmExport.Status == nil || vmExport.Status.ServiceName == "" {
		// Service name not yet set by export controller, retry later
		return nil
	}
	ca, err := ctrl.exportCaManager.GetCurrentRaw()
	if err != nil {
		return syncInfoError(err)
	}
	keyPair, err := ctrl.generateBackupTunnelCert(backup)
	if err != nil {
		return syncInfoError(err)
	}
	exportAddr := fmt.Sprintf("%s.%s.svc", vmExport.Status.ServiceName, vmExport.Namespace)
	serverName := fmt.Sprintf("%s.cluster.local", exportAddr)
	backupOptions := &backupv1.BackupOptions{
		BackupName:       backup.Name,
		Cmd:              backupv1.Export,
		BackupStartTime:  &backup.CreationTimestamp,
		Mode:             *backup.Spec.Mode,
		ExportServerAddr: &exportAddr,
		ExportServerName: &serverName,
		BackupKey:        pointer.P(string(cert.EncodePrivateKeyPEM(keyPair.Key))),
		BackupCert:       pointer.P(string(cert.EncodeCertPEM(keyPair.Cert))),
		CACert:           pointer.P(string(ca)),
	}
	if err := ctrl.client.VirtualMachineInstance(vmi.Namespace).Backup(context.Background(), vmi.Name, backupOptions); err != nil {
		return syncInfoError(err)
	}
	return &SyncInfo{
		event:  backupExportInitiatedEvent,
		reason: backupExportInitiated,
	}
}

func (ctrl *VMBackupController) getOrCreateBackupExport(vmi *v1.VirtualMachineInstance, backup *backupv1.VirtualMachineBackup) (*SyncInfo, *exportv1.VirtualMachineExport) {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
	obj, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		err = fmt.Errorf("error getting VMExport from store: %w", err)
		log.Log.Error(err.Error())
		return syncInfoError(err), nil
	}
	if exists {
		vmExport := obj.(*exportv1.VirtualMachineExport)
		if !metav1.IsControlledBy(vmExport, backup) {
			return syncInfoError(fmt.Errorf(exportExistsWithDifferentOwner, vmExport.Name, backup.Name)), nil
		}
		return nil, vmExport
	}

	return ctrl.createBackupExport(backup, vmi), nil
}

func (ctrl *VMBackupController) createBackupExport(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
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
		return syncInfoError(fmt.Errorf(failedExportCreate, err))
	}

	return &SyncInfo{
		event:  backupPreparingVMExportEvent,
		reason: backupPreparingVMExport,
	}
}

func (ctrl *VMBackupController) waitForBackupExportReady(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
	obj, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		err = fmt.Errorf("error getting VMExport from store: %w", err)
		log.Log.Error(err.Error())
		return syncInfoError(err)
	}
	if !exists {
		return syncInfoError(fmt.Errorf("associated export does not exist"))
	}
	vmExport := obj.(*exportv1.VirtualMachineExport)

	if vmExport.Status == nil || vmExport.Status.Phase != exportv1.Ready {
		return nil
	}

	if len(backup.Status.IncludedVolumes) == 0 {
		return nil
	}

	links := vmExport.Status.Links
	hasInternalLinks := links != nil && links.Internal != nil && len(links.Internal.Backups) > 0
	hasExternalLinks := links != nil && links.External != nil && len(links.External.Backups) > 0

	if !hasInternalLinks && !hasExternalLinks {
		return syncInfoError(fmt.Errorf("associated export ready but has no backup links"))
	}

	iterableLinks := links.External
	if !hasExternalLinks {
		iterableLinks = links.Internal
	}

	if iterableLinks.Cert == "" {
		return syncInfoError(fmt.Errorf("associated export ready but has no cert exposed"))
	}

	syncInfo := &SyncInfo{
		event:  backupExportReadyEvent,
		reason: backupExportReady,
		caCert: &iterableLinks.Cert,
	}
	endpointMap := make(map[string][]exportv1.VirtualMachineExportBackupEndpoint)
	for _, backupEndpoint := range iterableLinks.Backups {
		endpointMap[backupEndpoint.Name] = backupEndpoint.Endpoints
	}
	for _, volume := range backup.Status.IncludedVolumes {
		if endpoints, ok := endpointMap[volume.VolumeName]; ok {
			for _, link := range endpoints {
				switch link.Endpoint {
				case exportv1.Data:
					volume.DataEndpoint = link.Url
				case exportv1.Map:
					volume.MapEndpoint = link.Url
				}
			}
		}
		syncInfo.includedVolumes = append(syncInfo.includedVolumes, volume)
	}
	return syncInfo
}

func (ctrl *VMBackupController) validateExportHealth(backup *backupv1.VirtualMachineBackup) *SyncInfo {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
	_, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		return syncInfoError(fmt.Errorf("error getting VMExport from store: %w", err))
	}

	if exists {
		return nil
	}

	return &SyncInfo{
		event:  backupPreparingVMExportEvent,
		reason: backupPreparingVMExport,
	}
}

func (ctrl *VMBackupController) generateBackupTunnelCert(backup *backupv1.VirtualMachineBackup) (*triple.KeyPair, error) {
	caCert := ctrl.caCertManager.Current()
	caKeyPair := &triple.KeyPair{
		Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
		Cert: caCert.Leaf,
	}
	keyPair, err := triple.NewClientKeyPair(
		caKeyPair,
		fmt.Sprintf("kubevirt.io:system:client:%s", backup.UID),
		nil,
		getPullBackupRemainingTTL(backup).Duration,
	)
	return keyPair, err
}

func (ctrl *VMBackupController) handlePullModeTTLExpiry(backup *backupv1.VirtualMachineBackup, vmi *v1.VirtualMachineInstance) *SyncInfo {
	if hasVMIBackupStatus(vmi) && !vmi.Status.ChangedBlockTracking.BackupStatus.Completed {
		if syncInfo := ctrl.handleAbort(backup, vmi); syncInfo != nil {
			if syncInfo.reason != "" {
				syncInfo.reason = fmt.Sprintf("%s: %s", backupTTLExpiredMsg, syncInfo.reason)
			}
			return syncInfo
		}
		return nil
	}
	return nil
}

func (ctrl *VMBackupController) cleanupBackupExport(backup *backupv1.VirtualMachineBackup) *SyncInfo {
	objKey := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
	_, exists, err := ctrl.vmExportStore.GetByKey(objKey)
	if err != nil {
		return syncInfoError(fmt.Errorf("error getting VMExport from store during TTL expiry: %w", err))
	}
	if exists {
		if err := ctrl.client.VirtualMachineExport(backup.Namespace).Delete(context.Background(), backup.Name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return syncInfoError(fmt.Errorf("failed to delete VMExport during TTL expiry: %w", err))
		}
		return nil
	}
	return nil
}
