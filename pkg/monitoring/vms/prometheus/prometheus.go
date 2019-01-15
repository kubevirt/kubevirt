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
		"kubevirt_vm_storage_iops",
		"I/O operation performed.",
		[]string{"domain", "drive", "type"},
		nil,
	)
	// from now on: TODO: validate
	vcpuUsageDesc = prometheus.NewDesc(
		"kubevirt_vm_vcpu_time",
		"Vcpu elapsed time, seconds.",
		[]string{"domain", "id", "state"},
		nil,
	)
	networkTrafficDesc = prometheus.NewDesc(
		"kubevirt_vm_network_traffic_bytes",
		"network traffic, bytes.",
		[]string{"domain", "interface", "type"},
		nil,
	)
	memoryUsageDesc = prometheus.NewDesc(
		"kubevirt_vm_memory_amount_bytes",
		"memory amount, bytes.",
		[]string{"domain", "type"},
		nil,
	)
)

func updateMemory(vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	if vmStats.Memory.UnusedSet {
		mv, err := prometheus.NewConstMetric(
			memoryUsageDesc, prometheus.GaugeValue,
			float64(vmStats.Memory.Unused),
			vmStats.Name, "unused",
		)
		if err == nil {
			ch <- mv
		}
	}
	if vmStats.Memory.AvailableSet {
		mv, err := prometheus.NewConstMetric(
			memoryUsageDesc, prometheus.GaugeValue,
			float64(vmStats.Memory.Available),
			vmStats.Name, "available",
		)
		if err == nil {
			ch <- mv
		}
	}
	if vmStats.Memory.ActualBalloonSet {
		mv, err := prometheus.NewConstMetric(
			memoryUsageDesc, prometheus.GaugeValue,
			float64(vmStats.Memory.ActualBalloon),
			vmStats.Name, "balloon",
		)
		if err == nil {
			ch <- mv
		}
	}
	if vmStats.Memory.RSSSet {
		mv, err := prometheus.NewConstMetric(
			memoryUsageDesc, prometheus.GaugeValue,
			float64(vmStats.Memory.RSS),
			vmStats.Name, "resident",
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
		if block.FlReqsSet {
			mv, err := prometheus.NewConstMetric(
				storageIopsDesc, prometheus.CounterValue,
				float64(block.FlReqs),
				vmStats.Name, block.Name, "flush",
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
	virtShareDir string
}

func SetupCollector(virtShareDir string) *Collector {
	log.Log.Infof("Starting collector: sharedir=%v", virtShareDir)
	co := &Collector{
		virtShareDir: virtShareDir,
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
	ch <- memoryUsageDesc
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

	cc := concurrentCollector{
		Scraper: &prometheusScraper{
			ch: ch,
		},
	}
	cc.Collect(socketFiles)
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

	// GetDomainStats() may hang for a long time. So, when it wakes up, we must ensure not to
	// send back stale information. We just check the freshness of the informations we collected
	// against a threshold.
	elapsed := time.Now().Sub(ts)
	if elapsed > statsMaxAge {
		log.Log.Infof("took too long (%v) to collect stats from %s: ignored", elapsed, socketFile)
		return
	}

	updateMemory(vmStats, ps.ch)
	updateVcpu(vmStats, ps.ch)
	updateBlock(vmStats, ps.ch)
	updateNetwork(vmStats, ps.ch)

}
