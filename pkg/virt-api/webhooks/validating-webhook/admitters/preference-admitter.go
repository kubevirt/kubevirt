package admitters

import (
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypeapiv1beta1 "kubevirt.io/api/instancetype/v1beta1"

	instancetype "kubevirt.io/kubevirt/pkg/instancetype"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

type PreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &PreferenceAdmitter{}

func (f *PreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitPreference(ar.Request, instancetypeapi.PluralPreferenceResourceName)
}

type ClusterPreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterPreferenceAdmitter{}

func (f *ClusterPreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitPreference(ar.Request, instancetypeapi.ClusterPluralPreferenceResourceName)
}

func validatePreferenceSpec(field *k8sfield.Path, spec *instancetypeapiv1beta1.VirtualMachinePreferenceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateSpreadOptions(field, spec)...)
	return causes
}

const spreadAcrossCoresThreadsRatioErr = "only a ratio of 2 (1 core 2 threads) is allowed when spreading vCPUs over cores and threads"

func validateSpreadOptions(field *k8sfield.Path, spec *instancetypeapiv1beta1.VirtualMachinePreferenceSpec) []metav1.StatusCause {
	if spec.CPU == nil || spec.CPU.SpreadOptions == nil || spec.CPU.PreferredCPUTopology == nil || *spec.CPU.PreferredCPUTopology != instancetypeapiv1beta1.PreferSpread {
		return nil
	}
	ratio, across := instancetype.GetSpreadOptions(spec)
	if across == instancetypeapiv1beta1.SpreadAcrossCoresThreads && ratio != 2 {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: spreadAcrossCoresThreadsRatioErr,
			Field:   field.Child("cpu", "spreadOptions", "ratio").String(),
		}}
	}
	return nil
}

func admitPreference(request *admissionv1.AdmissionRequest, resource string) *admissionv1.AdmissionResponse {
	// Only handle create and update
	if request.Operation != admissionv1.Create && request.Operation != admissionv1.Update {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	if resp := validateInstancetypeRequestResource(request.Resource, resource); resp != nil {
		return resp
	}

	gvk := schema.GroupVersionKind{
		Group:   instancetypeapiv1beta1.SchemeGroupVersion.Group,
		Kind:    resource,
		Version: request.Resource.Version,
	}
	if resp := webhookutils.ValidateSchema(gvk, request.Object.Raw); resp != nil {
		return resp
	}

	preferenceSpec, _, err := webhookutils.GetPreferenceSpecFromAdmissionRequest(request)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}
	causes := validatePreferenceSpec(k8sfield.NewPath("spec"), preferenceSpec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
