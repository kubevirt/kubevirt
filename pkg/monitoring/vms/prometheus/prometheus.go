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
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/version"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

const statsMaxAge time.Duration = collectionTimeout + 2*time.Second // "a bit more" than timeout, heuristic again

var (
	// see https://www.robustperception.io/exposing-the-software-version-to-prometheus
	versionDesc = prometheus.NewDesc(
		"kubevirt_info",
		"Version information",
		[]string{"goversion", "kubeversion"},
		nil,
	)

	storageIopsDesc = prometheus.NewDesc(
		"kubevirt_vm_storage_iops_total",
		"I/O operation performed.",
		[]string{"domain", "drive", "type"},
		nil,
	)
	// from now on: TODO: validate
	vcpuUsageDesc = prometheus.NewDesc(
		"kubevirt_vm_vcpu_seconds",
		"Vcpu elapsed time.",
		[]string{"domain", "id", "state"},
		nil,
	)
	networkTrafficDesc = prometheus.NewDesc(
		"kubevirt_vm_network_traffic_bytes_total",
		"network traffi.",
		[]string{"domain", "interface", "type"},
		nil,
	)
	memoryAvailableDesc = prometheus.NewDesc(
		"kubevirt_vm_memory_available_bytes",
		"amount of usable memory as seen by the domain.",
		[]string{"domain"},
		nil,
	)
	memoryResidentDesc = prometheus.NewDesc(
		"kubevirt_vm_memory_resident_bytes",
		"resident set size of the process running the domain",
		[]string{"domain"},
		nil,
	)
)

func updateMemory(vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	if vmStats.Memory.AvailableSet {
		mv, err := prometheus.NewConstMetric(
			memoryAvailableDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.Available)*1024,
			vmStats.Name,
		)
		if err == nil {
			ch <- mv
		}
	}
	if vmStats.Memory.RSSSet {
		mv, err := prometheus.NewConstMetric(
			memoryResidentDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.RSS)*1024,
			vmStats.Name,
		)
		if err == nil {
			ch <- mv
		}
	}
}

func updateVcpu(vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	for vcpuId, vcpu := range vmStats.Vcpu {
		if !vcpu.StateSet || !vcpu.TimeSet {
			continue
		}
		mv, err := prometheus.NewConstMetric(
			vcpuUsageDesc, prometheus.GaugeValue,
			float64(vcpu.Time/1000000000),
			vmStats.Name, fmt.Sprintf("%v", vcpuId), fmt.Sprintf("%v", vcpu.State),
		)
		if err != nil {
			continue
		}
		ch <- mv
	}

}

func updateBlock(vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	for _, block := range vmStats.Block {
		if !block.NameSet {
			continue
		}

		if block.RdReqsSet {
			mv, err := prometheus.NewConstMetric(
				storageIopsDesc, prometheus.CounterValue,
				float64(block.RdReqs),
				vmStats.Name, block.Name, "read",
			)
			if err == nil {
				ch <- mv
			}
		}
		if block.WrReqsSet {
			mv, err := prometheus.NewConstMetric(
				storageIopsDesc, prometheus.CounterValue,
				float64(block.WrReqs),
				vmStats.Name, block.Name, "write",
			)
			if err == nil {
				ch <- mv
			}
		}
	}

}

func updateNetwork(vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	for _, net := range vmStats.Net {
		if !net.NameSet {
			continue
		}
		if net.RxBytesSet {
			mv, err := prometheus.NewConstMetric(
				networkTrafficDesc, prometheus.CounterValue,
				float64(net.RxBytes),
				vmStats.Name, net.Name, "rx",
			)
			if err == nil {
				ch <- mv
			}
		}
		if net.TxBytesSet {
			mv, err := prometheus.NewConstMetric(
				networkTrafficDesc, prometheus.CounterValue,
				float64(net.TxBytes),
				vmStats.Name, net.Name, "tx",
			)
			if err == nil {
				ch <- mv
			}
		}
	}
}

func updateVersion(ch chan<- prometheus.Metric) {
	verinfo := version.Get()
	ch <- prometheus.MustNewConstMetric(
		versionDesc, prometheus.GaugeValue,
		1.0,
		verinfo.GoVersion, verinfo.GitVersion,
	)
}

type Collector struct {
	virtShareDir  string
	concCollector *concurrentCollector
}

func SetupCollector(virtShareDir string) *Collector {
	log.Log.Infof("Starting collector: sharedir=%v", virtShareDir)
	co := &Collector{
		virtShareDir:  virtShareDir,
		concCollector: NewConcurrentCollector(),
	}
	prometheus.MustRegister(co)
	return co
}

func (co *Collector) Describe(ch chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
	ch <- versionDesc
	ch <- storageIopsDesc
	ch <- vcpuUsageDesc
	ch <- networkTrafficDesc
	ch <- memoryAvailableDesc
	ch <- memoryResidentDesc
}

// Note that Collect could be called concurrently
func (co *Collector) Collect(ch chan<- prometheus.Metric) {
	updateVersion(ch)

	socketFiles, err := cmdclient.ListAllSockets(co.virtShareDir)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to list all sockets in '%s'", co.virtShareDir)
		return
	}

	if len(socketFiles) == 0 {
		log.Log.V(2).Infof("No VMs detected")
		return
	}

	scraper := &prometheusScraper{ch: ch}
	co.concCollector.Collect(socketFiles, scraper, collectionTimeout)
	return
}

type prometheusScraper struct {
	ch chan<- prometheus.Metric
}

func (ps *prometheusScraper) Scrape(socketFile string) {
	ts := time.Now()
	cli, err := cmdclient.GetClient(socketFile)
	if err != nil {
		log.Log.Reason(err).Error("failed to connect to cmd client socket")
		// Ignore failure to connect to client.
		// These are all local connections via unix socket.
		// A failure to connect means there's nothing on the other
		// end listening.
		return
	}
	defer cli.Close()

	vmStats, exists, err := cli.GetDomainStats()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to update stats from socket %s", socketFile)
		return
	}
	if !exists || vmStats.Name == "" {
		log.Log.V(2).Infof("disappearing VM on %s, ignored", socketFile) // VM may be shutting down
		return
	}

	// GetDomainStats() may hang for a long time.
	// If it wakes up past the timeout, there is no point in send back any metric.
	// In the best case the information is stale, in the worst case the information is stale *and*
	// the reporting channel is already closed, leading to a possible panic - see below
	elapsed := time.Now().Sub(ts)
	if elapsed > statsMaxAge {
		log.Log.Infof("took too long (%v) to collect stats from %s: ignored", elapsed, socketFile)
		return
	}

	ps.Report(socketFile, vmStats)
}

func (ps *prometheusScraper) Report(socketFile string, vmStats *stats.DomainStats) {
	// statsMaxAge is an estimation - and there is not better way to do that. So it is possible that
	// GetDomainStats() takes enough time to lag behind, but not enough to trigger the statsMaxAge check.
	// In this case the next functions will end up writing on a closed channel. This will panic.
	// It is actually OK in this case to abort the goroutine that panicked -that's what we want anyway,
	// and the very reason we collect in throwaway goroutines. We need however to avoid dump stacktraces in the logs.
	// Since this is a known failure condition, let's handle it explicitely.
	defer func() {
		if err := recover(); err != nil {
			log.Log.V(2).Warningf("collector goroutine panicked for VM %s: %s", socketFile, err)
		}
	}()

	updateMemory(vmStats, ps.ch)
	updateVcpu(vmStats, ps.ch)
	updateBlock(vmStats, ps.ch)
	updateNetwork(vmStats, ps.ch)
}
