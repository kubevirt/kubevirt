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

package domainstats

import (
	"fmt"
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var guestDeviceDriverDateSeconds = operatormetrics.NewGauge(
	operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_device_driver_date_seconds",
		Help: "Timestamp of the VirtIO device driver date as reported by the guest agent, in seconds since epoch.",
	},
)

type deviceMetrics struct{}

func (deviceMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		guestDeviceDriverDateSeconds,
	}
}

func (deviceMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	if !settings.clusterConfig.GuestDeviceMetricsEnabled() {
		return nil
	}

	var crs []operatormetrics.CollectorResult
	seen := map[string]struct{}{}
	for _, device := range vmiReport.vmiStats.DeviceStats {
		deviceID := fmt.Sprintf("%x", device.DeviceID)

		// Deduplicate devices sharing the same label set, otherwise Gather()
		// fails on duplicate label sets and the whole /metrics endpoint errors.
		dedupeKey := fmt.Sprintf("%s|%s|%s", device.DriverName, device.DriverVersion, deviceID)
		if _, ok := seen[dedupeKey]; ok {
			continue
		}
		seen[dedupeKey] = struct{}{}

		deviceLabels := map[string]string{
			"driver_name":    device.DriverName,
			"driver_version": device.DriverVersion,
			"device_id":      deviceID,
		}

		driverDateSeconds := float64(device.DriverDate) / float64(time.Second)

		crs = append(crs,
			vmiReport.newCollectorResultWithLabels(guestDeviceDriverDateSeconds, driverDateSeconds, deviceLabels),
		)
	}
	return crs
}
