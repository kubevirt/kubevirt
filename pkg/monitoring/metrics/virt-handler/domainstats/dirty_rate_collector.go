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
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
)

var (
	DomainDirtyRateStatsCollector = operatormetrics.Collector{
		Metrics:         domainStatsMetrics(dirtyRateMetrics{}),
		CollectCallback: domainDirtyRateStatsCollectorCallback,
	}
)

func domainDirtyRateStatsCollectorCallback() []operatormetrics.CollectorResult {
	cachedObjs := settings.vmiInformer.GetStore().List()
	if len(cachedObjs) == 0 {
		log.Log.V(4).Infof("No VMIs detected")
		return []operatormetrics.CollectorResult{}
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	concCollector := collector.NewConcurrentCollector(settings.maxRequestsInFlight)
	return execDomainDirtyRateStatsCollector(concCollector, vmis)
}

func execDomainDirtyRateStatsCollector(concCollector collector.Collector, vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	scraper := NewDomainsDirtyRateStatsScraper(len(vmis))
	go concCollector.Collect(vmis, scraper, PrometheusCollectionTimeout)

	var crs []operatormetrics.CollectorResult

	dirtyRateResourceMetric := dirtyRateMetrics{}
	for vmiReport := range scraper.ch {
		crs = append(crs, dirtyRateResourceMetric.Collect(vmiReport)...)
	}

	return crs
}
