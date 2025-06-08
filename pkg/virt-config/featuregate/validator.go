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

package featuregate

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	v1 "kubevirt.io/api/core/v1"
)

func ValidateFeatureGates(featureGates []string, vmiSpec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	for _, fgName := range featureGates {
		fg := FeatureGateInfo(fgName)
		if fg != nil && fg.State == Discontinued && fg.VmiSpecUsed != nil {
			if used := fg.VmiSpecUsed(vmiSpec); used {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: fg.Message,
				})
			}
		}
	}
	return causes
}

func GetEnabledFeatureGates(featureGates []v1.FeatureGateConfiguration, legacyFeatureGates []string) []string {
	enabledFeatureGates := sets.New[string](legacyFeatureGates...)

	for _, fgConfig := range featureGates {
		if fgConfig.IsEnabled() {
			enabledFeatureGates.Insert(fgConfig.Name)
		} else {
			enabledFeatureGates.Delete(fgConfig.Name)
		}
	}

	if len(enabledFeatureGates) == 0 {
		return nil
	}

	enabledFeatureGatesSlice := enabledFeatureGates.UnsortedList()

	// Sort the feature gates to ensure a consistent order.
	// This is important for any comparison logic, as we want ["fg1", "fg2"] to be considered equal to ["fg2", "fg1"].
	// Without this, the test suite, as well as out-of-tree components would need to re-implement the sorting logic.
	// In addition, since this list is expected to be very small, the performance impact of sorting is negligible.
	slices.Sort(enabledFeatureGatesSlice)

	return enabledFeatureGatesSlice
}
