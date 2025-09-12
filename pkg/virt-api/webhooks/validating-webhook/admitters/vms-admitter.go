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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	instancetypeWebhooks "kubevirt.io/kubevirt/pkg/instancetype/webhooks/vm"
	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	netadmitter "kubevirt.io/kubevirt/pkg/network/admitter"
	storageadmitters "kubevirt.io/kubevirt/pkg/storage/admitters"
	migrationutil "kubevirt.io/kubevirt/pkg/util/migrations"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var validRunStrategies = []v1.VirtualMachineRunStrategy{v1.RunStrategyHalted, v1.RunStrategyManual, v1.RunStrategyAlways, v1.RunStrategyRerunOnFailure, v1.RunStrategyOnce}

type instancetypeVMsAdmitter interface {
	ApplyToVM(vm *v1.VirtualMachine) (
		*instancetypev1beta1.VirtualMachineInstancetypeSpec,
		*instancetypev1beta1.VirtualMachinePreferenceSpec,
		[]metav1.StatusCause,
	)
	Check(*instancetypev1beta1.VirtualMachineInstancetypeSpec,
		*instancetypev1beta1.VirtualMachinePreferenceSpec,
		*v1.VirtualMachineInstanceSpec,
	) (conflict.Conflicts, error)
}

type VMsAdmitter struct {
	VirtClient              kubecli.KubevirtClient
	DataSourceInformer      cache.SharedIndexInformer
	NamespaceInformer       cache.SharedIndexInformer
	InstancetypeAdmitter    instancetypeVMsAdmitter
	ClusterConfig           *virtconfig.ClusterConfig
	KubeVirtServiceAccounts map[string]struct{}
}

func NewVMsAdmitter(clusterConfig *virtconfig.ClusterConfig, virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, informers *webhooks.Informers, kubeVirtServiceAccounts map[string]struct{}) *VMsAdmitter {
	return &VMsAdmitter{
		VirtClient:              virtClient,
		DataSourceInformer:      informers.DataSourceInformer,
		NamespaceInformer:       informers.NamespaceInformer,
		InstancetypeAdmitter:    instancetypeWebhooks.NewAdmitter(virtClient, k8sClient),
		ClusterConfig:           clusterConfig,
		KubeVirtServiceAccounts: kubeVirtServiceAccounts,
	}
}

func (admitter *VMsAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
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

	// We apply any referenced instancetype and preferences early here to the VirtualMachine in order to
	// validate the resulting VirtualMachineInstanceSpec below. As we don't want to persist these changes
	// we pass a copy of the original VirtualMachine here and to the validation call below.
	vmCopy := vm.DeepCopy()
	instancetypeSpec, preferenceSpec, causes := admitter.InstancetypeAdmitter.ApplyToVM(vmCopy)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	// Set VirtualMachine defaults on the copy before validating
	if err = defaults.SetDefaultVirtualMachineInstanceSpec(admitter.ClusterConfig, &vmCopy.Spec.Template.Spec); err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// With the defaults now set we can check that the VM meets the requirements of any provided preference
	if conflicts, err := admitter.InstancetypeAdmitter.Check(instancetypeSpec, preferenceSpec, &vmCopy.Spec.Template.Spec); err != nil {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("failure checking preference requirements: %v", err),
			Field:   conflicts.String(),
		}})
	}

	if ar.Request.Operation == admissionv1.Create {
		clusterCfg := admitter.ClusterConfig.GetConfig()
		if devCfg := clusterCfg.DeveloperConfiguration; devCfg != nil {
			if causes = featuregate.ValidateFeatureGates(devCfg.FeatureGates, &vm.Spec.Template.Spec); len(causes) > 0 {
				return webhookutils.ToAdmissionResponse(causes)
			}
		}

		netValidator := netadmitter.NewValidator(k8sfield.NewPath("spec"), &vmCopy.Spec.Template.Spec, admitter.ClusterConfig)
		if causes = netValidator.ValidateCreation(); len(causes) > 0 {
			return webhookutils.ToAdmissionResponse(causes)
		}
	}

	_, isKubeVirtServiceAccount := admitter.KubeVirtServiceAccounts[ar.Request.UserInfo.Username]
	causes = ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &vmCopy.Spec, admitter.ClusterConfig, isKubeVirtServiceAccount)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes, err = storageadmitters.Admit(admitter.VirtClient, ctx, ar.Request, &vm, admitter.ClusterConfig)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes, err = admitter.validateVolumeRequests(ctx, &vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	} else if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	isDryRun := ar.Request.DryRun != nil && *ar.Request.DryRun
	if !isDryRun && ar.Request.Operation == admissionv1.Create {
		metrics.NewVMCreated(&vm)
	}

	warnings := warnDeprecatedAPIs(&vm.Spec.Template.Spec, admitter.ClusterConfig)
	if vm.Spec.Running != nil {
		warnings = append(warnings, "spec.running is deprecated, please use spec.runStrategy instead.")
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnings,
	}
}

