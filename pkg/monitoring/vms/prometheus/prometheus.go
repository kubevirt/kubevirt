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
	"strings"
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

	// Formatter used to sanitize k8s metadata into metric labels
	labelFormatter = strings.NewReplacer(".", "_", "/", "_", "-", "_")

	// Preffixes used when transforming K8s metadata into metric labels
	labelPrefix = "kubernetes_vmi_label_"

	// see https://www.robustperception.io/exposing-the-software-version-to-prometheus
	versionDesc = prometheus.NewDesc(
		"kubevirt_info",
		"Version information",
		[]string{"goversion", "kubeversion"},
		nil,
	)

	// higher-level, telemetry-friendly metrics
	vmiCountDesc = prometheus.NewDesc(
		"kubevirt_vmi_phase_count",
		"VMI phase.",
		[]string{
			"node", "phase",
		},
		nil,
	)
)

func tryToPushMetric(desc *prometheus.Desc, mv prometheus.Metric, err error, ch chan<- prometheus.Metric) {
	if err != nil {
		log.Log.V(4).Warningf("Error creating the new const metric for %s: %s", desc, err)
		return
	}
	ch <- mv
}

func (metrics *vmiMetrics) updateMemory(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric, k8sLabels []string, k8sLabelValues []string) {
	if vmStats.Memory.RSSSet {
		// Initial label set for a given metric
		var memoryResidentLabels = []string{"node", "namespace", "name"}
		// Kubernetes labels added afterwards
		memoryResidentLabels = append(memoryResidentLabels, k8sLabels...)
		metrics.memoryResidentDesc = prometheus.NewDesc(
			"kubevirt_vmi_memory_resident_bytes",
			"resident set size of the process running the domain.",
			memoryResidentLabels,
			nil,
		)

		var memoryResidentLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name}
		memoryResidentLabelValues = append(memoryResidentLabelValues, k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			metrics.memoryResidentDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.RSS)*1024,
			memoryResidentLabelValues...,
		)
		tryToPushMetric(metrics.memoryResidentDesc, mv, err, ch)
	}

	if vmStats.Memory.AvailableSet {
		var memoryAvailableLabels = []string{"node", "namespace", "name"}
		memoryAvailableLabels = append(memoryAvailableLabels, k8sLabels...)
		metrics.memoryAvailableDesc = prometheus.NewDesc(
			"kubevirt_vmi_memory_available_bytes",
			"amount of usable memory as seen by the domain.",
			memoryAvailableLabels,
			nil,
		)

		var memoryAvailableLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name}
		memoryAvailableLabelValues = append(memoryAvailableLabelValues, k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			metrics.memoryAvailableDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.Available)*1024,
			memoryAvailableLabelValues...,
		)
		tryToPushMetric(metrics.memoryAvailableDesc, mv, err, ch)
	}

	if vmStats.Memory.SwapInSet || vmStats.Memory.SwapOutSet {
		var swapTrafficLabels = []string{"node", "namespace", "name", "type"}
		swapTrafficLabels = append(swapTrafficLabels, k8sLabels...)
		metrics.swapTrafficDesc = prometheus.NewDesc(
			"kubevirt_vmi_memory_swap_traffic_bytes_total",
			"swap memory traffic.",
			swapTrafficLabels,
			nil,
		)

		if vmStats.Memory.SwapInSet {
			var swapTrafficInLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, "in"}
			swapTrafficInLabelValues = append(swapTrafficInLabelValues, k8sLabelValues...)

			mv, err := prometheus.NewConstMetric(
				metrics.swapTrafficDesc, prometheus.GaugeValue,
				// the libvirt value is in KiB
				float64(vmStats.Memory.SwapIn)*1024,
				swapTrafficInLabelValues...,
			)
			tryToPushMetric(metrics.swapTrafficDesc, mv, err, ch)
		}
		if vmStats.Memory.SwapOutSet {
			var swapTrafficOutLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, "out"}
			swapTrafficOutLabelValues = append(swapTrafficOutLabelValues, k8sLabelValues...)

			mv, err := prometheus.NewConstMetric(
				metrics.swapTrafficDesc, prometheus.GaugeValue,
				// the libvirt value is in KiB
				float64(vmStats.Memory.SwapOut)*1024,
				swapTrafficOutLabelValues...,
			)
			tryToPushMetric(metrics.swapTrafficDesc, mv, err, ch)
		}
	}
}

