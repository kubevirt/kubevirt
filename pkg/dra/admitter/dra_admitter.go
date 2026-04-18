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

// draCapableDevice abstracts the common DRA-relevant fields shared by v1.GPU and v1.HostDevice.
type draCapableDevice interface {
	isDRA() bool
	getDeviceName() string
	getClaimRequest() *v1.ClaimRequest
}

type gpuAdapter v1.GPU

func (g gpuAdapter) isDRA() bool                       { return g.DeviceName == "" && g.ClaimRequest != nil }
func (g gpuAdapter) getDeviceName() string             { return g.DeviceName }
func (g gpuAdapter) getClaimRequest() *v1.ClaimRequest { return g.ClaimRequest }

type hostDeviceAdapter v1.HostDevice

func (h hostDeviceAdapter) isDRA() bool                       { return h.DeviceName == "" && h.ClaimRequest != nil }
func (h hostDeviceAdapter) getDeviceName() string             { return h.DeviceName }
func (h hostDeviceAdapter) getClaimRequest() *v1.ClaimRequest { return h.ClaimRequest }

type deviceValidationConfig struct {
	fieldPath   string
	typeName    string
	gateEnabled bool
	rejectMixed bool
}

type indexedDevice struct {
	idx    int
	device draCapableDevice
}

func validateDRAGPUs(field *k8sfield.Path, gpus []v1.GPU, checker DRAConfigChecker) ([]metav1.StatusCause, sets.Set[string]) {
	devices := make([]draCapableDevice, len(gpus))
	for i, g := range gpus {
		devices[i] = gpuAdapter(g)
	}
	return validateDRADevices(field, devices, deviceValidationConfig{
		fieldPath:   "gpus",
		typeName:    "GPU",
		gateEnabled: checker.GPUsWithDRAGateEnabled(),
		rejectMixed: true,
	})
}

func validateDRAHostDevices(field *k8sfield.Path, hostDevices []v1.HostDevice, checker DRAConfigChecker) ([]metav1.StatusCause, sets.Set[string]) {
	devices := make([]draCapableDevice, len(hostDevices))
	for i, hd := range hostDevices {
		devices[i] = hostDeviceAdapter(hd)
	}
	return validateDRADevices(field, devices, deviceValidationConfig{
		fieldPath:   "hostDevices",
		typeName:    "HostDevice",
		gateEnabled: checker.HostDevicesWithDRAEnabled(),
		rejectMixed: false,
	})
}

func validateDRADevices(field *k8sfield.Path, devices []draCapableDevice, cfg deviceValidationConfig) ([]metav1.StatusCause, sets.Set[string]) {
	var (
		causes     []metav1.StatusCause
		draDevs    []indexedDevice
		nonDRADevs []indexedDevice
	)
	devField := field.Child("domain", "devices", cfg.fieldPath)

	for i, d := range devices {
		if d.isDRA() {
			draDevs = append(draDevs, indexedDevice{i, d})
		} else {
			nonDRADevs = append(nonDRADevs, indexedDevice{i, d})
		}
	}

	if cfg.rejectMixed && len(nonDRADevs) > 0 && len(draDevs) > 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("vmi.spec.domain.devices.%s contains both DRA and non-DRA %ss; each %s must be either DRA or non-DRA", cfg.fieldPath, cfg.typeName, cfg.typeName),
			Field:   devField.String(),
		})
		return causes, sets.New[string]()
	}

	for _, nd := range nonDRADevs {
		if nd.device.getDeviceName() == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("vmi.spec.domain.devices.%s contains %ss without deviceName or claimRequest; each %s must specify either a deviceName or a claimRequest", cfg.fieldPath, cfg.typeName, cfg.typeName),
				Field:   devField.String(),
			})
		}
		if nd.device.getDeviceName() != "" && nd.device.getClaimRequest() != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("vmi.spec.domain.devices.%s contains %ss with both deviceName and claimRequest", cfg.fieldPath, cfg.typeName),
				Field:   devField.String(),
			})
		}
	}

	if len(draDevs) > 0 && !cfg.gateEnabled {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("vmi.spec.domain.devices.%s contains DRA enabled %ss but feature gate is not enabled", cfg.fieldPath, cfg.typeName),
			Field:   devField.String(),
		})
		return causes, sets.New[string]()
	}

	var validDRA []indexedDevice
	for _, id := range draDevs {
		valid := true
		cr := id.device.getClaimRequest()
		if cr.ClaimName == nil || *cr.ClaimName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("claimName is required for DRA %s", cfg.typeName),
				Field:   devField.Index(id.idx).Child("claimName").String(),
			})
			valid = false
		}
		if cr.RequestName == nil || *cr.RequestName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf("requestName is required for DRA %s", cfg.typeName),
				Field:   devField.Index(id.idx).Child("requestName").String(),
			})
			valid = false
		}
		if valid {
			validDRA = append(validDRA, id)
		}
	}

	claimRequestPairs := sets.New[string]()
	for _, id := range validDRA {
		cr := id.device.getClaimRequest()
		key := *cr.ClaimName + "/" + *cr.RequestName
		if claimRequestPairs.Has(key) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("duplicate claimName/requestName pair %q", key),
				Field:   devField.Index(id.idx).String(),
			})
		}
		claimRequestPairs.Insert(key)
	}

	claimNames := sets.New[string]()
	for _, id := range validDRA {
		claimNames.Insert(*id.device.getClaimRequest().ClaimName)
	}

	return causes, claimNames
}

func ValidateCreation(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, clusterCfg *virtconfig.ClusterConfig) []metav1.StatusCause {
	return NewValidator(field, vmiSpec, clusterCfg).ValidateCreation()
}
