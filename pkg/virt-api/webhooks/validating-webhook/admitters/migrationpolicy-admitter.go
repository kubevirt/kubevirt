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
	"math"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/migrations"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// MigrationPolicyAdmitter validates MigrationPolicy resources
type MigrationPolicyAdmitter struct {
	clusterConfig *virtconfig.ClusterConfig
}

// NewMigrationPolicyAdmitter creates a MigrationPolicyAdmitter
func NewMigrationPolicyAdmitter(clusterConfig *virtconfig.ClusterConfig) *MigrationPolicyAdmitter {
	return &MigrationPolicyAdmitter{
		clusterConfig: clusterConfig,
	}
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
	if spec.MaxDowntimeMs != nil {
		var oldMaxDowntimeMs *uint64
		if ar.Request.OldObject.Raw != nil {
			oldPolicy := &migrationsv1.MigrationPolicy{}
			if err := json.Unmarshal(ar.Request.OldObject.Raw, oldPolicy); err == nil {
				oldMaxDowntimeMs = oldPolicy.Spec.MaxDowntimeMs
			}
		}
		if !equality.Semantic.DeepEqual(oldMaxDowntimeMs, spec.MaxDowntimeMs) &&
			!admitter.clusterConfig.MigrationStallDetectionEnabled() &&
			!admitter.clusterConfig.MigrationDowntimeTuningEnabled() {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("maxDowntimeMs cannot be modified without enabling the %s or %s feature gate", featuregate.MigrationStallDetection, featuregate.MigrationDowntimeTuning),
				Field:   sourceField.Child("maxDowntimeMs").String(),
			})
		}
	}

	if spec.ExperimentalMigrationOptions != nil && spec.ExperimentalMigrationOptions.DowntimeTuning != nil {
		var oldDowntimeTuning any
		if ar.Request.OldObject.Raw != nil {
			oldPolicy := &migrationsv1.MigrationPolicy{}
			if err := json.Unmarshal(ar.Request.OldObject.Raw, oldPolicy); err == nil {
				if oldPolicy.Spec.ExperimentalMigrationOptions != nil {
					oldDowntimeTuning = oldPolicy.Spec.ExperimentalMigrationOptions.DowntimeTuning
				}
			}
		}
		if !equality.Semantic.DeepEqual(oldDowntimeTuning, spec.ExperimentalMigrationOptions.DowntimeTuning) &&
			!admitter.clusterConfig.MigrationDowntimeTuningEnabled() {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("experimental.downtimeTuning cannot be modified without enabling the %s feature gate", featuregate.MigrationDowntimeTuning),
				Field:   sourceField.Child("experimental", "downtimeTuning").String(),
			})
		}
	}

	if spec.ExperimentalMigrationOptions != nil && spec.ExperimentalMigrationOptions.StallDetector != nil {
		var oldStallDetector any
		if ar.Request.OldObject.Raw != nil {
			oldPolicy := &migrationsv1.MigrationPolicy{}
			if err := json.Unmarshal(ar.Request.OldObject.Raw, oldPolicy); err == nil {
				if oldPolicy.Spec.ExperimentalMigrationOptions != nil {
					oldStallDetector = oldPolicy.Spec.ExperimentalMigrationOptions.StallDetector
				}
			}
		}
		if !equality.Semantic.DeepEqual(oldStallDetector, spec.ExperimentalMigrationOptions.StallDetector) &&
			!admitter.clusterConfig.MigrationStallDetectionEnabled() {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("experimental.stallDetector cannot be modified without enabling the %s feature gate", featuregate.MigrationStallDetection),
				Field:   sourceField.Child("experimental", "stallDetector").String(),
			})
		}

		stallDetectorField := sourceField.Child("experimental", "stallDetector")
		sd := spec.ExperimentalMigrationOptions.StallDetector
		causes = append(causes, validateStallDetectorFactor(
			stallDetectorField.Child("ewmaAlpha"),
			sd.EwmaAlpha,
			0, 1,
			true,
		)...)
		causes = append(causes, validateStallDetectorFactor(
			stallDetectorField.Child("precopyPossibleFactor"),
			sd.PrecopyPossibleFactor,
			1, math.MaxFloat64,
			false,
		)...)
		causes = append(causes, validateStallDetectorFactor(
			stallDetectorField.Child("patienceWindowDecayFactor"),
			sd.PatienceWindowDecayFactor,
			0, 1,
			false,
		)...)
		causes = append(causes, validateStallDetectorFactor(
			stallDetectorField.Child("completionTimeoutFactor"),
			sd.CompletionTimeoutFactor,
			1, math.MaxFloat64,
			false,
		)...)
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

func validateStallDetectorFactor(field *k8sfield.Path, value *string, min, max float64, exclusiveMin bool) []metav1.StatusCause {
	if value == nil {
		return nil
	}

	factor, err := virtconfig.ParseFactor(*value, virtconfig.StallDetectorFactorPrecision)
	if err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
			Field:   field.String(),
		}}
	}

	if exclusiveMin {
		if factor <= 0 {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "must be greater than 0",
				Field:   field.String(),
			}}
		}
	} else if factor < min {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("must be greater than or equal to %g", min),
			Field:   field.String(),
		}}
	}

	if factor > max {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("must be less than or equal to %g", max),
			Field:   field.String(),
		}}
	}

	return nil
}
