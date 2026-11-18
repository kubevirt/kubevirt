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

package gvk

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"
)

// GenerateKubeVirtGroupVersionKind ensures a provided object registered with
// KubeVirt's generated scheme has GVK set correctly. This is required as
// client-go continues to return objects without TypeMeta set, as documented
// in https://github.com/kubernetes/client-go/issues/413
func GenerateKubeVirtGroupVersionKind(obj runtime.Object) (runtime.Object, error) {
	objCopy := obj.DeepCopyObject()
	gvks, _, err := generatedscheme.Scheme.ObjectKinds(objCopy)
	if err != nil {
		return nil, fmt.Errorf("could not get GroupVersionKind for object: %w", err)
	}
	objCopy.GetObjectKind().SetGroupVersionKind(gvks[0])

	return objCopy, nil
}
