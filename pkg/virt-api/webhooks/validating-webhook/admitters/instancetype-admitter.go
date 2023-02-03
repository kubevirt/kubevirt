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

type InstancetypeAdmitter struct{}

var _ validating_webhooks.Admitter = &InstancetypeAdmitter{}

func (f *InstancetypeAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitInstancetype(ar,
		metav1.GroupVersionResource{
			Group:    instancetypev1alpha2.SchemeGroupVersion.Group,
			Version:  instancetypev1alpha2.SchemeGroupVersion.Version,
			Resource: instancetype.PluralResourceName,
		},
		func(raw []byte) (*instancetypev1alpha2.VirtualMachineInstancetype, error) {
			instancetypeObj := &instancetypev1alpha2.VirtualMachineInstancetype{}
			err := json.Unmarshal(raw, &instancetypeObj)
			if err != nil {
				return nil, err
			}
			return instancetypeObj, nil
		},
	)
}

type extractInstancetypeFunc = func([]byte) (*instancetypev1alpha2.VirtualMachineInstancetype, error)

func admitInstancetype(ar *admissionv1.AdmissionReview, expectedGvr metav1.GroupVersionResource, extractInstancetype extractInstancetypeFunc) *admissionv1.AdmissionResponse {
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

	_, err := extractInstancetype(ar.Request.Object.Raw)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

type ClusterInstancetypeAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterInstancetypeAdmitter{}

func (f *ClusterInstancetypeAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitClusterInstancetype(ar,
		metav1.GroupVersionResource{
			Group:    instancetypev1alpha2.SchemeGroupVersion.Group,
			Version:  instancetypev1alpha2.SchemeGroupVersion.Version,
			Resource: instancetype.ClusterPluralResourceName,
		},
		func(raw []byte) (*instancetypev1alpha2.VirtualMachineClusterInstancetype, error) {
			clusterInstancetypeObj := &instancetypev1alpha2.VirtualMachineClusterInstancetype{}
			err := json.Unmarshal(raw, &clusterInstancetypeObj)
			if err != nil {
				return nil, err
			}
			return clusterInstancetypeObj, nil
		},
	)
}

type extractClusterInstancetypeFunc = func([]byte) (*instancetypev1alpha2.VirtualMachineClusterInstancetype, error)

func admitClusterInstancetype(ar *admissionv1.AdmissionReview, expectedGvr metav1.GroupVersionResource, extractClusterInstancetype extractClusterInstancetypeFunc) *admissionv1.AdmissionResponse {
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

	_, err := extractClusterInstancetype(ar.Request.Object.Raw)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
