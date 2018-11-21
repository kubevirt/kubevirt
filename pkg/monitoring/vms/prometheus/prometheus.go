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

	"github.com/prometheus/client_golang/prometheus"

	"kubevirt.io/kubevirt/pkg/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var (
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

func Update(cli cmdclient.LauncherClient, ch chan<- prometheus.Metric) error {
	vmStats, exists, err := cli.GetDomainStats()
	if err != nil {
		return err
	}
	if !exists || vmStats.Name == "" {
		return nil
	}

	updateMemory(vmStats, ch)
	updateVcpu(vmStats, ch)
	updateBlock(vmStats, ch)
	updateNetwork(vmStats, ch)

	return nil
}

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

type Collector struct {
	virtShareDir string
}

func SetupCollector(virtShareDir string) *Collector {
	log.Log.Infof("starting updater: sharedir=%v", virtShareDir)
	co := &Collector{
		virtShareDir: virtShareDir,
	}
	prometheus.MustRegister(co)
	return co
}

func (co Collector) Describe(ch chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
	ch <- storageIopsDesc
	ch <- vcpuUsageDesc
	ch <- networkTrafficDesc
	ch <- memoryUsageDesc
}

// Note that Collect could be called concurrently
func (co Collector) Collect(ch chan<- prometheus.Metric) {
	socketFiles, err := cmdclient.ListAllSockets(co.virtShareDir)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to list all sockets in '%s'", co.virtShareDir)
		return
	}
	for _, socketFile := range socketFiles {
		log.Log.V(3).Infof("Getting stats from sock %s", socketFile)
		client, err := cmdclient.GetClient(socketFile)
		if err != nil {
			log.Log.Reason(err).Error("failed to connect to cmd client socket")
			// Ignore failure to connect to client.
			// These are all local connections via unix socket.
			// A failure to connect means there's nothing on the other
			// end listening.
			continue
		}
		defer client.Close()

		err = Update(client, ch)
		if err != nil {
			log.Log.Reason(err).Error("failed to connect to update stats from socket")
			continue
		}
		log.Log.V(3).Infof("Updated stats from sock %s", socketFile)
	}
	return
}
