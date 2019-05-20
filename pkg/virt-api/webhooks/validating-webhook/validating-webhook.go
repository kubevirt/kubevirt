/*
 * This file is part of the KubeVirt project
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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package validating_webhook

import (
	"encoding/json"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type admitter interface {
	Admit(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
}

func serve(resp http.ResponseWriter, req *http.Request, a admitter) {
	response := v1beta1.AdmissionReview{}
	review, err := webhooks.GetAdmissionReview(req)

	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	reviewResponse := a.Admit(review)
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}
	// reset the Object and OldObject, they are not needed in a response.
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
	resp.WriteHeader(http.StatusOK)
}

func ServeVMICreate(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	serve(resp, req, &admitters.VMICreateAdmitter{ClusterConfig: clusterConfig})
}

func ServeVMIUpdate(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, &admitters.VMIUpdateAdmitter{})
}

func ServeVMs(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	serve(resp, req, &admitters.VMsAdmitter{ClusterConfig: clusterConfig})
}

func ServeVMIRS(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	serve(resp, req, &admitters.VMIRSAdmitter{ClusterConfig: clusterConfig})
}

func ServeVMIPreset(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, &admitters.VMIPresetAdmitter{})
}

func ServeMigrationCreate(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	serve(resp, req, &admitters.MigrationCreateAdmitter{ClusterConfig: clusterConfig})
}

func ServeMigrationUpdate(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, &admitters.MigrationUpdateAdmitter{})
}
