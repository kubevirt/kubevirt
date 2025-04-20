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
 */

package cleanup

import (
	"fmt"
)

const (
	KubeVirtTestLabelPrefix = "test.kubevirt.io"
)

// TestLabelForNamespace is used to mark non-namespaces resources with a label bound to a test namespace.
// This will be used to clean up non-namespaced resources after a test case was executed.
func TestLabelForNamespace(namespace string) string {
	return fmt.Sprintf("%s/%s", KubeVirtTestLabelPrefix, namespace)
}
