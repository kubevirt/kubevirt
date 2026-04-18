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

func validateDiscontinuedBindings(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	for idx, ifaceSpec := range spec.Domain.Devices.Interfaces {
		if ifaceSpec.DeprecatedSlirp != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Slirp interface support has been discontinued since v1.3",
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("slirp").String(),
			})
		}
		if ifaceSpec.InterfaceBindingMethod.DeprecatedPasst != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Passt network binding has been discontinued since v1.3",
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("passt").String(),
			})
		}

		if ifaceSpec.InterfaceBindingMethod.DeprecatedMacvtap != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Macvtap network binding has been discontinued since v1.3",
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("macvtap").String(),
			})
		}
	}
	return causes
}
