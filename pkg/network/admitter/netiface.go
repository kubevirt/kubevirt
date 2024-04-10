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
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

func validateNetworksAssignedToInterfaces(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	const nameOfTypeNotFoundMessagePattern = "%s '%s' not found."
	interfaceSet := vmispec.IndexInterfaceSpecByName(spec.Domain.Devices.Interfaces)
	for i, network := range spec.Networks {
		if _, exists := interfaceSet[network.Name]; !exists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: fmt.Sprintf(nameOfTypeNotFoundMessagePattern, field.Child("networks").Index(i).Child("name").String(), network.Name),
				Field:   field.Child("networks").Index(i).Child("name").String(),
			})
		}
	}
	return causes
}

func validateNetworkNameUnique(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	networkSet := map[string]struct{}{}
	for i, network := range spec.Networks {
		if _, exists := networkSet[network.Name]; exists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("Network with name %q already exists, every network must have a unique name", network.Name),
				Field:   field.Child("networks").Index(i).Child("name").String(),
			})
		}
		networkSet[network.Name] = struct{}{}
	}
	return causes
}
