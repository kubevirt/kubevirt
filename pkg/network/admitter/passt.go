/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
)

func validatePasstBinding(
	fieldPath *field.Path, idx int, iface v1.Interface, net v1.Network, config clusterConfigChecker,
) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if iface.PasstBinding != nil && !config.PasstBindingEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "PasstBinding feature gate is not enabled",
			Field:   fieldPath.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
		})
	}

	if iface.PasstBinding != nil && net.Pod == nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "PasstBinding interface only implemented with pod network",
			Field:   fieldPath.Child("domain", "devices", "interfaces").Index(idx).Child("name").String(),
		})
	}

	return causes
}