func (admitter *VMsAdmitter) AdmitStatus(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	vm, _, err := webhookutils.GetVMFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes, err := admitter.validateVolumeRequests(ctx, vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	} else if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	causes = storageadmitters.AdmitStatus(admitter.VirtClient, ctx, ar.Request, vm, admitter.ClusterConfig)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ValidateVirtualMachineSpec(field *k8sfield.Path, spec *v1.VirtualMachineSpec, config *virtconfig.ClusterConfig, isKubeVirtServiceAccount bool) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   field.Child("template").String(),
		})
	}

	causes = append(causes, ValidateVirtualMachineInstanceMetadata(field.Child("template", "metadata"), &spec.Template.ObjectMeta, config, isKubeVirtServiceAccount)...)
	causes = append(causes, ValidateVirtualMachineInstanceSpec(field.Child("template", "spec"), &spec.Template.Spec, config)...)

	causes = append(causes, storageadmitters.ValidateDataVolumeTemplate(field, spec)...)
	causes = append(causes, validateRunStrategy(field, spec, config)...)
	causes = append(causes, validateLiveUpdateFeatures(field, spec, config)...)

	return causes
}

func validateRunStrategy(field *k8sfield.Path, spec *v1.VirtualMachineSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if spec.Running != nil && spec.RunStrategy != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Running and RunStrategy are mutually exclusive. Note that Running is deprecated, please use RunStrategy instead"),
			Field:   field.Child("running").String(),
		})
	}

	if spec.Running == nil && spec.RunStrategy == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("RunStrategy must be specified"),
			Field:   field.Child("running").String(),
		})
	}

	if spec.RunStrategy != nil {
		if *spec.RunStrategy == v1.RunStrategyWaitAsReceiver {
			if config.DecentralizedLiveMigrationEnabled() {
				// Only allow wait as receiver if feature gate is set.
				return
			}
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid RunStrategy (%s), %s feature gate is not enabled in kubevirt resource", *spec.RunStrategy, featuregate.DecentralizedLiveMigration),
				Field:   field.Child("runStrategy").String(),
			})
		} else {
			validRunStrategy := false
			for _, strategy := range validRunStrategies {
				if *spec.RunStrategy == strategy {
					validRunStrategy = true
					break
				}
			}
			if !validRunStrategy {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("Invalid RunStrategy (%s)", *spec.RunStrategy),
					Field:   field.Child("runStrategy").String(),
				})
			}
		}
	}
	return causes
}

func validateLiveUpdateFeatures(field *k8sfield.Path, spec *v1.VirtualMachineSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if !config.IsVMRolloutStrategyLiveUpdate() {
		return causes
	}

	if spec.Template.Spec.Domain.CPU != nil {
		causes = append(causes, validateLiveUpdateCPU(field, &spec.Template.Spec.Domain)...)
	}

	if spec.Template.Spec.Domain.Memory != nil && spec.Template.Spec.Domain.Memory.MaxGuest != nil {
		if err := memory.ValidateLiveUpdateMemory(&spec.Template.Spec, spec.Template.Spec.Domain.Memory.MaxGuest); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: err.Error(),
				Field:   field.Child("template", "spec", "domain", "memory", "guest").String(),
			})
		}
	}

	return causes
}

func validateLiveUpdateCPU(field *k8sfield.Path, domain *v1.DomainSpec) (causes []metav1.StatusCause) {
	if domain.CPU.Sockets != 0 && domain.CPU.MaxSockets != 0 && domain.CPU.Sockets > domain.CPU.MaxSockets {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Number of sockets in CPU topology is greater than the maximum sockets allowed"),
			Field:   field.Child("template", "spec", "domain", "cpu", "sockets").String(),
		})
	}

	return causes
}

func (admitter *VMsAdmitter) validateVolumeRequests(ctx context.Context, vm *v1.VirtualMachine) ([]metav1.StatusCause, error) {
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
		var err error

		vmi, err = admitter.VirtClient.VirtualMachineInstance(vm.Namespace).Get(ctx, vm.Name, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return nil, err
		} else if err == nil && vmi.DeletionTimestamp == nil {
			// ignore validating the vmi if it is being deleted
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
			invalidDiskStatusCause := storageadmitters.ValidateHotplugDiskConfiguration(
				volumeRequest.AddVolumeOptions.Disk, name,
				"AddVolume request",
				k8sfield.NewPath("Status", "volumeRequests").String(),
			)
			if invalidDiskStatusCause != nil {
				return invalidDiskStatusCause, nil
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
			if ok && !equality.Semantic.DeepEqual(newVolume, vmVolume) {
				return []metav1.StatusCause{{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("AddVolume request for [%s] conflicts with an existing volume of the same name on the vmi template.", name),
					Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
				}}, nil
			}

			vmiVolume, ok := vmiVolumeMap[name]
			if ok && !equality.Semantic.DeepEqual(newVolume, vmiVolume) {
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

		if migrationutil.IsMigrating(vmi) {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("Cannot handle volume requests while VMI migration is in progress"),
				Field:   k8sfield.NewPath("spec").String(),
			}}, nil
		}
	}

	return nil, nil

}
