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

	virt "kubevirt.io/api/core"
	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/api/snapshot"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	pvc            = "PersistentVolumeClaim"
	vmSnapshotKind = "VirtualMachineSnapshot"
	vmKind         = "VirtualMachine"
)

// VMExportAdmitter validates VirtualMachineExports
type VMExportAdmitter struct {
	Config *virtconfig.ClusterConfig
}

// NewVMExportAdmitter creates a VMExportAdmitter
func NewVMExportAdmitter(config *virtconfig.ClusterConfig) *VMExportAdmitter {
	return &VMExportAdmitter{
		Config: config,
	}
}

// Admit validates an AdmissionReview
func (admitter *VMExportAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != exportv1.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != "virtualmachineexports" {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.VMExportEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("vm export feature gate not enabled"))
	}

	vmExport := &exportv1.VirtualMachineExport{}
	// TODO ideally use UniversalDeserializer here
	err := json.Unmarshal(ar.Request.Object.Raw, vmExport)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	switch ar.Request.Operation {
	case admissionv1.Create:
		sourceField := k8sfield.NewPath("spec", "source")

		switch vmExport.Spec.Source.Kind {
		case pvc:
			causes = append(causes, admitter.validatePVCName(sourceField.Child("name"), vmExport.Spec.Source.Name)...)
			causes = append(causes, admitter.validatePVCApiGroup(sourceField.Child("APIGroup"), vmExport.Spec.Source.APIGroup)...)
		case vmSnapshotKind:
			causes = append(causes, admitter.validateVMSnapshotName(sourceField.Child("name"), vmExport.Spec.Source.Name)...)
			causes = append(causes, admitter.validateVMSnapshotApiGroup(sourceField.Child("APIGroup"), vmExport.Spec.Source.APIGroup)...)
		case vmKind:
			causes = append(causes, admitter.validateVMName(sourceField.Child("name"), vmExport.Spec.Source.Name)...)
			causes = append(causes, admitter.validateVMApiGroup(sourceField.Child("APIGroup"), vmExport.Spec.Source.APIGroup)...)
		default:
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "invalid kind",
					Field:   sourceField.Child("kind").String(),
				},
			}
		}

	case admissionv1.Update:
		prevObj := &exportv1.VirtualMachineExport{}
		err = json.Unmarshal(ar.Request.OldObject.Raw, prevObj)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		if !equality.Semantic.DeepEqual(prevObj.Spec, vmExport.Spec) {
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "spec in immutable after creation",
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

func (admitter *VMExportAdmitter) validatePVCName(field *k8sfield.Path, name string) []metav1.StatusCause {
	if name == "" {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "PVC name must not be empty",
				Field:   field.String(),
			},
		}
	}

	return []metav1.StatusCause{}
}

func (admitter *VMExportAdmitter) validatePVCApiGroup(field *k8sfield.Path, apigroup *string) []metav1.StatusCause {
	if apigroup != nil && *apigroup != "" {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "PVC API group must be missing or blank",
				Field:   field.String(),
			},
		}
	}

	return []metav1.StatusCause{}
}

func (admitter *VMExportAdmitter) validateVMSnapshotName(field *k8sfield.Path, name string) []metav1.StatusCause {
	if name == "" {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "VMSnapshot name must not be empty",
				Field:   field.String(),
			},
		}
	}

	return []metav1.StatusCause{}
}

func (admitter *VMExportAdmitter) validateVMSnapshotApiGroup(field *k8sfield.Path, apigroup *string) []metav1.StatusCause {
	if apigroup == nil || *apigroup != snapshot.GroupName {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "VMSnapshot API group must be " + snapshot.GroupName,
				Field:   field.String(),
			},
		}
	}

	return []metav1.StatusCause{}
}

func (admitter *VMExportAdmitter) validateVMName(field *k8sfield.Path, name string) []metav1.StatusCause {
	if name == "" {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Virtual Machine name must not be empty",
				Field:   field.String(),
			},
		}
	}

	return []metav1.StatusCause{}
}

func (admitter *VMExportAdmitter) validateVMApiGroup(field *k8sfield.Path, apigroup *string) []metav1.StatusCause {
	if apigroup == nil || *apigroup != virt.GroupName {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "VM API group must be " + virt.GroupName,
				Field:   field.String(),
			},
		}
	}

	return []metav1.StatusCause{}
}
