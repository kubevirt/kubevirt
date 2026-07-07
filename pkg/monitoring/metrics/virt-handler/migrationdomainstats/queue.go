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

package migrationdomainstats

import (
	"container/ring"
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	domstatsCollector "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/domainstats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

const (
	collectionTimeout     = 10 * time.Second
	pollingInterval       = 5 * time.Second
	bufferSize            = 10
	completedStatsTimeout = 3 * time.Minute

	logVerbosityWarning = 2
	logVerbosityDebug   = 4
)

type result struct {
	vmi       string
	namespace string
	node      string

	domainJobInfo stats.DomainJobInfo
	timestamp     time.Time
}

type queue struct {
	sync.Mutex

	vmiStore    cache.Store
	vmi         *v1.VirtualMachineInstance
	collector   domstatsCollector.Collector
	results     *ring.Ring
	finished    bool
	completedAt *time.Time

	ctx       context.Context
	ctxCancel context.CancelFunc
}

func newQueue(vmiStore cache.Store, vmi *v1.VirtualMachineInstance) *queue {
	return &queue{
		vmiStore:  vmiStore,
		vmi:       vmi,
		collector: domstatsCollector.NewConcurrentCollector(1),
		results:   ring.New(bufferSize),
	}
}

func (q *queue) startPolling() {
	q.ctx, q.ctxCancel = context.WithCancel(context.Background())

	ticker := time.NewTicker(pollingInterval)
	go func() {
		log.Log.V(logVerbosityDebug).Infof("collecting domain stats for VMI %s/%s (initial)", q.vmi.Namespace, q.vmi.Name)
		q.collect()

		for {
			select {
			case <-q.ctx.Done():
				log.Log.V(logVerbosityWarning).Infof("stopping domain stats collection for VMI %s/%s", q.vmi.Namespace, q.vmi.Name)
				ticker.Stop()
				return
			case <-ticker.C:
				log.Log.V(logVerbosityDebug).Infof("collecting domain stats for VMI %s/%s", q.vmi.Namespace, q.vmi.Name)
				q.collect()
			}
		}
	}()
}

func (q *queue) collect() {
	if q.isFinished() {
		return
	}

	// Check whether the VMI has already transitioned out of the migrating
	// state, but keep scraping until virt-launcher's completed migration event
	// stats (Downtime, DowntimeNet) are captured before the poller exits.
	finished := q.isMigrationFinished()

	values, err := q.scrapeDomainStats()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to scrape domain stats for VMI %s/%s", q.vmi.Namespace, q.vmi.Name)
		if q.shouldStopPolling(finished, stats.DomainJobInfo{}) {
			q.stopPolling()
		}
		return
	}

	r := result{
		vmi:       q.vmi.Name,
		namespace: q.vmi.Namespace,
		node:      q.vmi.Status.NodeName,

		domainJobInfo: *values.MigrateDomainJobInfo,
		timestamp:     time.Now(),
	}

	q.Lock()
	if q.finished {
		q.Unlock()
		return
	}
	q.results.Value = r
	q.results = q.results.Next()
	q.Unlock()

	if q.shouldStopPolling(finished, r.domainJobInfo) {
		q.stopPolling()
	}
}

func (q *queue) scrapeDomainStats() (*stats.DomainStats, error) {
	scraper := domainstats.NewDomainstatsScraper(1)
	vmis := []*v1.VirtualMachineInstance{q.vmi}
	q.collector.Collect(vmis, scraper, collectionTimeout)

	values := scraper.GetValues()
	if len(values) != 1 {
		return nil, fmt.Errorf("expected 1 value from DomainstatsScraper, got %d", len(values))
	}

	return values[0].GetVmiStats().DomainStats, nil
}

func (q *queue) addCompletedStats(domainJobInfo stats.DomainJobInfo) {
	r := result{
		vmi:       q.vmi.Name,
		namespace: q.vmi.Namespace,
		node:      q.vmi.Status.NodeName,

		domainJobInfo: domainJobInfo,
		timestamp:     time.Now(),
	}

	q.Lock()
	q.results.Value = r
	q.results = q.results.Next()
	q.finished = true
	if q.ctxCancel != nil {
		q.ctxCancel()
	}
	q.Unlock()
}

func (q *queue) isFinished() bool {
	q.Lock()
	defer q.Unlock()

	return q.finished
}

func (q *queue) stopPolling() {
	q.Lock()
	defer q.Unlock()

	if q.finished {
		return
	}

	q.finished = true
	if q.ctxCancel != nil {
		q.ctxCancel()
	}
}

func hasCompletedDowntimeStats(domainJobInfo stats.DomainJobInfo) bool {
	return domainJobInfo.DowntimeSet || domainJobInfo.DowntimeNetSet
}

func (q *queue) shouldStopPolling(finished bool, domainJobInfo stats.DomainJobInfo) bool {
	if !finished {
		q.completedAt = nil
		return false
	}

	if hasCompletedDowntimeStats(domainJobInfo) {
		return true
	}

	if q.completedAt == nil {
		completedAt := time.Now()
		q.completedAt = &completedAt
		return false
	}

	return time.Since(*q.completedAt) >= completedStatsTimeout
}

func (q *queue) all() ([]result, bool) {
	q.Lock()
	defer q.Unlock()

	var results []result

	q.results.Do(func(r interface{}) {
		if r != nil {
			results = append(results, r.(result))
		}
	})
	q.results = q.results.Unlink(q.results.Len())

	return results, q.finished
}

func (q *queue) isMigrationFinished() bool {
	vmiRaw, exists, err := q.vmiStore.Get(q.vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get VMI %s/%s", q.vmi.Namespace, q.vmi.Name)
		return true
	}
	if !exists {
		return true
	}

	vmi := vmiRaw.(*v1.VirtualMachineInstance)
	return vmi.Status.MigrationState == nil || vmi.Status.MigrationState.Completed
}
