package admitters

import (
	admissionv1 "k8s.io/api/admission/v1"

	"kubevirt.io/api/instancetype"

	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

type PreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &PreferenceAdmitter{}

func (f *PreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitInstancetypeOrPreference(ar.Request, instancetype.PluralPreferenceResourceName)
}

type ClusterPreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterPreferenceAdmitter{}

func (f *ClusterPreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitInstancetypeOrPreference(ar.Request, instancetype.ClusterPluralPreferenceResourceName)
}
