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
	"strconv"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var (
	storageIopsRead = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_iops_read_total",
			Help: "Total number of I/O read operations.",
		},
	)

	storageIopsWrite = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_iops_write_total",
			Help: "Total number of I/O write operations.",
		},
	)

	storageReadTrafficBytes = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_read_traffic_bytes_total",
			Help: "Total number of bytes read from storage.",
		},
	)

	storageWriteTrafficBytes = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_write_traffic_bytes_total",
			Help: "Total number of written bytes.",
		},
	)

	storageReadTimesSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_read_times_seconds_total",
			Help: "Total time spent on read operations.",
		},
	)

	storageWriteTimesSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_write_times_seconds_total",
			Help: "Total time spent on write operations.",
		},
	)

	storageFlushRequests = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_flush_requests_total",
			Help: "Total storage flush requests.",
		},
	)

	storageFlushTimesSeconds = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_flush_times_seconds_total",
			Help: "Total time spent on cache flushing.",
		},
	)

	storageReadLatencySeconds = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_read_latency_seconds_bucket",
			Help: "Cumulative read latency histogram bucket for block devices.",
		},
	)
	storageWriteLatencySeconds = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_write_latency_seconds_bucket",
			Help: "Cumulative write latency histogram bucket for block devices.",
		},
	)
	storageFlushLatencySeconds = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_storage_flush_latency_seconds_bucket",
			Help: "Cumulative flush latency histogram bucket for block devices.",
		},
	)
)

type blockMetrics struct{}

func (blockMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		storageIopsRead,
		storageIopsWrite,
		storageReadTrafficBytes,
		storageWriteTrafficBytes,
		storageReadTimesSeconds,
		storageWriteTimesSeconds,
		storageFlushRequests,
		storageFlushTimesSeconds,
		storageReadLatencySeconds,
		storageWriteLatencySeconds,
		storageFlushLatencySeconds,
	}
}

func (blockMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil || vmiReport.vmiStats.DomainStats.Block == nil {
		return crs
	}

	for blockIdx, block := range vmiReport.vmiStats.DomainStats.Block {
		if !block.NameSet {
			log.Log.Warningf("Name not set for block device#%d", blockIdx)
			continue
		}

		blkLabels := map[string]string{"drive": block.Name}
		if block.Alias != "" {
			blkLabels["drive"] = block.Alias
		}

		if block.RdReqsSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageIopsRead, float64(block.RdReqs), blkLabels))
		}

		if block.WrReqsSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageIopsWrite, float64(block.WrReqs), blkLabels))
		}

		if block.RdBytesSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageReadTrafficBytes, float64(block.RdBytes), blkLabels))
		}

		if block.WrBytesSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageWriteTrafficBytes, float64(block.WrBytes), blkLabels))
		}

		if block.RdTimesSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageReadTimesSeconds, nanosecondsToSeconds(block.RdTimes), blkLabels))
		}

		if block.WrTimesSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageWriteTimesSeconds, nanosecondsToSeconds(block.WrTimes), blkLabels))
		}

		if block.FlReqsSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageFlushRequests, float64(block.FlReqs), blkLabels))
		}

		if block.FlTimesSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(storageFlushTimesSeconds, nanosecondsToSeconds(block.FlTimes), blkLabels))
		}

		crs = append(crs, emitLatencyHistogramBuckets(
			vmiReport, block.LatencyHistograms.Read,
			storageReadLatencySeconds, blkLabels)...)

		crs = append(crs, emitLatencyHistogramBuckets(
			vmiReport, block.LatencyHistograms.Write,
			storageWriteLatencySeconds, blkLabels)...)

		crs = append(crs, emitLatencyHistogramBuckets(
			vmiReport, block.LatencyHistograms.Flush,
			storageFlushLatencySeconds, blkLabels)...)

	}

	return crs
}

func emitLatencyHistogramBuckets(
	vmiReport *VirtualMachineInstanceReport,
	histogram *stats.DomainStatsBlockLatencyHistogram,
	metric operatormetrics.Metric,
	baseLabels map[string]string,
) []operatormetrics.CollectorResult {
	if histogram == nil || len(histogram.Bins) == 0 {
		return nil
	}

	bins := histogram.Bins

	// Validate the histogram before emitting anything.
	//
	// Libvirt reports each bin using its lower boundary ("start"):
	//
	//   start=0       value=5   => [0, 1ms)
	//   start=1ms     value=7   => [1ms, 10ms)
	//   start=10ms    value=2   => [10ms, +Inf)
	//
	// Prometheus histogram buckets instead use the upper boundary ("le").
	// Therefore, the upper boundary of bin i is the start of bin i+1.
	for i, bin := range bins {
		if !bin.StartSet || !bin.ValueSet {
			return nil
		}

		if i > 0 && bin.Start <= bins[i-1].Start {
			return nil
		}
	}

	results := make([]operatormetrics.CollectorResult, 0, len(bins))
	var cumulative uint64

	// Every bin except the last one gets a finite Prometheus upper
	// boundary derived from the start of the following libvirt bin.
	for i := 0; i < len(bins)-1; i++ {
		cumulative += bins[i].Value

		upperBoundSeconds := float64(bins[i+1].Start) / 1e9

		labels := make(map[string]string, len(baseLabels)+1)
		for key, value := range baseLabels {
			labels[key] = value
		}
		labels["le"] = strconv.FormatFloat(
			upperBoundSeconds,
			'g',
			-1,
			64,
		)

		results = append(
			results,
			vmiReport.newCollectorResultWithLabels(
				metric,
				float64(cumulative),
				labels,
			),
		)
	}
	// The last libvirt bin has no finite upper boundary, so it maps to
	// Prometheus's +Inf bucket.
	cumulative += bins[len(bins)-1].Value

	infLabels := make(map[string]string, len(baseLabels)+1)
	for key, value := range baseLabels {
		infLabels[key] = value
	}
	infLabels["le"] = "+Inf"

	results = append(
		results,
		vmiReport.newCollectorResultWithLabels(
			metric,
			float64(cumulative),
			infLabels,
		),
	)

	return results
}
