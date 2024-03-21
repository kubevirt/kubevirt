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

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	corev1 "k8s.io/api/core/v1"

	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	migrationutil "kubevirt.io/kubevirt/pkg/util/migrations"

	"kubevirt.io/kubevirt/pkg/instancetype"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var validRunStrategies = []v1.VirtualMachineRunStrategy{v1.RunStrategyHalted, v1.RunStrategyManual, v1.RunStrategyAlways, v1.RunStrategyRerunOnFailure, v1.RunStrategyOnce}

type CloneAuthFunc func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error)

type VMsAdmitter struct {
	VirtClient          kubecli.KubevirtClient
	DataSourceInformer  cache.SharedIndexInformer
	NamespaceInformer   cache.SharedIndexInformer
	InstancetypeMethods instancetype.Methods
	ClusterConfig       *virtconfig.ClusterConfig
	cloneAuthFunc       CloneAuthFunc
}

type authProxy struct {
	client             kubecli.KubevirtClient
	dataSourceInformer cache.SharedIndexInformer
	namespaceInformer  cache.SharedIndexInformer
}

func (p *authProxy) CreateSar(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
	return p.client.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
}

func (p *authProxy) GetNamespace(name string) (*corev1.Namespace, error) {
	obj, exists, err := p.namespaceInformer.GetStore().GetByKey(name)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("namespace %s does not exist", name)
	}

	ns := obj.(*corev1.Namespace).DeepCopy()
	return ns, nil
}

func (p *authProxy) GetDataSource(namespace, name string) (*cdiv1.DataSource, error) {
	key := fmt.Sprintf("%s/%s", namespace, name)
	obj, exists, err := p.dataSourceInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("dataSource %s does not exist", key)
	}

	ds := obj.(*cdiv1.DataSource).DeepCopy()
	return ds, nil
}

func NewVMsAdmitter(clusterConfig *virtconfig.ClusterConfig, client kubecli.KubevirtClient, informers *webhooks.Informers) *VMsAdmitter {
	return &VMsAdmitter{
		VirtClient:          client,
		DataSourceInformer:  informers.DataSourceInformer,
		NamespaceInformer:   informers.NamespaceInformer,
		InstancetypeMethods: &instancetype.InstancetypeMethods{Clientset: client},
		ClusterConfig:       clusterConfig,
		cloneAuthFunc: func(dv *cdiv1.DataVolume, requestNamespace, requestName string, proxy cdiv1.AuthorizationHelperProxy, saNamespace, saName string) (bool, string, error) {
			response, err := dv.AuthorizeSA(requestNamespace, requestName, proxy, saNamespace, saName)
			return response.Allowed, response.Reason, err
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
	instancetypeSpec, preferenceSpec, causes := admitter.applyInstancetypeToVm(vmCopy)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	// Set VirtualMachine defaults on the copy before validating
	if err = webhooks.SetDefaultVirtualMachine(admitter.ClusterConfig, vmCopy); err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// With the defaults now set we can check that the VM meets the requirements of any provided preference
	if preferenceSpec != nil {
		if conflicts, err := admitter.InstancetypeMethods.CheckPreferenceRequirements(instancetypeSpec, preferenceSpec, &vmCopy.Spec.Template.Spec); err != nil {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueNotFound,
				Message: fmt.Sprintf("failure checking preference requirements: %v", err),
				Field:   conflicts.String(),
			}})
		}
	}

	causes = ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &vmCopy.Spec, admitter.ClusterConfig, accountName)
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

	causes = validateRestoreStatus(ar.Request, &vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	isDryRun := ar.Request.DryRun != nil && *ar.Request.DryRun
	if !isDryRun && ar.Request.Operation == admissionv1.Create {
		metrics.NewVMCreated(&vm)
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnDeprecatedAPIs(&vm.Spec.Template.Spec, admitter.ClusterConfig),
	}

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

	causes = validateRestoreStatus(ar.Request, vm)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func (admitter *VMsAdmitter) applyInstancetypeToVm(vm *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, *instancetypev1beta1.VirtualMachinePreferenceSpec, []metav1.StatusCause) {
	instancetypeSpec, err := admitter.InstancetypeMethods.FindInstancetypeSpec(vm)
	if err != nil {
		return nil, nil, []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("Failure to find instancetype: %v", err),
			Field:   k8sfield.NewPath("spec", "instancetype").String(),
		}}
	}

	preferenceSpec, err := admitter.InstancetypeMethods.FindPreferenceSpec(vm)
	if err != nil {
		return nil, nil, []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: fmt.Sprintf("Failure to find preference: %v", err),
			Field:   k8sfield.NewPath("spec", "preference").String(),
		}}
	}

	if topology := instancetype.GetPreferredTopology(preferenceSpec); instancetypeSpec != nil && topology == instancetypev1beta1.PreferSpread {
		ratio := preferenceSpec.PreferSpreadSocketToCoreRatio
		if ratio == 0 {
			ratio = instancetype.DefaultSpreadRatio
		}

		if (instancetypeSpec.CPU.Guest % ratio) > 0 {
			return nil, nil, []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Instancetype CPU Guest is not divisible by PreferSpreadSocketToCoreRatio",
				Field:   k8sfield.NewPath("instancetype.spec.cpu.guest").String(),
			}}
		}
	}

	conflicts := admitter.InstancetypeMethods.ApplyToVmi(k8sfield.NewPath("spec", "template", "spec"), instancetypeSpec, preferenceSpec, &vm.Spec.Template.Spec, &vm.Spec.Template.ObjectMeta)

	if len(conflicts) == 0 {
		return instancetypeSpec, preferenceSpec, nil
	}

	causes := make([]metav1.StatusCause, 0, len(conflicts))
	for _, conflict := range conflicts {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf(instancetype.VMFieldConflictErrorFmt, conflict.String()),
			Field:   conflict.String(),
		})
	}
	return nil, nil, causes
}

