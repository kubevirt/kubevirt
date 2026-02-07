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

// CalculateVirtualMachineInstanceID calculates a stable and unique identifier for a VMI based on its name attribute.
// For VMI names longer than 63 characters, the name is a truncated and hashed to ensure uniqueness.
func CalculateVirtualMachineInstanceID(vmiName string) string {
	if len(vmiName) <= validation.DNS1035LabelMaxLength {
		return vmiName
	}

	const (
		hashLength             = 8
		vmiNamePrefixMaxLength = validation.DNS1035LabelMaxLength - hashLength - 1
	)

	truncatedVMIName := vmiName[:vmiNamePrefixMaxLength]

	hasher := sha1.New()
	hasher.Write([]byte(vmiName))
	vmiNameHash := fmt.Sprintf("%x", hasher.Sum(nil))

	return fmt.Sprintf("%s-%s", truncatedVMIName, vmiNameHash[:hashLength])
}
