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
	GuestAgentInfoCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			guestLoad1M,
			guestLoad5M,
			guestLoad15M,
		},
		CollectCallback: guestAgentInfoCollectorCallback,
	}

	guestLoad1M = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_1m",
			Help: "Guest system load average over 1 minute as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O.",
		},
	)

	guestLoad5M = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_5m",
			Help: "Guest system load average over 5 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O.",
		},
	)

	guestLoad15M = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_load_15m",
			Help: "Guest system load average over 15 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O.",
		},
	)

	scraper = NewGuestAgentInfoScraper()
)

func guestAgentInfoCollectorCallback() []operatormetrics.CollectorResult {
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
	return execGuestAgentInfoCollector(concCollector, vmis)
}

func execGuestAgentInfoCollector(concCollector collector.Collector, vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	scraper.Reset()
	go concCollector.Collect(vmis, scraper, PrometheusCollectionTimeout)

	var crs []operatormetrics.CollectorResult

	for vmiReport := range scraper.ch {
		if vmiReport.vmiStats.GuestAgentInfo == nil {
			log.Log.Warningf("Guest agent info is nil for VMI %s/%s", vmiReport.vmi.Name, vmiReport.vmi.Namespace)
			continue
		}

		guestInfo := vmiReport.vmiStats.GuestAgentInfo

		crs = append(crs, vmiReport.newCollectorResult(guestLoad1M, guestInfo.Load.Load1m))
		crs = append(crs, vmiReport.newCollectorResult(guestLoad5M, guestInfo.Load.Load5m))
		crs = append(crs, vmiReport.newCollectorResult(guestLoad15M, guestInfo.Load.Load15m))
	}

	return crs
}
