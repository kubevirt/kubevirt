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

package domainstats

import (
	"fmt"
	"time"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type DomainstatsScraper struct {
	ch chan *VirtualMachineInstanceReport
}

func NewDomainstatsScraper(channelLength int) *DomainstatsScraper {
	return &DomainstatsScraper{
		ch: make(chan *VirtualMachineInstanceReport, channelLength),
	}
}

func (d DomainstatsScraper) Scrape(socketFile string, vmi *k6tv1.VirtualMachineInstance) {
	ts := time.Now()

	exists, vmStats, err := d.gatherMetrics(socketFile)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to scrape metrics from %s", socketFile)
		return
	}

	if !exists || vmStats.DomainStats.Name == "" {
		log.Log.V(2).Infof("disappearing VM on %s, ignored", socketFile) // VM may be shutting down
		return
	}

	// GetDomainStats() may hang for a long time.
	// If it wakes up past the timeout, there is no point in send back any metric.
	// In the best case the information is stale, in the worst case the information is stale *and*
	// the reporting channel is already closed, leading to a possible panic - see below
	elapsed := time.Now().Sub(ts)
	if elapsed > collector.StatsMaxAge {
		log.Log.Infof("took too long (%v) to collect stats from %s: ignored", elapsed, socketFile)
		return
	}

	report(vmi, vmStats, d.ch)
}

func (d DomainstatsScraper) GetValues() []VirtualMachineInstanceReport {
	var reports []VirtualMachineInstanceReport
	for report := range d.ch {
		reports = append(reports, *report)
	}
	return reports
}

func (d DomainstatsScraper) Complete() {
	close(d.ch)
}

func (d DomainstatsScraper) gatherMetrics(socketFile string) (bool, *VirtualMachineInstanceStats, error) {
	cli, err := cmdclient.NewClient(socketFile)
	if err != nil {
		// Ignore failure to connect to client.
		// These are all local connections via unix socket.
		// A failure to connect means there's nothing on the other
		// end listening.
		return false, nil, fmt.Errorf("failed to connect to cmd client socket: %w", err)
	}
	defer cli.Close()

	vmStats := &VirtualMachineInstanceStats{}
	var exists bool

	vmStats.DomainStats, exists, err = cli.GetDomainStats()
	if err != nil {
		return false, nil, fmt.Errorf("failed to update domain stats from socket %s: %w", socketFile, err)
	}

	vmStats.FsStats, err = cli.GetFilesystems()
	if err != nil {
		return false, nil, fmt.Errorf("failed to update filesystem stats from socket %s: %w", socketFile, err)
	}

	return exists, vmStats, nil
}
