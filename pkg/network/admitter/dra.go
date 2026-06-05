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
	"slices"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type networkDRAConfigChecker interface {
	NetworkDevicesWithDRAGateEnabled() bool
}

func validateNetworkDevicesWithDRA(
	field *k8sfield.Path,
	spec *v1.VirtualMachineInstanceSpec,
	checker networkDRAConfigChecker,
) []metav1.StatusCause {
	if !vmispec.HasDRANetwork(spec.Networks) {
		return []metav1.StatusCause{}
	}

	if !checker.NetworkDevicesWithDRAGateEnabled() {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.networks contains DRA networks but NetworkDevicesWithDRA feature gate is not enabled",
			Field:   field.Child("networks").String(),
		}}
	}

	if slices.ContainsFunc(spec.Networks, func(network v1.Network) bool {
		return network.Multus != nil
	}) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "mixing Multus and DRA resourceClaim networks in the same VMI is not supported",
			Field:   field.Child("networks").String(),
		}}
	}

	var causes []metav1.StatusCause
	tupleIdxByKey := vmispec.ExtractDRANetworkClaimRequestTuples(spec)
	for idx, net := range spec.Networks {
		if net.ResourceClaim == nil {
			continue
		}

		causes = append(causes, validateDRANetworkInterfaceBinding(field, spec.Domain.Devices.Interfaces, net.Name)...)

		claimName, requestName := net.ResourceClaim.ClaimName, net.ResourceClaim.RequestName
		causes = append(causes, validateDRAClaimAndRequestNames(field, idx, claimName, requestName)...)

		if claimName == "" || requestName == "" {
			continue
		}

		causes = append(causes, validateDRANetworkClaimReference(field, idx, claimName, spec.ResourceClaims)...)
		causes = append(causes, validateDRAClaimRequestUniqueness(field, idx, claimName+"/"+requestName, tupleIdxByKey)...)
	}

	return causes
}

func validateDRANetworkClaimReference(
	field *k8sfield.Path,
	idx int,
	claimName string,
	resourceClaims []k8sv1.PodResourceClaim,
) []metav1.StatusCause {
	for _, rc := range resourceClaims {
		if rc.Name == claimName {
			return nil
		}
	}

	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueNotFound,
		Message: fmt.Sprintf("network references resourceClaim %q which is not defined in spec.resourceClaims", claimName),
		Field:   field.Child("networks").Index(idx).Child("resourceClaim", "claimName").String(),
	}}
}

func validateDRAClaimRequestUniqueness(
	field *k8sfield.Path,
	idx int,
	key string,
	tupleFirstIndexByKey map[string]int,
) []metav1.StatusCause {
	if firstIdx, exists := tupleFirstIndexByKey[key]; !exists || firstIdx == idx {
		return nil
	}
	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueDuplicate,
		Message: fmt.Sprintf("duplicate claimName/requestName combination %q", key),
		Field:   field.Child("networks").Index(idx).String(),
	}}
}

func validateDRAClaimAndRequestNames(field *k8sfield.Path, idx int, claimName, requestName string) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if claimName == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "claimName is required for DRA network",
			Field:   field.Child("networks").Index(idx).Child("resourceClaim", "claimName").String(),
		})
	}

	if requestName == "" {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "requestName is required for DRA network",
			Field:   field.Child("networks").Index(idx).Child("resourceClaim", "requestName").String(),
		})
	}

	return causes
}

func validateDRANetworkInterfaceBinding(field *k8sfield.Path, interfaces []v1.Interface, networkName string) []metav1.StatusCause {
	iface := vmispec.LookupInterfaceByName(interfaces, networkName)
	if iface == nil || iface.SRIOV != nil || iface.Binding != nil {
		return nil
	}
	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf("DRA network %q requires an SR-IOV or binding plugin interface", networkName),
		Field:   field.Child("domain", "devices", "interfaces").String(),
	}}
}
