/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
)

var DomainDirtyRateStatsCollector = operatormetrics.Collector{
	Metrics:         domainStatsMetrics(dirtyRateMetrics{}),
	CollectCallback: domainDirtyRateStatsCollectorCallback,
}

func domainDirtyRateStatsCollectorCallback() []operatormetrics.CollectorResult {
	cachedObjs := settings.vmiInformer.GetStore().List()
	if len(cachedObjs) == 0 {
		log.Log.V(logVerbosityDebug).Infof("No VMIs detected")
		return []operatormetrics.CollectorResult{}
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	concCollector := collector.NewConcurrentCollector(settings.maxRequestsInFlight)
	return execDomainDirtyRateStatsCollector(concCollector, vmis)
}

func execDomainDirtyRateStatsCollector(
	concCollector collector.Collector, vmis []*k6tv1.VirtualMachineInstance,
) []operatormetrics.CollectorResult {
	scraper := NewDomainsDirtyRateStatsScraper(len(vmis))
	go concCollector.Collect(vmis, scraper, PrometheusCollectionTimeout)

	var crs []operatormetrics.CollectorResult

	dirtyRateResourceMetric := dirtyRateMetrics{}
	for vmiReport := range scraper.ch {
		crs = append(crs, dirtyRateResourceMetric.Collect(vmiReport)...)
	}

	return crs
}
