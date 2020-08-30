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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package admitters

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/api/admission/v1beta1"
	k8svalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	cdiclone "kubevirt.io/containerized-data-importer/pkg/clone"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var validRunStrategies = []v1.VirtualMachineRunStrategy{v1.RunStrategyHalted, v1.RunStrategyManual, v1.RunStrategyAlways, v1.RunStrategyRerunOnFailure}

type CloneAuthFunc func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error)

type VMsAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
	cloneAuthFunc CloneAuthFunc
}

func NewVMsAdmitter(clusterConfig *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VMsAdmitter {
	return &VMsAdmitter{
		ClusterConfig: clusterConfig,
		cloneAuthFunc: func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
			return cdiclone.CanServiceAccountClonePVC(client, pvcNamespace, pvcName, saNamespace, saName)
		},
	}
}

func (admitter *VMsAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	accountName := ar.Request.UserInfo.Username
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &vm.Spec, admitter.ClusterConfig, accountName)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes, err = admitter.authorizeVirtualMachineSpec(ar.Request, &vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateStateChangeRequests(ar.Request, &vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateSnapshotStatus(ar.Request, &vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func (admitter *VMsAdmitter) AdmitStatus(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	vm, _, err := webhookutils.GetVMFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := validateStateChangeRequests(ar.Request, vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateSnapshotStatus(ar.Request, vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func (admitter *VMsAdmitter) authorizeVirtualMachineSpec(ar *v1beta1.AdmissionRequest, vm *v1.VirtualMachine) ([]metav1.StatusCause, error) {
	var causes []metav1.StatusCause

	for idx, dataVolume := range vm.Spec.DataVolumeTemplates {
		pvcSource := dataVolume.Spec.Source.PVC
		if pvcSource != nil {
			sourceNamespace := pvcSource.Namespace
			if sourceNamespace == "" {
				if vm.Namespace != "" {
					sourceNamespace = vm.Namespace
				} else {
					sourceNamespace = ar.Namespace
				}
			}

			if sourceNamespace == "" || pvcSource.Name == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotFound,
					Message: fmt.Sprintf("Clone source %s/%s invalid", sourceNamespace, pvcSource.Name),
					Field:   k8sfield.NewPath("spec", "dataVolumeTemplates").Index(idx).String(),
				})
			} else {
				targetNamespace := vm.Namespace
				if targetNamespace == "" {
					targetNamespace = ar.Namespace
				}

				serviceAccount := "default"
				for _, vol := range vm.Spec.Template.Spec.Volumes {
					if vol.ServiceAccount != nil {
						serviceAccount = vol.ServiceAccount.ServiceAccountName
					}
				}

				allowed, message, err := admitter.cloneAuthFunc(sourceNamespace, pvcSource.Name, targetNamespace, serviceAccount)
				if err != nil {
					return nil, err
				}

				if !allowed {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "Authorization failed, message is: " + message,
						Field:   k8sfield.NewPath("spec", "dataVolumeTemplates").Index(idx).String(),
					})
				}
			}
		}
	}

	return causes, nil
}

func ValidateVirtualMachineSpec(field *k8sfield.Path, spec *v1.VirtualMachineSpec, config *virtconfig.ClusterConfig, accountName string) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   field.Child("template").String(),
		})
	}

	causes = append(causes, ValidateVirtualMachineInstanceMetadata(field.Child("template", "metadata"), &spec.Template.ObjectMeta, config, accountName)...)
	causes = append(causes, ValidateVirtualMachineInstanceSpec(field.Child("template", "spec"), &spec.Template.Spec, config)...)

	if len(spec.DataVolumeTemplates) > 0 {

		for idx, dataVolume := range spec.DataVolumeTemplates {
			if dataVolume.Name == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Message: fmt.Sprintf("'name' field must not be empty for DataVolumeTemplate entry %s.", field.Child("dataVolumeTemplate").Index(idx).String()),
					Field:   field.Child("dataVolumeTemplate").Index(idx).Child("name").String(),
				})
			}

			dataVolumeRefFound := false
			for _, volume := range spec.Template.Spec.Volumes {
				if volume.VolumeSource.DataVolume == nil {
					continue
				} else if volume.VolumeSource.DataVolume.Name == dataVolume.Name {
					dataVolumeRefFound = true
					break
				}
			}

			if !dataVolumeRefFound {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Message: fmt.Sprintf("DataVolumeTemplate entry %s must be referenced in the VMI template's 'volumes' list", field.Child("dataVolumeTemplate").Index(idx).String()),
					Field:   field.Child("dataVolumeTemplate").Index(idx).String(),
				})
			}
		}
	}

	// Validate RunStrategy
	if spec.Running != nil && spec.RunStrategy != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Running and RunStrategy are mutually exclusive"),
			Field:   field.Child("running").String(),
		})
	}

	if spec.Running == nil && spec.RunStrategy == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("One of Running or RunStrategy must be specified"),
			Field:   field.Child("running").String(),
		})
	}

	if spec.RunStrategy != nil {
		validRunStrategy := false
		for _, strategy := range validRunStrategies {
			if *spec.RunStrategy == strategy {
				validRunStrategy = true
				break
			}
		}
		if validRunStrategy == false {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid RunStrategy (%s)", *spec.RunStrategy),
				Field:   field.Child("runStrategy").String(),
			})
		}
	}

	return causes
}

