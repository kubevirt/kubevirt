/*
Copyright 2024 The KubeVirt Authors.

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

package apitesting

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	kubevirtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/api/apitesting/roundtrip"
)

var groups = []runtime.SchemeBuilder{
	kubevirtv1.SchemeBuilder,
	instancetypev1beta1.SchemeBuilder,
}

func TestCompatibility(t *testing.T) {
	scheme := runtime.NewScheme()

	for _, builder := range groups {
		if err := builder.AddToScheme(scheme); err != nil {
			t.Fatalf("failed to add to scheme: %v", err)
		}
	}

	roundtrip.NewCompatibilityTestOptions(scheme).Complete(t).Run(t)
}
