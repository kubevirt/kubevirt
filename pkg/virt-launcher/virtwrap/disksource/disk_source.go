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

package disksource

import (
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type ResolvedDiskSource struct {
	sourcePath     string
	backendPath    string
	backendIsBlock bool
	hasOverlay     bool
}

func Resolve(d api.Disk) ResolvedDiskSource {
	rds := ResolvedDiskSource{}
	if d.Source.Dev != "" {
		rds.sourcePath = d.Source.Dev
		rds.backendPath = d.Source.Dev
		rds.backendIsBlock = true
	}
	if d.Source.File != "" {
		rds.sourcePath = d.Source.File
		rds.backendPath = d.Source.File
		rds.backendIsBlock = false
	}
	if d.Source.DataStore != nil && d.Source.DataStore.Source != nil {
		rds.hasOverlay = true
		if d.Source.DataStore.Source.Dev != "" {
			rds.backendPath = d.Source.DataStore.Source.Dev
			rds.backendIsBlock = true
		}
		if d.Source.DataStore.Source.File != "" {
			rds.backendPath = d.Source.DataStore.Source.File
			rds.backendIsBlock = false
		}
	}
	return rds
}

func (rds ResolvedDiskSource) SourcePath() string {
	return rds.sourcePath
}

func (rds ResolvedDiskSource) BackendPath() string {
	return rds.backendPath
}

func (rds ResolvedDiskSource) BackendIsBlock() bool {
	return rds.backendIsBlock
}

func (rds ResolvedDiskSource) HasOverlay() bool {
	return rds.hasOverlay
}

func (rds ResolvedDiskSource) IsHotplugDisk() bool {
	return strings.HasPrefix(rds.backendPath, v1.HotplugDiskDir)
}

func (rds ResolvedDiskSource) IsHotplugOrEmpty() bool {
	return rds.IsHotplugDisk() || rds.backendPath == ""
}
