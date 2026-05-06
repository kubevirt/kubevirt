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
 * Copyright The KubeVirt Authors.
 *
 */

package admitters

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/api/worker"
	workerv1 "kubevirt.io/api/worker/v1alpha1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type WorkerPoolAdmitter struct{}

func NewWorkerPoolAdmitter() *WorkerPoolAdmitter {
	return &WorkerPoolAdmitter{}
}

func (admitter *WorkerPoolAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != workerv1.SchemeGroupVersion.Group ||
		ar.Request.Resource.Resource != worker.ResourceWorkerPools {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	pool := &workerv1.WorkerPool{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pool); err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	if pool.Spec.VirtHandlerImage == "" && pool.Spec.VirtLauncherImage == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "pool must specify at least one of virtHandlerImage or virtLauncherImage",
			Field:   "spec",
		})
	}

	if len(pool.Spec.Selector.DeviceNames) == 0 && pool.Spec.Selector.VMLabels == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "pool selector must define at least one of deviceNames or vmLabels",
			Field:   "spec.selector",
		})
	}

	if len(pool.Spec.NodeSelector) == 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "pool must specify a non-empty nodeSelector",
			Field:   "spec.nodeSelector",
		})
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
