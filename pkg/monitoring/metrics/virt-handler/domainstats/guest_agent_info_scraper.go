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
	"fmt"
	"sync"
	"time"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

const (
	cacheTimeout = 1 * time.Minute
)

type guestAgentInfoCache struct {
	timestamp time.Time
	info      *k6tv1.VirtualMachineInstanceGuestAgentInfo
}

type GuestAgentInfoScraper struct {
	ch       chan *VirtualMachineInstanceReport
	cache    map[string]*guestAgentInfoCache
	mutex    sync.Mutex
	closed   bool
	closeMux sync.Mutex
}

func NewGuestAgentInfoScraper() *GuestAgentInfoScraper {
	return &GuestAgentInfoScraper{
		ch:     make(chan *VirtualMachineInstanceReport),
		cache:  make(map[string]*guestAgentInfoCache),
		closed: false,
	}
}

func (d *GuestAgentInfoScraper) Scrape(socketFile string, vmi *k6tv1.VirtualMachineInstance) {
	ts := time.Now()

	vmStats, err := d.gatherMetrics(socketFile, vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to scrape metrics from %s", socketFile)
		return
	}

	// GetDomainStats() may hang for a long time.
	// If it wakes up past the timeout, there is no point in send back any metric.
	// In the best case the information is stale, in the worst case the information is stale *and*
	// the reporting channel is already closed, leading to a possible panic - see below
	elapsed := time.Since(ts)
	if elapsed > collector.StatsMaxAge {
		log.Log.Infof("took too long (%v) to collect stats from %s: ignored", elapsed, socketFile)
		return
	}

	report(vmi, vmStats, d.ch)
}

func (d *GuestAgentInfoScraper) Complete() {
	d.closeMux.Lock()
	defer d.closeMux.Unlock()

	if !d.closed {
		close(d.ch)
		d.closed = true
	}
}

func (d *GuestAgentInfoScraper) Reset() {
	d.closeMux.Lock()
	defer d.closeMux.Unlock()

	if d.closed {
		d.ch = make(chan *VirtualMachineInstanceReport)
		d.closed = false
	}
}

func (d *GuestAgentInfoScraper) gatherMetrics(socketFile string, vmi *k6tv1.VirtualMachineInstance) (*VirtualMachineInstanceStats, error) {
	cached, _, exists := d.getCacheEntry(socketFile)
	if exists {
		return cached, nil
	}

	cli, err := cmdclient.NewClient(socketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cmd client socket: %w", err)
	}
	defer cli.Close()

	vmStats := &VirtualMachineInstanceStats{}

	vmStats.GuestAgentInfo, err = cli.GetGuestInfo(vmi, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get guest agent info: %w", err)
	}

	d.addCacheEntry(socketFile, time.Now(), vmStats.GuestAgentInfo)

	return vmStats, nil
}

func (d *GuestAgentInfoScraper) addCacheEntry(socketFile string, timestamp time.Time, info *k6tv1.VirtualMachineInstanceGuestAgentInfo) {
	if info == nil || info.Hostname == "" {
		log.Log.V(3).Infof("skipping empty guest agent info for socket %s", socketFile)
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.cache[socketFile] = &guestAgentInfoCache{
		timestamp: timestamp,
		info:      info.DeepCopy(),
	}
}

func (d *GuestAgentInfoScraper) getCacheEntry(socketFile string) (*VirtualMachineInstanceStats, time.Time, bool) {
	now := time.Now()

	d.mutex.Lock()

	// Clean up expired cache entries
	for socketFile, cached := range d.cache {
		if now.Sub(cached.timestamp) >= cacheTimeout {
			delete(d.cache, socketFile)
		}
	}

	// Check if the entry exists and is still valid
	cached, exists := d.cache[socketFile]

	d.mutex.Unlock()

	if exists {
		vmStats := &VirtualMachineInstanceStats{
			GuestAgentInfo: cached.info,
		}
		return vmStats, cached.timestamp, true
	}

	return nil, time.Time{}, false
}
