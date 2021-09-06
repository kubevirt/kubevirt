package admitters

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"kubevirt.io/api/flavor"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

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
		func(raw []byte) ([]flavorv1alpha1.VirtualMachineFlavorProfile, error) {
			flavorObj := flavorv1alpha1.VirtualMachineFlavor{}
			err := json.Unmarshal(raw, &flavorObj)
			if err != nil {
				return nil, err
			}
			return flavorObj.Profiles, nil
		},
	)
}

type ClusterFlavorAdmitter struct{}

var _ validating_webhooks.Admitter = &ClusterFlavorAdmitter{}

func (f *ClusterFlavorAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return admitFlavor(ar,
		metav1.GroupVersionResource{
			Group:    flavorv1alpha1.SchemeGroupVersion.Group,
			Version:  flavorv1alpha1.SchemeGroupVersion.Version,
			Resource: flavor.ClusterPluralResourceName,
		},
		func(raw []byte) ([]flavorv1alpha1.VirtualMachineFlavorProfile, error) {
			clusterFlavorObj := flavorv1alpha1.VirtualMachineClusterFlavor{}
			err := json.Unmarshal(raw, &clusterFlavorObj)
			if err != nil {
				return nil, err
			}
			return clusterFlavorObj.Profiles, nil
		},
	)
}

type extractProfilesFunc = func([]byte) ([]flavorv1alpha1.VirtualMachineFlavorProfile, error)

func admitFlavor(ar *admissionv1.AdmissionReview, expectedGvr metav1.GroupVersionResource, extractProfiles extractProfilesFunc) *admissionv1.AdmissionResponse {
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

	profiles, err := extractProfiles(ar.Request.Object.Raw)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := ValidateFlavorProfiles(k8sfield.NewPath("profiles"), profiles)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func ValidateFlavorProfiles(basePath *k8sfield.Path, profiles []flavorv1alpha1.VirtualMachineFlavorProfile) []metav1.StatusCause {
	if len(profiles) == 0 {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "A flavor must have at least one profile.",
			Field:   basePath.String(),
		}}
	}

	defaultCount := 0
	var allCauses []metav1.StatusCause
	for i := range profiles {
		profile := &profiles[i]
		if profile.Default {
			defaultCount += 1
		}
		allCauses = append(allCauses, validateFlavorProfile(basePath.Child(strconv.Itoa(i)), profile)...)
	}

	if defaultCount > 1 {
		allCauses = append(allCauses, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Flavor contains more than one default profile. Found: %d", defaultCount),
			Field:   basePath.String(),
		})
	}

	return allCauses
}

func validateFlavorProfile(basePath *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile) []metav1.StatusCause {
	return validateUnusedFieldsAreZero(basePath.Child("domainTemplate"), profile.DomainTemplate)
}

func validateUnusedFieldsAreZero(basePath *k8sfield.Path, domainTemplate *flavorv1alpha1.VirtualMachineFlavorDomainTemplateSpec) []metav1.StatusCause {

	if domainTemplate == nil {
		return nil
	}

	// While VirtualMachineFlavorDomainTemplateSpec.Devices is accepted we don't want to allow Disks, Interfaces etc.
	unusedDeviceFields := []string{
		"Disks",
		"Interfaces",
		"Inputs",
		"GPUs",
		"Filesystems",
		"HostDevices",
		"ClientPassthrough",
		"Sound",
	}
	var causes []metav1.StatusCause
	devices := reflect.ValueOf(domainTemplate.Devices)
	for _, field := range unusedDeviceFields {
		if !reflect.Indirect(devices).FieldByName(field).IsZero() {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("%s is not supported on domainTemplate.devices", field),
				Field:   basePath.Child(field).String(),
			})
		}
	}
	return causes
}
