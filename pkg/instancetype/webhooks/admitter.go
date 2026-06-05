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
 */

package webhooks

import (
	"context"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

const percentValueMustBeInRangeMessagePattern = "%s '%d': must be in range between 0 and 100."

type InstancetypeAdmitter struct{}

func (f *InstancetypeAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitInstancetype(ar.Request, instancetype.PluralResourceName)
}

func ValidateInstanceTypeSpec(field *k8sfield.Path, spec *instancetypev1beta1.VirtualMachineInstancetypeSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateMemoryOvercommitPercentSetting(field, spec)...)
	causes = append(causes, validateMemoryOvercommitPercentNoHugepages(field, spec)...)
	return causes
}

func validateMemoryOvercommitPercentSetting(
	field *k8sfield.Path,
	spec *instancetypev1beta1.VirtualMachineInstancetypeSpec,
) (causes []metav1.StatusCause) {
	if spec.Memory.OvercommitPercent < 0 || spec.Memory.OvercommitPercent > 100 {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(percentValueMustBeInRangeMessagePattern, field.Child("memory", "overcommitPercent").String(),
				spec.Memory.OvercommitPercent),
			Field: field.Child("memory", "overcommitPercent").String(),
		})
	}
	return causes
}

func validateMemoryOvercommitPercentNoHugepages(
	field *k8sfield.Path,
	spec *instancetypev1beta1.VirtualMachineInstancetypeSpec,
) (causes []metav1.StatusCause) {
	if spec.Memory.OvercommitPercent != 0 && spec.Memory.Hugepages != nil {
		causes = append(causes, metav1.StatusCause{
			Type: metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s and %s should not be requested together.",
				field.Child("memory", "overcommitPercent").String(),
				field.Child("memory", "hugepages").String()),
			Field: field.Child("memory", "overcommitPercent").String(),
		})
	}
	return causes
}

type ClusterInstancetypeAdmitter struct{}

func (f *ClusterInstancetypeAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitInstancetype(ar.Request, instancetype.ClusterPluralResourceName)
}

func admitInstancetype(request *admissionv1.AdmissionRequest, resource string) *admissionv1.AdmissionResponse {
	// Only handle create and update
	if request.Operation != admissionv1.Create && request.Operation != admissionv1.Update {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	gvk := schema.GroupVersionKind{
		Group:   instancetypev1beta1.SchemeGroupVersion.Group,
		Kind:    resource,
		Version: request.Resource.Version,
	}
	if resp := webhookutils.ValidateSchema(gvk, request.Object.Raw); resp != nil {
		return resp
	}

	instancetypeSpecObj, _, err := webhookutils.GetInstanceTypeSpecFromAdmissionRequest(request)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}
	causes := ValidateInstanceTypeSpec(k8sfield.NewPath("spec"), instancetypeSpecObj)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
