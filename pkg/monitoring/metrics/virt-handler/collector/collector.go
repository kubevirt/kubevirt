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

package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

const (
	// "long enough", crude heuristic
	CollectionTimeout = 10 * time.Second
	// "a bit more" than timeout, heuristic again
	StatsMaxAge = CollectionTimeout + 2*time.Second

	// DefaultMaxConcurrentSources caps how many virt-launcher sockets may be
	// scraped in parallel. Unbounded fan-out starves virt-handler (including
	// /healthz) when many VMIs share a node.
	DefaultMaxConcurrentSources = 10

	logVerbosityInfo  = 3
	logVerbosityDebug = 4
)

type vmiSocketMap map[string]*k6tv1.VirtualMachineInstance

type Collector interface {
	Collect(vmis []*k6tv1.VirtualMachineInstance, scraper MetricsScraper, timeout time.Duration) (skipped []string, completed bool)
}

type ConcurrentCollector struct {
	lock                 sync.Mutex
	clientsPerKey        map[string]int
	maxClientsPerKey     int
	maxConcurrentSources int
	scrapeSlots          chan struct{}
	socketMapper         func(vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap
}

func NewConcurrentCollector(maxRequestsPerKey int) Collector {
	return NewConcurrentCollectorWithMapper(maxRequestsPerKey, newvmiSocketMapFromVMIs)
}

func NewConcurrentCollectorWithMapper(maxRequestsPerKey int, mapper func(vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap) Collector {
	return NewConcurrentCollectorWithLimits(maxRequestsPerKey, DefaultMaxConcurrentSources, mapper)
}

// NewConcurrentCollectorWithLimits creates a collector that:
//   - skips a socket key already at maxRequestsPerKey in-flight scrapes
//   - never runs more than maxConcurrentSources scrapes at once
func NewConcurrentCollectorWithLimits(
	maxRequestsPerKey, maxConcurrentSources int,
	mapper func(vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap,
) Collector {
	if maxRequestsPerKey < 1 {
		panic(fmt.Sprintf("maxRequestsPerKey must be >= 1, got %d", maxRequestsPerKey))
	}
	if maxConcurrentSources < 1 {
		panic(fmt.Sprintf("maxConcurrentSources must be >= 1, got %d", maxConcurrentSources))
	}
	return &ConcurrentCollector{
		clientsPerKey:        make(map[string]int),
		maxClientsPerKey:     maxRequestsPerKey,
		maxConcurrentSources: maxConcurrentSources,
		scrapeSlots:          make(chan struct{}, maxConcurrentSources),
		socketMapper:         mapper,
	}
}

func (cc *ConcurrentCollector) Collect(
	vmis []*k6tv1.VirtualMachineInstance, scraper MetricsScraper, timeout time.Duration,
) ([]string, bool) {
	socketToVMIs := cc.socketMapper(vmis)
	log.Log.V(logVerbosityInfo).Infof("Collecting VM metrics from %d sources (max concurrent %d)",
		len(socketToVMIs), cc.maxConcurrentSources)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var busyScrapers sync.WaitGroup

	var skipped []string
	for key, vmi := range socketToVMIs {
		reserved := cc.reserveKey(key)
		if !reserved {
			log.Log.Warningf("Source %s busy from a previous collection, skipped", key)
			skipped = append(skipped, key)
			continue
		}

		log.Log.V(logVerbosityDebug).Infof("Source %s responsive, scraping", key)
		busyScrapers.Add(1)
		go cc.collectFromSource(ctx, scraper, &busyScrapers, key, vmi)
	}

	completed := true
	c := make(chan struct{})
	go func() {
		busyScrapers.Wait()
		c <- struct{}{}
	}()
	select {
	case <-c:
		log.Log.V(logVerbosityInfo).Infof("Collection successful")
	case <-ctx.Done():
		log.Log.Warning("Collection timeout")
		completed = false
	}

	log.Log.V(logVerbosityDebug).Infof("Collection completed")
	scraper.Complete()

	return skipped, completed
}

func (cc *ConcurrentCollector) collectFromSource(
	ctx context.Context,
	scraper MetricsScraper, wg *sync.WaitGroup, socket string, vmi *k6tv1.VirtualMachineInstance,
) {
	defer wg.Done()
	defer cc.releaseKey(socket)

	// Bounded acquire: if Collect timed out (or previous hung scrapes still
	// hold every scrapeSlots slot), skip instead of blocking forever.
	select {
	case cc.scrapeSlots <- struct{}{}:
		defer func() { <-cc.scrapeSlots }()
	case <-ctx.Done():
		log.Log.Warningf("Timed out waiting for scrape slot for source %s, skipped", socket)
		return
	}

	log.Log.V(logVerbosityDebug).Infof("Getting stats from source %s", socket)
	scraper.Scrape(socket, vmi)
	log.Log.V(logVerbosityDebug).Infof("Updated stats from source %s", socket)
}

func (cc *ConcurrentCollector) reserveKey(key string) bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	count := cc.clientsPerKey[key]
	if count >= cc.maxClientsPerKey {
		return false
	}
	cc.clientsPerKey[key] += 1
	return true
}

func (cc *ConcurrentCollector) releaseKey(key string) {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	cc.clientsPerKey[key] -= 1
}

func newvmiSocketMapFromVMIs(vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap {
	if len(vmis) == 0 {
		return nil
	}

	ret := make(vmiSocketMap)
	for _, vmi := range vmis {
		socketPath, err := cmdclient.FindSocket(vmi)
		if err != nil {
			// nothing to scrape...
			// this means there's no socket or the socket
			// is currently unreachable for this vmi.
			continue
		}
		ret[socketPath] = vmi
	}
	return ret
}
