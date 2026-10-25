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

package diskpaths

import (
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/util"
)

var podsBaseDir = util.KubeletPodsDir
var mountBaseDir = filepath.Join(util.VirtShareDir, "/container-disks")

// SetLocalDirectory sets the base directory for container disks.
// It contains no filesystem side-effects.
func SetLocalDirectory(dir string) {
	mountBaseDir = dir
}

// GetMountBaseDir returns the current base directory for container disk mounts.
func GetMountBaseDir() string {
	return mountBaseDir
}

// SetKubeletPodsDirectory sets the base directory for kubelet pods.
func SetKubeletPodsDirectory(dir string) {
	podsBaseDir = dir
}

// GetPodsBaseDir returns the current base directory for kubelet pods.
func GetPodsBaseDir() string {
	return podsBaseDir
}
