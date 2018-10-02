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

package webhooks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	v1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

var webhookInformers *Informers
var once sync.Once

type Informers struct {
	VMIPresetInformer       cache.SharedIndexInformer
	NamespaceLimitsInformer cache.SharedIndexInformer
}

func GetInformers() *Informers {
	once.Do(func() {
		webhookInformers = newInformers()
	})
	return webhookInformers
}

// SetInformers created for unittest usage only
func SetInformers(informers *Informers) {
	once.Do(func() {
		webhookInformers = informers
	})
}

func newInformers() *Informers {
	kubeClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	kubeInformerFactory := controller.NewKubeInformerFactory(kubeClient.RestClient(), kubeClient)
	kubeInformerFactory.VMI()
	return &Informers{
		VMIPresetInformer:       kubeInformerFactory.VirtualMachinePreset(),
		NamespaceLimitsInformer: kubeInformerFactory.LimitRanges(),
	}
}

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
		Result: &metav1.Status{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		},
	}
}
