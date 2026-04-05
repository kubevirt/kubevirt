/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"
)

type clusterConfigChecker interface {
	IsBridgeInterfaceOnPodNetworkEnabled() bool
	PasstBindingEnabled() bool
}

type Validator struct {
	field         *k8sfield.Path
	vmiSpec       *v1.VirtualMachineInstanceSpec
	configChecker clusterConfigChecker
}

func NewValidator(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, configChecker clusterConfigChecker) *Validator {
	return &Validator{
		field:         field,
		vmiSpec:       vmiSpec,
		configChecker: configChecker,
	}
}

func (v Validator) Validate() []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateSinglePodNetwork(v.field, v.vmiSpec)...)
	causes = append(causes, validateSingleNetworkSource(v.field, v.vmiSpec)...)
	causes = append(causes, validateMultusNetworkSource(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfaceStateValue(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfaceBinding(v.field, v.vmiSpec, v.configChecker)...)
	causes = append(causes, validateNetworkNameUnique(v.field, v.vmiSpec)...)
	causes = append(causes, validateNetworksAssignedToInterfaces(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfaceNameUnique(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfacesAssignedToNetworks(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfacesFields(v.field, v.vmiSpec)...)

	return causes
}

func (v Validator) ValidateCreation() []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateDiscontinuedBindings(v.field, v.vmiSpec)...)

	return causes
}

func ValidateCreation(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, clusterCfg clusterConfigChecker) []metav1.StatusCause {
	networkValidator := NewValidator(field, vmiSpec, clusterCfg)
	return networkValidator.ValidateCreation()
}

func Validate(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, clusterCfg clusterConfigChecker) []metav1.StatusCause {
	netValidator := NewValidator(field, vmiSpec, clusterCfg)
	var statusCauses []metav1.StatusCause
	statusCauses = append(statusCauses, netValidator.ValidateCreation()...)
	statusCauses = append(statusCauses, netValidator.Validate()...)
	return statusCauses
}
