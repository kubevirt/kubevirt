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
	"crypto/sha256"
	"fmt"
	"hash"

	"k8s.io/apimachinery/pkg/util/validation"
)

const truncateHashLength = 8

func truncateWithHasher(value string, maxLength int, hasher hash.Hash) string {
	if len(value) <= maxLength {
		return value
	}
	hasher.Write([]byte(value))
	h := fmt.Sprintf("%x", hasher.Sum(nil))
	return fmt.Sprintf("%s-%s", value[:maxLength-truncateHashLength-1], h[:truncateHashLength])
}

func TruncateWithHash(value string, maxLength int) string {
	return truncateWithHasher(value, maxLength, sha256.New())
}

func TruncateLabelValue(value string) string {
	return TruncateWithHash(value, validation.LabelValueMaxLength)
}

// CalculateVirtualMachineInstanceID calculates a stable and unique identifier for a VMI based on its name attribute.
// For VMI names longer than 63 characters, the name is truncated and hashed to ensure uniqueness.
// This uses SHA1 for backward compatibility with existing pod labels (shipped in v1.7.0).
func CalculateVirtualMachineInstanceID(vmiName string) string {
	return truncateWithHasher(vmiName, validation.DNS1035LabelMaxLength, sha1.New())
}
