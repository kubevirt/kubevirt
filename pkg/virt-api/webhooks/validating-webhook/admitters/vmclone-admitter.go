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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package admitters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"kubevirt.io/api/snapshot/v1alpha1"

	"kubevirt.io/kubevirt/pkg/storage/snapshot"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/clone"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// VirtualMachineCloneAdmitter validates VirtualMachineClones
type VirtualMachineCloneAdmitter struct {
	Config *virtconfig.ClusterConfig
	Client kubecli.KubevirtClient
}

// NewMigrationPolicyAdmitter creates a MigrationPolicyAdmitter
func NewVMCloneAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VirtualMachineCloneAdmitter {
	return &VirtualMachineCloneAdmitter{
		Config: config,
		Client: client,
	}
}

// Admit validates an AdmissionReview
func (admitter *VirtualMachineCloneAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != clonev1alpha1.VirtualMachineCloneKind.Group {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected group: %+v. Expected group: %+v", ar.Request.Resource.Group, clonev1alpha1.VirtualMachineCloneKind.Group))
	}
	if ar.Request.Resource.Resource != clone.ResourceVMClonePlural {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource: %+v. Expected resource: %+v", ar.Request.Resource.Resource, clone.ResourceVMClonePlural))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.SnapshotEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("snapshot feature gate is not enabled"))
	}

	vmClone := &clonev1alpha1.VirtualMachineClone{}
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

	if newCauses := validateSource(admitter.Client, vmClone); newCauses != nil {
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

	errPattern := "%s filter %s is invalid: cannot contain a %s character (%s); FilterRules: %s"
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

		if filterWithoutFirstChar := filter[1:]; strings.Contains(filterWithoutFirstChar, negationChar) {
			addCause(fmt.Sprintf(errPattern, fieldName, filter, "negation", negationChar, "NegationChar can be only used at the beginning of the filter"))
		}
		if filterWithoutLastChar := filter[:len(filter)-1]; strings.Contains(filterWithoutLastChar, wildcardChar) {
			addCause(fmt.Sprintf(errPattern, fieldName, filter, "wildcard", wildcardChar, "WildcardChar can be only at the end of the filter"))
		}
	}

	return causes
}

func validateSourceAndTargetKind(vmClone *clonev1alpha1.VirtualMachineClone) []metav1.StatusCause {
	var causes []metav1.StatusCause = nil
	sourceField := k8sfield.NewPath("spec")

	supportedSourceTypes := []string{"VirtualMachine", "VirtualMachineSnapshot"}
	supportedTargetTypes := []string{"VirtualMachine"}

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

func validateSource(client kubecli.KubevirtClient, vmClone *clonev1alpha1.VirtualMachineClone) []metav1.StatusCause {
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
	if source.Kind != "" && source.Name != "" {
		switch source.Kind {
		case "VirtualMachine":
			causes = append(causes, validateCloneSourceVM(client, source.Name, vmClone.Namespace, sourceField.Child("Source"))...)
		case "VirtualMachineSnapshot":
			causes = append(causes, validateCloneSourceSnapshot(client, source.Name, vmClone.Namespace, sourceField.Child("Source"))...)
		default:
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Source's Kind is invalid",
				Field:   sourceField.Child("Source").String(),
			})
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

func validateCloneSourceExists(clientGetErr error, sourceField *k8sfield.Path, kind, name, namespace string) []metav1.StatusCause {
	if errors.IsNotFound(clientGetErr) {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s %s does not exist in namespace %s", kind, name, namespace),
				Field:   sourceField.String(),
			},
		}
	} else if clientGetErr != nil {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("error occurred while trying to get source %s: %v", kind, clientGetErr),
				Field:   sourceField.String(),
			},
		}
	}

	return nil
}

func validateCloneSourceVM(client kubecli.KubevirtClient, name, namespace string, sourceField *k8sfield.Path) []metav1.StatusCause {
	vm, err := client.VirtualMachine(namespace).Get(context.Background(), name, &metav1.GetOptions{})
	causes := validateCloneSourceExists(err, sourceField, "VirtualMachine", name, namespace)

	if causes != nil {
		return causes
	}

	// currently, cloning leverages vm snapshot/restore to copy volumes
	// snapshot/restore requires that volumes support CSI snapshots
	// this limitation should be removed eventually
	// probably by leveraging CDI cloning
	causes = append(causes, validateCloneVolumeSnapshotSupportVM(vm, sourceField)...)

	return causes
}

func validateCloneSourceSnapshot(client kubecli.KubevirtClient, name, namespace string, sourceField *k8sfield.Path) []metav1.StatusCause {
	vmSnapshot, err := client.VirtualMachineSnapshot(namespace).Get(context.Background(), name, metav1.GetOptions{})
	causes := validateCloneSourceExists(err, sourceField, "VirtualMachineSnapshot", name, namespace)
	if causes != nil {
		return causes
	}

	snapshotContent, err := snapshot.GetSnapshotContents(vmSnapshot, client)
	if err != nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("cannot get snapshot contents: %v", err),
			Field:   sourceField.String(),
		})
	}

	causes = append(causes, validateCloneVolumeSnapshotSupportVMSnapshotContent(snapshotContent, sourceField)...)
	return causes
}

func validateCloneVolumeSnapshotSupportVM(vm *v1.VirtualMachine, sourceField *k8sfield.Path) []metav1.StatusCause {
	var result []metav1.StatusCause

	// should never happen, but don't want to NPE
	if vm.Spec.Template == nil {
		return result
	}

	for _, v := range vm.Spec.Template.Spec.Volumes {
		if v.PersistentVolumeClaim != nil || v.DataVolume != nil {
			found := false
			for _, vss := range vm.Status.VolumeSnapshotStatuses {
				if v.Name == vss.Name {
					if !vss.Enabled {
						result = append(result, metav1.StatusCause{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("Virtual Machine volume %s does not support snapshots", v.Name),
							Field:   sourceField.String(),
						})
					}
					found = true
					break
				}
			}
			if !found {
				result = append(result, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Virtual Machine volume %s snapshot support unknown", v.Name),
					Field:   sourceField.String(),
				})
			}
		}
	}

	return result
}

func validateCloneVolumeSnapshotSupportVMSnapshotContent(snapshotContents *v1alpha1.VirtualMachineSnapshotContent, sourceField *k8sfield.Path) []metav1.StatusCause {
	var result []metav1.StatusCause

	if snapshotContents.Spec.VirtualMachineSnapshotName == nil {
		return []metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("cannot get snapshot name from content %s", snapshotContents.Name),
				Field:   sourceField.String(),
			},
		}
	}

	snapshotName := *snapshotContents.Spec.VirtualMachineSnapshotName
	vm := snapshotContents.Spec.Source.VirtualMachine

	addVolumeIsNotBackedUpCause := func(volumeName string) {
		result = append(result, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("volume %s is not backed up in snapshot %s", volumeName, snapshotName),
			Field:   sourceField.String(),
		})
	}

	if vm.Spec.Template == nil {
		return nil
	}

	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.PersistentVolumeClaim == nil && volume.DataVolume == nil {
			continue
		}

		foundBackup := false
		for _, volumeBackup := range snapshotContents.Spec.VolumeBackups {
			if volume.Name == volumeBackup.VolumeName {
				foundBackup = true
				break
			}
		}

		if !foundBackup {
			addVolumeIsNotBackedUpCause(volume.Name)
		}
	}

	return result
}
