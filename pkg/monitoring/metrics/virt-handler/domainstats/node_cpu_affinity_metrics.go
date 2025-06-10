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

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	nodeCpuAffinity = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_node_cpu_affinity",
			Help: "Number of VMI CPU affinities to node physical cores.",
		},
	)
)

type cpuAffinityMetrics struct{}

func (cpuAffinityMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{nodeCpuAffinity}
}

func (cpuAffinityMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	if vmiReport.vmiStats.DomainStats == nil || !vmiReport.vmiStats.DomainStats.CPUMapSet {
		return []operatormetrics.CollectorResult{}
	}

	affinityCount := 0.0

	for vidx := 0; vidx < len(vmiReport.vmiStats.DomainStats.CPUMap); vidx++ {
		for cidx := 0; cidx < len(vmiReport.vmiStats.DomainStats.CPUMap[vidx]); cidx++ {
			if vmiReport.vmiStats.DomainStats.CPUMap[vidx][cidx] {
				affinityCount++
			}
		}
	}

	return []operatormetrics.CollectorResult{
		vmiReport.newCollectorResult(nodeCpuAffinity, affinityCount),
	}
}
