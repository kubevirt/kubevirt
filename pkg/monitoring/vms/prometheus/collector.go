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

package prometheus

import (
	"sync"
	"time"

	k6tv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

const collectionTimeout time.Duration = 10 * time.Second // "long enough", crude heuristic

type vmiSocketMap map[string]*k6tv1.VirtualMachineInstance

type metricsScraper interface {
	Scrape(key string, vmi *k6tv1.VirtualMachineInstance)
}

type concurrentCollector struct {
	lock     sync.Mutex
	busyKeys map[string]bool
}

func NewConcurrentCollector() *concurrentCollector {
	return &concurrentCollector{
		busyKeys: make(map[string]bool),
	}
}

func (cc *concurrentCollector) Collect(socketToVMIs vmiSocketMap, scraper metricsScraper, timeout time.Duration) ([]string, bool) {
	log.Log.V(3).Infof("Collecting VM metrics from %d sources", len(socketToVMIs))
	var busyScrapers sync.WaitGroup

	skipped := []string{}
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

	return skipped, completed
}

func (cc *concurrentCollector) collectFromSource(scraper metricsScraper, wg *sync.WaitGroup, key string, vmi *k6tv1.VirtualMachineInstance) {
	defer wg.Done()
	defer cc.releaseKey(key)

	log.Log.V(4).Infof("Getting stats from source %s", key)
	scraper.Scrape(key, vmi)
	log.Log.V(4).Infof("Updated stats from source %s", key)
}

func (cc *concurrentCollector) reserveKey(key string) bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	busy := cc.busyKeys[key]
	if busy {
		return false
	}
	cc.busyKeys[key] = true
	return true
}

func (cc *concurrentCollector) releaseKey(key string) {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	cc.busyKeys[key] = false
}
