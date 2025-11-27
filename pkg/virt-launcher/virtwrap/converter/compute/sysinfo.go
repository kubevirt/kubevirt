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

type SysInfoDomainConfigurator struct{}

func (s SysInfoDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	domain.Spec.SysInfo = &api.SysInfo{}

	domain.Spec.SysInfo.System = buildSystem(vmi.Spec.Domain.Firmware)

	return nil
}

func buildSystem(firmware *v1.Firmware) []api.Entry {
	var systemEntries []api.Entry

	if firmware != nil {
		systemEntries = []api.Entry{{Name: "uuid", Value: string(firmware.UUID)}}

		if len(firmware.Serial) > 0 {
			systemEntries = append(systemEntries, api.Entry{Name: "serial", Value: firmware.Serial})
		}
	}

	return systemEntries
}
