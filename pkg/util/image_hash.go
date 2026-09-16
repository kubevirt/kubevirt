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

package util

import (
	"crypto/sha256"
	"fmt"
)

// Duplicated from pkg/virt-operator/util to avoid a virt-handler -> virt-operator dependency.
const VirtHandlerImageEnvName = "VIRT_HANDLER_IMAGE"

// ImageHashLabelValue fingerprints an image reference for use as a label value
// (image references are too long and contain invalid characters).
func ImageHashLabelValue(image string) string {
	if image == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(image))
	return fmt.Sprintf("%x", sum)[:32]
}
