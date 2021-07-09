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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package handler_launcher_com

import (
	"fmt"
	"sort"
)

func GetHighestCompatibleVersion(serverVersions []uint32, clientVersions []uint32) (uint32, error) {
	// sort serverversions descending
	sort.Slice(serverVersions, func(i, j int) bool { return serverVersions[i] > serverVersions[j] })
	for _, s := range serverVersions {
		for _, c := range clientVersions {
			if s == c {
				return s, nil
			}

		}
	}
	return 0, fmt.Errorf("no compatible version found, server: %v, client: %v", serverVersions, clientVersions)
}
