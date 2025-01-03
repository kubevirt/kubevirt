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
 * Copyright The KubeVirt Authors
 *
 */
package apply

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

func applyInstanceTypeAnnotations(annotations map[string]string, target metav1.Object) (conflicts conflict.Conflicts) {
	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}

	targetAnnotations := target.GetAnnotations()
	for key, value := range annotations {
		if targetValue, exists := targetAnnotations[key]; exists {
			if targetValue != value {
				conflicts = append(conflicts, conflict.New("annotations", key))
			}
			continue
		}
		targetAnnotations[key] = value
	}

	return conflicts
}