func validateStateChangeRequests(ar *v1beta1.AdmissionRequest, vm *v1.VirtualMachine) []metav1.StatusCause {
	// Only rename request is validated
	renameRequest := getRenameRequest(vm)

	// Prevent creation of VM with rename request
	if ar.Operation == v1beta1.Create {
		if renameRequest != nil {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Creating a VM with a rename request is not allowed",
				Field:   k8sfield.NewPath("Status", "stateChangeRequests").String(),
			}}
		}
	} else if ar.Operation == v1beta1.Update {
		existingVM := &v1.VirtualMachine{}
		err := json.Unmarshal(ar.OldObject.Raw, existingVM)

		if err != nil {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeUnexpectedServerResponse,
				Message: "Could not fetch old VM",
			}}
		}

		if getRenameRequest(existingVM) != nil {
			allowed := webhooks.GetAllowedServiceAccounts()
			if _, ok := allowed[ar.UserInfo.Username]; ok {
				if !reflect.DeepEqual(existingVM.Spec, vm.Spec) {
					return []metav1.StatusCause{{
						Type:    metav1.CauseTypeFieldValueNotSupported,
						Message: fmt.Sprint("Cannot update VM spec until rename process completes"),
						Field:   k8sfield.NewPath("spec").String(),
					}}

				}
			} else {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: fmt.Sprint("Modifying a VM during a rename process is restricted to Kubevirt core components"),
				}}
			}
		}
	}

	if renameRequest == nil {
		return nil
	}

	// Reject rename requests if the VM is running
	if vm.Status.Created {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Cannot rename a running VM",
			Field:   k8sfield.NewPath("spec", "running").String(),
		}}
	}

	newName, hasNewName := renameRequest.Data["newName"]
	if !hasNewName {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "New name not provided",
			Field:   k8sfield.NewPath("status", "stateChangeRequests").String(),
		}}
	}

	nameErrs := k8svalidation.NameIsDNSSubdomain(newName, false)
	if len(nameErrs) > 0 {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("The VM's new name is not valid: %s", strings.Join(nameErrs, "; ")),
			Field:   k8sfield.NewPath("status", "stateChangeRequests").String(),
		}}
	}

	return nil
}

func validateSnapshotStatus(ar *v1beta1.AdmissionRequest, vm *v1.VirtualMachine) []metav1.StatusCause {
	if ar.Operation != v1beta1.Update || vm.Status.SnapshotInProgress == nil {
		return nil
	}

	oldVM := &v1.VirtualMachine{}
	if err := json.Unmarshal(ar.OldObject.Raw, oldVM); err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeUnexpectedServerResponse,
			Message: "Could not fetch old VM",
		}}
	}

	if !reflect.DeepEqual(oldVM.Spec, vm.Spec) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Cannot update VM spec until snapshot %q completes", *vm.Status.SnapshotInProgress),
			Field:   k8sfield.NewPath("spec").String(),
		}}
	}

	return nil
}

func getRenameRequest(vm *v1.VirtualMachine) *v1.VirtualMachineStateChangeRequest {
	for _, req := range vm.Status.StateChangeRequests {
		if req.Action == v1.RenameRequest {
			return &req
		}
	}
	return nil
}
