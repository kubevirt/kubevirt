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
package apply

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

type annotations struct{}

func (annotations) Apply(
	_ *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	_ *v1beta1.VirtualMachinePreferenceSpec,
	_ *virtv1.VirtualMachineInstanceSpec,
	vmiMetadata *metav1.ObjectMeta,
) conflict.Conflicts {
	if instancetypeSpec == nil {
		return nil
	}

	if vmiMetadata.Annotations == nil {
		vmiMetadata.Annotations = make(map[string]string)
	}

	targetAnnotations := vmiMetadata.Annotations
	var conflicts conflict.Conflicts
	for key, value := range instancetypeSpec.Annotations {
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
