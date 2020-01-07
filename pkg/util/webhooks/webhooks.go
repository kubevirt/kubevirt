package webhooks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v12 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

// GetAdmissionReview
func GetAdmissionReview(r *http.Request) (*v1beta1.AdmissionReview, error) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return nil, fmt.Errorf("contentType=%s, expect application/json", contentType)
	}

	ar := &v1beta1.AdmissionReview{}
	err := json.Unmarshal(body, ar)
	return ar, err
}

// ToAdmissionResponseError
func ToAdmissionResponseError(err error) *v1beta1.AdmissionResponse {
	log.Log.Reason(err).Error("admission generic error")

	return &v1beta1.AdmissionResponse{
		Result: &v1.Status{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		},
	}
}

func ToAdmissionResponse(causes []v1.StatusCause) *v1beta1.AdmissionResponse {
	log.Log.Infof("rejected vmi admission")

	globalMessage := ""
	for _, cause := range causes {
		if globalMessage == "" {
			globalMessage = cause.Message
		} else {
			globalMessage = fmt.Sprintf("%s, %s", globalMessage, cause.Message)
		}
	}

	return &v1beta1.AdmissionResponse{
		Result: &v1.Status{
			Message: globalMessage,
			Reason:  v1.StatusReasonInvalid,
			Code:    http.StatusUnprocessableEntity,
			Details: &v1.StatusDetails{
				Causes: causes,
			},
		},
	}
}

func ValidationErrorsToAdmissionResponse(errs []error) *v1beta1.AdmissionResponse {
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

func ValidateSchema(gvk schema.GroupVersionKind, data []byte) *v1beta1.AdmissionResponse {
	in := map[string]interface{}{}
	err := json.Unmarshal(data, &in)
	if err != nil {
		return ToAdmissionResponseError(err)
	}
	errs := webhooks.Validator.Validate(gvk, in)
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
