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

package mutators

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"

	"kubevirt.io/api/migrations"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type MigrationPolicyMutator struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (mutator *MigrationPolicyMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != migrationsv1.MigrationPolicyKind.Group ||
		ar.Request.Resource.Resource != migrations.ResourceMigrationPolicies {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource %+v", ar.Request.Resource))
	}

	raw := ar.Request.Object.Raw
	policy := migrationsv1.MigrationPolicy{}

	err := json.Unmarshal(raw, &policy)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var oldPolicy migrationsv1.MigrationPolicy
	if len(ar.Request.OldObject.Raw) > 0 {
		err = json.Unmarshal(ar.Request.OldObject.Raw, &oldPolicy)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
	}

	var warnings []string

	// If the feature gate is disabled, clear new usage of the experimental section
	if !mutator.ClusterConfig.AdvancedMigrationOptionsEnabled() {
		if policy.Spec.AdvancedMigrationOptions != nil {
			if len(ar.Request.OldObject.Raw) == 0 || oldPolicy.Spec.AdvancedMigrationOptions == nil {
				policy.Spec.AdvancedMigrationOptions = nil
				warnings = append(warnings, "spec.advanced was ignored because the AdvancedMigrationOptions feature gate is disabled")
			} else if !apiequality.Semantic.DeepEqual(policy.Spec.AdvancedMigrationOptions, oldPolicy.Spec.AdvancedMigrationOptions) {
				// preserve existing data
				policy.Spec.AdvancedMigrationOptions = oldPolicy.Spec.AdvancedMigrationOptions
				warnings = append(warnings, "changes to spec.advanced were ignored because the AdvancedMigrationOptions feature gate is disabled")
			}
		}
	}

	patchBytes, err := patch.New(
		patch.WithReplace("/spec", policy.Spec),
	).GeneratePayload()
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
		Warnings:  warnings,
	}
}
