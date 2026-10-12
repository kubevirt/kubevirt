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

package disk

import (
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

// Hook rewrites disk paths for the target, then creates the CBT overlays
// the rewritten domain references.
func Hook(c *convertertypes.ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if err := DiskSourcePathHook(c, vmi, domain); err != nil {
		return err
	}
	return CBTOverlayHook(c, vmi, domain)
}
