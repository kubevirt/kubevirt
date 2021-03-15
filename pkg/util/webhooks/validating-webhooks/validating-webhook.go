package validating_webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/util/webhooks"

	"kubevirt.io/client-go/log"
)

type Admitter interface {
	Admit(*admissionv1.AdmissionReview) *admissionv1.AdmissionResponse
}

type AlwaysPassAdmitter struct {
}

func NewPassingAdmissionResponse() *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{Allowed: true}
}

func (*AlwaysPassAdmitter) Admit(*admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	return NewPassingAdmissionResponse()
}

func NewAdmissionResponse(causes []v1.StatusCause) *admissionv1.AdmissionResponse {
	if len(causes) == 0 {
		return NewPassingAdmissionResponse()
	}

	globalMessage := ""
	for _, cause := range causes {
		if globalMessage == "" {
			globalMessage = cause.Message
		} else {
			globalMessage = fmt.Sprintf("%s, %s", globalMessage, cause.Message)
		}
	}

	return &admissionv1.AdmissionResponse{
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

func Serve(resp http.ResponseWriter, req *http.Request, admitter Admitter) {
	review, err := webhooks.GetAdmissionReview(req)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	response := admissionv1.AdmissionReview{
		TypeMeta: v1.TypeMeta{
			// match the request version to be
			// backwards compatible with v1beta1
			APIVersion: review.APIVersion,
			Kind:       "AdmissionReview",
		},
	}
	reviewResponse := admitter.Admit(review)
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}
	// reset the Object and OldObject, they are not needed in admitter response.
	review.Request.Object = runtime.RawExtension{}
	review.Request.OldObject = runtime.RawExtension{}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Log.Reason(err).Errorf("failed json encode webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := resp.Write(responseBytes); err != nil {
		log.Log.Reason(err).Errorf("failed to write webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
}
