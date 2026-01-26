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
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"
)

var (
	cpuUsageSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_cpu_usage_seconds_total",
			Help: "Total CPU time spent in all modes (sum of both vcpu and hypervisor usage).",
		},
	)

	cpuUserUsageSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_cpu_user_usage_seconds_total",
			Help: "Total CPU time spent in user mode.",
		},
	)

	cpuSystemUsageSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_cpu_system_usage_seconds_total",
			Help: "Total CPU time spent in system mode.",
		},
	)

	guestLoad1m = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_1m",
			Help: "Guest system load average over 1 minute as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Requires qemu-guest-agent version 10.0.0 or above.",
		},
	)

	guestLoad5m = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_5m",
			Help: "Guest system load average over 5 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Requires qemu-guest-agent version 10.0.0 or above.",
		},
	)

	guestLoad15m = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_15m",
			Help: "Guest system load average over 15 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Requires qemu-guest-agent version 10.0.0 or above.",
		},
	)
)

type cpuMetrics struct{}

func (cpuMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		cpuUsageSeconds,
		cpuUserUsageSeconds,
		cpuSystemUsageSeconds,
		guestLoad1m,
		guestLoad5m,
		guestLoad15m,
	}
}

func (cpuMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil {
		return crs
	}

	if vmiReport.vmiStats.DomainStats.Cpu != nil {
		cpu := vmiReport.vmiStats.DomainStats.Cpu

		if !cpu.TimeSet && !cpu.UserSet && !cpu.SystemSet {
			log.Log.Warningf("No domain CPU stats is set for %s VMI.", vmiReport.vmi.Name)
		}

		if cpu.TimeSet {
			crs = append(crs, vmiReport.newCollectorResult(cpuUsageSeconds, nanosecondsToSeconds(cpu.Time)))
		}

		if cpu.UserSet {
			crs = append(crs, vmiReport.newCollectorResult(cpuUserUsageSeconds, nanosecondsToSeconds(cpu.User)))
		}

		if cpu.SystemSet {
			crs = append(crs, vmiReport.newCollectorResult(cpuSystemUsageSeconds, nanosecondsToSeconds(cpu.System)))
		}
	}

	if vmiReport.vmiStats.DomainStats.Load != nil {
		load := vmiReport.vmiStats.DomainStats.Load

		if load.Load1mSet {
			crs = append(crs, vmiReport.newCollectorResult(guestLoad1m, load.Load1m))
		}

		if load.Load5mSet {
			crs = append(crs, vmiReport.newCollectorResult(guestLoad5m, load.Load5m))
		}

		if load.Load15mSet {
			crs = append(crs, vmiReport.newCollectorResult(guestLoad15m, load.Load15m))
		}
	}

	return crs
}
