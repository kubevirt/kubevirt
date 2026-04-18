/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
