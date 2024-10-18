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
 * Copyright 2019 Red Hat, Inc.
 */

package mutators

import (
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/instancetype"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMsMutator struct {
	ClusterConfig       *virtconfig.ClusterConfig
	InstancetypeMethods instancetype.Methods
}

func (mutator *VMsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// If the VirtualMachine is being deleted return early and avoid racing any other in-flight resource deletions that might be happening
	if vm.DeletionTimestamp != nil {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Validate updates to the {Instancetype,Preference}Matchers
	if ar.Request.Operation == admissionv1.Update {
		newVM, oldVM, err := webhookutils.GetVMFromAdmissionReview(ar)
		if err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}
		if causes := validateInstancetypeMatcherUpdate(newVM.Spec.Instancetype, oldVM.Spec.Instancetype); len(causes) > 0 {
			return webhookutils.ToAdmissionResponse(causes)
		}
		if causes := validatePreferenceMatcherUpdate(newVM.Spec.Preference, oldVM.Spec.Preference); len(causes) > 0 {
			return webhookutils.ToAdmissionResponse(causes)
		}
	}

	// Validate the InstancetypeMatcher before proceeding, the schema check above isn't enough
	// as we need to ensure at least one of the optional Name or InferFromVolume attributes are present.
	if causes := validateInstancetypeMatcher(&vm); len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	if causes := validatePreferenceMatcher(&vm); len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	// Set VM defaults
	log.Log.Object(&vm).V(4).Info("Apply defaults")

	if err = mutator.InstancetypeMethods.InferDefaultInstancetype(&vm); err != nil {
		log.Log.Reason(err).Error("admission failed, unable to set default instancetype")
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			},
		}
	}

	if err = mutator.InstancetypeMethods.InferDefaultPreference(&vm); err != nil {
		log.Log.Reason(err).Error("admission failed, unable to set default preference")
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			},
		}
	}

	defaults.SetVirtualMachineDefaults(&vm, mutator.ClusterConfig, mutator.InstancetypeMethods)

	patchBytes, err := patch.New(
		patch.WithReplace("/spec", vm.Spec),
		patch.WithReplace("/metadata", vm.ObjectMeta),
	).GeneratePayload()

	if err != nil {
		log.Log.Reason(err).Error("admission failed to marshall patch to JSON")
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusInternalServerError,
			},
		}
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func validateInstancetypeMatcherUpdate(oldInstancetypeMatcher *v1.InstancetypeMatcher, newInstancetypeMatcher *v1.InstancetypeMatcher) []metav1.StatusCause {
	// Allow updates introducing or removing the matchers
	if oldInstancetypeMatcher == nil || newInstancetypeMatcher == nil {
		return nil
	}
	if err := validateMatcherUpdate(oldInstancetypeMatcher, newInstancetypeMatcher); err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
			Field:   k8sfield.NewPath("spec", "instancetype", "revisionName").String(),
		}}
	}
	return nil
}

func validatePreferenceMatcherUpdate(oldPreferenceMatcher *v1.PreferenceMatcher, newPreferenceMatcher *v1.PreferenceMatcher) []metav1.StatusCause {
	// Allow updates introducing or removing the matchers
	if oldPreferenceMatcher == nil || newPreferenceMatcher == nil {
		return nil
	}
	if err := validateMatcherUpdate(oldPreferenceMatcher, newPreferenceMatcher); err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
			Field:   k8sfield.NewPath("spec", "preference", "revisionName").String(),
		}}
	}
	return nil
}

func validateMatcherUpdate(oldMatcher, newMatcher v1.Matcher) error {
	// Do not check anything when the original matcher didn't have a revisionName as this is likely the VM Controller updating the matcher
	if oldMatcher.GetRevisionName() == "" {
		return nil
	}
	// If the matchers have changed ensure that the RevisionName is cleared when updating the Name
	if !equality.Semantic.DeepEqual(newMatcher, oldMatcher) {
		if oldMatcher.GetName() != newMatcher.GetName() && oldMatcher.GetRevisionName() == newMatcher.GetRevisionName() {
			return fmt.Errorf("the Matcher Name has been updated without updating the RevisionName")
		}
	}
	return nil
}

func validateInstancetypeMatcher(vm *v1.VirtualMachine) []metav1.StatusCause {
	if vm.Spec.Instancetype == nil {
		return nil
	}

	var causes []metav1.StatusCause
	if vm.Spec.Instancetype.Name == "" && vm.Spec.Instancetype.InferFromVolume == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("Either Name or InferFromVolume should be provided within the InstancetypeMatcher"),
			Field:   k8sfield.NewPath("spec", "instancetype").String(),
		})
	}
	if vm.Spec.Instancetype.InferFromVolume != "" {
		if vm.Spec.Instancetype.Name != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotFound,
				Message: fmt.Sprintf("Name should not be provided when InferFromVolume is used within the InstancetypeMatcher"),
				Field:   k8sfield.NewPath("spec", "instancetype", "name").String(),
			})
		}
		if vm.Spec.Instancetype.Kind != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotFound,
				Message: fmt.Sprintf("Kind should not be provided when InferFromVolume is used within the InstancetypeMatcher"),
				Field:   k8sfield.NewPath("spec", "instancetype", "kind").String(),
			})
		}
	}
	if vm.Spec.Instancetype.InferFromVolumeFailurePolicy != nil {
		failurePolicy := *vm.Spec.Instancetype.InferFromVolumeFailurePolicy
		if failurePolicy != v1.IgnoreInferFromVolumeFailure && failurePolicy != v1.RejectInferFromVolumeFailure {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid value '%s' for InferFromVolumeFailurePolicy", failurePolicy),
				Field:   k8sfield.NewPath("spec", "instancetype", "inferFromVolumeFailurePolicy").String(),
			})
		}
	}
	return causes
}

func validatePreferenceMatcher(vm *v1.VirtualMachine) []metav1.StatusCause {
	if vm.Spec.Preference == nil {
		return nil
	}

	var causes []metav1.StatusCause
	if vm.Spec.Preference.Name == "" && vm.Spec.Preference.InferFromVolume == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("Either Name or InferFromVolume should be provided within the PreferenceMatcher"),
			Field:   k8sfield.NewPath("spec", "preference").String(),
		})
	}
	if vm.Spec.Preference.InferFromVolume != "" {
		if vm.Spec.Preference.Name != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotFound,
				Message: fmt.Sprintf("Name should not be provided when InferFromVolume is used within the PreferenceMatcher"),
				Field:   k8sfield.NewPath("spec", "preference", "name").String(),
			})
		}
		if vm.Spec.Preference.Kind != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotFound,
				Message: fmt.Sprintf("Kind should not be provided when InferFromVolume is used within the PreferenceMatcher"),
				Field:   k8sfield.NewPath("spec", "preference", "kind").String(),
			})
		}
	}
	if vm.Spec.Preference.InferFromVolumeFailurePolicy != nil {
		failurePolicy := *vm.Spec.Preference.InferFromVolumeFailurePolicy
		if failurePolicy != v1.IgnoreInferFromVolumeFailure && failurePolicy != v1.RejectInferFromVolumeFailure {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid value '%s' for InferFromVolumeFailurePolicy", failurePolicy),
				Field:   k8sfield.NewPath("spec", "preference", "inferFromVolumeFailurePolicy").String(),
			})
		}
	}
	return causes
}
