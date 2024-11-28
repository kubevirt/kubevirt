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
)

type clusterConfigChecker interface {
	IsSlirpInterfaceEnabled() bool
	IsBridgeInterfaceOnPodNetworkEnabled() bool
	MacvtapEnabled() bool
	PasstEnabled() bool
}

type Validator struct {
	configChecker clusterConfigChecker
}

func NewValidator(configChecker clusterConfigChecker) *Validator {
	return &Validator{configChecker: configChecker}
}

func (v Validator) Validate(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateSinglePodNetwork(field, vmiSpec)...)
	causes = append(causes, validateSingleNetworkSource(field, vmiSpec)...)
	causes = append(causes, validateMultusNetworkSource(field, vmiSpec)...)
	causes = append(causes, validateInterfaceStateValue(field, vmiSpec)...)
	causes = append(causes, validateInterfaceBinding(field, vmiSpec, v.configChecker)...)
	causes = append(causes, validateSlirpBinding(field, vmiSpec, v.configChecker)...)
	causes = append(causes, validateNetworkNameUnique(field, vmiSpec)...)
	causes = append(causes, validateNetworksAssignedToInterfaces(field, vmiSpec)...)
	causes = append(causes, validateInterfaceNameUnique(field, vmiSpec)...)
	causes = append(causes, validateInterfacesAssignedToNetworks(field, vmiSpec)...)
	causes = append(causes, validateInterfacesFields(field, vmiSpec)...)

	return causes
}

func (v Validator) ValidateCreation(field *k8sfield.Path, vmiSpec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	causes = append(causes, validateCreationSlirpBinding(field, vmiSpec)...)

	return causes
}
