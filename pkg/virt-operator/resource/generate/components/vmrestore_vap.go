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

package components

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	vmRestoreVAPName        = "vmrestore-policy.snapshot.kubevirt.io"
	vmRestoreVAPBindingName = "vmrestore-policy-binding"

	VMRestoreErrFeatureGateDisabled          = "Snapshot/Restore feature gate not enabled"
	VMRestoreErrMissingAPIGroup              = "spec.target.apiGroup is required"
	VMRestoreErrInvalidAPIGroup              = "spec.target.apiGroup must be 'kubevirt.io'"
	VMRestoreErrInvalidKind                  = "spec.target.kind must be 'VirtualMachine' when apiGroup is 'kubevirt.io'"
	VMRestoreErrSpecImmutable                = "spec is immutable after creation"
	VMRestoreErrVolumeOverrideVolumeName     = "volumeRestoreOverrides[].volumeName is required"
	VMRestoreErrVolumeOverrideAtLeastOneField = "volumeRestoreOverrides[] must specify at least one field: restoreName, annotations, or labels"
	VMRestoreErrInvalidVolumeRestorePolicy   = "volumeRestorePolicy must be one of: InPlace, RandomizeNames, PrefixTargetName"
	VMRestoreErrInvalidVolumeOwnershipPolicy = "volumeOwnershipPolicy must be one of: Vm, None"
	VMRestoreErrInvalidPatchPath             = "patches[] can only modify paths under /spec/, /metadata/labels/, or /metadata/annotations/"
)

func NewVMRestoreValidatingAdmissionPolicy(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicy {
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmRestoreVAPName,
			Labels: map[string]string{
				v1.AppLabel:       vmSnapshotAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: pointer.P(admissionregistrationv1.Fail),
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"snapshot.kubevirt.io"},
								APIVersions: []string{"v1beta1", "v1alpha1"},
								Resources:   []string{"virtualmachinerestores"},
							},
						},
					},
				},
			},
			ParamKind: &admissionregistrationv1.ParamKind{
				APIVersion: "kubevirt.io/v1",
				Kind:       "KubeVirt",
			},
			Validations: []admissionregistrationv1.Validation{
				// CREATE validations
				{
					Expression: "request.operation != 'CREATE' || params.spec.configuration.developerConfiguration.featureGates.exists(fg, fg == 'Snapshot')",
					Message:    VMRestoreErrFeatureGateDisabled,
					Reason:     pointer.P(metav1.StatusReasonForbidden),
				},
				{
					Expression: "request.operation != 'CREATE' || has(object.spec.target.apiGroup)",
					Message:    VMRestoreErrMissingAPIGroup,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.target.apiGroup) || object.spec.target.apiGroup == 'kubevirt.io'",
					Message:    VMRestoreErrInvalidAPIGroup,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.target.apiGroup) || object.spec.target.apiGroup != 'kubevirt.io' || object.spec.target.kind == 'VirtualMachine'",
					Message:    VMRestoreErrInvalidKind,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Volume override validations
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.volumeRestoreOverrides) || object.spec.volumeRestoreOverrides.all(override, has(override.volumeName) && override.volumeName != '')",
					Message:    VMRestoreErrVolumeOverrideVolumeName,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.volumeRestoreOverrides) || object.spec.volumeRestoreOverrides.all(override, has(override.restoreName) || has(override.annotations) || has(override.labels))",
					Message:    VMRestoreErrVolumeOverrideAtLeastOneField,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Volume restore policy enum validation
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.volumeRestorePolicy) || object.spec.volumeRestorePolicy in ['InPlace', 'RandomizeNames', 'PrefixTargetName']",
					Message:    VMRestoreErrInvalidVolumeRestorePolicy,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Volume ownership policy enum validation
				{
					Expression: "request.operation != 'CREATE' || !has(object.spec.volumeOwnershipPolicy) || object.spec.volumeOwnershipPolicy in ['Vm', 'None']",
					Message:    VMRestoreErrInvalidVolumeOwnershipPolicy,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// Patches validation - basic check that patches target allowed paths
				{
					Expression: `request.operation != 'CREATE' || !has(object.spec.patches) || object.spec.patches.all(patch, patch.contains('"path"') && (patch.contains('"/spec/') || patch.contains('"/metadata/labels/') || patch.contains('"/metadata/annotations/')))`,
					Message:    VMRestoreErrInvalidPatchPath,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
				// UPDATE validation
				{
					Expression: "request.operation != 'UPDATE' || object.spec == oldObject.spec",
					Message:    VMRestoreErrSpecImmutable,
					Reason:     pointer.P(metav1.StatusReasonInvalid),
				},
			},
		},
	}
}

func NewVMRestoreValidatingAdmissionPolicyBinding(namespace string) *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: vmRestoreVAPBindingName,
			Labels: map[string]string{
				v1.AppLabel:       vmSnapshotAppLabelValue,
				v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: vmRestoreVAPName,
			ParamRef: &admissionregistrationv1.ParamRef{
				Name:      "kubevirt",
				Namespace: namespace,
			},
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
		},
	}
}
