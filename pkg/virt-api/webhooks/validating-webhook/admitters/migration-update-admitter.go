/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitters

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

type MigrationUpdateAdmitter struct {
}

func ensureSelectorLabelSafe(newMigration *v1.VirtualMachineInstanceMigration, oldMigration *v1.VirtualMachineInstanceMigration) []metav1.StatusCause {
	if newMigration.Status.Phase != v1.MigrationSucceeded && newMigration.Status.Phase != v1.MigrationFailed && oldMigration.Labels != nil {
		oldLabel, oldExists := oldMigration.Labels[v1.MigrationSelectorLabel]
		if newMigration.Labels == nil {
			if oldExists {
				return []metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueNotSupported,
						Message: "selector label can't be removed from an in-flight migration",
					},
				}
			}
		} else {
			newLabel, newExists := newMigration.Labels[v1.MigrationSelectorLabel]
			if oldExists && (!newExists || newLabel != oldLabel) {
				return []metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueNotSupported,
						Message: "selector label can't be modified on an in-flight migration",
					},
				}
			}
		}
	}

	return []metav1.StatusCause{}
}

func (admitter *MigrationUpdateAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Get new migration from admission response
	newMigration, oldMigration, err := getAdmissionReviewMigration(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceMigrationGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	// Reject Migration update if spec changed
	if !equality.Semantic.DeepEqual(newMigration.Spec, oldMigration.Spec) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "update of Migration object's spec is restricted",
			},
		})
	}

	// Reject Migration update if selector label changed on an in-flight migration
	causes := ensureSelectorLabelSafe(newMigration, oldMigration)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return validating_webhooks.NewPassingAdmissionResponse()
}
