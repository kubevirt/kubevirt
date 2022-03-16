package admitters

import (
	"encoding/json"
	"fmt"

	"kubevirt.io/api/flavor"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

type FlavorAdmitter struct{}

var _ validating_webhooks.Admitter = &FlavorAdmitter{}

func (f *FlavorAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitFlavor(ar,
		metav1.GroupVersionResource{
			Group:    flavorv1alpha1.SchemeGroupVersion.Group,
			Version:  flavorv1alpha1.SchemeGroupVersion.Version,
			Resource: flavor.PluralResourceName,
		},
		func(raw []byte) (*flavorv1alpha1.VirtualMachineFlavor, error) {
			flavorObj := &flavorv1alpha1.VirtualMachineFlavor{}
			err := json.Unmarshal(raw, &flavorObj)
			if err != nil {
				return nil, err
			}
			return flavorObj, nil
		},
	)
}

type extractFlavorFunc = func([]byte) (*flavorv1alpha1.VirtualMachineFlavor, error)

func admitFlavor(ar *admissionv1.AdmissionReview, expectedGvr metav1.GroupVersionResource, extractFlavor extractFlavorFunc) *admissionv1.AdmissionResponse {
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

	_, err := extractFlavor(ar.Request.Object.Raw)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

type ClusterFlavorAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterFlavorAdmitter{}

func (f *ClusterFlavorAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitClusterFlavor(ar,
		metav1.GroupVersionResource{
			Group:    flavorv1alpha1.SchemeGroupVersion.Group,
			Version:  flavorv1alpha1.SchemeGroupVersion.Version,
			Resource: flavor.ClusterPluralResourceName,
		},
		func(raw []byte) (*flavorv1alpha1.VirtualMachineClusterFlavor, error) {
			clusterFlavorObj := &flavorv1alpha1.VirtualMachineClusterFlavor{}
			err := json.Unmarshal(raw, &clusterFlavorObj)
			if err != nil {
				return nil, err
			}
			return clusterFlavorObj, nil
		},
	)
}

type extractClusterFlavorFunc = func([]byte) (*flavorv1alpha1.VirtualMachineClusterFlavor, error)

func admitClusterFlavor(ar *admissionv1.AdmissionReview, expectedGvr metav1.GroupVersionResource, extractClusterFlavor extractClusterFlavorFunc) *admissionv1.AdmissionResponse {
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

	_, err := extractClusterFlavor(ar.Request.Object.Raw)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
