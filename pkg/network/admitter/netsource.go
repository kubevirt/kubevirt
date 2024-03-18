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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

func validateSinglePodNetwork(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	if countPodNetworks(spec.Networks) > 1 {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: fmt.Sprintf("more than one interface is connected to a pod network in %s", field.Child("interfaces").String()),
			Field:   field.Child("interfaces").String(),
		}}
	}
	return nil
}

func countPodNetworks(networks []v1.Network) int {
	count := 0
	for _, net := range networks {
		if net.Pod != nil {
			count++
		}
	}
	return count
}

func validateSingleNetworkSource(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for idx, net := range spec.Networks {
		if net.Pod == nil && net.Multus == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "should have a network type",
				Field:   field.Child("networks").Index(idx).String(),
			})
		} else if net.Pod != nil && net.Multus != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "should have only one network type",
				Field:   field.Child("networks").Index(idx).String(),
			})
		}
	}
	return causes
}

func validateMultusNetworkSource(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	for idx, net := range spec.Networks {
		if net.Multus != nil && net.Multus.NetworkName == "" {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "CNI delegating plugin must have a networkName",
				Field:   field.Child("networks").Index(idx).String(),
			}}
		}
	}
	return nil
}
