package validating_webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/util/webhooks"

	"kubevirt.io/client-go/log"
)

type admitter interface {
	Admit(context.Context, *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse
}

func NewPassingAdmissionResponse() *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{Allowed: true}
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

func Serve(resp http.ResponseWriter, req *http.Request, admitter admitter) {
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

	ctx, cancel, err := getContextFromRequest(req)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(err.Error()))
		return
	}

	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	reviewResponse := admitter.Admit(ctx, review)
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

func getContextFromRequest(req *http.Request) (context.Context, context.CancelFunc, error) {
	ctx := req.Context()
	if _, timeoutDefined := ctx.Deadline(); timeoutDefined {
		return ctx, nil, nil
	}

	timeout := 10 * time.Second
	if timeoutStr := req.URL.Query().Get("timeout"); timeoutStr != "" {
		parsedTimeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			errTxt := fmt.Sprintf("failed to parse timeout duration of %q", timeoutStr)
			log.Log.Reason(err).Error(errTxt)
			return nil, nil, fmt.Errorf(errTxt)
		} else {
			timeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)

	return ctx, cancel, nil
}
