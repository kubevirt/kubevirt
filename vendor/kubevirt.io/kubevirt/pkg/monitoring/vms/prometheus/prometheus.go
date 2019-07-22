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
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	k6tv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/version"
	"kubevirt.io/kubevirt/pkg/util/lookup"
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
		[]string{
			"node", "namespace", "name",
			"domain", "drive", "type",
		},
		nil,
	)
	storageTrafficDesc = prometheus.NewDesc(
		"kubevirt_vm_storage_traffic_bytes_total",
		"storage traffic.",
		[]string{
			"node", "namespace", "name",
			"domain", "drive", "type",
		},
		nil,
	)
	storageTimesDesc = prometheus.NewDesc(
		"kubevirt_vm_storage_times_ms_total",
		"storage operation time.",
		[]string{
			"node", "namespace", "name",
			"domain", "drive", "type",
		},
		nil,
	)
	vcpuUsageDesc = prometheus.NewDesc(
		"kubevirt_vm_vcpu_seconds",
		"Vcpu elapsed time.",
		[]string{
			"node", "namespace", "name",
			"domain", "id", "state",
		},
		nil,
	)
	networkTrafficBytesDesc = prometheus.NewDesc(
		"kubevirt_vm_network_traffic_bytes_total",
		"network traffic.",
		[]string{
			"node", "namespace", "name",
			"domain", "interface", "type",
		},
		nil,
	)
	networkTrafficPktsDesc = prometheus.NewDesc(
		"kubevirt_vm_network_traffic_packets_total",
		"network traffic.",
		[]string{
			"node", "namespace", "name",
			"domain", "interface", "type",
		},
		nil,
	)
	networkErrorsDesc = prometheus.NewDesc(
		"kubevirt_vm_network_errors_total",
		"network errors.",
		[]string{
			"node", "namespace", "name",
			"domain", "interface", "type",
		},
		nil,
	)
	memoryAvailableDesc = prometheus.NewDesc(
		"kubevirt_vm_memory_available_bytes",
		"amount of usable memory as seen by the domain.",
		[]string{
			"node", "namespace", "name",
			"domain",
		},
		nil,
	)
	memoryResidentDesc = prometheus.NewDesc(
		"kubevirt_vm_memory_resident_bytes",
		"resident set size of the process running the domain",
		[]string{
			"node", "namespace", "name",
			"domain",
		},
		nil,
	)

	swapTrafficDesc = prometheus.NewDesc(
		"kubevirt_vm_memory_swap_traffic_bytes_total",
		"swap memory traffic.",
		[]string{
			"node", "namespace", "name",
			"domain", "type",
		},
		nil,
	)
)

func updateMemory(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	if vmStats.Memory.AvailableSet {
		mv, err := prometheus.NewConstMetric(
			memoryAvailableDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.Available)*1024,
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
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
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			vmStats.Name,
		)
		if err == nil {
			ch <- mv
		}
	}

	if vmStats.Memory.SwapInSet {
		mv, err := prometheus.NewConstMetric(
			swapTrafficDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.SwapIn)*1024,
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			vmStats.Name, "in",
		)
		if err == nil {
			ch <- mv
		}
	}
	if vmStats.Memory.SwapInSet {
		mv, err := prometheus.NewConstMetric(
			swapTrafficDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.SwapOut)*1024,
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			vmStats.Name, "out",
		)
		if err == nil {
			ch <- mv
		}
	}
}

func updateVcpu(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	for vcpuId, vcpu := range vmStats.Vcpu {
		if !vcpu.StateSet || !vcpu.TimeSet {
			continue
		}
		mv, err := prometheus.NewConstMetric(
			vcpuUsageDesc, prometheus.GaugeValue,
			float64(vcpu.Time/1000000000),
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			vmStats.Name, fmt.Sprintf("%v", vcpuId), fmt.Sprintf("%v", vcpu.State),
		)
		if err != nil {
			continue
		}
		ch <- mv
	}

}

func updateBlock(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	for _, block := range vmStats.Block {
		if !block.NameSet {
			continue
		}

		if block.RdReqsSet {
			mv, err := prometheus.NewConstMetric(
				storageIopsDesc, prometheus.CounterValue,
				float64(block.RdReqs),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
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
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, block.Name, "write",
			)
			if err == nil {
				ch <- mv
			}
		}

		if block.RdBytesSet {
			mv, err := prometheus.NewConstMetric(
				storageTrafficDesc, prometheus.CounterValue,
				float64(block.RdBytes),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, block.Name, "read",
			)
			if err == nil {
				ch <- mv
			}
		}
		if block.WrBytesSet {
			mv, err := prometheus.NewConstMetric(
				storageTrafficDesc, prometheus.CounterValue,
				float64(block.WrBytes),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, block.Name, "write",
			)
			if err == nil {
				ch <- mv
			}
		}

		if block.RdTimesSet {
			mv, err := prometheus.NewConstMetric(
				storageTimesDesc, prometheus.CounterValue,
				float64(block.RdTimes),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, block.Name, "read",
			)
			if err == nil {
				ch <- mv
			}
		}
		if block.WrTimesSet {
			mv, err := prometheus.NewConstMetric(
				storageTimesDesc, prometheus.CounterValue,
				float64(block.WrTimes),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, block.Name, "write",
			)
			if err == nil {
				ch <- mv
			}
		}
	}
}

