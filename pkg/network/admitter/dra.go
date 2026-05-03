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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type networkDRAConfigChecker interface {
	NetworkDevicesWithDRAGateEnabled() bool
}

func validateNetworkDevicesWithDRA(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec, checker networkDRAConfigChecker) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if !vmispec.HasNetworkDRA(spec.Networks) {
		return causes
	}

	if !checker.NetworkDevicesWithDRAGateEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "NetworkDevicesWithDRA feature gate is not enabled in kubevirt-config",
			Field:   field.Child("networks").String(),
		})
		return causes
	}

	hasMultusNetwork := false
	tupleFirstIndexByKey := ExtractDRANetworkClaimRequestTupleFirstIndex(spec, checker)
	for idx, net := range spec.Networks {
		if net.ResourceClaim == nil {
			if net.Multus != nil {
				hasMultusNetwork = true
			}
			continue
		}

		causes = append(causes, validateDRANetworkSRIOVBinding(field, spec.Domain.Devices.Interfaces, net.Name)...)

		claimName, requestName := draClaimAndRequestNames(net)
		causes = append(causes, validateDRAClaimAndRequestNames(field, idx, claimName, requestName)...)

		if claimName == "" || requestName == "" {
			continue
		}

		causes = append(causes, validateDRANetworkClaimReference(field, idx, claimName, spec.ResourceClaims)...)
		causes = append(causes, validateDRAClaimRequestUniqueness(field, idx, claimName+"/"+requestName, tupleFirstIndexByKey)...)
	}

	if hasMultusNetwork {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "mixing Multus and DRA resourceClaim networks in the same VMI is not supported",
			Field:   field.Child("networks").String(),
		})
	}

	return causes
}

// ExtractDRANetworkClaimRequestTupleFirstIndex returns the first network index
// for each valid <claimName,requestName> tuple.
func ExtractDRANetworkClaimRequestTupleFirstIndex(spec *v1.VirtualMachineInstanceSpec, checker networkDRAConfigChecker) map[string]int {
	validPairFirstIndexByKey := map[string]int{}
	if !checker.NetworkDevicesWithDRAGateEnabled() {
		return validPairFirstIndexByKey
	}

	for idx, net := range spec.Networks {
		if net.ResourceClaim == nil ||
			net.ResourceClaim.ClaimName == nil || *net.ResourceClaim.ClaimName == "" ||
			net.ResourceClaim.RequestName == nil || *net.ResourceClaim.RequestName == "" {
			continue
		}

		key := *net.ResourceClaim.ClaimName + "/" + *net.ResourceClaim.RequestName
		if _, exists := validPairFirstIndexByKey[key]; !exists {
			validPairFirstIndexByKey[key] = idx
		}
	}

	return validPairFirstIndexByKey
}

func resourceClaimExists(resourceClaims []k8sv1.PodResourceClaim, claimName string) bool {
	for _, rc := range resourceClaims {
		if rc.Name == claimName {
			return true
		}
	}
	return false
}

func validateDRANetworkClaimReference(field *k8sfield.Path, idx int, claimName string, resourceClaims []k8sv1.PodResourceClaim) []metav1.StatusCause {
	if resourceClaimExists(resourceClaims, claimName) {
		return nil
	}
	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueNotFound,
		Message: fmt.Sprintf("network references resourceClaim %q which is not defined in spec.resourceClaims", claimName),
		Field:   field.Child("networks").Index(idx).Child("resourceClaim", "claimName").String(),
	}}
}

func validateDRAClaimRequestUniqueness(field *k8sfield.Path, idx int, key string, tupleFirstIndexByKey map[string]int) []metav1.StatusCause {
	if firstIdx, exists := tupleFirstIndexByKey[key]; !exists || firstIdx == idx {
		return nil
	}
	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueDuplicate,
		Message: fmt.Sprintf("duplicate claimName/requestName combination %q", key),
		Field:   field.Child("networks").Index(idx).String(),
	}}
}

func draClaimAndRequestNames(net v1.Network) (string, string) {
	var claimName string
	if net.ResourceClaim.ClaimName != nil {
		claimName = *net.ResourceClaim.ClaimName
	}

	var requestName string
	if net.ResourceClaim.RequestName != nil {
		requestName = *net.ResourceClaim.RequestName
	}

	return claimName, requestName
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

func validateDRANetworkSRIOVBinding(field *k8sfield.Path, interfaces []v1.Interface, networkName string) []metav1.StatusCause {
	iface := vmispec.LookupInterfaceByName(interfaces, networkName)
	if iface == nil || iface.SRIOV != nil {
		return nil
	}
	return []metav1.StatusCause{{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: fmt.Sprintf("DRA network %q requires an SR-IOV interface binding", networkName),
		Field:   field.Child("domain", "devices", "interfaces").String(),
	}}
}
