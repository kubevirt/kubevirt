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
