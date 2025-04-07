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
 * Copyright the KubeVirt Authors.
 *
 */

package libvmi

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

// WithTablet adds tablet device with given name and bus
func WithTablet(name string, bus v1.InputBus) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Inputs = append(vmi.Spec.Domain.Devices.Inputs,
			v1.Input{
				Name: name,
				Bus:  bus,
				Type: v1.InputTypeTablet,
			},
		)
	}
}

func WithAutoattachGraphicsDevice(enable bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &enable
	}
}

// WithRng adds `rng` to the vmi devices.
func WithRng() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
	}
}

// WithWatchdog adds a watchdog to the vmi devices.
func WithWatchdog(action v1.WatchdogAction, arch string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		watchdog := &v1.Watchdog{
			Name: "watchdog",
		}
		if arch == "s390x" {
			watchdog.WatchdogDevice.Diag288 = &v1.Diag288Watchdog{Action: action}
		} else {
			watchdog.WatchdogDevice.I6300ESB = &v1.I6300ESBWatchdog{Action: action}
		}

		vmi.Spec.Domain.Devices.Watchdog = watchdog
	}
}

func WithDownwardMetricsVolume(volumeName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: volumeName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.DiskBusVirtio,
				},
			},
		})
	}
}

func WithDownwardMetricsChannel() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.DownwardMetrics = &v1.DownwardMetrics{}
	}
}

func WithoutSerialConsole() Option {
	return func(vmi *v1.VirtualMachineInstance) {
		enabled := false
		vmi.Spec.Domain.Devices.AutoattachSerialConsole = &enabled
	}
}

func WithTPM(persistent bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
			Persistent: pointer.P(persistent),
		}
	}
}
