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

package export

type ExportMapExtent struct {
	Offset      uint64 `json:"offset"`
	Length      uint64 `json:"length"`
	Type        uint64 `json:"type"`
	Description string `json:"description"`
}

type ExportMapResponse struct {
	Extents    []ExportMapExtent `json:"extents"`
	NextOffset *uint64           `json:"next_offset"`
}
