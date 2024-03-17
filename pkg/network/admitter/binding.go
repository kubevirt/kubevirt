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
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

func validateInterfaceBinding(fieldPath *field.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for idx, iface := range spec.Domain.Devices.Interfaces {
		if iface.Binding != nil {
			if hasInterfaceBindingMethod(iface) {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("logical %s interface cannot have both binding plugin and interface binding method", iface.Name),
					Field:   fieldPath.Child("domain", "devices", "interfaces").Index(idx).Child("binding").String(),
				})
			}
		}
	}
	return causes
}

func hasInterfaceBindingMethod(iface v1.Interface) bool {
	return iface.InterfaceBindingMethod.Bridge != nil ||
		iface.InterfaceBindingMethod.Slirp != nil ||
		iface.InterfaceBindingMethod.Masquerade != nil ||
		iface.InterfaceBindingMethod.SRIOV != nil ||
		iface.InterfaceBindingMethod.Macvtap != nil ||
		iface.InterfaceBindingMethod.Passt != nil
}
