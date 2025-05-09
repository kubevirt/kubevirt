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

package domainstats

import (
	"fmt"
	"time"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

type DomainDirtyRateStatsScraper struct {
	ch chan *VirtualMachineInstanceReport
}

func NewDomainsDirtyRateStatsScraper(channelLength int) *DomainDirtyRateStatsScraper {
	return &DomainDirtyRateStatsScraper{
		ch: make(chan *VirtualMachineInstanceReport, channelLength),
	}
}

func (d DomainDirtyRateStatsScraper) Scrape(socketFile string, vmi *k6tv1.VirtualMachineInstance) {
	ts := time.Now()

	vmStats, err := d.gatherMetrics(socketFile)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to scrape metrics from %s", socketFile)
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

func (d DomainDirtyRateStatsScraper) Complete() {
	close(d.ch)
}

func (d DomainDirtyRateStatsScraper) gatherMetrics(socketFile string) (*VirtualMachineInstanceStats, error) {
	cli, err := cmdclient.NewClient(socketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cmd client socket: %w", err)
	}
	defer cli.Close()

	vmStats := &VirtualMachineInstanceStats{
		DomainStats: &stats.DomainStats{
			DirtyRate: &stats.DomainStatsDirtyRate{
				MegabytesPerSecondSet: true,
			},
		},
	}

	vmStats.DomainStats.DirtyRate.MegabytesPerSecond, err = cli.GetDomainDirtyRateStats()
	if err != nil {
		return nil, fmt.Errorf("failed to update dirty rate stats from socket %s: %w", socketFile, err)
	}

	return vmStats, nil
}

func report(vmi *k6tv1.VirtualMachineInstance, vmStats *VirtualMachineInstanceStats, ch chan *VirtualMachineInstanceReport) {
	// statsMaxAge is an estimation - and there is no better way to do that. So it is possible that
	// GetDomainStats() takes enough time to lag behind, but not enough to trigger the statsMaxAge check.
	// In this case the next functions will end up writing on a closed channel. This will panic.
	// It is actually OK in this case to abort the goroutine that panicked -that's what we want anyway,
	// and the very reason we collect in throwaway goroutines. We need however to avoid dump stacktraces in the logs.
	// Since this is a known failure condition, let's handle it explicitly.
	defer func() {
		if err := recover(); err != nil {
			log.Log.Warningf("collector goroutine panicked for VM %s: %v", vmi.Name, err)
		}
	}()

	ch <- newVirtualMachineInstanceReport(vmi, vmStats)
}