func (metrics *vmiMetrics) updateVcpu(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric, k8sLabels []string, k8sLabelValues []string) {
	for vcpuId, vcpu := range vmStats.Vcpu {
		// Initial vcpu metrics labels
		if !vcpu.StateSet || !vcpu.TimeSet {
			log.Log.V(4).Warningf("State or time not set for vcpu#%d", vcpuId)
			continue
		}

		var vcpuUsageLabels = []string{"node", "namespace", "name", "id", "state"}
		vcpuUsageLabels = append(vcpuUsageLabels, k8sLabels...)
		metrics.vcpuUsageDesc = prometheus.NewDesc(
			"kubevirt_vmi_vcpu_seconds",
			"Vcpu elapsed time.",
			vcpuUsageLabels,
			nil,
		)

		var vcpuUsageLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, fmt.Sprintf("%v", vcpuId), fmt.Sprintf("%v", vcpu.State)}
		vcpuUsageLabelValues = append(vcpuUsageLabelValues, k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			metrics.vcpuUsageDesc, prometheus.GaugeValue,
			float64(vcpu.Time/1000000000),
			vcpuUsageLabelValues...,
		)
		tryToPushMetric(metrics.vcpuUsageDesc, mv, err, ch)
	}
}

func (metrics *vmiMetrics) updateBlock(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric, k8sLabels []string, k8sLabelValues []string) {
	for blockId, block := range vmStats.Block {
		if !block.NameSet {
			log.Log.V(4).Warningf("Name not set for block device#%d", blockId)
			continue
		}

		if block.RdReqsSet || block.WrReqsSet {
			// Initial label set for a given metric
			var storageIopsLabels = []string{"node", "namespace", "name", "drive", "type"}
			// Kubernetes labels added afterwards
			storageIopsLabels = append(storageIopsLabels, k8sLabels...)
			metrics.storageIopsDesc = prometheus.NewDesc(
				"kubevirt_vmi_storage_iops_total",
				"I/O operation performed.",
				storageIopsLabels,
				nil,
			)

			if block.RdReqsSet {
				var storageIopsReadLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, block.Name, "read"}
				storageIopsReadLabelValues = append(storageIopsReadLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.storageIopsDesc, prometheus.CounterValue,
					float64(block.RdReqs),
					storageIopsReadLabelValues...,
				)
				tryToPushMetric(metrics.storageIopsDesc, mv, err, ch)
			}
			if block.WrReqsSet {
				var storageIopsWriteLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, block.Name, "write"}
				storageIopsWriteLabelValues = append(storageIopsWriteLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.storageIopsDesc, prometheus.CounterValue,
					float64(block.WrReqs),
					storageIopsWriteLabelValues...,
				)
				tryToPushMetric(metrics.storageIopsDesc, mv, err, ch)
			}
		}

		if block.RdBytesSet || block.WrBytesSet {
			var storageTrafficLabels = []string{"node", "namespace", "name", "domain", "drive", "type"}
			storageTrafficLabels = append(storageTrafficLabels, k8sLabels...)
			metrics.storageTrafficDesc = prometheus.NewDesc(
				"kubevirt_vmi_storage_traffic_bytes_total",
				"storage traffic.",
				storageTrafficLabels,
				nil,
			)

			if block.RdBytesSet {
				var storageTrafficReadLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, block.Name, "read"}
				storageTrafficReadLabelValues = append(storageTrafficReadLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.storageTrafficDesc, prometheus.CounterValue,
					float64(block.RdBytes),
					storageTrafficReadLabelValues...,
				)
				tryToPushMetric(metrics.storageTrafficDesc, mv, err, ch)
			}
			if block.WrBytesSet {
				var storageTrafficWriteLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, block.Name, "write"}
				storageTrafficWriteLabelValues = append(storageTrafficWriteLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.storageTrafficDesc, prometheus.CounterValue,
					float64(block.WrBytes),
					storageTrafficWriteLabelValues...,
				)
				tryToPushMetric(metrics.storageTrafficDesc, mv, err, ch)
			}
		}

		if block.RdTimesSet || block.WrTimesSet {
			var storageTimesLabels = []string{"node", "namespace", "name", "drive", "type"}
			storageTimesLabels = append(storageTimesLabels, k8sLabels...)
			metrics.storageTimesDesc = prometheus.NewDesc(
				"kubevirt_vmi_storage_times_ms_total",
				"storage operation time.",
				storageTimesLabels,
				nil,
			)

			if block.RdTimesSet {
				var storageTimesReadLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, block.Name, "read"}
				storageTimesReadLabelValues = append(storageTimesReadLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.storageTimesDesc, prometheus.CounterValue,
					float64(block.RdTimes),
					storageTimesReadLabelValues...,
				)
				tryToPushMetric(metrics.storageTimesDesc, mv, err, ch)
			}
			if block.WrTimesSet {
				var storageTimesWriteLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, block.Name, "write"}
				storageTimesWriteLabelValues = append(storageTimesWriteLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.storageTimesDesc, prometheus.CounterValue,
					float64(block.WrTimes),
					storageTimesWriteLabelValues...,
				)
				tryToPushMetric(metrics.storageTimesDesc, mv, err, ch)
			}
		}
	}
}

