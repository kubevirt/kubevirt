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

import "k8s.io/client-go/testing"

// FilterActions returns the actions which satisfy the passed verb and resource name.
func FilterActions(fake *testing.Fake, verb, resourceName string) []testing.Action {
	var filtered []testing.Action
	for _, action := range fake.Actions() {
		if action.GetVerb() == verb && action.GetResource().Resource == resourceName {
			filtered = append(filtered, action)
		}
	}
	return filtered
}
