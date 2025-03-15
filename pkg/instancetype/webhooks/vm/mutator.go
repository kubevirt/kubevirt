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
 * Copyright The KubeVirt Authors
 *
 */
package vm

import (
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/instancetype/infer"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type inferHandler interface {
	Instancetype(vm *virtv1.VirtualMachine) error
	Preference(vm *virtv1.VirtualMachine) error
}

type findPreferenceSpecHandler interface {
	FindPreference(vm *virtv1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error)
}

type mutator struct {
	inferHandler
	findPreferenceSpecHandler
}

func NewMutator(virtClient kubecli.KubevirtClient) *mutator {
	return &mutator{
		inferHandler: infer.New(virtClient),
		// TODO(lyarwood): Wire up informers for use here to speed up lookups
		findPreferenceSpecHandler: preferenceFind.NewSpecFinder(nil, nil, nil, virtClient),
	}
}

func (m *mutator) Mutate(vm, oldVM *virtv1.VirtualMachine, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if response := m.validateMatchers(vm, oldVM, ar); response != nil {
		return response
	}

	if response := m.inferMatchers(vm); response != nil {
		return response
	}

	return nil
}

func (m *mutator) validateMatchers(vm, oldVM *virtv1.VirtualMachine, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	// Validate updates to the {Instancetype,Preference}Matchers
	if ar.Request.Operation == admissionv1.Update {
		if causes := validateInstancetypeMatcherUpdate(vm.Spec.Instancetype, oldVM.Spec.Instancetype); len(causes) > 0 {
			return webhookutils.ToAdmissionResponse(causes)
		}
		if causes := validatePreferenceMatcherUpdate(vm.Spec.Preference, oldVM.Spec.Preference); len(causes) > 0 {
			return webhookutils.ToAdmissionResponse(causes)
		}
	}

	// Validate the InstancetypeMatcher before proceeding, the schema check above isn't enough
	// as we need to ensure at least one of the optional Name or InferFromVolume attributes are present.
	if causes := validateInstancetypeMatcher(vm); len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	if causes := validatePreferenceMatcher(vm); len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return nil
}

func validateInstancetypeMatcherUpdate(oldInstancetypeMatcher, newInstancetypeMatcher *virtv1.InstancetypeMatcher) []metav1.StatusCause {
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

func validatePreferenceMatcherUpdate(oldPreferenceMatcher, newPreferenceMatcher *virtv1.PreferenceMatcher) []metav1.StatusCause {
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

func validateMatcherUpdate(oldMatcher, newMatcher virtv1.Matcher) error {
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

// validateMatcher is a generic function to validate both instancetype and preference matchers
func validateMatcher(matcherType string, matcher virtv1.Matcher) []metav1.StatusCause {
	if matcher == nil {
		return nil
	}

	fieldPath := k8sfield.NewPath("spec", matcherType)
	var causes []metav1.StatusCause

	if matcher.GetName() == "" && matcher.GetInferFromVolume() == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("Either Name or InferFromVolume should be provided within the %s matcher", matcherType),
			Field:   fieldPath.String(),
		})
	}
	if matcher.GetInferFromVolume() != "" {
		if matcher.GetName() != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotFound,
				Message: fmt.Sprintf("Name should not be provided when InferFromVolume is used within the %s matcher", matcherType),
				Field:   fieldPath.Child("name").String(),
			})
		}
		if matcher.GetKind() != "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotFound,
				Message: fmt.Sprintf("Kind should not be provided when InferFromVolume is used within the %s matcher", matcherType),
				Field:   fieldPath.Child("kind").String(),
			})
		}
	}
	if matcher.GetInferFromVolumeFailurePolicy() != nil {
		failurePolicy := *matcher.GetInferFromVolumeFailurePolicy()
		if failurePolicy != virtv1.IgnoreInferFromVolumeFailure && failurePolicy != virtv1.RejectInferFromVolumeFailure {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid value '%s' for InferFromVolumeFailurePolicy", failurePolicy),
				Field:   fieldPath.Child("inferFromVolumeFailurePolicy").String(),
			})
		}
	}
	return causes
}

func validateInstancetypeMatcher(vm *virtv1.VirtualMachine) []metav1.StatusCause {
	return validateMatcher("instancetype", vm.Spec.Instancetype)
}

func validatePreferenceMatcher(vm *virtv1.VirtualMachine) []metav1.StatusCause {
	return validateMatcher("preference", vm.Spec.Preference)
}

func (m *mutator) inferMatchers(vm *virtv1.VirtualMachine) *admissionv1.AdmissionResponse {
	if err := m.inferHandler.Instancetype(vm); err != nil {
		log.Log.Reason(err).Error("admission failed, unable to set default instancetype")
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			},
		}
	}

	if err := m.inferHandler.Preference(vm); err != nil {
		log.Log.Reason(err).Error("admission failed, unable to set default preference")
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			},
		}
	}

	return nil
}
