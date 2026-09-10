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

package disksize

import "kubevirt.io/client-go/log"

// AlignImageSizeTo1MiB rounds down size to the nearest multiple of 1 MiB.
// A warning or error is logged when the size is not already aligned.
// The caller is responsible for ensuring the rounded-down size is not 0.
func AlignImageSizeTo1MiB(size int64, logger *log.FilteredLogger) int64 {
	remainder := size % (1024 * 1024)
	if remainder == 0 {
		return size
	}
	newSize := size - remainder
	if logger != nil {
		if newSize == 0 {
			logger.Errorf("disks must be at least 1MiB, %d bytes is too small", size)
		} else {
			logger.V(4).Infof("disk size is not 1MiB-aligned. Adjusting from %d down to %d.", size, newSize)
		}
	}
	return newSize
}
