/*
 * This file is part of the CDI project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2019 Red Hat, Inc.
 *
 */

package webhooks

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/appscode/jsonpatch"

	"k8s.io/api/admission/v1beta1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	cdicorev1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/token"
)

// Admitter is the interface implemented by admission webhooks
type Admitter interface {
	Admit(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
}

type admissionHandler struct {
	a Admitter
}

// NewDataVolumeValidatingWebhook creates a new DataVolumeValidation webhook
func NewDataVolumeValidatingWebhook(client kubernetes.Interface) http.Handler {
	return newAdmissionHandler(&dataVolumeValidatingWebhook{client: client})
}

// NewDataVolumeMutatingWebhook creates a new DataVolumeMutation webhook
func NewDataVolumeMutatingWebhook(client kubernetes.Interface, key *rsa.PrivateKey) http.Handler {
	generator := newCloneTokenGenerator(key)
	return newAdmissionHandler(&dataVolumeMutatingWebhook{client: client, tokenGenerator: generator})
}

func newCloneTokenGenerator(key *rsa.PrivateKey) token.Generator {
	return token.NewGenerator(common.CloneTokenIssuer, key, 5*time.Minute)
}

func newAdmissionHandler(a Admitter) http.Handler {
	return &admissionHandler{a: a}
}

func (h *admissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	klog.V(2).Info(fmt.Sprintf("handling request: %s", body))

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := v1beta1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := v1beta1.AdmissionReview{}

	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Error(err)
		responseAdmissionReview.Response = toAdmissionResponseError(err)
	} else {
		if requestedAdmissionReview.Request == nil {
			responseAdmissionReview.Response = toAdmissionResponseError(fmt.Errorf("AdmissionReview.Request is nil"))
		} else {
			// pass to Admitter
			responseAdmissionReview.Response = h.a.Admit(requestedAdmissionReview)
		}
	}

	// Return the same UID
	if requestedAdmissionReview.Request != nil {
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	}

	klog.V(2).Info(fmt.Sprintf("sending response: %v", responseAdmissionReview.Response))

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error(err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
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

func allowedAdmissionResponse() *admissionv1beta1.AdmissionResponse {
	return &admissionv1beta1.AdmissionResponse{
		Allowed: true,
	}
}

func validateDataVolumeResource(ar v1beta1.AdmissionReview) error {
	resource := metav1.GroupVersionResource{
		Group:    cdicorev1alpha1.SchemeGroupVersion.Group,
		Version:  cdicorev1alpha1.SchemeGroupVersion.Version,
		Resource: "datavolumes",
	}
	if ar.Request.Resource != resource {
		klog.Errorf("resource is %s but request is: %s", resource, ar.Request.Resource)
		return fmt.Errorf("expect resource to be '%s'", resource.Resource)
	}
	return nil
}

func toPatchResponse(original, current interface{}) *admissionv1beta1.AdmissionResponse {
	patchType := admissionv1beta1.PatchTypeJSONPatch

	ob, err := json.Marshal(original)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	cb, err := json.Marshal(current)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	patches, err := jsonpatch.CreatePatch(ob, cb)
	if err != nil {
		toAdmissionResponseError(err)
	}

	pb, err := json.Marshal(patches)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	klog.V(3).Infof("sending patches\n%s", pb)

	return &admissionv1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     pb,
		PatchType: &patchType,
	}
}
