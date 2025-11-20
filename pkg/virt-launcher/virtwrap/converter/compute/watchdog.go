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

type watchdogConverter interface {
	ConvertWatchdog(source *v1.Watchdog, watchdog *api.Watchdog) error
}

type WatchdogDomainConfigurator struct {
	watchdogConverter watchdogConverter
}

func NewWatchdogDomainConfigurator(watchdogConverter watchdogConverter) WatchdogDomainConfigurator {
	return WatchdogDomainConfigurator{
		watchdogConverter: watchdogConverter,
	}
}

func (w WatchdogDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Devices.Watchdog != nil {
		newWatchdog := &api.Watchdog{}
		err := w.watchdogConverter.ConvertWatchdog(vmi.Spec.Domain.Devices.Watchdog, newWatchdog)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Watchdogs = append(domain.Spec.Devices.Watchdogs, *newWatchdog)
	}

	return nil
}
