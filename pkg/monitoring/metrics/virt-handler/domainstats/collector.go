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
 */

package domainstats

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
)

type resourceMetrics interface {
	Describe() []operatormetrics.Metric
	Collect(report *VirtualMachineInstanceReport) []operatormetrics.CollectorResult
}

var (
	rms = []resourceMetrics{
		memoryMetrics{},
		cpuMetrics{},
		vcpuMetrics{},
		blockMetrics{},
		networkMetrics{},
		cpuAffinityMetrics{},
		migrationMetrics{},
		filesystemMetrics{},
	}

	Collector = operatormetrics.Collector{
		Metrics:         domainStatsMetrics(rms...),
		CollectCallback: domainStatsCollectorCallback,
	}
)

func domainStatsMetrics(rms ...resourceMetrics) []operatormetrics.Metric {
	var metrics []operatormetrics.Metric

	for _, rm := range rms {
		metrics = append(metrics, rm.Describe()...)
	}

	return metrics
}

func domainStatsCollectorCallback() []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult
	var vmis []k6tv1.VirtualMachineInstance

	for _, vmi := range vmis {
		crs = append(crs, collect(&vmi)...)
	}

	return crs
}

func collect(vmi *k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	vmiStats := scrape(vmi)

	if vmiStats == nil {
		return crs
	}

	vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)
	for _, rm := range rms {
		crs = append(crs, rm.Collect(vmiReport)...)
	}

	return crs
}

func scrape(vmi *k6tv1.VirtualMachineInstance) *VirtualMachineInstanceStats {
	return nil
}
