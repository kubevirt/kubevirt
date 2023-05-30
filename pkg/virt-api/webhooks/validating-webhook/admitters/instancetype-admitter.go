package admitters

import (
	"fmt"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

var supportedInstancetypeVersions = []string{
	instancetypev1alpha1.SchemeGroupVersion.Version,
	instancetypev1alpha2.SchemeGroupVersion.Version,
	instancetypev1beta1.SchemeGroupVersion.Version,
}

type InstancetypeAdmitter struct{}

var _ validating_webhooks.Admitter = &InstancetypeAdmitter{}

func (f *InstancetypeAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitInstancetypeOrPreference(ar.Request, instancetype.PluralResourceName)
}

type ClusterInstancetypeAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterInstancetypeAdmitter{}

func (f *ClusterInstancetypeAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitInstancetypeOrPreference(ar.Request, instancetype.ClusterPluralResourceName)
}

func admitInstancetypeOrPreference(request *admissionv1.AdmissionRequest, resource string) *admissionv1.AdmissionResponse {
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

func validateInstancetypeRequestResource(request metav1.GroupVersionResource, resource string) *admissionv1.AdmissionResponse {
	gvr := metav1.GroupVersionResource{
		Group:    instancetypev1beta1.SchemeGroupVersion.Group,
		Resource: resource,
	}

	for _, version := range supportedInstancetypeVersions {
		gvr.Version = version
		if request == gvr {
			return nil
		}
	}

	return webhookutils.ToAdmissionResponseError(
		fmt.Errorf("expected '%s.%s' with versions [%s], got '%s'",
			resource, gvr.Group, strings.Join(supportedInstancetypeVersions, ","), request),
	)
}
