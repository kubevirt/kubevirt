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
	"strings"

	"kubevirt.io/kubevirt/pkg/network/link"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	clonebase "kubevirt.io/api/clone"
	clone "kubevirt.io/api/clone/v1beta1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	virtualMachineKind         = "VirtualMachine"
	virtualMachineSnapshotKind = "VirtualMachineSnapshot"
)

// VirtualMachineCloneAdmitter validates VirtualMachineClones
type VirtualMachineCloneAdmitter struct {
	Config *virtconfig.ClusterConfig
	Client kubecli.KubevirtClient
}

// NewVMCloneAdmitter creates a VM Clone Admitter
func NewVMCloneAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VirtualMachineCloneAdmitter {
	return &VirtualMachineCloneAdmitter{
		Config: config,
		Client: client,
	}
}

// Admit validates an AdmissionReview
func (admitter *VirtualMachineCloneAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != clone.VirtualMachineCloneKind.Group {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected group: %+v. Expected group: %+v", ar.Request.Resource.Group, clone.VirtualMachineCloneKind.Group))
	}
	if ar.Request.Resource.Resource != clonebase.ResourceVMClonePlural {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource: %+v. Expected resource: %+v", ar.Request.Resource.Resource, clonebase.ResourceVMClonePlural))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.SnapshotEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("snapshot feature gate is not enabled"))
	}

	vmClone := &clone.VirtualMachineClone{}
	err := json.Unmarshal(ar.Request.Object.Raw, vmClone)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	if newCauses := validateFilters(vmClone.Spec.AnnotationFilters, "spec.annotations"); newCauses != nil {
		causes = append(causes, newCauses...)
	}
	if newCauses := validateFilters(vmClone.Spec.LabelFilters, "spec.labels"); newCauses != nil {
		causes = append(causes, newCauses...)
	}
	if newCauses := validateFilters(vmClone.Spec.Template.AnnotationFilters, "spec.template.annotations"); newCauses != nil {
		causes = append(causes, newCauses...)
	}
	if newCauses := validateFilters(vmClone.Spec.Template.LabelFilters, "spec.template.labels"); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := validateSourceAndTargetKind(vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := validateSource(ctx, admitter.Client, vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := validateTarget(vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := validateNewMacAddresses(vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{
		Allowed: true,
	}
	return &reviewResponse
}

func validateFilters(filters []string, fieldName string) (causes []metav1.StatusCause) {
	if filters == nil {
		return nil
	}

	addCause := func(message string) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: message,
			Field:   fieldName,
		})
	}
	const negationChar = "!"
	const wildcardChar = "*"

	for _, filter := range filters {
		if len(filter) == 1 {
			if filter == negationChar {
				addCause("a negation character is not a valid filter")
			}
			continue
		}

		const errPattern = "%s filter %s is invalid: cannot contain a %s character (%s); FilterRules: %s"

		if filterWithoutFirstChar := filter[1:]; strings.Contains(filterWithoutFirstChar, negationChar) {
			addCause(fmt.Sprintf(errPattern, fieldName, filter, "negation", negationChar, "NegationChar can be only used at the beginning of the filter"))
		}

		if filterWithoutLastChar := filter[:len(filter)-1]; strings.Contains(filterWithoutLastChar, wildcardChar) {
			addCause(fmt.Sprintf(errPattern, fieldName, filter, "wildcard", wildcardChar, "WildcardChar can be only at the end of the filter"))
		}
	}

	return causes
}

func validateSourceAndTargetKind(vmClone *clone.VirtualMachineClone) []metav1.StatusCause {
	var causes []metav1.StatusCause = nil
	sourceField := k8sfield.NewPath("spec")

	supportedSourceTypes := []string{virtualMachineKind, virtualMachineSnapshotKind}
	supportedTargetTypes := []string{virtualMachineKind}

	if !doesSliceContainStr(supportedSourceTypes, vmClone.Spec.Source.Kind) {
		causes = []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Source kind is not supported",
			Field:   sourceField.Child("Source").Child("Kind").String(),
		}}
	}

	if vmClone.Spec.Target != nil && !doesSliceContainStr(supportedTargetTypes, vmClone.Spec.Target.Kind) {
		if causes == nil {
			causes = []metav1.StatusCause{}
		}
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Target kind is not supported",
			Field:   sourceField.Child("Target").Child("Kind").String(),
		})
	}

	return causes
}

func validateSource(ctx context.Context, client kubecli.KubevirtClient, vmClone *clone.VirtualMachineClone) []metav1.StatusCause {
	var causes []metav1.StatusCause = nil
	sourceField := k8sfield.NewPath("spec")
	source := vmClone.Spec.Source

	if source == nil {
		causes = []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Source cannot be nil",
			Field:   sourceField.Child("Source").String(),
		}}
		return causes
	}
	if source.APIGroup == nil || *source.APIGroup == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Source's APIGroup cannot be empty",
			Field:   sourceField.Child("Source").Child("APIGroup").String(),
		})
	}
	if source.Kind == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Source's Kind cannot be empty",
			Field:   sourceField.Child("Source").String(),
		})
	}
	if source.Name == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Source's name cannot be empty",
			Field:   sourceField.Child("Source").Child("Name").String(),
		})
	}
	return causes
}

func validateTarget(vmClone *clone.VirtualMachineClone) []metav1.StatusCause {
	var causes []metav1.StatusCause

	source := vmClone.Spec.Source
	target := vmClone.Spec.Target

	if source != nil &&
		target != nil &&
		source.Kind == virtualMachineKind &&
		target.Kind == virtualMachineKind &&
		target.Name == source.Name {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Target name cannot be equal to source name when both are VirtualMachines",
			Field:   k8sfield.NewPath("spec").Child("target").Child("name").String(),
		})
	}

	return causes
}

func validateNewMacAddresses(vmClone *clone.VirtualMachineClone) []metav1.StatusCause {
	var causes []metav1.StatusCause

	for ifaceName, ifaceMac := range vmClone.Spec.NewMacAddresses {
		if ifaceMac != "" {
			if err := link.ValidateMacAddress(ifaceMac); err != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("interface %s has malformed MAC address (%s).", ifaceName, ifaceMac),
					Field:   k8sfield.NewPath("spec").Child("newMacAddresses").Child(ifaceName).String(),
				})
			}
		}
	}

	return causes
}

func doesSliceContainStr(slice []string, str string) (isFound bool) {
	for _, curSliceStr := range slice {
		if curSliceStr == str {
			isFound = true
			break
		}
	}

	return isFound
}
