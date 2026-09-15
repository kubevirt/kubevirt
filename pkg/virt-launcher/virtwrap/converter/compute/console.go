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
	architecture        string
}

func NewConsoleDomainConfigurator(useSerialConsoleLog bool, architecture string) ConsoleDomainConfigurator {
	return ConsoleDomainConfigurator{
		useSerialConsoleLog: useSerialConsoleLog,
		architecture:        architecture,
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
		consoleSCLPType = "sclp"
		serialTypeUnix  = "unix"
		bindMode        = "bind"
		logAppend       = "on"
	)

	socketPath := fmt.Sprintf("%s/%s/virt-serial%d", util.VirtPrivateDir, vmi.ObjectMeta.UID, serialPortIndex)

	// On s390x, libvirt maps both <serial> and <console> devices to the SCLP console.
	// When a <serial type="unix"> is submitted, libvirt automatically creates a <console>
	// alias and copies the <log> element to it, causing every log line to be written twice.
	// To avoid this, on s390x we submit only a <console type="unix"> with target type "sclp".
	// Libvirt does NOT create a reverse serial alias in that case, so there is only one
	// logging device.
	if c.architecture == "s390x" {
		console := api.Console{
			Type: serialTypeUnix,
			Source: &api.ConsoleSource{
				Mode: bindMode,
				Path: socketPath,
			},
			Target: &api.ConsoleTarget{
				Type: pointer.P(consoleSCLPType),
				Port: pointer.P(serialPortIndex),
			},
		}
		if c.useSerialConsoleLog {
			console.Log = &api.SerialLog{
				File:   fmt.Sprintf("%s-log", socketPath),
				Append: logAppend,
			}
		}
		domain.Spec.Devices.Consoles = []api.Console{console}
		return nil
	}

	domain.Spec.Devices.Consoles = []api.Console{
		{
			Type: consoleType,
			Target: &api.ConsoleTarget{
				Type: pointer.P(serialType),
				Port: pointer.P(serialPortIndex),
			},
		},
	}

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
