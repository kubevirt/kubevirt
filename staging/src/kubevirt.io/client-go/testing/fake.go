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

package testing

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/testing"
)

// FilterActions returns the actions which satisfy the passed verb and optionally the resource and subresource name.
func FilterActions(fake *testing.Fake, verb string, resources ...string) []testing.Action {
	var filtered []testing.Action
	for _, action := range fake.Actions() {
		if action.GetVerb() == verb {
			if len(resources) > 0 && action.GetResource().Resource != resources[0] {
				continue
			}
			if len(resources) > 1 && action.GetSubresource() != resources[1] {
				continue
			}
			filtered = append(filtered, action)
		}
	}
	return filtered
}

// PrependGenerateNameCreateReactor prepends a reactor to the specified resource
// that generates a name starting from the meta.generateName field.
func PrependGenerateNameCreateReactor(fake *testing.Fake, resourceName string) {
	fake.PrependReactor("create", resourceName, func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		ret = action.(testing.CreateAction).GetObject()
		meta, ok := ret.(metav1.Object)
		if !ok {
			return
		}

		if meta.GetName() == "" && meta.GetGenerateName() != "" {
			meta.SetName(names.SimpleNameGenerator.GenerateName(meta.GetGenerateName()))
		}

		return
	})
}
