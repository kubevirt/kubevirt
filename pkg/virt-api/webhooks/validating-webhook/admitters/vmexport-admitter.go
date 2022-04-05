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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package admitters

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	exportv1 "kubevirt.io/api/export/v1alpha1"
	"kubevirt.io/client-go/log"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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
func (admitter *VMExportAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	log.DefaultLogger().Info("**************** Calling VM export admitter")
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

		if vmExport.Spec.Source.APIGroup == nil {
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueNotFound,
					Message: "missing apiGroup",
					Field:   sourceField.Child("apiGroup").String(),
				},
			}
			break
		}

		switch *vmExport.Spec.Source.APIGroup {
		case "v1":
			switch vmExport.Spec.Source.Kind {
			case "PersistentVolumeClaim":
				causes, err = admitter.validatePVC(sourceField.Child("name"), ar.Request.Namespace, vmExport.Spec.Source.Name)
				if err != nil {
					return webhookutils.ToAdmissionResponseError(err)
				}
			default:
				causes = []metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "invalid kind",
						Field:   sourceField.Child("kind").String(),
					},
				}
			}
		default:
			causes = []metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "invalid apiGroup",
					Field:   sourceField.Child("apiGroup").String(),
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

func (admitter *VMExportAdmitter) validatePVC(field *k8sfield.Path, namespace, name string) ([]metav1.StatusCause, error) {
	if name == "" {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "PVC name must not be empty",
				Field:   field.String(),
			},
		}, nil
	}

	return []metav1.StatusCause{}, nil
}
