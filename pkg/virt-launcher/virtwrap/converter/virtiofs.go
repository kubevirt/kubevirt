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
 */

package converter

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

func convertFileSystems(fileSystems []v1.Filesystem) []api.FilesystemDevice {
	domainFileSystems := []api.FilesystemDevice{}
	for _, fs := range fileSystems {
		if fs.Virtiofs == nil {
			continue
		}

		domainFileSystems = append(domainFileSystems,
			api.FilesystemDevice{
				Type:       "mount",
				AccessMode: "passthrough",
				Driver: &api.FilesystemDriver{
					Type:  "virtiofs",
					Queue: "1024",
				},
				Source: &api.FilesystemSource{
					Socket: virtiofs.VirtioFSSocketPath(fs.Name),
				},
				Target: &api.FilesystemTarget{
					Dir: fs.Name,
				},
			})
	}

	return domainFileSystems
}
