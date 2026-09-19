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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

const DependencyWarningPattern = "feature gate %s depends on feature gate %s, which is not enabled"

// WarnMissingDependencies returns a warning for every enabled feature gate that
// declares a dependency on a feature gate which is not enabled
func WarnMissingDependencies(devConfig *v1.DeveloperConfiguration) []string {
	var warnings []string
	for _, fg := range GetRegisteredFeatureGates() {
		if !IsEnabled(fg.Name, devConfig) {
			continue
		}
		for _, dep := range fg.Dependencies {
			if !IsEnabled(dep, devConfig) {
				warnings = append(warnings, fmt.Sprintf(DependencyWarningPattern, fg.Name, dep))
			}
		}
	}
	return warnings
}

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
