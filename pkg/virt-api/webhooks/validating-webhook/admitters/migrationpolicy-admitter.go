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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package admitters

import (
	"encoding/json"
	"fmt"

	"kubevirt.io/api/migrations"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

// MigrationPolicyAdmitter validates VirtualMachineSnapshots
type MigrationPolicyAdmitter struct {
	Client kubecli.KubevirtClient
}

// NewMigrationPolicyAdmitter creates a MigrationPolicyAdmitter
func NewMigrationPolicyAdmitter(client kubecli.KubevirtClient) *MigrationPolicyAdmitter {
	return &MigrationPolicyAdmitter{
		Client: client,
	}
}

// Admit validates an AdmissionReview
func (admitter *MigrationPolicyAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != migrationsv1.MigrationPolicyKind.Group ||
		ar.Request.Resource.Resource != migrations.ResourceMigrationPolicies {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	policy := &migrationsv1.MigrationPolicy{}
	err := json.Unmarshal(ar.Request.Object.Raw, policy)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	switch ar.Request.Operation {
	case admissionv1.Create:
		break

	case admissionv1.Update:
		break

	default:
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected operation %s", ar.Request.Operation))
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{
		Allowed: true,
	}
	return &reviewResponse
}
