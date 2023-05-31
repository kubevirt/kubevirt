package admitters

import (
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

type PreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &PreferenceAdmitter{}

func (f *PreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitPreference(ar.Request, instancetype.PluralPreferenceResourceName)
}

type ClusterPreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterPreferenceAdmitter{}

func (f *ClusterPreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitPreference(ar.Request, instancetype.ClusterPluralPreferenceResourceName)
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
		Group:   instancetypev1beta1.SchemeGroupVersion.Group,
		Kind:    resource,
		Version: request.Resource.Version,
	}
	if resp := webhookutils.ValidateSchema(gvk, request.Object.Raw); resp != nil {
		return resp
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
