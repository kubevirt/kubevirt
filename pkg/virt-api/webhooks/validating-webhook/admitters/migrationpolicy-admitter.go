/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitters

import (
	"context"
	"encoding/json"
	"fmt"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/migrations"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

// MigrationPolicyAdmitter validates VirtualMachineSnapshots
type MigrationPolicyAdmitter struct {
}

// NewMigrationPolicyAdmitter creates a MigrationPolicyAdmitter
func NewMigrationPolicyAdmitter() *MigrationPolicyAdmitter {
	return &MigrationPolicyAdmitter{}
}

// Admit validates an AdmissionReview
func (admitter *MigrationPolicyAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
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

	sourceField := k8sfield.NewPath("spec")

	spec := policy.Spec
	if spec.CompletionTimeoutPerGiB != nil && *spec.CompletionTimeoutPerGiB < 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "must not be negative",
			Field:   sourceField.Child("completionTimeoutPerGiB").String(),
		})
	}

	if spec.BandwidthPerMigration != nil {
		quantity, ok := spec.BandwidthPerMigration.AsInt64()
		if !ok {
			dec := spec.BandwidthPerMigration.AsDec()
			quantity = int64(dec.Sign())
		}

		if quantity < 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "must not be negative",
				Field:   sourceField.Child("bandwidthPerMigration").String(),
			})
		}
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{
		Allowed: true,
	}
	return &reviewResponse
}
