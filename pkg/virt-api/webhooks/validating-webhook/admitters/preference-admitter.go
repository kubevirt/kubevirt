package admitters

import (
	"encoding/json"
	"fmt"

	"kubevirt.io/api/instancetype"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

type PreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &InstancetypeAdmitter{}

func (f *PreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitPreference(ar,
		metav1.GroupVersionResource{
			Group:    instancetypev1alpha2.SchemeGroupVersion.Group,
			Version:  instancetypev1alpha2.SchemeGroupVersion.Version,
			Resource: instancetype.PluralPreferenceResourceName,
		},
		func(raw []byte) (*instancetypev1alpha2.VirtualMachinePreference, error) {
			preferenceObj := &instancetypev1alpha2.VirtualMachinePreference{}
			err := json.Unmarshal(raw, &preferenceObj)
			if err != nil {
				return nil, err
			}
			return preferenceObj, nil
		},
	)
}

type extractPreferenceFunc = func([]byte) (*instancetypev1alpha2.VirtualMachinePreference, error)

func admitPreference(ar *admissionv1.AdmissionReview, expectedGvr metav1.GroupVersionResource, extractPreference extractPreferenceFunc) *admissionv1.AdmissionResponse {
	// Only handle create and update
	if ar.Request.Operation != admissionv1.Create && ar.Request.Operation != admissionv1.Update {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	if ar.Request.Resource != expectedGvr {
		return webhookutils.ToAdmissionResponseError(
			fmt.Errorf("expected '%s' got '%s'", expectedGvr, ar.Request.Resource),
		)
	}

	_, err := extractPreference(ar.Request.Object.Raw)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

type ClusterPreferenceAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterInstancetypeAdmitter{}

func (f *ClusterPreferenceAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitClusterPreference(ar,
		metav1.GroupVersionResource{
			Group:    instancetypev1alpha2.SchemeGroupVersion.Group,
			Version:  instancetypev1alpha2.SchemeGroupVersion.Version,
			Resource: instancetype.ClusterPluralPreferenceResourceName,
		},
		func(raw []byte) (*instancetypev1alpha2.VirtualMachineClusterPreference, error) {
			clusterPreferenceObj := &instancetypev1alpha2.VirtualMachineClusterPreference{}
			err := json.Unmarshal(raw, &clusterPreferenceObj)
			if err != nil {
				return nil, err
			}
			return clusterPreferenceObj, nil
		},
	)
}

type extractClusterPreferenceFunc = func([]byte) (*instancetypev1alpha2.VirtualMachineClusterPreference, error)

func admitClusterPreference(ar *admissionv1.AdmissionReview, expectedGvr metav1.GroupVersionResource, extractClusterPreference extractClusterPreferenceFunc) *admissionv1.AdmissionResponse {
	// Only handle create and update
	if ar.Request.Operation != admissionv1.Create && ar.Request.Operation != admissionv1.Update {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	if ar.Request.Resource != expectedGvr {
		return webhookutils.ToAdmissionResponseError(
			fmt.Errorf("expected '%s' got '%s'", expectedGvr, ar.Request.Resource),
		)
	}

	_, err := extractClusterPreference(ar.Request.Object.Raw)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
