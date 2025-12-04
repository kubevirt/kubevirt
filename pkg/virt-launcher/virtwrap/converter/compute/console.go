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

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type ConsoleDomainConfigurator struct {
	useSerialConsoleLog bool
}

func NewConsoleDomainConfigurator(useSerialConsoleLog bool) ConsoleDomainConfigurator {
	return ConsoleDomainConfigurator{
		useSerialConsoleLog: useSerialConsoleLog,
	}
}

func (c ConsoleDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Devices.AutoattachSerialConsole != nil && !*vmi.Spec.Domain.Devices.AutoattachSerialConsole {
		return nil
	}

	const (
		serialPortIndex = uint(0)
		serialType      = "serial"
		consoleType     = "pty"
		serialTypeUnix  = "unix"
		bindMode        = "bind"
		logAppend       = "on"
	)

	domain.Spec.Devices.Consoles = []api.Console{
		{
			Type: consoleType,
			Target: &api.ConsoleTarget{
				Type: pointer.P(serialType),
				Port: pointer.P(serialPortIndex),
			},
		},
	}

	socketPath := fmt.Sprintf("%s/%s/virt-serial%d", util.VirtPrivateDir, vmi.ObjectMeta.UID, serialPortIndex)
	serial := api.Serial{
		Type: serialTypeUnix,
		Target: &api.SerialTarget{
			Port: pointer.P(serialPortIndex),
		},
		Source: &api.SerialSource{
			Mode: bindMode,
			Path: socketPath,
		},
	}

	if c.useSerialConsoleLog {
		serial.Log = &api.SerialLog{
			File:   fmt.Sprintf("%s-log", socketPath),
			Append: logAppend,
		}
	}

	domain.Spec.Devices.Serials = []api.Serial{serial}

	return nil
}
