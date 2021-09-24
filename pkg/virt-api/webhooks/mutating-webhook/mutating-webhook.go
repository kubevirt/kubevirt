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

package mutating_webhook

import (
	"encoding/json"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/log"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type mutator interface {
	Mutate(*admissionv1.AdmissionReview) *admissionv1.AdmissionResponse
}

func serve(resp http.ResponseWriter, req *http.Request, m mutator) {
	review, err := webhookutils.GetAdmissionReview(req)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	response := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionv1.SchemeGroupVersion.String(),
			Kind:       "AdmissionReview",
		},
	}
	reviewResponse := m.Mutate(review)
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
}

func ServeVMs(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig) {
	serve(resp, req, &mutators.VMsMutator{ClusterConfig: clusterConfig})
}

func ServeVMIs(resp http.ResponseWriter, req *http.Request, clusterConfig *virtconfig.ClusterConfig, informers *webhooks.Informers) {
	serve(resp, req, &mutators.VMIsMutator{ClusterConfig: clusterConfig, VMIPresetInformer: informers.VMIPresetInformer, NamespaceLimitsInformer: informers.NamespaceLimitsInformer})
}

func ServeMigrationCreate(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, &mutators.MigrationCreateMutator{})
}
