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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package collector

import (
	"sync"
	"time"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

const (
	CollectionTimeout = 10 * time.Second                  // "long enough", crude heuristic
	StatsMaxAge       = CollectionTimeout + 2*time.Second // "a bit more" than timeout, heuristic again
)

type vmiSocketMap map[string]*k6tv1.VirtualMachineInstance

type Collector interface {
	Collect(vmis []*k6tv1.VirtualMachineInstance, scraper MetricsScraper, timeout time.Duration) (skipped []string, completed bool)
}

type ConcurrentCollector struct {
	lock             sync.Mutex
	clientsPerKey    map[string]int
	maxClientsPerKey int
	socketMapper     func(vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap
}

func NewConcurrentCollector(MaxRequestsPerKey int) Collector {
	return NewConcurrentCollectorWithMapper(MaxRequestsPerKey, newvmiSocketMapFromVMIs)
}

func NewConcurrentCollectorWithMapper(MaxRequestsPerKey int, mapper func(vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap) Collector {
	return &ConcurrentCollector{
		clientsPerKey:    make(map[string]int),
		maxClientsPerKey: MaxRequestsPerKey,
		socketMapper:     mapper,
	}
}

func (cc *ConcurrentCollector) Collect(vmis []*k6tv1.VirtualMachineInstance, scraper MetricsScraper, timeout time.Duration) ([]string, bool) {
	socketToVMIs := cc.socketMapper(vmis)
	log.Log.V(3).Infof("Collecting VM metrics from %d sources", len(socketToVMIs))
	var busyScrapers sync.WaitGroup

	var skipped []string
	for key, vmi := range socketToVMIs {
		reserved := cc.reserveKey(key)
		if !reserved {
			log.Log.Warningf("Source %s busy from a previous collection, skipped", key)
			skipped = append(skipped, key)
			continue
		}

		log.Log.V(4).Infof("Source %s responsive, scraping", key)
		busyScrapers.Add(1)
		go cc.collectFromSource(scraper, &busyScrapers, key, vmi)
	}

	completed := true
	c := make(chan struct{})
	go func() {
		busyScrapers.Wait()
		c <- struct{}{}
	}()
	select {
	case <-c:
		log.Log.V(3).Infof("Collection successful")
	case <-time.After(timeout):
		log.Log.Warning("Collection timeout")
		completed = false
	}

	log.Log.V(4).Infof("Collection completed")
	scraper.Complete()

	return skipped, completed
}

func (cc *ConcurrentCollector) collectFromSource(scraper MetricsScraper, wg *sync.WaitGroup, socket string, vmi *k6tv1.VirtualMachineInstance) {
	defer wg.Done()
	defer cc.releaseKey(socket)

	log.Log.V(4).Infof("Getting stats from source %s", socket)
	scraper.Scrape(socket, vmi)
	log.Log.V(4).Infof("Updated stats from source %s", socket)
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
		socketPath, err := cmdclient.FindSocketOnHost(vmi)
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
