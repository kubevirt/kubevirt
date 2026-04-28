/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitter

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/vmispec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

func validateInterfaceStateValue(field *k8sfield.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for idx, iface := range spec.Domain.Devices.Interfaces {
		if iface.State != "" &&
			iface.State != v1.InterfaceStateAbsent &&
			iface.State != v1.InterfaceStateLinkDown &&
			iface.State != v1.InterfaceStateLinkUp {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("logical %s interface state value is unsupported: %s", iface.Name, iface.State),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("state").String(),
			})
		}

		if iface.SRIOV != nil &&
			(iface.State == v1.InterfaceStateLinkDown || iface.State == v1.InterfaceStateLinkUp) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%q interface's state %q is not supported for SR-IOV NICs", iface.Name, iface.State),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("state").String(),
			})
		}

		if iface.State == v1.InterfaceStateAbsent && iface.Bridge == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%q interface's state %q is supported only for bridge binding", iface.Name, iface.State),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("state").String(),
			})
		}
		defaultNetwork := vmispec.LookUpDefaultNetwork(spec.Networks)
		if iface.State == v1.InterfaceStateAbsent && defaultNetwork != nil && defaultNetwork.Name == iface.Name {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%q interface's state %q is not supported on default networks", iface.Name, iface.State),
				Field:   field.Child("domain", "devices", "interfaces").Index(idx).Child("state").String(),
			})
		}
	}
	return causes
}
