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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	"kubevirt.io/api/core"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
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

	vmBackup := &backupv1.VirtualMachineBackup{}
	err := json.Unmarshal(ar.Request.Object.Raw, vmBackup)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	switch ar.Request.Operation {
	case admissionv1.Create:
		causes, err = admitter.validateSingleBackup(vmBackup, ar.Request.Namespace, causes)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
		causes = validateSource(&vmBackup.Spec.Source, causes)
		causes = validateBackupMode(vmBackup, causes)

	case admissionv1.Update:
		prevObj := &backupv1.VirtualMachineBackup{}
		err = json.Unmarshal(ar.Request.OldObject.Raw, prevObj)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		if !equality.Semantic.DeepEqual(prevObj.Spec, vmBackup.Spec) {
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "spec is immutable after creation",
					Field:   k8sfield.NewPath("spec").String(),
				},
			}
		}
	default:
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected operation %s", ar.Request.Operation))
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{
		Allowed: true,
	}
	return &reviewResponse
}

func (admitter *VMBackupAdmitter) validateSingleBackup(vmBackup *backupv1.VirtualMachineBackup, namespace string, causes []metav1.StatusCause) ([]metav1.StatusCause, error) {
	objects, err := admitter.VMBackupInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return causes, err
	}

	sourceField := k8sfield.NewPath("spec", "source")
	for _, obj := range objects {
		vmbackup2 := obj.(*backupv1.VirtualMachineBackup)
		if vmbackup2.Name == vmBackup.Name {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachineBackup %q already exists", vmbackup2.Name),
				Field:   k8sfield.NewPath("metadata", "name").String(),
			})
			return causes, nil
		}
		// Reject if another backup is in progress for the same source
		if equality.Semantic.DeepEqual(vmbackup2.Spec.Source, vmBackup.Spec.Source) &&
			!backup.IsBackupDone(vmbackup2.Status) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("VirtualMachineBackup %q in progress for source", vmbackup2.Name),
				Field:   sourceField.String(),
			})
			return causes, nil
		}
	}
	return causes, nil
}

func validateSource(source *corev1.TypedLocalObjectReference, causes []metav1.StatusCause) []metav1.StatusCause {
	sourceField := k8sfield.NewPath("spec", "source")
	if source.APIGroup == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: "missing apiGroup",
			Field:   sourceField.Child("apiGroup").String(),
		})
		return causes
	}

	switch *source.APIGroup {
	case core.GroupName:
		switch source.Kind {
		case "VirtualMachine":
		default:
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "invalid kind",
				Field:   sourceField.Child("kind").String(),
			})
			return causes
		}
	default:
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "invalid apiGroup",
			Field:   sourceField.Child("apiGroup").String(),
		})
		return causes
	}
	if source.Name == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "name is required",
			Field:   sourceField.Child("name").String(),
		})
		return causes
	}
	return causes
}

func validateBackupMode(vmBackup *backupv1.VirtualMachineBackup, causes []metav1.StatusCause) []metav1.StatusCause {
	// Mode is optional, default to push mode
	if vmBackup.Spec.Mode == nil {
		vmBackup.Spec.Mode = pointer.P(backupv1.PushMode)
	}

	switch *vmBackup.Spec.Mode {
	case backupv1.PushMode:
		return validatePVCNameExists(vmBackup, causes)
	default:
		modeField := k8sfield.NewPath("spec", "mode")
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "invalid mode",
			Field:   modeField.String(),
		})
		return causes
	}
}

func validatePVCNameExists(vmBackup *backupv1.VirtualMachineBackup, causes []metav1.StatusCause) []metav1.StatusCause {
	if vmBackup.Spec.PvcName == nil || *vmBackup.Spec.PvcName == "" {
		pvcNameField := k8sfield.NewPath("spec", "pvcName")
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "pvcName must be provided in push mode",
			Field:   pvcNameField.String(),
		})
	}
	return causes
}
