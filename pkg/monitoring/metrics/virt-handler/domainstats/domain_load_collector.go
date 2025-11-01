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
 */

package domainstats

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	guestLoad1m = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_1m",
			Help: "Guest system load average over 1 minute as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O.",
		},
	)

	guestLoad5m = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_5m",
			Help: "Guest system load average over 5 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O.",
		},
	)

	guestLoad15m = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_15m",
			Help: "Guest system load average over 15 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O.",
		},
	)
)

// domainLoadMetrics emits kubevirt_vmi_guest_load_* based on DomainStats
type domainLoadMetrics struct{}

func (m domainLoadMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		guestLoad1m,
		guestLoad5m,
		guestLoad15m,
	}
}

func (m domainLoadMetrics) Collect(report *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult
	ds := report.vmiStats.DomainStats
	if ds == nil {
		return crs
	}

	if ds.Load != nil {
		if ds.Load.Load1mSet {
			crs = append(crs, report.newCollectorResult(guestLoad1m, ds.Load.Load1m))
		}
		if ds.Load.Load5mSet {
			crs = append(crs, report.newCollectorResult(guestLoad5m, ds.Load.Load5m))
		}
		if ds.Load.Load15mSet {
			crs = append(crs, report.newCollectorResult(guestLoad15m, ds.Load.Load15m))
		}
	}

	return crs
}
