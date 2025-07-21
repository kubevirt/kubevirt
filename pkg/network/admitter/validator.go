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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package admitter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
)

type clusterConfigChecker interface {
	IsSlirpInterfaceEnabled() bool
	IsBridgeInterfaceOnPodNetworkEnabled() bool
	MacvtapEnabled() bool
	PasstEnabled() bool
	NetworkBindingPlugingsEnabled() bool
}

type Validator struct {
	field         *k8sfield.Path
	vmiSpec       *v1.VirtualMachineInstanceSpec
	configChecker clusterConfigChecker

	networkByName map[string]v1.Network
}

func NewValidator(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec, configChecker clusterConfigChecker) *Validator {
	return &Validator{
		field:         field,
		vmiSpec:       vmiSpec,
		configChecker: configChecker,
		networkByName: netvmispec.IndexNetworkSpecByName(vmiSpec.Networks),
	}
}

func (v Validator) Validate() []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validatePodNetwork(v.field, v.vmiSpec)...)
	causes = append(causes, validateSingleNetworkSource(v.field, v.vmiSpec)...)
	causes = append(causes, validateMultusNetworkSource(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfaceStateValue(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfaceBinding(v.field, v.vmiSpec, v.configChecker)...)
	causes = append(causes, validateSlirpBinding(v.field, v.vmiSpec, v.configChecker)...)
	causes = append(causes, validateNetworkNameUnique(v.field, v.vmiSpec)...)
	causes = append(causes, validateNetworksAssignedToInterfaces(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfaceNameUnique(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfacesAssignedToNetworks(v.field, v.vmiSpec)...)
	causes = append(causes, validateInterfacesFields(v.field, v.vmiSpec)...)

	return causes
}

func (v Validator) ValidateCreation() []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateCreationSlirpBinding(v.field, v.vmiSpec)...)

	return causes
}
