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

package watchdog

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type DeviceConfigurator interface {
	ConfigureWatchdogDevice(watchdogDevice *v1.Watchdog) (api.Watchdog, error)
}

func NewDeviceConfigurator(architecture string) DeviceConfigurator {
	switch architecture {
	case "amd64":
		return amd64WatchdogDeviceConfigurator{}
	case "arm64":
		return arm64WatchdogDeviceConfigurator{}
	case "s390x":
		return s390xWatchdogDeviceConfigurator{}
	default:
		return nil
	}
}

type amd64WatchdogDeviceConfigurator struct{}

func (amd64WatchdogDeviceConfigurator) ConfigureWatchdogDevice(watchdogDevice *v1.Watchdog) (api.Watchdog, error) {
	if watchdogDevice.I6300ESB == nil {
		return api.Watchdog{}, fmt.Errorf("watchdog %s can't be mapped, no watchdog type specified", watchdogDevice.Name)
	}

	return newWatchdog(watchdogDevice.Name, "i6300esb", string(watchdogDevice.I6300ESB.Action)), nil
}

type arm64WatchdogDeviceConfigurator struct{}

func (arm64WatchdogDeviceConfigurator) ConfigureWatchdogDevice(_ *v1.Watchdog) (api.Watchdog, error) {
	return api.Watchdog{}, fmt.Errorf("watchdog is not supported on architecture ARM64")
}

type s390xWatchdogDeviceConfigurator struct{}

func (s390xWatchdogDeviceConfigurator) ConfigureWatchdogDevice(watchdogDevice *v1.Watchdog) (api.Watchdog, error) {
	if watchdogDevice.Diag288 == nil {
		return api.Watchdog{}, fmt.Errorf("watchdog %s can't be mapped, no watchdog type specified", watchdogDevice.Name)
	}

	return newWatchdog(watchdogDevice.Name, "diag288", string(watchdogDevice.Diag288.Action)), nil
}

func newWatchdog(name, model, action string) api.Watchdog {
	return api.Watchdog{
		Alias:  api.NewUserDefinedAlias(name),
		Model:  model,
		Action: action,
	}
}
