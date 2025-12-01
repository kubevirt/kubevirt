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
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type ConsoleDomainConfigurator struct {
	serialConsoleLog bool
}

func NewConsoleDomainConfigurator(serialConsoleLog bool) ConsoleDomainConfigurator {
	return ConsoleDomainConfigurator{
		serialConsoleLog: serialConsoleLog,
	}
}

func (c ConsoleDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Devices.AutoattachSerialConsole == nil || *vmi.Spec.Domain.Devices.AutoattachSerialConsole {
		var serialPort uint = 0
		var serialType string = "serial"
		domain.Spec.Devices.Consoles = []api.Console{
			{
				Type: "pty",
				Target: &api.ConsoleTarget{
					Type: &serialType,
					Port: &serialPort,
				},
			},
		}

		socketPath := fmt.Sprintf("%s/%s/virt-serial%d", util.VirtPrivateDir, vmi.ObjectMeta.UID, serialPort)
		domain.Spec.Devices.Serials = []api.Serial{
			{
				Type: "unix",
				Target: &api.SerialTarget{
					Port: &serialPort,
				},
				Source: &api.SerialSource{
					Mode: "bind",
					Path: socketPath,
				},
			},
		}

		if c.serialConsoleLog {
			domain.Spec.Devices.Serials[0].Log = &api.SerialLog{
				File:   fmt.Sprintf("%s-log", socketPath),
				Append: "on",
			}
		}

	}

	return nil
}
