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
 * Copyright the KubeVirt Authors.
 */

package migrationdomainstats

import (
	"sync"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/domainstats"
	domstatsCollector "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/domainstats/collector"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

const (
	collectionTimeout = 10 * time.Second
	pollingInterval   = 1 * time.Second
)

type Result struct {
	VMI       string
	VMIM      string
	Namespace string

	DomainJobInfo stats.DomainJobInfo
	Timestamp     time.Time
}

type queue struct {
	vmi     *v1.VirtualMachineInstance
	vmim    *v1.VirtualMachineInstanceMigration
	results []Result

	isActive  bool
	mutex     *sync.Mutex
	collector domstatsCollector.Collector
}

func newQueue(vmi *v1.VirtualMachineInstance, vmim *v1.VirtualMachineInstanceMigration) *queue {
	return &queue{
		vmi:     vmi,
		vmim:    vmim,
		results: make([]Result, 0),

		isActive:  false,
		mutex:     &sync.Mutex{},
		collector: domstatsCollector.NewConcurrentCollector(1),
	}
}

func (q *queue) startPolling() {
	q.isActive = true

	ticker := time.NewTicker(pollingInterval)
	go func() {
		for range ticker.C {
			if !q.isActive {
				ticker.Stop()
				return
			}
			q.collect()
		}
	}()
}

func (q *queue) stopPolling() {
	q.isActive = false
}

func (q *queue) collect() {
	scraper := domainstats.NewDomainstatsScraper(1)
	vmis := []*v1.VirtualMachineInstance{q.vmi}
	q.collector.Collect(vmis, scraper, collectionTimeout)

	values := scraper.GetValues()
	if len(values) != 1 {
		log.Log.Errorf("Expected 1 value from DomainstatsScraper, got %d", len(values))
		return
	}

	result := Result{
		VMI:       q.vmim.Spec.VMIName,
		VMIM:      q.vmim.Name,
		Namespace: q.vmim.Namespace,

		DomainJobInfo: *values[0].GetVmiStats().DomainStats.MigrateDomainJobInfo,
		Timestamp:     time.Now(),
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.results = append(q.results, result)
}

func (q *queue) all() []Result {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	out := q.results
	q.results = make([]Result, 0)

	return out
}