func updateNetwork(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric) {
	for _, net := range vmStats.Net {
		if !net.NameSet {
			continue
		}
		if net.RxBytesSet {
			mv, err := prometheus.NewConstMetric(
				networkTrafficBytesDesc, prometheus.CounterValue,
				float64(net.RxBytes),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, net.Name, "rx",
			)
			if err == nil {
				ch <- mv
			}
		}
		if net.RxPktsSet {
			mv, err := prometheus.NewConstMetric(
				networkTrafficPktsDesc, prometheus.CounterValue,
				float64(net.RxPkts),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, net.Name, "rx",
			)
			if err == nil {
				ch <- mv
			}
		}
		if net.RxErrsSet {
			mv, err := prometheus.NewConstMetric(
				networkErrorsDesc, prometheus.CounterValue,
				float64(net.RxErrs),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, net.Name, "rx",
			)
			if err == nil {
				ch <- mv
			}
		}

		if net.TxBytesSet {
			mv, err := prometheus.NewConstMetric(
				networkTrafficBytesDesc, prometheus.CounterValue,
				float64(net.TxBytes),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, net.Name, "tx",
			)
			if err == nil {
				ch <- mv
			}
		}
		if net.TxPktsSet {
			mv, err := prometheus.NewConstMetric(
				networkTrafficPktsDesc, prometheus.CounterValue,
				float64(net.TxPkts),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				vmStats.Name, net.Name, "tx",
			)
			if err == nil {
				ch <- mv
			}
		}
		if net.TxErrsSet {
			mv, err := prometheus.NewConstMetric(
				networkErrorsDesc, prometheus.CounterValue,
				float64(net.TxErrs),
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
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
	virtCli       kubecli.KubevirtClient
	virtShareDir  string
	nodeName      string
	concCollector *concurrentCollector
}

func SetupCollector(virtCli kubecli.KubevirtClient, virtShareDir, nodeName string) *Collector {
	log.Log.Infof("Starting collector: node name=%v", nodeName)
	co := &Collector{
		virtCli:       virtCli,
		virtShareDir:  virtShareDir,
		nodeName:      nodeName,
		concCollector: NewConcurrentCollector(),
	}
	prometheus.MustRegister(co)
	return co
}

func (co *Collector) Describe(ch chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
	ch <- versionDesc
	ch <- storageIopsDesc
	ch <- storageTrafficDesc
	ch <- storageTimesDesc
	ch <- vcpuUsageDesc
	ch <- networkTrafficBytesDesc
	ch <- networkTrafficPktsDesc
	ch <- networkErrorsDesc
	ch <- memoryAvailableDesc
	ch <- memoryResidentDesc
}

func newvmiSocketMapFromVMIs(baseDir string, vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap {
	if len(vmis) == 0 {
		return nil
	}

	ret := make(vmiSocketMap)
	for _, vmi := range vmis {
		socketPath := cmdclient.SocketFromUID(baseDir, string(vmi.UID))
		ret[socketPath] = vmi
	}
	return ret
}

// Note that Collect could be called concurrently
func (co *Collector) Collect(ch chan<- prometheus.Metric) {
	updateVersion(ch)

	vmis, err := lookup.VirtualMachinesOnNode(co.virtCli, co.nodeName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to list all VMIs in '%s': %s", co.nodeName, err)
		return
	}

	if len(vmis) == 0 {
		log.Log.V(2).Infof("No VMIs detected")
		return
	}

	socketToVMIs := newvmiSocketMapFromVMIs(co.virtShareDir, vmis)
	scraper := &prometheusScraper{ch: ch}
	co.concCollector.Collect(socketToVMIs, scraper, collectionTimeout)
	return
}

type prometheusScraper struct {
	ch chan<- prometheus.Metric
}

type vmiStatsInfo struct {
	vmiSpec  *k6tv1.VirtualMachineInstance
	vmiStats *stats.DomainStats
}

func (ps *prometheusScraper) Scrape(socketFile string, vmi *k6tv1.VirtualMachineInstance) {
	ts := time.Now()
	cli, err := cmdclient.NewClient(socketFile)
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

	ps.Report(socketFile, vmi, vmStats)
}

func (ps *prometheusScraper) Report(socketFile string, vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats) {
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

	updateMemory(vmi, vmStats, ps.ch)
	updateVcpu(vmi, vmStats, ps.ch)
	updateBlock(vmi, vmStats, ps.ch)
	updateNetwork(vmi, vmStats, ps.ch)
}

func Handler(MaxRequestsInFlight int) http.Handler {
	return promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(
			prometheus.DefaultGatherer,
			promhttp.HandlerOpts{
				MaxRequestsInFlight: MaxRequestsInFlight,
			}),
	)
}
