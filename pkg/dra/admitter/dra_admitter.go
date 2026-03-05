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

package admitter

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type Validator struct {
	field         *k8sfield.Path
	vmiSpec       *v1.VirtualMachineInstanceSpec
	configChecker DRAConfigChecker
}

type DRAConfigChecker interface {
	GPUsWithDRAGateEnabled() bool
	HostDevicesWithDRAEnabled() bool
}

func NewValidator(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, configChecker DRAConfigChecker) *Validator {
	return &Validator{
		field:         field,
		vmiSpec:       vmiSpec,
		configChecker: configChecker,
	}
}

func (v Validator) ValidateCreation() []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateCreationDRA(v.field, v.vmiSpec, v.configChecker)...)

	return causes
}

func (v Validator) Validate() []metav1.StatusCause {
	return validateCreationDRA(v.field, v.vmiSpec, v.configChecker)
}

func validateCreationDRA(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, checker DRAConfigChecker) []metav1.StatusCause {
	var causes []metav1.StatusCause

	rcField := field.Child("resourceClaims")
	rcNames := sets.New[string]()
	for i, rc := range spec.ResourceClaims {
		if rcNames.Has(rc.Name) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("duplicate resourceClaims name %q", rc.Name),
				Field:   rcField.Index(i).Child("name").String(),
			})
		}
		rcNames.Insert(rc.Name)
	}

	gpuCauses, gpuClaimNames := validateDRAGPUs(field, spec.Domain.Devices.GPUs, checker)
	causes = append(causes, gpuCauses...)

	hdCauses, hdClaimNames := validateDRAHostDevices(field, spec.Domain.Devices.HostDevices, checker)
	causes = append(causes, hdCauses...)

	allClaimNames := gpuClaimNames.Union(hdClaimNames)

	claimNamesFromRC := sets.New[string]()
	for _, rc := range spec.ResourceClaims {
		claimNamesFromRC.Insert(rc.Name)
	}

	if !claimNamesFromRC.IsSuperset(allClaimNames) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.resourceClaims must specify all claims used in vmi.spec.domain.devices.gpus and vmi.spec.domain.devices.hostDevices",
			Field:   field.Child("resourceClaims").String(),
		})
	}

	return causes
}

func validateDRAGPUs(field *k8sfield.Path, gpus []v1.GPU, checker DRAConfigChecker) ([]metav1.StatusCause, sets.Set[string]) {
	var (
		causes     []metav1.StatusCause
		draGPUs    []v1.GPU
		nonDRAGPUs []v1.GPU
	)
	gpusField := field.Child("domain", "devices", "gpus")

	for _, gpu := range gpus {
		if drautil.IsGPUDRA(gpu) {
			draGPUs = append(draGPUs, gpu)
		} else {
			nonDRAGPUs = append(nonDRAGPUs, gpu)
		}
	}

	// in case of GPUs having device plugin and DRA drivers on the same node will be problematic
	// hence this validation is added to reject the creation of VMI if both DRA and non-DRA GPUs are present
	if len(nonDRAGPUs) > 0 && len(draGPUs) > 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.domain.devices.gpus contains both DRA and non-DRA GPUs; each GPU must be either DRA or non-DRA",
			Field:   gpusField.String(),
		})
		return causes, sets.Set[string](sets.NewString())
	}

	for _, gpu := range nonDRAGPUs {
		if gpu.DeviceName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "vmi.spec.domain.devices.gpus contains GPUs without deviceName or claimRequest; each GPU must specify either a deviceName or a claimRequest",
				Field:   gpusField.String(),
			})
		}
		if gpu.DeviceName != "" && gpu.ClaimRequest != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "vmi.spec.domain.devices.gpus contains GPUs with both deviceName and claimRequest",
				Field:   gpusField.String(),
			})
		}
	}

	// returns early because feature gate is not enabled
	if len(draGPUs) > 0 && !checker.GPUsWithDRAGateEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.domain.devices.gpus contains DRA enabled GPUs but feature gate is not enabled",
			Field:   gpusField.String(),
		})
		return causes, sets.New[string]()
	}

	type indexedGPU struct {
		index int
		gpu   v1.GPU
	}
	var validDRAGPUs []indexedGPU
	for i, gpu := range draGPUs {
		valid := true
		if gpu.ClaimName == nil || *gpu.ClaimName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "claimName is required for DRA GPU",
				Field:   gpusField.Index(i).Child("claimName").String(),
			})
			valid = false
		}
		if gpu.RequestName == nil || *gpu.RequestName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "requestName is required for DRA GPU",
				Field:   gpusField.Index(i).Child("requestName").String(),
			})
			valid = false
		}
		if valid {
			validDRAGPUs = append(validDRAGPUs, indexedGPU{index: i, gpu: gpu})
		}
	}

	claimRequestPairs := sets.New[string]()
	for _, vg := range validDRAGPUs {
		key := *vg.gpu.ClaimName + "/" + *vg.gpu.RequestName
		if claimRequestPairs.Has(key) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("duplicate claimName/requestName pair %q", key),
				Field:   gpusField.Index(vg.index).String(),
			})
		}
		claimRequestPairs.Insert(key)
	}

	claimNames := sets.New[string]()
	for _, vg := range validDRAGPUs {
		claimNames.Insert(*vg.gpu.ClaimName)
	}

	return causes, claimNames
}

