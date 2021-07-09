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
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	cdiclone "kubevirt.io/containerized-data-importer/pkg/clone"
	"kubevirt.io/kubevirt/pkg/controller"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var validRunStrategies = []v1.VirtualMachineRunStrategy{v1.RunStrategyHalted, v1.RunStrategyManual, v1.RunStrategyAlways, v1.RunStrategyRerunOnFailure}

type CloneAuthFunc func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error)

type VMsAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
	cloneAuthFunc CloneAuthFunc
	virtClient    kubecli.KubevirtClient
}

type sarProxy struct {
	client kubecli.KubevirtClient
}

func (p *sarProxy) Create(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
	return p.client.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
}

func NewVMsAdmitter(clusterConfig *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VMsAdmitter {
	proxy := &sarProxy{client: client}

	return &VMsAdmitter{
		ClusterConfig: clusterConfig,
		virtClient:    client,
		cloneAuthFunc: func(pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
			return cdiclone.CanServiceAccountClonePVC(proxy, pvcNamespace, pvcName, saNamespace, saName)
		},
	}
}

func (admitter *VMsAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
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

	causes, err = admitter.validateVolumeRequests(&vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	} else if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateSnapshotStatus(ar.Request, &vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func (admitter *VMsAdmitter) AdmitStatus(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	vm, _, err := webhookutils.GetVMFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes, err := admitter.validateVolumeRequests(vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	} else if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = validateSnapshotStatus(ar.Request, vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func (admitter *VMsAdmitter) authorizeVirtualMachineSpec(ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine) ([]metav1.StatusCause, error) {
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
				// TODO: Assuming here that PVC name == DV name which might not be the case in the future
				if volume.VolumeSource.PersistentVolumeClaim != nil && volume.VolumeSource.PersistentVolumeClaim.ClaimName == dataVolume.Name {
					dataVolumeRefFound = true
					break
				} else if volume.VolumeSource.DataVolume != nil && volume.VolumeSource.DataVolume.Name == dataVolume.Name {
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

func (admitter *VMsAdmitter) validateVolumeRequests(vm *v1.VirtualMachine) ([]metav1.StatusCause, error) {
	if len(vm.Status.VolumeRequests) == 0 {
		return nil, nil
	}

	curVMAddRequestsMap := make(map[string]*v1.VirtualMachineVolumeRequest)
	curVMRemoveRequestsMap := make(map[string]*v1.VirtualMachineVolumeRequest)

	vmVolumeMap := make(map[string]v1.Volume)
	vmiVolumeMap := make(map[string]v1.Volume)

	vmi := &v1.VirtualMachineInstance{}
	vmiExists := false
	// get VMI if vm is active
	if vm.Status.Ready {
		retrievedVMI, err := admitter.virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		if errors.IsNotFound(err) {
			// ignore if VMI doesn't exist. This is best effort
		} else if err != nil {
			return nil, err
		} else if retrievedVMI.DeletionTimestamp == nil {
			// If VMI exists, lets simulate whether the new volume will be successful
			vmi = retrievedVMI
			vmiExists = true
		}
	}

	if vmiExists {
		for _, volume := range vmi.Spec.Volumes {
			vmiVolumeMap[volume.Name] = volume
		}
	}

	for _, volume := range vm.Spec.Template.Spec.Volumes {
		vmVolumeMap[volume.Name] = volume
	}

	newSpec := vm.Spec.Template.Spec.DeepCopy()
	for _, volumeRequest := range vm.Status.VolumeRequests {
		volumeRequest := volumeRequest
		name := ""
		if volumeRequest.AddVolumeOptions != nil && volumeRequest.RemoveVolumeOptions != nil {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "VolumeRequests require either addVolumeOptions or removeVolumeOptions to be set, not both",
				Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
			}}, nil
		} else if volumeRequest.AddVolumeOptions != nil {
			name = volumeRequest.AddVolumeOptions.Name

			_, ok := curVMAddRequestsMap[name]
			if ok {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("AddVolume request for [%s] aleady exists", name),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			}

			// Validate the disk is configured properly
			if volumeRequest.AddVolumeOptions.Disk == nil {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("AddVolume request for [%s] requires the disk field to be set.", name),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			} else if volumeRequest.AddVolumeOptions.Disk.DiskDevice.Disk == nil {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("AddVolume request for [%s] requires diskDevice of type 'disk' to be used.", name),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			} else if volumeRequest.AddVolumeOptions.Disk.DiskDevice.Disk.Bus != "scsi" {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("AddVolume request for [%s] requires disk bus to be 'scsi'. [%s] is not permitted", name, volumeRequest.AddVolumeOptions.Disk.DiskDevice.Disk.Bus),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			}

			newVolume := v1.Volume{
				Name: volumeRequest.AddVolumeOptions.Name,
			}
			if volumeRequest.AddVolumeOptions.VolumeSource.PersistentVolumeClaim != nil {
				newVolume.VolumeSource.PersistentVolumeClaim = volumeRequest.AddVolumeOptions.VolumeSource.PersistentVolumeClaim
			} else if volumeRequest.AddVolumeOptions.VolumeSource.DataVolume != nil {
				newVolume.VolumeSource.DataVolume = volumeRequest.AddVolumeOptions.VolumeSource.DataVolume
			}

			vmVolume, ok := vmVolumeMap[name]
			if ok && !reflect.DeepEqual(newVolume, vmVolume) {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("AddVolume request for [%s] conflicts with an existing volume of the same name on the vmi template.", name),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			}

			vmiVolume, ok := vmiVolumeMap[name]
			if ok && !reflect.DeepEqual(newVolume, vmiVolume) {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("AddVolume request for [%s] conflicts with an existing volume of the same name on currently running vmi", name),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			}

			curVMAddRequestsMap[name] = &volumeRequest
		} else if volumeRequest.RemoveVolumeOptions != nil {
			name = volumeRequest.RemoveVolumeOptions.Name

			_, ok := curVMRemoveRequestsMap[name]
			if ok {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("RemoveVolume request for [%s] aleady exists", name),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			}

			curVMRemoveRequestsMap[name] = &volumeRequest
		} else {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "VolumeRequests require one of either addVolumeOptions or removeVolumeOptions to be set",
				Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
			}}, nil
		}
		newSpec = controller.ApplyVolumeRequestOnVMISpec(newSpec, &volumeRequest)

		if vmiExists {
			vmi.Spec = *controller.ApplyVolumeRequestOnVMISpec(&vmi.Spec, &volumeRequest)
		}
	}

	// this simulates injecting the changes into the VMI template and validates it will work.
	causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec", "template", "spec"), newSpec, admitter.ClusterConfig)
	if len(causes) > 0 {
		return causes, nil
	}

	// This simulates injecting the changes directly into the vmi, if the vmi exists
	if vmiExists {
		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec", "template", "spec"), &vmi.Spec, admitter.ClusterConfig)
		if len(causes) > 0 {
			return causes, nil
		}
	}

	return nil, nil

}

func validateSnapshotStatus(ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine) []metav1.StatusCause {
	if ar.Operation != admissionv1.Update || vm.Status.SnapshotInProgress == nil {
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
