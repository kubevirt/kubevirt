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

	corev1 "k8s.io/api/core/v1"

	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/controller"
	migrationutil "kubevirt.io/kubevirt/pkg/util/migrations"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype"
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

	if ar.Request.Operation == admissionv1.Update {
		oldVM := v1.VirtualMachine{}
		if err := json.Unmarshal(ar.Request.OldObject.Raw, &oldVM); err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		if !equality.Semantic.DeepEqual(&oldVM.Spec, &vm.Spec) {
			causes = admitter.validateVMUpdate(&oldVM, &vm)
			if len(causes) > 0 {
				return webhookutils.ToAdmissionResponse(causes)
			}
		}
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

	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil, nil, nil
	}

	if topology := instancetype.GetPreferredTopology(preferenceSpec); topology == instancetypev1beta1.PreferSpread {
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
	if spec.LiveUpdateFeatures != nil && !config.VMLiveUpdateFeaturesEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s feature gate is not enabled in kubevirt-config", virtconfig.VMLiveUpdateFeaturesGate),
			Field:   field.Child("liveUpdateFeatures").String(),
		})
	}

	if spec.Template.Spec.Domain.CPU != nil && spec.Template.Spec.Domain.CPU.MaxSockets != 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("CPU topology maxSockets cannot be set directy in VM template"),
			Field:   field.Child("template.spec.domain.cpu.maxSockets").String(),
		})
	}

	if spec.LiveUpdateFeatures != nil {
		if spec.Instancetype != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("Live update features cannot be used when instance type is configured"),
				Field:   field.Child("liveUpdateFeatures").String(),
			})
		}
	}

	if spec.LiveUpdateFeatures != nil && spec.LiveUpdateFeatures.CPU != nil {

		if spec.Template.Spec.Domain.CPU.Sockets != 0 {
			if spec.LiveUpdateFeatures.CPU.MaxSockets != nil {
				if spec.Template.Spec.Domain.CPU.Sockets > *spec.LiveUpdateFeatures.CPU.MaxSockets {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("Number of sockets in CPU topology is greater than the maximum sockets allowed"),
						Field:   field.Child("liveUpdateFeatures").String(),
					})
				}
			}
		}

		if hasCPURequestsOrLimits(&spec.Template.Spec.Domain.Resources) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Configuration of CPU resource requirements is not allowed when CPU live update is enabled"),
				Field:   field.Child("liveUpdateFeatures").String(),
			})
		}
	}

	// Validate Memory Hotplug
	if spec.Template.Spec.Domain.Memory != nil && spec.Template.Spec.Domain.Memory.MaxGuest != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: fmt.Sprintf("Memory maxGuest cannot be set directy in VM template"),
			Field:   field.Child("template.spec.domain.memory.maxGuest").String(),
		})
	}

	if spec.LiveUpdateFeatures != nil &&
		spec.LiveUpdateFeatures.Memory != nil &&
		spec.LiveUpdateFeatures.Memory.MaxGuest != nil {

		if hasMemoryLimits(&spec.Template.Spec.Domain.Resources) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Configuration of Memory limits is not allowed when Memory live update is enabled"),
				Field:   field.Child("liveUpdateFeatures").String(),
			})
		}

		if spec.Template.Spec.Domain.CPU != nil &&
			spec.Template.Spec.Domain.CPU.Realtime != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Memory hotplug is not compatible with realtime VMs"),
				Field:   field.Child("template", "spec", "domain", "cpu", "realtime").String(),
			})
		}

		if spec.Template.Spec.Domain.CPU != nil &&
			spec.Template.Spec.Domain.CPU.NUMA != nil &&
			spec.Template.Spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Memory hotplug is not compatible with guest mapping passthrough"),
				Field:   field.Child("template", "spec", "domain", "cpu", "numa", "guestMappingPassthrough").String(),
			})
		}

		if spec.Template.Spec.Domain.LaunchSecurity != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Memory hotplug is not compatible with encrypted VMs"),
				Field:   field.Child("template", "spec", "domain", "launchSecurity").String(),
			})
		}

		if spec.Template.Spec.Domain.CPU != nil &&
			spec.Template.Spec.Domain.CPU.DedicatedCPUPlacement {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Memory hotplug is not compatible with dedicated CPUs"),
				Field:   field.Child("template", "spec", "domain", "cpu", "dedicatedCpuPlacement").String(),
			})
		}

		if spec.Template.Spec.Domain.Memory != nil &&
			spec.Template.Spec.Domain.Memory.Hugepages != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Memory hotplug is not compatible with hugepages"),
				Field:   field.Child("template", "spec", "domain", "memory", "hugepages").String(),
			})
		}

		if spec.Template.Spec.Domain.Memory == nil ||
			spec.Template.Spec.Domain.Memory.Guest == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Guest memory must be configured when memory hotplug is enabled"),
				Field:   field.Child("template", "spec", "domain", "memory", "guest").String(),
			})
		} else if spec.Template.Spec.Domain.Memory.Guest.Cmp(*spec.LiveUpdateFeatures.Memory.MaxGuest) > 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Guest memory is greater than the configured maxGuest memory"),
				Field:   field.Child("template", "spec", "domain", "memory", "guest").String(),
			})
		} else if spec.Template.Spec.Domain.Memory.Guest.Value()%converter.MemoryHotplugBlockAlignmentBytes != 0 {
			alignment := resource.NewQuantity(converter.MemoryHotplugBlockAlignmentBytes, resource.BinarySI)
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Guest memory must be %s aligned", alignment),
				Field:   field.Child("template", "spec", "domain", "memory", "guest").String(),
			})
		}

		if spec.LiveUpdateFeatures.Memory.MaxGuest.Value()%converter.MemoryHotplugBlockAlignmentBytes != 0 {
			alignment := resource.NewQuantity(converter.MemoryHotplugBlockAlignmentBytes, resource.BinarySI)
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("MaxGuest must be %s aligned", alignment),
				Field:   field.Child("liveUpdateFeatures", "MaxGuest").String(),
			})
		}

		if spec.Template.Spec.Architecture != "amd64" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("Memory hotplug is only available for x86_64 VMs"),
				Field:   field.Child("template", "spec", "architecture").String(),
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
		var err error

		vmi, err = admitter.VirtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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

