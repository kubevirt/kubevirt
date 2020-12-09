package validating_webhooks

import (
	"encoding/json"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/util/webhooks"

	"kubevirt.io/client-go/log"
)

type Admitter interface {
	Admit(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
}

type AlwaysPassAdmitter struct {
}

func NewPassingAdmissionResponse() *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{Allowed: true}
}

func (*AlwaysPassAdmitter) Admit(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	return NewPassingAdmissionResponse()
}

func Serve(resp http.ResponseWriter, req *http.Request, admitter Admitter) {
	response := v1beta1.AdmissionReview{}
	review, err := webhooks.GetAdmissionReview(req)

	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
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
