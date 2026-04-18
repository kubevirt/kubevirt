/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package apply

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func applyPreferenceAnnotations(annotations map[string]string, target metav1.Object) {
	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}

	targetAnnotations := target.GetAnnotations()
	for key, value := range annotations {
		if _, exists := targetAnnotations[key]; exists {
			continue
		}
		targetAnnotations[key] = value
	}
}
