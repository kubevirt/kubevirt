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
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func validateInterfaceBinding(
	fieldPath *field.Path, spec *v1.VirtualMachineInstanceSpec, config clusterConfigChecker,
) []metav1.StatusCause {
	var causes []metav1.StatusCause
	networksByName := vmispec.IndexNetworkSpecByName(spec.Networks)
	for idx, iface := range spec.Domain.Devices.Interfaces {
		causes = append(causes, validateInterfaceBindingExists(fieldPath, idx, iface)...)
		causes = append(causes, validateMasqueradeBinding(fieldPath, idx, iface, networksByName[iface.Name])...)
		causes = append(causes, validateBridgeBinding(fieldPath, idx, iface, networksByName[iface.Name], config)...)
		causes = append(causes, validateMacvtapBinding(fieldPath, idx, iface, networksByName[iface.Name], config)...)
		causes = append(causes, validatePasstBinding(fieldPath, idx, iface, networksByName[iface.Name], config)...)
	}
	return causes
}

func validateInterfaceBindingExists(fieldPath *field.Path, idx int, iface v1.Interface) []metav1.StatusCause {
	if iface.Binding != nil && hasInterfaceBindingMethod(iface) {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("logical %s interface cannot have both binding plugin and interface binding method", iface.Name),
			Field:   fieldPath.Child("domain", "devices", "interfaces").Index(idx).Child("binding").String(),
		}}
	}
	return nil
}

func hasInterfaceBindingMethod(iface v1.Interface) bool {
	return iface.InterfaceBindingMethod.Bridge != nil ||
		iface.InterfaceBindingMethod.DeprecatedSlirp != nil ||
		iface.InterfaceBindingMethod.Masquerade != nil ||
		iface.InterfaceBindingMethod.SRIOV != nil ||
		iface.InterfaceBindingMethod.DeprecatedMacvtap != nil ||
		iface.InterfaceBindingMethod.DeprecatedPasst != nil
}

func validateMasqueradeBinding(fieldPath *field.Path, idx int, iface v1.Interface, net v1.Network) []metav1.StatusCause {
	var causes []metav1.StatusCause
	if iface.Masquerade != nil && net.Pod == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Masquerade interface only implemented with pod network",
			Field:   fieldPath.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
		})
	}
	if iface.Masquerade != nil && link.IsReserved(iface.MacAddress) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "The requested MAC address is reserved for the in-pod bridge. Please choose another one.",
			Field:   fieldPath.Child("domain", "devices", "interfaces").Index(idx).Child("macAddress").String(),
		})
	}
	return causes
}

func validateBridgeBinding(
	fieldPath *field.Path, idx int, iface v1.Interface, net v1.Network, config clusterConfigChecker,
) []metav1.StatusCause {
	if iface.InterfaceBindingMethod.Bridge != nil && net.NetworkSource.Pod != nil && !config.IsBridgeInterfaceOnPodNetworkEnabled() {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "Bridge on pod network configuration is not enabled under kubevirt-config",
			Field:   fieldPath.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
		}}
	}
	return nil
}
