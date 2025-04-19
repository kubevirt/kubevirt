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
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func validateSinglePodNetwork(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	podNetworks := vmispec.FilterNetworksSpec(spec.Networks, func(n v1.Network) bool {
		return n.Pod != nil
	})
	if len(podNetworks) > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: fmt.Sprintf("more than one interface is connected to a pod network in %s", field.Child("interfaces").String()),
			Field:   field.Child("interfaces").String(),
		})
	}

	multusDefaultNetworks := vmispec.FilterNetworksSpec(spec.Networks, func(n v1.Network) bool {
		return n.Multus != nil && n.Multus.Default
	})
	if len(multusDefaultNetworks) > 1 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Multus CNI should only have one default network",
			Field:   field.Child("networks").String(),
		})
	}

	if len(podNetworks) > 0 && len(multusDefaultNetworks) > 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Pod network cannot be defined when Multus default network is defined",
			Field:   field.Child("networks").String(),
		})
	}
	return causes
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
