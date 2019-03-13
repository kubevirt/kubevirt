/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mergeLabelsAndAnnotations(src, dest metav1.Object) {
	// allow users to add labels but not change ours
	for k, v := range src.GetLabels() {
		if dest.GetLabels() == nil {
			dest.SetLabels(map[string]string{})
		}
		_, exists := dest.GetLabels()[k]
		if !exists {
			dest.GetLabels()[k] = v
		}
	}

	// same for annotations
	for k, v := range src.GetAnnotations() {
		if dest.GetAnnotations() == nil {
			dest.SetAnnotations(map[string]string{})
		}
		_, exists := dest.GetAnnotations()[k]
		if !exists {
			dest.GetAnnotations()[k] = v
		}
	}
}

func deployClusterResources() bool {
	return strings.ToLower(os.Getenv("DEPLOY_CLUSTER_RESOURCES")) != "false"
}
