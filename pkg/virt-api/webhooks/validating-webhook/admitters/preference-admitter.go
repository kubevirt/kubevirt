package admitters

import (
	"fmt"
	"slices"

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

	causes = append(causes, validatePreferredCPUTopology(field, spec)...)
	causes = append(causes, validateSpreadOptions(field, spec)...)
	return causes
}

const preferredCPUTopologyUnknownErrFmt = "unknown preferredCPUTopology %s"

func validatePreferredCPUTopology(field *k8sfield.Path, spec *instancetypeapiv1beta1.VirtualMachinePreferenceSpec) []metav1.StatusCause {
	if spec.CPU == nil || spec.CPU.PreferredCPUTopology == nil {
		return nil
	}
	topology := *spec.CPU.PreferredCPUTopology
	if !instancetype.IsPreferredTopologySupported(topology) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(preferredCPUTopologyUnknownErrFmt, topology),
			Field:   field.Child("cpu", "preferredCPUTopology").String(),
		}}
	}
	return nil
}

const (
	spreadAcrossCoresThreadsRatioErr = "only a ratio of 2 (1 core 2 threads) is allowed when spreading vCPUs over cores and threads"
	spreadAcrossUnsupportedErrFmt    = "across %s is not supported"
)

func validateSpreadOptions(field *k8sfield.Path, spec *instancetypeapiv1beta1.VirtualMachinePreferenceSpec) []metav1.StatusCause {
	if spec.CPU == nil || spec.CPU.SpreadOptions == nil || spec.CPU.PreferredCPUTopology == nil || *spec.CPU.PreferredCPUTopology != instancetypeapiv1beta1.Spread {
		return nil
	}
	ratio, across := instancetype.GetSpreadOptions(spec)

	supportedSpreadAcross := []instancetypeapiv1beta1.SpreadAcross{
		instancetypeapiv1beta1.SpreadAcrossCoresThreads,
		instancetypeapiv1beta1.SpreadAcrossSocketsCores,
		instancetypeapiv1beta1.SpreadAcrossSocketsCoresThreads,
	}
	if !slices.Contains(supportedSpreadAcross, across) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(spreadAcrossUnsupportedErrFmt, across),
			Field:   field.Child("cpu", "spreadOptions", "across").String(),
		}}
	}

	if across == instancetypeapiv1beta1.SpreadAcrossCoresThreads && ratio != 2 {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: spreadAcrossCoresThreadsRatioErr,
			Field:   field.Child("cpu", "spreadOptions", "ratio").String(),
		}}
	}
	return nil
}

const deprecatedPreferredCPUTopologyErrFmt = "PreferredCPUTopology %s is deprecated for removal in a future release, please use %s instead"

var deprecatedTopologies = map[instancetypeapiv1beta1.PreferredCPUTopology]instancetypeapiv1beta1.PreferredCPUTopology{
	instancetypeapiv1beta1.DeprecatedPreferSockets: instancetypeapiv1beta1.Sockets,
	instancetypeapiv1beta1.DeprecatedPreferCores:   instancetypeapiv1beta1.Cores,
	instancetypeapiv1beta1.DeprecatedPreferThreads: instancetypeapiv1beta1.Threads,
	instancetypeapiv1beta1.DeprecatedPreferSpread:  instancetypeapiv1beta1.Spread,
	instancetypeapiv1beta1.DeprecatedPreferAny:     instancetypeapiv1beta1.Any,
}

func checkForDeprecatedPreferredCPUTopology(spec *instancetypeapiv1beta1.VirtualMachinePreferenceSpec) []string {
	if spec.CPU == nil || spec.CPU.PreferredCPUTopology == nil {
		return nil
	}
	topology := *spec.CPU.PreferredCPUTopology
	if _, ok := deprecatedTopologies[topology]; !ok {
		return nil
	}
	return []string{fmt.Sprintf(deprecatedPreferredCPUTopologyErrFmt, topology, deprecatedTopologies[topology])}
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
		Allowed:  true,
		Warnings: checkForDeprecatedPreferredCPUTopology(preferenceSpec),
	}
}
