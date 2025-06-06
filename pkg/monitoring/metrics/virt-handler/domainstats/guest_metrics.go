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
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

var (
	guestHostname = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_hostname_info",
			Help: "Guest hostname reported by the guest agent. The value is always 1.",
		},
	)

	guestLoad1M = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_1m",
			Help: "Guest system load average over 1 minute as reported by the guest agent.",
		},
	)

	guestLoad5M = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_5m",
			Help: "Guest system load average over 5 minutes as reported by the guest agent.",
		},
	)

	guestLoad15M = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_15m",
			Help: "Guest system load average over 15 minutes as reported by the guest agent.",
		},
	)
)

type guestMetrics struct{}

func (guestMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		guestHostname,
		guestLoad1M,
		guestLoad5M,
		guestLoad15M,
	}
}

func (guestMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.GuestAgentInfo == nil {
		return crs
	}

	guestInfo := vmiReport.vmiStats.GuestAgentInfo

	if guestInfo.Hostname != "" {
		hostnameLabels := map[string]string{"hostname": guestInfo.Hostname}
		crs = append(crs, vmiReport.newCollectorResultWithLabels(guestHostname, 1.0, hostnameLabels))
	}

	if guestInfo.Load.Load1m != 0 {
		crs = append(crs, vmiReport.newCollectorResult(guestLoad1M, guestInfo.Load.Load1m))
	}

	if guestInfo.Load.Load5m != 0 {
		crs = append(crs, vmiReport.newCollectorResult(guestLoad5M, guestInfo.Load.Load5m))
	}

	if guestInfo.Load.Load15m != 0 {
		crs = append(crs, vmiReport.newCollectorResult(guestLoad15M, guestInfo.Load.Load15m))
	}

	return crs
}
