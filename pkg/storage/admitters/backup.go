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

package admitters

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"

	"kubevirt.io/client-go/kubecli"

	backup "kubevirt.io/kubevirt/pkg/storage/cbt"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// VMBackupAdmitter validates VirtualMachineBackups
type VMBackupAdmitter struct {
	Config           *virtconfig.ClusterConfig
	Client           kubecli.KubevirtClient
	VMBackupInformer cache.SharedIndexInformer
}

// NewVMBackupAdmitter creates a VMBackupAdmitter
func NewVMBackupAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient, vmBackupInformer cache.SharedIndexInformer) *VMBackupAdmitter {
	return &VMBackupAdmitter{
		Config:           config,
		Client:           client,
		VMBackupInformer: vmBackupInformer,
	}
}

// Admit validates an AdmissionReview
func (admitter *VMBackupAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != backupv1.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != "virtualmachinebackups" {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.IncrementalBackupEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("IncrementalBackup feature gate not enabled"))
	}

	// Only need to validate on Create - spec immutability is now enforced by CEL
	if ar.Request.Operation != admissionv1.Create {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	vmBackup := &backupv1.VirtualMachineBackup{}
	if err := json.Unmarshal(ar.Request.Object.Raw, vmBackup); err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Validate that only one backup is in progress per source
	causes, err := admitter.validateSingleBackup(vmBackup, ar.Request.Namespace)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{Allowed: true}
}

func (admitter *VMBackupAdmitter) validateSingleBackup(vmBackup *backupv1.VirtualMachineBackup, namespace string) ([]metav1.StatusCause, error) {
	objects, err := admitter.VMBackupInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return nil, err
	}

	sourceField := k8sfield.NewPath("spec", "source")
	for _, obj := range objects {
		existingBackup := obj.(*backupv1.VirtualMachineBackup)
		if existingBackup.Name == vmBackup.Name {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachineBackup %q already exists", existingBackup.Name),
				Field:   k8sfield.NewPath("metadata", "name").String(),
			}}, nil
		}
		// Reject if another backup is in progress for the same source
		if equality.Semantic.DeepEqual(existingBackup.Spec.Source, vmBackup.Spec.Source) &&
			!backup.IsBackupDone(existingBackup.Status) {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachineBackup %q in progress for source", existingBackup.Name),
				Field:   sourceField.String(),
			}}, nil
		}
	}
	return nil, nil
}

// VMBackupTrackerAdmitter validates VirtualMachineBackupTrackers
type VMBackupTrackerAdmitter struct {
	Config *virtconfig.ClusterConfig
}

// NewVMBackupTrackerAdmitter creates a VMBackupTrackerAdmitter
func NewVMBackupTrackerAdmitter(config *virtconfig.ClusterConfig) *VMBackupTrackerAdmitter {
	return &VMBackupTrackerAdmitter{
		Config: config,
	}
}

// Admit validates an AdmissionReview for VirtualMachineBackupTracker
func (admitter *VMBackupTrackerAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != backupv1.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != "virtualmachinebackuptrackers" {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.IncrementalBackupEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("IncrementalBackup feature gate not enabled"))
	}

	return &admissionv1.AdmissionResponse{Allowed: true}
}
