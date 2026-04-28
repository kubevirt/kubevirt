/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

import (
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

	Collector = operatormetrics.Collector{
		Metrics:         domainStatsMetrics(domainStatsResourceMetrics...),
		CollectCallback: domainStatsCollectorCallback,
	}

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

func domainStatsCollectorCallback() []operatormetrics.CollectorResult {
	cachedObjs := settings.vmiInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		log.Log.V(logVerbosityDebug).Infof("No VMIs detected")
		return []operatormetrics.CollectorResult{}
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	concCollector := collector.NewConcurrentCollector(settings.maxRequestsInFlight)
	return execDomainStatsCollector(concCollector, vmis)
}

func execDomainStatsCollector(concCollector collector.Collector, vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	scraper := NewDomainstatsScraper(len(vmis))
	go concCollector.Collect(vmis, scraper, PrometheusCollectionTimeout)

	var crs []operatormetrics.CollectorResult

	for vmiReport := range scraper.ch {
		for _, rm := range domainStatsResourceMetrics {
			crs = append(crs, rm.Collect(vmiReport)...)
		}
	}

	return crs
}
