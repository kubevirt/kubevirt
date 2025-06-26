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

package virthandler

import (
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

func passtSocketDirOnHost(path isolation.IsolationResult) (string, error) {
	root, err := path.MountRoot()
	if err != nil {
		return "", err
	}

	const repairSocketDir = "var/run/libvirt/qemu/run/passt"
	safePath, err := root.AppendAndResolveWithRelativeRoot(repairSocketDir)
	if err != nil {
		return "", err
	}

	return unsafepath.UnsafeAbsolute(safePath.Raw()), nil
}
