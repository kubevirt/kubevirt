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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package v1alpha2

import (
	conversion "k8s.io/apimachinery/pkg/conversion"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

// Manually defined function to convert from pointer to value
func Convert_v1beta1_CPUPreferences_To_v1alpha2_CPUPreferences(in *v1beta1.CPUPreferences, out *CPUPreferences, s conversion.Scope) error {
	if in.PreferredCPUTopology != nil {
		out.PreferredCPUTopology = (PreferredCPUTopology)(*in.PreferredCPUTopology)
	}

	return autoConvert_v1beta1_CPUPreferences_To_v1alpha2_CPUPreferences(in, out, s)
}

// Manually defined function to convert from value to pointer
func Convert_v1alpha2_CPUPreferences_To_v1beta1_CPUPreferences(in *CPUPreferences, out *v1beta1.CPUPreferences, s conversion.Scope) error {
	if in.PreferredCPUTopology != "" {
		out.PreferredCPUTopology = (*v1beta1.PreferredCPUTopology)(&in.PreferredCPUTopology)
	}

	return autoConvert_v1alpha2_CPUPreferences_To_v1beta1_CPUPreferences(in, out, s)
}
