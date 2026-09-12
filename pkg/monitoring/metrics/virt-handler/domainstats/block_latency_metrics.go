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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var storageIOLatencySeconds = operatormetrics.NewHistogram(
	operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_io_latency_seconds",
		Help: "I/O latency distribution for block devices.",
	},
	prometheus.HistogramOpts{},
)

var histogramLabelNames = []string{"node", "namespace", "name", "drive", "operation"}

var histogramDesc = prometheus.NewDesc(
	storageIOLatencySeconds.GetOpts().Name,
	storageIOLatencySeconds.GetOpts().Help,
	histogramLabelNames,
	nil,
)

func collectBlockLatencyHistograms(
	vmiReport *VirtualMachineInstanceReport,
	ch chan<- prometheus.Metric,
) {
	if vmiReport.vmiStats.DomainStats == nil ||
		vmiReport.vmiStats.DomainStats.Block == nil {
		return
	}

	for _, block := range vmiReport.vmiStats.DomainStats.Block {
		if !block.NameSet {
			continue
		}

		drive := block.Name
		if block.Alias != "" {
			drive = block.Alias
		}

		operations := []struct {
			name         string
			histogram    *stats.DomainStatsBlockLatencyHistogram
			totalTimeSet bool
			totalTime    uint64
		}{
			{
				name:         "read",
				histogram:    block.LatencyHistograms.Read,
				totalTimeSet: block.RdTimesSet,
				totalTime:    block.RdTimes,
			},
			{
				name:         "write",
				histogram:    block.LatencyHistograms.Write,
				totalTimeSet: block.WrTimesSet,
				totalTime:    block.WrTimes,
			},
			{
				name:         "flush",
				histogram:    block.LatencyHistograms.Flush,
				totalTimeSet: block.FlTimesSet,
				totalTime:    block.FlTimes,
			},
		}

		for _, operation := range operations {
			if !operation.totalTimeSet {
				continue
			}

			buckets, count, ok := convertLatencyHistogram(operation.histogram)
			if !ok {
				continue
			}

			metric, err := prometheus.NewConstHistogram(
				histogramDesc,
				count,
				nanosecondsToSeconds(operation.totalTime),
				buckets,
				vmiReport.vmi.Status.NodeName,
				vmiReport.vmi.Namespace,
				vmiReport.vmi.Name,
				drive,
				operation.name,
			)
			if err != nil {
				log.Log.Warningf(
					"failed to create latency histogram for drive %s operation %s: %v",
					drive,
					operation.name,
					err,
				)
				continue
			}

			ch <- metric
		}
	}
}

func convertLatencyHistogram(
	histogram *stats.DomainStatsBlockLatencyHistogram,
) (map[float64]uint64, uint64, bool) {
	if histogram == nil || len(histogram.Bins) == 0 {
		return nil, 0, false
	}

	bins := histogram.Bins

	for i, bin := range bins {
		if !bin.StartSet || !bin.ValueSet {
			return nil, 0, false
		}

		if i > 0 && bin.Start <= bins[i-1].Start {
			return nil, 0, false
		}
	}

	buckets := make(map[float64]uint64, len(bins)-1)

	var cumulative uint64

	for i := 0; i < len(bins)-1; i++ {
		cumulative += bins[i].Value

		upperBoundSeconds := nanosecondsToSeconds(bins[i+1].Start)
		buckets[upperBoundSeconds] = cumulative
	}

	cumulative += bins[len(bins)-1].Value

	return buckets, cumulative, true
}
