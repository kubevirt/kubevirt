package validatingwebhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	datavolumev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
)

type admitFunc func(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

func toAdmissionReview(r *http.Request) (*v1beta1.AdmissionReview, error) {
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

func toRejectedAdmissionResponse(causes []metav1.StatusCause) *v1beta1.AdmissionResponse {
	globalMessage := ""
	for _, cause := range causes {
		globalMessage = fmt.Sprintf("%s %s", globalMessage, cause.Message)
	}

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: globalMessage,
			Code:    http.StatusUnprocessableEntity,
			Details: &metav1.StatusDetails{
				Causes: causes,
			},
		},
	}
}

func toAdmissionResponseError(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		},
	}
}

func validateSourceURL(sourceURL string) string {
	if sourceURL == "" {
		return "source URL is empty"
	}
	url, err := url.ParseRequestURI(sourceURL)
	if err != nil {
		return fmt.Sprintf("Invalid source URL: %s", sourceURL)
	}
	if url.Scheme != "http" && url.Scheme != "https" {
		return fmt.Sprintf("Invalid source URL scheme: %s", sourceURL)
	}
	return ""
}

func validateDataVolumeSpec(field *k8sfield.Path, spec *datavolumev1alpha1.DataVolumeSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	var url string
	var sourceType string
	// spec source field should not be empty
	if &spec.Source == nil || (spec.Source.HTTP == nil && spec.Source.S3 == nil && spec.Source.PVC == nil && spec.Source.Upload == nil &&
		spec.Source.Blank == nil) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Missing Data volume source"),
			Field:   field.Child("source").String(),
		})
		return causes
	}

	if (spec.Source.HTTP != nil && (spec.Source.S3 != nil || spec.Source.PVC != nil || spec.Source.Upload != nil)) ||
		(spec.Source.S3 != nil && (spec.Source.PVC != nil || spec.Source.Upload != nil)) ||
		(spec.Source.PVC != nil && spec.Source.Upload != nil) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Multiple Data volume sources"),
			Field:   field.Child("source").String(),
		})
		return causes
	}
	// if source types are HTTP or S3, check if URL is valid
	if spec.Source.HTTP != nil || spec.Source.S3 != nil {
		if spec.Source.HTTP != nil {
			url = spec.Source.HTTP.URL
			sourceType = field.Child("source", "HTTP", "url").String()
		} else if spec.Source.S3 != nil {
			url = spec.Source.S3.URL
			sourceType = field.Child("source", "S3", "url").String()
		}
		err := validateSourceURL(url)
		if err != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s %s", field.Child("source").String(), err),
				Field:   sourceType,
			})
			return causes
		}
	}

	if spec.Source.PVC != nil && (spec.Source.PVC.Namespace == "" || spec.Source.PVC.Name == "") {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s source PVC is not valid", field.Child("source", "PVC").String()),
			Field:   field.Child("source", "PVC").String(),
		})
		return causes
	}

	if spec.PVC == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Missing Data volume PVC"),
			Field:   field.Child("PVC").String(),
		})
		return causes
	}
	pvcSize := spec.PVC.Resources.Requests["storage"]
	if pvcSize.IsZero() || pvcSize.Value() < 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("PVC size can't be equal or less than zero"),
			Field:   field.Child("PVC", "resources", "requests", "size").String(),
		})
		return causes
	}

	return causes
}

func admitDVs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := metav1.GroupVersionResource{
		Group:    datavolumev1alpha1.SchemeGroupVersion.Group,
		Version:  datavolumev1alpha1.SchemeGroupVersion.Version,
		Resource: "datavolumes",
	}
	if ar.Request.Resource != resource {
		glog.Errorf("resource is %s but request is: %s", resource, ar.Request.Resource)
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	dv := datavolumev1alpha1.DataVolume{}

	err := json.Unmarshal(raw, &dv)

	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := validateDataVolumeSpec(k8sfield.NewPath("spec"), &dv.Spec)
	if len(causes) > 0 {
		glog.Infof("rejected DataVolume admission")
		return toRejectedAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func serve(resp http.ResponseWriter, req *http.Request, admit admitFunc) {

	response := v1beta1.AdmissionReview{}
	review, err := toAdmissionReview(req)

	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	reviewResponse := admit(review)
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}
	// reset the Object and OldObject, they are not needed in a response.
	review.Request.Object = runtime.RawExtension{}
	review.Request.OldObject = runtime.RawExtension{}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		glog.Errorf("failed json encode webhook response: %s", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := resp.Write(responseBytes); err != nil {
		glog.Errorf("failed to write webhook response: %s", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	resp.WriteHeader(http.StatusOK)
}

// ServeDVs ..
func ServeDVs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitDVs)
}
