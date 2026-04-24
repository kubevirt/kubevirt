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
package naming

import (
	"crypto/sha256"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation"
)

const hashLength = 8

func TruncateWithHash(value string, maxLength int) string {
	if len(value) <= maxLength {
		return value
	}
	hasher := sha256.New()
	hasher.Write([]byte(value))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	return fmt.Sprintf("%s-%s", value[:maxLength-hashLength-1], hash[:hashLength])
}

func TruncateLabelValue(value string) string {
	return TruncateWithHash(value, validation.LabelValueMaxLength)
}
