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

package domainstats

import (
	"fmt"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var (
	vcpuSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_vcpu_seconds_total",
			Help: "Total amount of time spent in each state by each vcpu (cpu_time excluding hypervisor time). Where `id` is the vcpu identifier and `state` can be one of the following: [`OFFLINE`, `RUNNING`, `BLOCKED`].",
		},
	)

	vcpuWaitSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_vcpu_wait_seconds_total",
			Help: "Amount of time spent by each vcpu while waiting on I/O.",
		},
	)

	vcpuDelaySeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_vcpu_delay_seconds_total",
			Help: "Amount of time spent by each vcpu waiting in the queue instead of running.",
		},
	)
)

type vcpuMetrics struct{}

func (vcpuMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		vcpuSeconds,
		vcpuWaitSeconds,
		vcpuDelaySeconds,
	}
}

func (vcpuMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil || vmiReport.vmiStats.DomainStats.Vcpu == nil {
		return crs
	}

	for vcpuIdx, vcpu := range vmiReport.vmiStats.DomainStats.Vcpu {
		stringVcpuIdx := fmt.Sprintf("%d", vcpuIdx)

		if vcpu.TimeSet {
			additionalLabels := map[string]string{
				"id":    stringVcpuIdx,
				"state": humanReadableState(vcpu.State),
			}
			crs = append(crs, vmiReport.newCollectorResultWithLabels(vcpuSeconds, nanosecondsToSeconds(vcpu.Time), additionalLabels))
		}

		if vcpu.WaitSet {
			additionalLabels := map[string]string{
				"id": stringVcpuIdx,
			}
			crs = append(crs, vmiReport.newCollectorResultWithLabels(vcpuWaitSeconds, nanosecondsToSeconds(vcpu.Wait), additionalLabels))
		}

		if vcpu.DelaySet {
			additionalLabels := map[string]string{
				"id": stringVcpuIdx,
			}
			crs = append(crs, vmiReport.newCollectorResultWithLabels(vcpuDelaySeconds, nanosecondsToSeconds(vcpu.Delay), additionalLabels))
		}
	}

	return crs
}

func humanReadableState(state int) string {
	switch state {
	case stats.VCPUOffline:
		return "offline"
	case stats.VCPUBlocked:
		return "blocked"
	case stats.VCPURunning:
		return "running"
	default:
		return "unknown"
	}
}
