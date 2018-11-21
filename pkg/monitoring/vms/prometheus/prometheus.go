/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package prometheus creates and registers prometheus metrics with
// rest clients. To use this package, you just have to import it.
package prometheus

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var (
	// requestLatency is a Prometheus Summary metric type partitioned by
	// "verb" and "url" labels. It is used for the rest client latency metrics.
	storageIops = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kubevirt",
			Subsystem: "vm",
			Name:      "storage_iops",
			Help:      "I/O operation performed.",
		},
		[]string{"domain", "drive", "type"},
	)
	// from now on: for demo purposes
	vcpuUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubevirt",
			Subsystem: "vm",
			Name:      "vcpu_time",
			Help:      "Vcpu elapsed time, seconds.",
		},
		[]string{"domain", "id", "state"},
	)
	networkTraffic = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kubevirt",
			Subsystem: "vm",
			Name:      "network_traffic_bytes",
			Help:      "network traffic, bytes.",
		},
		[]string{"domain", "interface", "type"},
	)
	memoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kubevirt",
			Subsystem: "vm",
			Name:      "memory_amount_bytes",
			Help:      "memory amount, bytes.",
		},
		[]string{"domain", "type"},
	)
)

func init() {
	prometheus.MustRegister(storageIops)
	prometheus.MustRegister(vcpuUsage)
	prometheus.MustRegister(networkTraffic)
	prometheus.MustRegister(memoryUsage)
}

func Update(cli cmdclient.LauncherClient) error {
	stats, exists, err := cli.GetDomainStats()
	if err != nil {
		return err
	}
	if !exists || stats.Name == "" {
		return nil
	}

	if stats.Memory.UnusedSet {
		memoryUsage.With(prometheus.Labels{"domain": stats.Name, "type": "unused"}).Set(float64(stats.Memory.Unused))
	}
	if stats.Memory.AvailableSet {
		memoryUsage.With(prometheus.Labels{"domain": stats.Name, "type": "available"}).Set(float64(stats.Memory.Available))
	}
	if stats.Memory.ActualBalloonSet {
		memoryUsage.With(prometheus.Labels{"domain": stats.Name, "type": "balloon"}).Set(float64(stats.Memory.ActualBalloon))
	}
	if stats.Memory.RSSSet {
		memoryUsage.With(prometheus.Labels{"domain": stats.Name, "type": "resident"}).Set(float64(stats.Memory.RSS))
	}

	for vcpuId, vcpu := range stats.Vcpu {
		if !vcpu.StateSet || !vcpu.TimeSet {
			continue
		}
		vcpuUsage.With(prometheus.Labels{"domain": stats.Name, "id": fmt.Sprintf("%v", vcpuId), "state": fmt.Sprintf("%v", vcpu.State)}).Set(float64(vcpu.Time / 1000000000))
	}

	for _, block := range stats.Block {
		if !block.NameSet {
			continue
		}
		if block.RdReqsSet {
			storageIops.With(prometheus.Labels{"domain": stats.Name, "drive": block.Name, "type": "read"}).Add(float64(block.RdReqs))
		}
		if block.WrReqsSet {
			storageIops.With(prometheus.Labels{"domain": stats.Name, "drive": block.Name, "type": "write"}).Add(float64(block.WrReqs))
		}
		if block.FlReqsSet {
			storageIops.With(prometheus.Labels{"domain": stats.Name, "drive": block.Name, "type": "flush"}).Add(float64(block.FlReqs))
		}
	}

	for _, net := range stats.Net {
		if !net.NameSet {
			continue
		}
		if net.RxBytesSet {
			networkTraffic.With(prometheus.Labels{"domain": stats.Name, "interface": net.Name, "type": "rx"}).Add(float64(net.RxBytes))
		}
		if net.TxBytesSet {
			networkTraffic.With(prometheus.Labels{"domain": stats.Name, "interface": net.Name, "type": "tx"}).Add(float64(net.TxBytes))
		}
	}

	return nil
}
