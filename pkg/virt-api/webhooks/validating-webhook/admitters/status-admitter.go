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
 */

package admitters

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"

	webhooks2 "kubevirt.io/kubevirt/pkg/virt-api/webhooks"

	"kubevirt.io/kubevirt/pkg/util/webhooks"
)

type StatusAdmitter struct {
	VmsAdmitter *VMsAdmitter
}

func (s *StatusAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resp := webhooks.ValidateStatus(ar.Request.Object.Raw); resp != nil {
		return resp
	}

	if webhooks.ValidateRequestResource(ar.Request.Resource, webhooks2.VirtualMachineGroupVersionResource.Group, webhooks2.VirtualMachineGroupVersionResource.Resource) {
		return s.VmsAdmitter.AdmitStatus(ctx, ar)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}