func (admitter *VMsAdmitter) authorizeVirtualMachineSpec(ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine) ([]metav1.StatusCause, error) {
	var causes []metav1.StatusCause

	for idx, dataVolume := range vm.Spec.DataVolumeTemplates {
		targetNamespace := vm.Namespace
		if targetNamespace == "" {
			targetNamespace = ar.Namespace
		}
		if dataVolume.Namespace != "" && dataVolume.Namespace != targetNamespace {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Embedded DataVolume namespace %s differs from VM namespace %s", dataVolume.Namespace, targetNamespace),
				Field:   k8sfield.NewPath("spec", "dataVolumeTemplates").Index(idx).String(),
			})

			continue
		}
		serviceAccountName := "default"
		for _, vol := range vm.Spec.Template.Spec.Volumes {
			if vol.ServiceAccount != nil {
				serviceAccountName = vol.ServiceAccount.ServiceAccountName
			}
		}

		proxy := &authProxy{client: admitter.VirtClient, dataSourceInformer: admitter.DataSourceInformer, namespaceInformer: admitter.NamespaceInformer}
		dv := &cdiv1.DataVolume{
			ObjectMeta: dataVolume.ObjectMeta,
			Spec:       dataVolume.Spec,
		}
		dv.Namespace = targetNamespace
		allowed, message, err := admitter.cloneAuthFunc(dv, ar.Namespace, ar.Name, proxy, targetNamespace, serviceAccountName)
		if err != nil && err != cdiv1.ErrNoTokenOkay {
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

	causes = append(causes, validateDataVolumeTemplate(field, spec)...)
	causes = append(causes, validateRunStrategy(field, spec)...)
	causes = append(causes, validateLiveUpdateFeatures(field, spec, config)...)

	return causes
}

func validateDataVolumeTemplate(field *k8sfield.Path, spec *v1.VirtualMachineSpec) (causes []metav1.StatusCause) {
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
	return causes
}

func validateRunStrategy(field *k8sfield.Path, spec *v1.VirtualMachineSpec) (causes []metav1.StatusCause) {
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
		if !validRunStrategy {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Invalid RunStrategy (%s)", *spec.RunStrategy),
				Field:   field.Child("runStrategy").String(),
			})
		}
	}
	return causes
}

