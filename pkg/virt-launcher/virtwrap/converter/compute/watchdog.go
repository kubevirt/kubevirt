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

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type WatchdogDomainConfigurator struct {
	architecture string
}

func NewWatchdogDomainConfigurator(architecture string) WatchdogDomainConfigurator {
	return WatchdogDomainConfigurator{
		architecture: architecture,
	}
}

func (w WatchdogDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	vmiWatchdog := vmi.Spec.Domain.Devices.Watchdog
	if vmiWatchdog == nil {
		return nil
	}

	newWatchdog := api.Watchdog{}

	switch w.architecture {
	case "amd64":
		if vmiWatchdog.I6300ESB == nil {
			return fmt.Errorf("watchdog %s can't be mapped, no watchdog type specified", vmiWatchdog.Name)
		}

		newWatchdog.Alias = api.NewUserDefinedAlias(vmiWatchdog.Name)
		newWatchdog.Model = "i6300esb"
		newWatchdog.Action = string(vmiWatchdog.I6300ESB.Action)
	case "arm64":
		return fmt.Errorf("watchdog is not supported on architecture ARM64")
	case "s390x":
		if vmiWatchdog.Diag288 == nil {
			return fmt.Errorf("watchdog %s can't be mapped, no watchdog type specified", vmiWatchdog.Name)
		}

		newWatchdog.Alias = api.NewUserDefinedAlias(vmiWatchdog.Name)
		newWatchdog.Model = "diag288"
		newWatchdog.Action = string(vmiWatchdog.Diag288.Action)
	}

	domain.Spec.Devices.Watchdogs = append(domain.Spec.Devices.Watchdogs, newWatchdog)

	return nil
}
