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
 * Copyright 2019 Red Hat, Inc.
 *
 */

/*
 * hyperv utilities are in the webhooks package because they are used both
 * by validation and mutation webhooks.
 */
package webhooks

import (
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

var _true bool = true

func enableFeatureState(fs **v1.FeatureState) {
	var val *v1.FeatureState
	if *fs != nil {
		val = *fs
	} else {
		val = &v1.FeatureState{}
	}
	val.Enabled = &_true
	*fs = val
}

func isFeatureStateMissing(fs **v1.FeatureState) bool {
	return *fs == nil || (*fs).Enabled == nil
}

// TODO: this dupes code in pkg/virt-controller/services/template.go
func isFeatureStateEnabled(fs **v1.FeatureState) bool {
	return !isFeatureStateMissing(fs) && *((*fs).Enabled)
}

type HypervFeature struct {
	State    **v1.FeatureState
	Field    *k8sfield.Path
	Requires *HypervFeature
}

func (hf HypervFeature) isRequirementOK() bool {
	if !isFeatureStateEnabled(hf.State) {
		return true
	}
	if hf.Requires == nil {
		return true
	}
	return isFeatureStateEnabled(hf.Requires.State)
}

// a requirement is compatible if
// 1. it is already enabled (either by the user or by us previously)
// 2. the user has not set it, so we can do on its behalf
func (hf HypervFeature) TryToSetRequirement() error {
	if !isFeatureStateEnabled(hf.State) || hf.Requires == nil {
		// not enabled or no requirements: nothing to do
		return nil
	}

	if isFeatureStateMissing(hf.Requires.State) {
		enableFeatureState(hf.Requires.State)
		return nil
	}

	if isFeatureStateEnabled(hf.Requires.State) {
		return nil
	}

	return fmt.Errorf("%s", hf.String())
}

func (hf HypervFeature) IsRequirementFulfilled() (metav1.StatusCause, bool) {
	if !hf.isRequirementOK() {
		return metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: hf.String(),
			Field:   hf.Field.String(),
		}, false
	}
	return metav1.StatusCause{}, true
}

func (hf HypervFeature) String() string {
	if hf.Requires == nil {
		return fmt.Sprintf("'%s' is missing", hf.Field.String())
	}
	return fmt.Sprintf("'%s' requires '%s', which was disabled.", hf.Field.String(), hf.Requires.Field.String())
}

func getHypervFeatureDependencies(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []HypervFeature {
	if spec.Domain.Features == nil || spec.Domain.Features.Hyperv == nil {
		return []HypervFeature{}
	}

	hyperv := spec.Domain.Features.Hyperv                      // shortcut
	hypervField := field.Child("domain", "features", "hyperv") // shortcut

	vpindex := HypervFeature{
		State: &hyperv.VPIndex,
		Field: hypervField.Child("vpindex"),
	}
	synic := HypervFeature{
		State:    &hyperv.SyNIC,
		Field:    hypervField.Child("synic"),
		Requires: &vpindex,
	}

	vapic := HypervFeature{
		State: &hyperv.VAPIC,
		Field: hypervField.Child("vapic"),
	}

	syNICTimer := &v1.FeatureState{}
	if hyperv.SyNICTimer != nil {
		syNICTimer.Enabled = hyperv.SyNICTimer.Enabled
	}

	features := []HypervFeature{
		// keep in REVERSE order: leaves first.
		{
			State:    &hyperv.EVMCS,
			Field:    hypervField.Child("evmcs"),
			Requires: &vapic,
		},
		{
			State:    &hyperv.IPI,
			Field:    hypervField.Child("ipi"),
			Requires: &vpindex,
		},
		{
			State:    &hyperv.TLBFlush,
			Field:    hypervField.Child("tlbflush"),
			Requires: &vpindex,
		},
		{
			State:    &syNICTimer,
			Field:    hypervField.Child("synictimer"),
			Requires: &synic,
		},
		synic,
	}

	return features
}

func ValidateVirtualMachineInstanceHypervFeatureDependencies(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if features := getHypervFeatureDependencies(field, spec); features != nil {
		for _, feat := range features {
			if cause, ok := feat.IsRequirementFulfilled(); !ok {
				causes = append(causes, cause)
			}
		}
	}

	if spec.Domain.Features != nil && spec.Domain.Features.Hyperv != nil && spec.Domain.Features.Hyperv.EVMCS != nil {
		if spec.Domain.CPU == nil || spec.Domain.CPU.Features == nil || len(spec.Domain.CPU.Features) == 0 {
			causes = append(causes, metav1.StatusCause{Type: metav1.CauseTypeFieldValueRequired, Message: "vmx cpu feature is required when evmcs is set", Field: "spec.domain.cpu.features"})
		} else if spec.Domain.CPU != nil || spec.Domain.CPU.Features != nil {
			for i, f := range spec.Domain.CPU.Features {
				if f.Name == nodelabellerutil.VmxFeature {
					if f.Policy != nodelabellerutil.RequirePolicy {
						causes = append(causes, metav1.StatusCause{Type: metav1.CauseTypeFieldValueInvalid, Message: "vmx cpu feature has to be set to " + nodelabellerutil.RequirePolicy + " policy", Field: "spec.domain.cpu.features[" + strconv.Itoa(i) + "].policy"})
					}
				}
			}
		}
	}

	return causes
}

func SetVirtualMachineInstanceHypervFeatureDependencies(vmi *v1.VirtualMachineInstance) error {
	path := k8sfield.NewPath("spec")

	if features := getHypervFeatureDependencies(path, &vmi.Spec); features != nil {
		for _, feat := range features {
			if err := feat.TryToSetRequirement(); err != nil {
				return err
			}
		}
	}

	//Check if vmi has EVMCS feature enabled. If yes, we have to add vmx cpu feature
	if vmi.Spec.Domain.Features != nil && vmi.Spec.Domain.Features.Hyperv != nil && vmi.Spec.Domain.Features.Hyperv.EVMCS != nil {
		setEVMCSDependency(vmi)
	}

	return nil
}

func setEVMCSDependency(vmi *v1.VirtualMachineInstance) {
	vmxFeature := v1.CPUFeature{
		Name:   nodelabellerutil.VmxFeature,
		Policy: nodelabellerutil.RequirePolicy,
	}

	cpuFeatures := []v1.CPUFeature{
		vmxFeature,
	}

	if vmi.Spec.Domain.CPU == nil {
		vmi.Spec.Domain.CPU = &v1.CPU{
			Features: cpuFeatures,
		}
	} else if len(vmi.Spec.Domain.CPU.Features) == 0 {
		vmi.Spec.Domain.CPU.Features = cpuFeatures
	} else {
		vmxFound := false
		for i, f := range vmi.Spec.Domain.CPU.Features {
			if f.Name == nodelabellerutil.VmxFeature {
				vmxFound = true
				if f.Policy != nodelabellerutil.RequirePolicy {
					vmi.Spec.Domain.CPU.Features[i].Policy = nodelabellerutil.RequirePolicy
				}
			}
		}
		if !vmxFound {
			vmi.Spec.Domain.CPU.Features = append(vmi.Spec.Domain.CPU.Features, vmxFeature)
		}
	}
}
