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
	"k8s.io/client-go/tools/cache"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
)

const (
	PrometheusCollectionTimeout = collector.CollectionTimeout

	logVerbosityDebug = 4
)

type domainStatsPrometheusCollector struct{}

func (domainStatsPrometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	scalarCollector := operatormetrics.Collector{
		Metrics: domainStatsMetrics(domainStatsResourceMetrics...),
	}

	scalarCollector.Describe(ch)

	ch <- histogramDesc
}

func (domainStatsPrometheusCollector) Collect(ch chan<- prometheus.Metric) {
	vmis := cachedVMIs()
	if len(vmis) == 0 {
		return
	}

	concCollector := collector.NewConcurrentCollector(
		settings.maxRequestsInFlight,
	)

	reports := collectDomainStatsReports(
		concCollector,
		vmis,
	)

	results := collectDomainStatsResults(reports)

	// Reuse operator-observability-toolkit for all existing scalar
	// metrics. No additional domain stats scrape is performed here.
	scalarCollector := operatormetrics.Collector{
		Metrics: domainStatsMetrics(domainStatsResourceMetrics...),
		CollectCallback: func() []operatormetrics.CollectorResult {
			return results
		},
	}

	scalarCollector.Collect(ch)

	for _, report := range reports {
		collectBlockLatencyHistograms(report, ch)
	}
}

var (
	domainStatsResourceMetrics = []resourceMetrics{
		memoryMetrics{},
		cpuMetrics{},
		vcpuMetrics{},
		blockMetrics{},
		networkMetrics{},
		cpuAffinityMetrics{},
		filesystemMetrics{},
	}

	Collector = domainStatsPrometheusCollector{}

	settings *collectorSettings
)

type resourceMetrics interface {
	Describe() []operatormetrics.Metric
	Collect(report *VirtualMachineInstanceReport) []operatormetrics.CollectorResult
}

type collectorSettings struct {
	maxRequestsInFlight int
	vmiInformer         cache.SharedIndexInformer
}

func SetupDomainStatsCollector(maxRequestsInFlight int, vmiInformer cache.SharedIndexInformer) {
	settings = &collectorSettings{
		maxRequestsInFlight: maxRequestsInFlight,
		vmiInformer:         vmiInformer,
	}
}

func domainStatsMetrics(rms ...resourceMetrics) []operatormetrics.Metric {
	var metrics []operatormetrics.Metric

	for _, rm := range rms {
		metrics = append(metrics, rm.Describe()...)
	}

	return metrics
}

func ListMetrics() []operatormetrics.Metric {
	metrics := domainStatsMetrics(domainStatsResourceMetrics...)
	return append(metrics, storageIOLatencySeconds)
}

func execDomainStatsCollector(
	concCollector collector.Collector,
	vmis []*k6tv1.VirtualMachineInstance,
) []operatormetrics.CollectorResult {
	reports := collectDomainStatsReports(concCollector, vmis)
	return collectDomainStatsResults(reports)
}

func collectDomainStatsReports(
	concCollector collector.Collector,
	vmis []*k6tv1.VirtualMachineInstance,
) []*VirtualMachineInstanceReport {
	scraper := NewDomainstatsScraper(len(vmis))

	go concCollector.Collect(
		vmis,
		scraper,
		PrometheusCollectionTimeout,
	)

	var reports []*VirtualMachineInstanceReport

	for report := range scraper.ch {
		reports = append(reports, report)
	}

	return reports
}

func collectDomainStatsResults(
	reports []*VirtualMachineInstanceReport,
) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	for _, report := range reports {
		for _, rm := range domainStatsResourceMetrics {
			crs = append(crs, rm.Collect(report)...)
		}
	}

	return crs
}

func cachedVMIs() []*k6tv1.VirtualMachineInstance {
	cachedObjs := settings.vmiInformer.GetIndexer().List()

	if len(cachedObjs) == 0 {
		log.Log.V(logVerbosityDebug).Infof("No VMIs detected")
		return nil
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	return vmis
}
