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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package admitters

import (
	"encoding/json"
	"fmt"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

type VMIPresetAdmitter struct {
}

func (admitter *VMIPresetAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if !webhooks.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineInstancePresetGroupVersionResource.Group, webhooks.VirtualMachineInstancePresetGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineInstancePresetGroupVersionResource.Resource)
		return webhooks.ToAdmissionResponseError(err)
	}

	if resp := webhooks.ValidateSchema(v1.VirtualMachineInstancePresetGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vmipreset := v1.VirtualMachineInstancePreset{}

	err := json.Unmarshal(raw, &vmipreset)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	causes := ValidateVMIPresetSpec(k8sfield.NewPath("spec"), &vmipreset.Spec)
	if len(causes) > 0 {
		return webhooks.ToAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ValidateVMIPresetSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstancePresetSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Domain == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing domain."),
			Field:   field.Child("domain").String(),
		})
	}

	causes = append(causes, validateDomainPresetSpec(field.Child("domain"), spec.Domain)...)
	return causes
}

func validateDomainPresetSpec(field *k8sfield.Path, spec *v1.DomainSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	causes = append(causes, validateDevices(field.Child("devices"), &spec.Devices)...)
	causes = append(causes, validateFirmware(field.Child("firmware"), spec.Firmware)...)
	return causes
}
