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

package hardware

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

const pciHole64KiB = int64(1024)

func PCIHole64SizeToKiB(size resource.Quantity) (uint, error) {
	bytes := size.Value()
	if bytes <= 0 {
		return 0, fmt.Errorf("pcihole64 size must be greater than zero")
	}

	kib := (uint64(bytes) + uint64(pciHole64KiB) - 1) / uint64(pciHole64KiB)
	if kib > uint64(^uint(0)) {
		return 0, fmt.Errorf("pcihole64 size %s is too large", size.String())
	}

	return uint(kib), nil
}
