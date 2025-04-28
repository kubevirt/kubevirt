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
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
)

const (
	PrometheusCollectionTimeout = collector.CollectionTimeout
)

var (
	rms = []resourceMetrics{
		memoryMetrics{},
		cpuMetrics{},
		vcpuMetrics{},
		blockMetrics{},
		networkMetrics{},
		cpuAffinityMetrics{},
		filesystemMetrics{},
	}

	Collector = operatormetrics.Collector{
		Metrics:         domainStatsMetrics(rms...),
		CollectCallback: domainStatsCollectorCallback,
	}

	settings *collectorSettings
)

type resourceMetrics interface {
	Describe() []operatormetrics.Metric
	Collect(report *VirtualMachineInstanceReport) []operatormetrics.CollectorResult
}

type collectorSettings struct {
	virtShareDir        string
	nodeName            string
	maxRequestsInFlight int
	vmiInformer         cache.SharedIndexInformer
}

func SetupDomainStatsCollector(virtShareDir, nodeName string, maxRequestsInFlight int, vmiInformer cache.SharedIndexInformer) {
	settings = &collectorSettings{
		virtShareDir:        virtShareDir,
		nodeName:            nodeName,
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
		log.Log.V(4).Infof("No VMIs detected")
		return []operatormetrics.CollectorResult{}
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	concCollector := collector.NewConcurrentCollector(settings.maxRequestsInFlight)
	return execCollector(concCollector, vmis)
}

func execCollector(concCollector collector.Collector, vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	scraper := NewDomainstatsScraper(len(vmis))
	go concCollector.Collect(vmis, scraper, PrometheusCollectionTimeout)

	var crs []operatormetrics.CollectorResult

	for vmiReport := range scraper.ch {
		for _, rm := range rms {
			crs = append(crs, rm.Collect(vmiReport)...)
		}
	}

	return crs
}
