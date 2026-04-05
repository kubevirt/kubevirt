/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
