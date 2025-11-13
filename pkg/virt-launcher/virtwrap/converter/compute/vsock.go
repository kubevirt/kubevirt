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

package compute

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type VSOCKDomainConfigurator struct{}

func (v VSOCKDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	vsockCID := vmi.Status.VSOCKCID
	if vsockCID == nil {
		return nil
	}

	domain.Spec.Devices.VSOCK = &api.VSOCK{
		// Force virtio v1 for vhost-vsock-pci.
		// https://gitlab.com/qemu-project/qemu/-/commit/6209070503989cf4f28549f228989419d4f0b236
		Model: "virtio-non-transitional",
		CID: api.CID{
			Auto:    "no",
			Address: *vsockCID,
		},
	}

	return nil
}