func validateLiveUpdateFeatures(field *k8sfield.Path, spec *v1.VirtualMachineSpec, config *virtconfig.ClusterConfig) (causes []metav1.StatusCause) {
	if !config.IsVMRolloutStrategyLiveUpdate() {
		return causes
	}

	if spec.Instancetype != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Cannot configure instance type when the vmRolloutStrategy is LiveUpdate"),
			Field:   field.Child("instancetype").String(),
		})
	}

	causes = append(causes, validateLiveUpdateCPU(field, &spec.Template.Spec.Domain)...)

	if hasCPURequestsOrLimits(&spec.Template.Spec.Domain.Resources) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Configuration of CPU resource requirements is not allowed when CPU live update is enabled"),
			Field:   field.Child("template", "spec", "domain", "resources").String(),
		})
	}

	if spec.Template.Spec.Domain.Memory != nil && spec.Template.Spec.Domain.Memory.MaxGuest != nil {
		causes = append(causes, validateLiveUpdateMemory(field, &spec.Template.Spec.Domain, spec.Template.Spec.Architecture)...)
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

func validateLiveUpdateMemory(field *k8sfield.Path, domain *v1.DomainSpec, architecture string) (causes []metav1.StatusCause) {
	if domain.CPU != nil &&
		domain.CPU.Realtime != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Memory hotplug is not compatible with realtime VMs"),
			Field:   field.Child("template", "spec", "domain", "cpu", "realtime").String(),
		})
	}

	if domain.CPU != nil &&
		domain.CPU.NUMA != nil &&
		domain.CPU.NUMA.GuestMappingPassthrough != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Memory hotplug is not compatible with guest mapping passthrough"),
			Field:   field.Child("template", "spec", "domain", "cpu", "numa", "guestMappingPassthrough").String(),
		})
	}

	if domain.LaunchSecurity != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Memory hotplug is not compatible with encrypted VMs"),
			Field:   field.Child("template", "spec", "domain", "launchSecurity").String(),
		})
	}

	if domain.CPU != nil &&
		domain.CPU.DedicatedCPUPlacement {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Memory hotplug is not compatible with dedicated CPUs"),
			Field:   field.Child("template", "spec", "domain", "cpu", "dedicatedCpuPlacement").String(),
		})
	}

	if domain.Memory != nil &&
		domain.Memory.Hugepages != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Memory hotplug is not compatible with hugepages"),
			Field:   field.Child("template", "spec", "domain", "memory", "hugepages").String(),
		})
	}

	if domain.Memory == nil ||
		domain.Memory.Guest == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Guest memory must be configured when memory hotplug is enabled"),
			Field:   field.Child("template", "spec", "domain", "memory", "guest").String(),
		})
	} else if domain.Memory.Guest.Cmp(*domain.Memory.MaxGuest) > 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Guest memory is greater than the configured maxGuest memory"),
			Field:   field.Child("template", "spec", "domain", "memory", "guest").String(),
		})
	} else if domain.Memory.Guest.Value()%converter.MemoryHotplugBlockAlignmentBytes != 0 {
		alignment := resource.NewQuantity(converter.MemoryHotplugBlockAlignmentBytes, resource.BinarySI)
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Guest memory must be %s aligned", alignment),
			Field:   field.Child("template", "spec", "domain", "memory", "guest").String(),
		})
	}

	if domain.Memory.MaxGuest.Value()%converter.MemoryHotplugBlockAlignmentBytes != 0 {
		alignment := resource.NewQuantity(converter.MemoryHotplugBlockAlignmentBytes, resource.BinarySI)
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("MaxGuest must be %s aligned", alignment),
			Field:   field.Child("template", "spec", "domain", "memory", "maxGuest").String(),
		})
	}

	if architecture != "amd64" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("Memory hotplug is only available for x86_64 VMs"),
			Field:   field.Child("template", "spec", "architecture").String(),
		})
		return causes
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
		var err error

		vmi, err = admitter.VirtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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
			invalidDiskStatusCause := validateDiskConfiguration(volumeRequest.AddVolumeOptions.Disk, name)
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

func validateDiskConfiguration(disk *v1.Disk, name string) []metav1.StatusCause {
	var bus v1.DiskBus
	// Validate the disk is configured properly
	if disk == nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("AddVolume request for [%s] requires the disk field to be set.", name),
			Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
		}}
	}
	if disk.DiskDevice.Disk == nil && disk.DiskDevice.LUN == nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("AddVolume request for [%s] requires diskDevice of type 'disk' or 'lun' to be used.", name),
			Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
		}}
	}
	if disk.DiskDevice.Disk != nil {
		bus = disk.DiskDevice.Disk.Bus
	} else {
		bus = disk.DiskDevice.LUN.Bus
	}
	if bus != "scsi" {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("AddVolume request for [%s] requires disk bus to be 'scsi'. [%s] is not permitted", name, bus),
			Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
		}}
	}
	if disk.DedicatedIOThread != nil && *disk.DedicatedIOThread {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "IOThreads are not supported by scsi bus.",
			Field:   k8sfield.NewPath("Status", "volumeRequests").String(),
		}}
	}

	return nil
}

func validateRestoreStatus(ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine) []metav1.StatusCause {
	if ar.Operation != admissionv1.Update || vm.Status.RestoreInProgress == nil {
		return nil
	}

	oldVM := &v1.VirtualMachine{}
	if err := json.Unmarshal(ar.OldObject.Raw, oldVM); err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeUnexpectedServerResponse,
			Message: "Could not fetch old VM",
		}}
	}

	if !equality.Semantic.DeepEqual(oldVM.Spec, vm.Spec) {
		strategy, _ := vm.RunStrategy()
		if strategy != v1.RunStrategyHalted {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("Cannot start VM until restore %q completes", *vm.Status.RestoreInProgress),
				Field:   k8sfield.NewPath("spec").String(),
			}}
		}
	}

	return nil
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

	if !equality.Semantic.DeepEqual(oldVM.Spec, vm.Spec) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Cannot update VM spec until snapshot %q completes", *vm.Status.SnapshotInProgress),
			Field:   k8sfield.NewPath("spec").String(),
		}}
	}

	return nil
}

func (admitter *VMsAdmitter) isMigrationInProgress(vmi *v1.VirtualMachineInstance) error {
	if vmi.Status.MigrationState != nil &&
		!vmi.Status.MigrationState.Completed {
		return fmt.Errorf("cannot update while VMI migration is in progress")
	}

	err := EnsureNoMigrationConflict(admitter.VirtClient, vmi.Name, vmi.Namespace)
	if err != nil {
		return fmt.Errorf("cannot update while VMI migration is in progress: %v", err)
	}
	return nil
}

func hasCPURequestsOrLimits(rr *v1.ResourceRequirements) bool {
	if _, ok := rr.Requests[corev1.ResourceCPU]; ok {
		return true
	}
	if _, ok := rr.Limits[corev1.ResourceCPU]; ok {
		return true
	}

	return false
}
