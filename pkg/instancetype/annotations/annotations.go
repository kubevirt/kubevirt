/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package annotations

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	api "kubevirt.io/api/instancetype"
)

func Set(vm *virtv1.VirtualMachine, target metav1.Object) {
	if vm.Spec.Instancetype == nil {
		return
	}

	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}
	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case api.PluralResourceName, api.SingularResourceName:
		target.GetAnnotations()[virtv1.InstancetypeAnnotation] = vm.Spec.Instancetype.Name
	case "", api.ClusterPluralResourceName, api.ClusterSingularResourceName:
		target.GetAnnotations()[virtv1.ClusterInstancetypeAnnotation] = vm.Spec.Instancetype.Name
	}
}
