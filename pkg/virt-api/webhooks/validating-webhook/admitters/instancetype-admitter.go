package admitters

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/instancetype"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

type InstancetypeAdmitter struct{}

var _ validating_webhooks.Admitter = &InstancetypeAdmitter{}

func (f *InstancetypeAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Only handle create and update
	if ar.Request.Operation != admissionv1.Create && ar.Request.Operation != admissionv1.Update {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	err := validateRequestResource(ar, instancetype.PluralResourceName)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Get instancetypeObj from AdmissionReview
	instancetypeObj, err := getInstanceTypeFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := validateDedicatedCPUPlacement(&instancetypeObj.Spec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateInstanceTypeCPU(&instancetypeObj.Spec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateInstanceTypeMemory(&instancetypeObj.Spec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func getInstanceTypeFromAdmissionReview(ar *admissionv1.AdmissionReview) (*instancetypev1alpha2.VirtualMachineInstancetype, error) {
	raw := ar.Request.Object.Raw
	newInstanceType := instancetypev1alpha2.VirtualMachineInstancetype{}

	if err := json.Unmarshal(raw, &newInstanceType); err != nil {
		return nil, err
	}

	return &newInstanceType, nil
}

type ClusterInstancetypeAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterInstancetypeAdmitter{}

func (f *ClusterInstancetypeAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Only handle create and update
	if ar.Request.Operation != admissionv1.Create && ar.Request.Operation != admissionv1.Update {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	err := validateRequestResource(ar, instancetype.ClusterPluralResourceName)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Get instancetypeObj from AdmissionReview
	instancetypeObj, err := getClusterInstanceTypeFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := validateDedicatedCPUPlacement(&instancetypeObj.Spec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateInstanceTypeCPU(&instancetypeObj.Spec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateInstanceTypeMemory(&instancetypeObj.Spec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func getClusterInstanceTypeFromAdmissionReview(ar *admissionv1.AdmissionReview) (*instancetypev1alpha2.VirtualMachineClusterInstancetype, error) {
	raw := ar.Request.Object.Raw
	newInstanceType := instancetypev1alpha2.VirtualMachineClusterInstancetype{}

	if err := json.Unmarshal(raw, &newInstanceType); err != nil {
		return nil, err
	}

	return &newInstanceType, nil
}

func validateRequestResource(ar *admissionv1.AdmissionReview, resourceType string) error {
	instanceTypeResource := metav1.GroupVersionResource{
		Group:    instancetypev1alpha2.SchemeGroupVersion.Group,
		Version:  instancetypev1alpha2.SchemeGroupVersion.Version,
		Resource: resourceType,
	}

	if ar.Request.Resource != instanceTypeResource {
		return fmt.Errorf("expected '%s' got '%s'", &instanceTypeResource, ar.Request.Resource)
	}

	return nil
}

func validateDedicatedCPUPlacement(instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec) []metav1.StatusCause {
	if instancetypeSpec == nil {
		return nil
	}

	if instancetypeSpec.CPU.DedicatedCPUPlacement {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "dedicatedCPUPlacement is not currently supported",
			Field:   k8sfield.NewPath("spec", "cpu", "dedictatedCPUPlacement").String(),
		}}
	}

	return nil
}

func validateInstanceTypeCPU(instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec) []metav1.StatusCause {
	if instancetypeSpec == nil {
		return nil
	}

	hasRequestedCPU := false
	if instancetypeSpec.Resources != nil {
		_, hasRequestedCPU = instancetypeSpec.Resources.Requests[k8sv1.ResourceCPU]
	}

	if (instancetypeSpec.CPU == nil || instancetypeSpec.CPU.Guest == 0) && !hasRequestedCPU {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "at least one of spec.resources.requests.CPU or spec.CPU should be defined",
			Field:   k8sfield.NewPath("spec", "instancetype").String(),
		}}
	}

	if instancetypeSpec.CPU == nil {
		return nil
	}

	if instancetypeSpec.CPU.Guest > 0 && hasRequestedCPU {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: "spec.resources.requests.CPU and spec.CPU can not be both defined",
			Field:   k8sfield.NewPath("spec", "instancetype").String(),
		}}
	}

	return nil
}

func validateInstanceTypeMemory(instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec) []metav1.StatusCause {
	if instancetypeSpec == nil {
		return nil
	}

	hasRequestedMemory := false
	if instancetypeSpec.Resources != nil {
		_, hasRequestedMemory = instancetypeSpec.Resources.Requests[k8sv1.ResourceMemory]
	}

	if (instancetypeSpec.Memory == nil || instancetypeSpec.Memory.Guest.IsZero()) && !hasRequestedMemory {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "at least one of spec.resources.requests.Memory or spec.Memory should be defined",
			Field:   k8sfield.NewPath("spec", "instancetype").String(),
		}}
	}

	if instancetypeSpec.Memory == nil {
		return nil
	}

	if !instancetypeSpec.Memory.Guest.IsZero() && hasRequestedMemory {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: "spec.resources.requests.Memory and spec.Memory can not be both defined",
			Field:   k8sfield.NewPath("spec", "instancetype").String(),
		}}
	}

	return nil
}
