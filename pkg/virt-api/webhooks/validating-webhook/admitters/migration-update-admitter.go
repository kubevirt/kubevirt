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

package admitters

import (
	"reflect"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

type MigrationUpdateAdmitter struct {
}

func (admitter *MigrationUpdateAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	// Get new migration from admission response
	newMigration, oldMigration, err := getAdmissionReviewMigration(ar)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	if resp := webhooks.ValidateSchema(v1.VirtualMachineInstanceMigrationGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	// Reject Migration update if spec changed
	if !reflect.DeepEqual(newMigration.Spec, oldMigration.Spec) {
		return webhooks.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "update of Migration object's spec is restricted",
			},
		})
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}