func (metrics *vmiMetrics) updateNetwork(vmi *k6tv1.VirtualMachineInstance, vmStats *stats.DomainStats, ch chan<- prometheus.Metric, k8sLabels []string, k8sLabelValues []string) {
	for _, net := range vmStats.Net {
		if !net.NameSet {
			continue
		}

		if net.RxBytesSet || net.TxBytesSet {
			// Initial label set for a given metric
			var networkTrafficBytesLabels = []string{"node", "namespace", "name", "interface", "type"}
			// Kubernetes labels added afterwards
			networkTrafficBytesLabels = append(networkTrafficBytesLabels, k8sLabels...)
			metrics.networkTrafficBytesDesc = prometheus.NewDesc(
				"kubevirt_vmi_network_traffic_bytes_total",
				"network traffic.",
				networkTrafficBytesLabels,
				nil,
			)

			if net.RxBytesSet {
				var networkTrafficBytesRxLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, net.Name, "rx"}
				networkTrafficBytesRxLabelValues = append(networkTrafficBytesRxLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.networkTrafficBytesDesc, prometheus.CounterValue,
					float64(net.RxBytes),
					networkTrafficBytesRxLabelValues...,
				)
				tryToPushMetric(metrics.networkTrafficBytesDesc, mv, err, ch)
			}
			if net.TxBytesSet {
				var networkTrafficBytesTxLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, net.Name, "tx"}
				networkTrafficBytesTxLabelValues = append(networkTrafficBytesTxLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.networkTrafficBytesDesc, prometheus.CounterValue,
					float64(net.TxBytes),
					networkTrafficBytesTxLabelValues...,
				)
				tryToPushMetric(metrics.networkTrafficBytesDesc, mv, err, ch)
			}
		}

		if net.RxPktsSet || net.TxPktsSet {
			var networkTrafficPktsLabels = []string{"node", "namespace", "name", "interface", "type"}
			networkTrafficPktsLabels = append(networkTrafficPktsLabels, k8sLabels...)
			metrics.networkTrafficPktsDesc = prometheus.NewDesc(
				"kubevirt_vmi_network_traffic_packets_total",
				"network traffic.",
				networkTrafficPktsLabels,
				nil,
			)

			if net.RxPktsSet {
				var networkTrafficPktsRxLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, net.Name, "rx"}
				networkTrafficPktsRxLabelValues = append(networkTrafficPktsRxLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.networkTrafficPktsDesc, prometheus.CounterValue,
					float64(net.RxPkts),
					networkTrafficPktsRxLabelValues...,
				)
				tryToPushMetric(metrics.networkTrafficPktsDesc, mv, err, ch)
			}
			if net.TxPktsSet {
				var networkTrafficPktsTxLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, net.Name, "tx"}
				networkTrafficPktsTxLabelValues = append(networkTrafficPktsTxLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.networkTrafficPktsDesc, prometheus.CounterValue,
					float64(net.TxPkts),
					networkTrafficPktsTxLabelValues...,
				)
				tryToPushMetric(metrics.networkTrafficPktsDesc, mv, err, ch)
			}
		}

		if net.RxErrsSet || net.TxErrsSet {
			var networkErrorsLabels = []string{"node", "namespace", "name", "interface", "type"}
			networkErrorsLabels = append(networkErrorsLabels, k8sLabels...)
			metrics.networkErrorsDesc = prometheus.NewDesc(
				"kubevirt_vmi_network_errors_total",
				"network errors.",
				networkErrorsLabels,
				nil,
			)

			if net.RxErrsSet {
				var networkErrorsRxLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, net.Name, "rx"}
				networkErrorsRxLabelValues = append(networkErrorsRxLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.networkErrorsDesc, prometheus.CounterValue,
					float64(net.RxErrs),
					networkErrorsRxLabelValues...,
				)
				tryToPushMetric(metrics.networkErrorsDesc, mv, err, ch)
			}
			if net.TxErrsSet {
				var networkErrorsTxLabelValues = []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, net.Name, "tx"}
				networkErrorsTxLabelValues = append(networkErrorsTxLabelValues, k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					metrics.networkErrorsDesc, prometheus.CounterValue,
					float64(net.TxErrs),
					networkErrorsTxLabelValues...,
				)
				tryToPushMetric(metrics.networkErrorsDesc, mv, err, ch)
			}
		}
	}
}

