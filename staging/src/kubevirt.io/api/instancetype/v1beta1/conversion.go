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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/conversion"

	v1 "kubevirt.io/api/core/v1"
	instancetype "kubevirt.io/api/instancetype"
)

// DeprecatedTopologyToNew maps deprecated PreferredCPUTopology values to their new equivalents.
var DeprecatedTopologyToNew = map[PreferredCPUTopology]instancetype.PreferredCPUTopology{
	DeprecatedPreferCores:   instancetype.Cores,
	DeprecatedPreferSockets: instancetype.Sockets,
	DeprecatedPreferThreads: instancetype.Threads,
	DeprecatedPreferSpread:  instancetype.Spread,
	DeprecatedPreferAny:     instancetype.Any,
}

// Convert_v1beta1_CPUPreferences_To_instancetype_CPUPreferences handles conversion of CPUPreferences,
// mapping deprecated PreferredCPUTopology values to their new equivalents.
func Convert_v1beta1_CPUPreferences_To_instancetype_CPUPreferences(in *CPUPreferences, out *instancetype.CPUPreferences, s conversion.Scope) error {
	if err := autoConvert_v1beta1_CPUPreferences_To_instancetype_CPUPreferences(in, out, s); err != nil {
		return err
	}

	// Convert deprecated topology values to new values
	if out.PreferredCPUTopology != nil {
		if newValue, ok := DeprecatedTopologyToNew[PreferredCPUTopology(*out.PreferredCPUTopology)]; ok {
			out.PreferredCPUTopology = &newValue
		}
	}

	return nil
}

// Convert_v1beta1_FirmwarePreferences_To_instancetype_FirmwarePreferences handles conversion of FirmwarePreferences,
// converting deprecated DeprecatedPreferredUseEfi and DeprecatedPreferredUseSecureBoot to PreferredEfi.
func Convert_v1beta1_FirmwarePreferences_To_instancetype_FirmwarePreferences(in *FirmwarePreferences, out *instancetype.FirmwarePreferences, s conversion.Scope) error {
	if err := autoConvert_v1beta1_FirmwarePreferences_To_instancetype_FirmwarePreferences(in, out, s); err != nil {
		return err
	}

	// Convert deprecated fields to PreferredEfi if PreferredEfi is not already set
	if out.PreferredEfi == nil {
		if in.DeprecatedPreferredUseEfi != nil && *in.DeprecatedPreferredUseEfi {
			out.PreferredEfi = &v1.EFI{}
			if in.DeprecatedPreferredUseSecureBoot != nil {
				out.PreferredEfi.SecureBoot = in.DeprecatedPreferredUseSecureBoot
			}
		}
	}

	return nil
}
