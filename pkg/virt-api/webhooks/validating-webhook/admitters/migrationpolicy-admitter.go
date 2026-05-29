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
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	migrationutils "kubevirt.io/kubevirt/pkg/util/migrations"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
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
	if err := json.Unmarshal(ar.Request.Object.Raw, policy); err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var oldOptions *v1.VMIMConfigurationOptions
	if ar.Request.OldObject.Raw != nil {
		oldPolicy := &migrationsv1.MigrationPolicy{}
		if err := json.Unmarshal(ar.Request.OldObject.Raw, oldPolicy); err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
		oldOptions = policySpecToOptions(&oldPolicy.Spec)
	}

	newOptions := policySpecToOptions(&policy.Spec)
	sourceField := k8sfield.NewPath("spec")
	causes := migrationutils.ValidateMigrationConfigurationOptions(
		sourceField,
		oldOptions,
		newOptions,
		admitter.clusterConfig.GetConfig().DeveloperConfiguration,
	)

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{Allowed: true}
}

func policySpecToOptions(spec *migrationsv1.MigrationPolicySpec) *v1.VMIMConfigurationOptions {
	return &v1.VMIMConfigurationOptions{
		AllowAutoConverge:            spec.AllowAutoConverge,
		BandwidthPerMigration:        spec.BandwidthPerMigration,
		CompletionTimeoutPerGiB:      spec.CompletionTimeoutPerGiB,
		MaxDowntimeMs:                spec.MaxDowntimeMs,
		AllowPostCopy:                spec.AllowPostCopy,
		AllowWorkloadDisruption:      spec.AllowWorkloadDisruption,
		ExperimentalMigrationOptions: spec.ExperimentalMigrationOptions,
	}
}
