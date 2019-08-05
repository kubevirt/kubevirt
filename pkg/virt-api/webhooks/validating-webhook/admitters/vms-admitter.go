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

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	cdiclone "kubevirt.io/containerized-data-importer/pkg/clone"
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
	if !webhooks.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineGroupVersionResource.Resource)
		return webhooks.ToAdmissionResponseError(err)
	}

	if resp := webhooks.ValidateSchema(v1.VirtualMachineGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	causes := ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &vm.Spec, admitter.ClusterConfig)
	if len(causes) > 0 {
		return webhooks.ToAdmissionResponse(causes)
	}

	causes, err = admitter.authorizeVirtualMachineSpec(ar.Request, &vm)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	if len(causes) > 0 {
		return webhooks.ToAdmissionResponse(causes)
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

func ValidateVirtualMachineSpec(field *k8sfield.Path, spec *v1.VirtualMachineSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   field.Child("template").String(),
		})
	}

	causes = append(causes, ValidateVirtualMachineInstanceMetadata(field.Child("template", "metadata"), &spec.Template.ObjectMeta, config)...)
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