func makeVMIsPhasesMap(vmis []*k6tv1.VirtualMachineInstance) map[string]uint64 {
	phasesMap := make(map[string]uint64)

	for _, vmi := range vmis {
		phasesMap[strings.ToLower(string(vmi.Status.Phase))] += 1
	}

	return phasesMap
}

func updateVMIsPhase(nodeName string, vmis []*k6tv1.VirtualMachineInstance, ch chan<- prometheus.Metric) {
	phasesMap := makeVMIsPhasesMap(vmis)

	for phase, count := range phasesMap {
		mv, err := prometheus.NewConstMetric(
			vmiCountDesc, prometheus.GaugeValue,
			float64(count),
			nodeName, phase,
		)
		if err != nil {
			continue
		}
		ch <- mv
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

type vmiMetrics struct {
	storageIopsDesc         *prometheus.Desc
	storageTrafficDesc      *prometheus.Desc
	storageTimesDesc        *prometheus.Desc
	vcpuUsageDesc           *prometheus.Desc
	networkTrafficBytesDesc *prometheus.Desc
	networkTrafficPktsDesc  *prometheus.Desc
	networkErrorsDesc       *prometheus.Desc
	memoryAvailableDesc     *prometheus.Desc
	memoryResidentDesc      *prometheus.Desc
	swapTrafficDesc         *prometheus.Desc
}

func newVmiMetrics() *vmiMetrics {
	return &vmiMetrics{}
}

type Collector struct {
	virtCli       kubecli.KubevirtClient
	virtShareDir  string
	nodeName      string
	concCollector *concurrentCollector
}

func SetupCollector(virtCli kubecli.KubevirtClient, virtShareDir, nodeName string, MaxRequestsInFlight int) *Collector {
	log.Log.Infof("Starting collector: node name=%v", nodeName)
	co := &Collector{
		virtCli:       virtCli,
		virtShareDir:  virtShareDir,
		nodeName:      nodeName,
		concCollector: NewConcurrentCollector(MaxRequestsInFlight),
	}
	prometheus.MustRegister(co)
	return co
}

func (co *Collector) Describe(ch chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

func newvmiSocketMapFromVMIs(baseDir string, vmis []*k6tv1.VirtualMachineInstance) vmiSocketMap {
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

// Note that Collect could be called concurrently
func (co *Collector) Collect(ch chan<- prometheus.Metric) {
	updateVersion(ch)

	vmis, err := lookup.VirtualMachinesOnNode(co.virtCli, co.nodeName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to list all VMIs in '%s': %s", co.nodeName, err)
		return
	}

	if len(vmis) == 0 {
		log.Log.V(4).Infof("No VMIs detected")
		return
	}

	socketToVMIs := newvmiSocketMapFromVMIs(co.virtShareDir, vmis)
	scraper := &prometheusScraper{ch: ch}
	co.concCollector.Collect(socketToVMIs, scraper, collectionTimeout)

	updateVMIsPhase(co.nodeName, vmis, ch)
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

	vmiMetrics := newVmiMetrics()
	k8sLabels, k8sLabelValues := updateKubernetesLabels(vmi)

	vmiMetrics.updateMemory(vmi, vmStats, ps.ch, k8sLabels, k8sLabelValues)
	vmiMetrics.updateVcpu(vmi, vmStats, ps.ch, k8sLabels, k8sLabelValues)
	vmiMetrics.updateBlock(vmi, vmStats, ps.ch, k8sLabels, k8sLabelValues)
	vmiMetrics.updateNetwork(vmi, vmStats, ps.ch, k8sLabels, k8sLabelValues)
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

func updateKubernetesLabels(vmi *k6tv1.VirtualMachineInstance) (k8sLabels []string, k8sLabelValues []string) {
	for label, val := range vmi.Labels {
		k8sLabels = append(k8sLabels, labelPrefix+labelFormatter.Replace(label))
		k8sLabelValues = append(k8sLabelValues, val)
	}

	return k8sLabels, k8sLabelValues
}
