package webhooks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v12 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

// GetAdmissionReview
func GetAdmissionReview(r *http.Request) (*admissionv1.AdmissionReview, error) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return nil, fmt.Errorf("contentType=%s, expect application/json", contentType)
	}

	ar := &admissionv1.AdmissionReview{}
	err := json.Unmarshal(body, ar)
	return ar, err
}

// ToAdmissionResponseError
func ToAdmissionResponseError(err error) *admissionv1.AdmissionResponse {
	log.Log.Reason(err).Error("admission generic error")

	return &admissionv1.AdmissionResponse{
		Result: &v1.Status{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		},
	}
}

func ToAdmissionResponse(causes []v1.StatusCause) *admissionv1.AdmissionResponse {
	log.Log.Infof("rejected vmi admission")

	causeLen := len(causes)

	lenDiff := 0
	if causeLen > 10 {
		causeLen = 10
		lenDiff = len(causes) - 10
	}

	globalMessage := ""
	for _, cause := range causes[:causeLen] {
		if globalMessage == "" {
			globalMessage = cause.Message
		} else {
			globalMessage = fmt.Sprintf("%s, %s", globalMessage, cause.Message)
		}
	}

	if lenDiff > 0 {
		globalMessage = fmt.Sprintf("%s, and %v more validation errors", globalMessage, lenDiff)
	}

	return &admissionv1.AdmissionResponse{
		Result: &v1.Status{
			Message: globalMessage,
			Reason:  v1.StatusReasonInvalid,
			Code:    http.StatusUnprocessableEntity,
			Details: &v1.StatusDetails{
				Causes: causes[:causeLen],
			},
		},
	}
}

func ValidationErrorsToAdmissionResponse(errs []error) *admissionv1.AdmissionResponse {
	var causes []v1.StatusCause
	for _, e := range errs {
		causes = append(causes,
			v1.StatusCause{
				Message: e.Error(),
			},
		)
	}
	return ToAdmissionResponse(causes)
}

func ValidateSchema(gvk schema.GroupVersionKind, data []byte) *admissionv1.AdmissionResponse {
	in := map[string]interface{}{}
	err := json.Unmarshal(data, &in)
	if err != nil {
		return ToAdmissionResponseError(err)
	}
	errs := definitions.Validator.Validate(gvk, in)
	if len(errs) > 0 {
		return ValidationErrorsToAdmissionResponse(errs)
	}
	return nil
}

func ValidateRequestResource(request v1.GroupVersionResource, group string, resource string) bool {
	gvr := v1.GroupVersionResource{Group: group, Resource: resource}

	for _, version := range v12.ApiSupportedWebhookVersions {
		gvr.Version = version
		if gvr == request {
			return true
		}
	}

	return false
}

func ValidateStatus(data []byte) *admissionv1.AdmissionResponse {
	in := map[string]interface{}{}
	err := json.Unmarshal(data, &in)
	if err != nil {
		return ToAdmissionResponseError(err)
	}
	obj := unstructured.Unstructured{Object: in}
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" {
		return ValidationErrorsToAdmissionResponse([]error{fmt.Errorf("could not determine object kind")})
	}
	errs := definitions.Validator.ValidateStatus(gvk, in)
	if len(errs) > 0 {
		return ValidationErrorsToAdmissionResponse(errs)
	}
	return nil
}

func GetVMIFromAdmissionReview(ar *admissionv1.AdmissionReview) (new *v12.VirtualMachineInstance, old *v12.VirtualMachineInstance, err error) {

	if !ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineInstanceGroupVersionResource.Group, webhooks.VirtualMachineInstanceGroupVersionResource.Resource) {
		return nil, nil, fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineInstanceGroupVersionResource.Resource)
	}

	raw := ar.Request.Object.Raw
	newVMI := v12.VirtualMachineInstance{}

	err = json.Unmarshal(raw, &newVMI)
	if err != nil {
		return nil, nil, err
	}

	if ar.Request.Operation == admissionv1.Update {
		raw := ar.Request.OldObject.Raw
		oldVMI := v12.VirtualMachineInstance{}

		err = json.Unmarshal(raw, &oldVMI)
		if err != nil {
			return nil, nil, err
		}
		return &newVMI, &oldVMI, nil
	}

	return &newVMI, nil, nil
}

func GetVMFromAdmissionReview(ar *admissionv1.AdmissionReview) (new *v12.VirtualMachine, old *v12.VirtualMachine, err error) {

	if !ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		return nil, nil, fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineGroupVersionResource.Resource)
	}

	raw := ar.Request.Object.Raw
	newVM := v12.VirtualMachine{}

	err = json.Unmarshal(raw, &newVM)
	if err != nil {
		return nil, nil, err
	}

	if ar.Request.Operation == admissionv1.Update {
		raw := ar.Request.OldObject.Raw
		oldVM := v12.VirtualMachine{}

		err = json.Unmarshal(raw, &oldVM)
		if err != nil {
			return nil, nil, err
		}
		return &newVM, &oldVM, nil
	}

	return &newVM, nil, nil
}

func GetInstanceTypeSpecFromAdmissionRequest(request *admissionv1.AdmissionRequest) (new *instancetypev1beta1.VirtualMachineInstancetypeSpec, old *instancetypev1beta1.VirtualMachineInstancetypeSpec, err error) {

	raw := request.Object.Raw
	instancetypeObj := instancetypev1beta1.VirtualMachineInstancetype{}

	err = json.Unmarshal(raw, &instancetypeObj)
	if err != nil {
		return nil, nil, err
	}

	if request.Operation == admissionv1.Update {
		raw := request.OldObject.Raw
		oldInstancetypeObj := instancetypev1beta1.VirtualMachineInstancetype{}

		err = json.Unmarshal(raw, &oldInstancetypeObj)
		if err != nil {
			return nil, nil, err
		}
		return &instancetypeObj.Spec, &oldInstancetypeObj.Spec, nil
	}

	return &instancetypeObj.Spec, nil, nil
}
