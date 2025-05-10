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

import "github.com/machadovilaca/operator-observability/pkg/operatormetrics"

var (
	memoryResident = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_resident_bytes",
			Help: "Resident set size of the process running the domain.",
		},
	)

	memoryAvailable = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_available_bytes",
			Help: "Amount of usable memory as seen by the domain. This value may not be accurate if a balloon driver is in use or if the guest OS does not initialize all assigned pages",
		},
	)

	memoryUnused = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_unused_bytes",
			Help: "The amount of memory left completely unused by the system. Memory that is available but used for reclaimable caches should NOT be reported as free.",
		},
	)

	memoryCached = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_cached_bytes",
			Help: "The amount of memory that is being used to cache I/O and is available to be reclaimed, corresponds to the sum of `Buffers` + `Cached` + `SwapCached` in `/proc/meminfo`.",
		},
	)

	memorySwapInTrafficBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_swap_in_traffic_bytes",
			Help: "The total amount of data read from swap space of the guest in bytes.",
		},
	)

	memorySwapOutTrafficBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_swap_out_traffic_bytes",
			Help: "The total amount of memory written out to swap space of the guest in bytes.",
		},
	)

	memoryPgmajfaultTotal = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_pgmajfault_total",
			Help: "The number of page faults when disk IO was required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is required, it is considered as major fault.",
		},
	)

	memoryPgminfaultTotal = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_pgminfault_total",
			Help: "The number of other page faults, when disk IO was not required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is NOT required, it is considered as minor fault.",
		},
	)

	memoryActualBallon = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_actual_balloon_bytes",
			Help: "Current balloon size in bytes.",
		},
	)

	memoryUsableBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_usable_bytes",
			Help: "The amount of memory which can be reclaimed by balloon without pushing the guest system to swap, corresponds to 'Available' in /proc/meminfo.",
		},
	)

	memoryDomainBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_domain_bytes",
			Help: "The amount of memory in bytes allocated to the domain. The `memory` value in domain xml file.",
		},
	)
)

type memoryMetrics struct{}

func (memoryMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		memoryResident,
		memoryAvailable,
		memoryUnused,
		memoryCached,
		memorySwapInTrafficBytes,
		memorySwapOutTrafficBytes,
		memoryPgmajfaultTotal,
		memoryPgminfaultTotal,
		memoryActualBallon,
		memoryUsableBytes,
		memoryDomainBytes,
	}
}

func (memoryMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil || vmiReport.vmiStats.DomainStats.Memory == nil {
		return crs
	}

	mem := vmiReport.vmiStats.DomainStats.Memory

	if mem.RSSSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryResident, kibibytesToBytes(mem.RSS)))
	}

	if mem.AvailableSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryAvailable, kibibytesToBytes(mem.Available)))
	}

	if mem.UnusedSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryUnused, kibibytesToBytes(mem.Unused)))
	}

	if mem.CachedSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryCached, kibibytesToBytes(mem.Cached)))
	}

	if mem.SwapInSet {
		crs = append(crs, vmiReport.newCollectorResult(memorySwapInTrafficBytes, kibibytesToBytes(mem.SwapIn)))
	}

	if mem.SwapOutSet {
		crs = append(crs, vmiReport.newCollectorResult(memorySwapOutTrafficBytes, kibibytesToBytes(mem.SwapOut)))
	}

	if mem.MajorFaultSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryPgmajfaultTotal, float64(mem.MajorFault)))
	}

	if mem.MinorFaultSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryPgminfaultTotal, float64(mem.MinorFault)))
	}

	if mem.ActualBalloonSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryActualBallon, kibibytesToBytes(mem.ActualBalloon)))
	}

	if mem.UsableSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryUsableBytes, kibibytesToBytes(mem.Usable)))
	}

	if mem.TotalSet {
		crs = append(crs, vmiReport.newCollectorResult(memoryDomainBytes, kibibytesToBytes(mem.Total)))
	}

	return crs
}