func validateDRAHostDevices(field *k8sfield.Path, hostDevices []v1.HostDevice, checker DRAConfigChecker) ([]metav1.StatusCause, sets.Set[string]) {
	var causes []metav1.StatusCause
	hdField := field.Child("domain", "devices", "hostDevices")

	type indexedHD struct {
		index int
		hd    v1.HostDevice
	}

	var draHDs []indexedHD
	hasDRA := false
	for i, hd := range hostDevices {
		if drautil.IsHostDeviceDRA(hd) {
			draHDs = append(draHDs, indexedHD{index: i, hd: hd})
			hasDRA = true
		} else {
			if hd.DeviceName == "" {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueRequired,
					Message: "vmi.spec.domain.devices.hostDevices contains HostDevices without deviceName or claimRequest; each HostDevice must specify either a deviceName or a claimRequest",
					Field:   hdField.String(),
				})
			}
			if hd.DeviceName != "" && hd.ClaimRequest != nil {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "vmi.spec.domain.devices.hostDevices contains HostDevices with both deviceName and claimRequest",
					Field:   hdField.String(),
				})
			}
		}
	}

	if hasDRA && !checker.HostDevicesWithDRAEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.domain.devices.hostDevices contains DRA enabled HostDevices but feature gate is not enabled",
			Field:   hdField.String(),
		})
		return causes, sets.New[string]()
	}

	var validDRAHDs []indexedHD
	for _, ih := range draHDs {
		valid := true
		if ih.hd.ClaimName == nil || *ih.hd.ClaimName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "claimName is required for DRA HostDevice",
				Field:   hdField.Index(ih.index).Child("claimName").String(),
			})
			valid = false
		}
		if ih.hd.RequestName == nil || *ih.hd.RequestName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "requestName is required for DRA HostDevice",
				Field:   hdField.Index(ih.index).Child("requestName").String(),
			})
			valid = false
		}
		if valid {
			validDRAHDs = append(validDRAHDs, ih)
		}
	}

	claimRequestPairs := sets.New[string]()
	for _, vh := range validDRAHDs {
		key := *vh.hd.ClaimName + "/" + *vh.hd.RequestName
		if claimRequestPairs.Has(key) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("duplicate claimName/requestName pair %q", key),
				Field:   hdField.Index(vh.index).String(),
			})
		}
		claimRequestPairs.Insert(key)
	}

	claimNames := sets.New[string]()
	for _, vh := range validDRAHDs {
		claimNames.Insert(*vh.hd.ClaimName)
	}

	return causes, claimNames
}

func ValidateCreation(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, clusterCfg *virtconfig.ClusterConfig) []metav1.StatusCause {
	return NewValidator(field, vmiSpec, clusterCfg).ValidateCreation()
}
