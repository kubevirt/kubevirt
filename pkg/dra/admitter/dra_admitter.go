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
	configChecker GPUDRAConfigChecker
}

type GPUDRAConfigChecker interface {
	GPUsWithDRAGateEnabled() bool
}

func NewValidator(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, configChecker GPUDRAConfigChecker) *Validator {
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

func validateCreationDRA(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, checker GPUDRAConfigChecker) []metav1.StatusCause {
	var (
		causes     []metav1.StatusCause
		draGPUs    []v1.GPU
		nonDRAGPUs []v1.GPU
	)
	for _, gpu := range spec.Domain.Devices.GPUs {
		if drautil.IsGPUDRA(gpu) {
			draGPUs = append(draGPUs, gpu)
		} else {
			nonDRAGPUs = append(nonDRAGPUs, gpu)
		}
	}

	// If a gpu is non-DRA, it must have only deviceName configured
	for _, gpu := range nonDRAGPUs {
		if gpu.DeviceName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "vmi.spec.domain.devices.gpus contains GPUs without deviceName",
				Field:   field.Child("spec", "domain", "devices", "gpus").String(),
			})
			return causes
		}
		if gpu.DeviceName != "" && gpu.ClaimRequest != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "vmi.spec.domain.devices.gpus contains GPUs with both deviceName and claimRequest",
				Field:   field.Child("spec", "domain", "devices", "gpus").String(),
			})
		}
	}

	if len(draGPUs) > 0 && !checker.GPUsWithDRAGateEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.domain.devices.gpus contains DRA enabled GPUs but feature gate is not enabled",
			Field:   field.Child("spec", "domain", "devices", "gpus").String(),
		})
		return causes
	}

	claimNamesFromGPUs := sets.New[string]()
	for _, gpu := range draGPUs {
		claimNamesFromGPUs.Insert(*gpu.ClaimName)
	}

	claimNamesFromRC := sets.New[string]()
	for _, rc := range spec.ResourceClaims {
		claimNamesFromRC.Insert(rc.Name)
	}

	if !claimNamesFromRC.IsSuperset(claimNamesFromGPUs) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.resourceClaims must specify all claims used in vmi.spec.domain.devices.gpus",
			Field:   field.Child("resourceClaims").String(),
		})
		return causes
	}

	return causes
}

func ValidateCreation(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, clusterCfg *virtconfig.ClusterConfig) []metav1.StatusCause {
	return NewValidator(field, vmiSpec, clusterCfg).ValidateCreation()
}