func (admitter *VMsAdmitter) validateVMUpdate(oldVM, newVM *v1.VirtualMachine) []metav1.StatusCause {
	if newVM.Status.Ready {
		if !equality.Semantic.DeepEqual(&oldVM.Spec.LiveUpdateFeatures, &newVM.Spec.LiveUpdateFeatures) {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: fmt.Sprintf("Cannot update VM live features while VM is running"),
				Field:   k8sfield.NewPath("spec").String(),
			}}
		}

		if newVM.Spec.LiveUpdateFeatures != nil {

			// CPU hotplug
			if newVM.Spec.LiveUpdateFeatures.CPU != nil {
				oldTopology := oldVM.Spec.Template.Spec.Domain.CPU
				newTopology := newVM.Spec.Template.Spec.Domain.CPU
				if oldTopology != nil && newTopology != nil {
					if oldTopology.Cores != newTopology.Cores {
						return []metav1.StatusCause{{
							Type:    metav1.CauseTypeFieldValueNotSupported,
							Message: fmt.Sprintf("Cannot update CPU cores while live update features configured"),
							Field:   k8sfield.NewPath("spec.template.spec.domain.cpu.cores").String(),
						}}
					}
					if oldTopology.Threads != newTopology.Threads {
						return []metav1.StatusCause{{
							Type:    metav1.CauseTypeFieldValueNotSupported,
							Message: fmt.Sprintf("Cannot update CPU threads while live update features configured"),
							Field:   k8sfield.NewPath("spec.template.spec.domain.cpu.threads").String(),
						}}
					}
					if oldTopology.Sockets != newTopology.Sockets {
						if causeErr := admitter.shouldAllowCPUHotPlug(oldVM); causeErr != nil {
							return []metav1.StatusCause{{
								Type:    metav1.CauseTypeFieldValueNotSupported,
								Message: causeErr.Error(),
								Field:   k8sfield.NewPath("spec.template.spec.domain.cpu.sockets").String(),
							}}
						}

					}
				}
			}

			// Memory Hotplug
			if newVM.Spec.LiveUpdateFeatures.Memory != nil {
				oldGuestMemory := oldVM.Spec.Template.Spec.Domain.Memory.Guest
				newGuestMemory := newVM.Spec.Template.Spec.Domain.Memory.Guest

				if !oldGuestMemory.Equal(*newGuestMemory) {
					if causeErr := admitter.shouldAllowMemoryHotPlug(newVM); causeErr != nil {
						return []metav1.StatusCause{{
							Type:    metav1.CauseTypeFieldValueNotSupported,
							Message: causeErr.Error(),
							Field:   k8sfield.NewPath("spec.template.spec.domain.memory.guest").String(),
						}}
					}
				}
			}
		}
	}

	return nil
}

func (admitter *VMsAdmitter) shouldAllowCPUHotPlug(vm *v1.VirtualMachine) error {
	vmi, err := admitter.VirtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, c := range vmi.Status.Conditions {
		if c.Type == v1.VirtualMachineInstanceVCPUChange &&
			c.Status == corev1.ConditionTrue {
			return fmt.Errorf("cannot update CPU sockets while another CPU change is in progress")
		}
	}

	if err := admitter.isMigrationInProgress(vmi); err != nil {
		return err
	}
	return nil
}

func (admitter *VMsAdmitter) shouldAllowMemoryHotPlug(vm *v1.VirtualMachine) error {
	vmi, err := admitter.VirtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
	if err != nil {
		return err
	}

	if vm.Spec.Template.Spec.Domain.Memory.Guest.Cmp(*vmi.Status.Memory.GuestAtBoot) < 0 {
		return fmt.Errorf("cannot set less memory than what the guest booted with")
	}

	for _, c := range vmi.Status.Conditions {
		if c.Type == v1.VirtualMachineInstanceMemoryChange &&
			c.Status == corev1.ConditionTrue {
			return fmt.Errorf("cannot update memory while another memory change is in progress")
		}
	}

	if err := admitter.isMigrationInProgress(vmi); err != nil {
		return err
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

func hasMemoryLimits(rr *v1.ResourceRequirements) bool {
	_, ok := rr.Limits[corev1.ResourceMemory]
	return ok
}
