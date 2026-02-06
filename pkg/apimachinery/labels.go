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

package apimachinery

import (
	"crypto/sha1"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation"
)

// CalculateValidUniqueID calculates a stable and unique identifier for a resource based on its name attribute.
// For names longer than 63 characters, the name is a truncated and hashed to ensure uniqueness.
func CalculateValidUniqueID(resName string) string {
	if len(resName) <= validation.DNS1035LabelMaxLength {
		return resName
	}

	const (
		hashLength             = 8
		resNamePrefixMaxLength = validation.DNS1035LabelMaxLength - hashLength - 1
	)

	truncatedResName := resName[:resNamePrefixMaxLength]

	hasher := sha1.New()
	hasher.Write([]byte(resName))
	resNameHash := fmt.Sprintf("%x", hasher.Sum(nil))

	return fmt.Sprintf("%s-%s", truncatedResName, resNameHash[:hashLength])
}
