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

	libvirt "libvirt.org/libvirt-go"

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

func (metrics *vmiMetrics) updateMemory(vmStats *stats.DomainStats) {
	if vmStats.Memory.RSSSet {
		// Initial label set for a given metric
		memoryResidentLabels := []string{"node", "namespace", "name"}
		// Kubernetes labels added afterwards
		memoryResidentLabels = append(memoryResidentLabels, metrics.k8sLabels...)
		memoryResidentDesc := prometheus.NewDesc(
			"kubevirt_vmi_memory_resident_bytes",
			"resident set size of the process running the domain.",
			memoryResidentLabels,
			nil,
		)

		memoryResidentLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name}
		memoryResidentLabelValues = append(memoryResidentLabelValues, metrics.k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			memoryResidentDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.RSS)*1024,
			memoryResidentLabelValues...,
		)
		tryToPushMetric(memoryResidentDesc, mv, err, metrics.ch)
	}

	if vmStats.Memory.AvailableSet {
		memoryAvailableLabels := []string{"node", "namespace", "name"}
		memoryAvailableLabels = append(memoryAvailableLabels, metrics.k8sLabels...)
		memoryAvailableDesc := prometheus.NewDesc(
			"kubevirt_vmi_memory_available_bytes",
			"amount of usable memory as seen by the domain.",
			memoryAvailableLabels,
			nil,
		)

		memoryAvailableLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name}
		memoryAvailableLabelValues = append(memoryAvailableLabelValues, metrics.k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			memoryAvailableDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.Available)*1024,
			memoryAvailableLabelValues...,
		)
		tryToPushMetric(memoryAvailableDesc, mv, err, metrics.ch)
	}

	if vmStats.Memory.UnusedSet {
		memoryUnusedLabels := []string{"node", "namespace", "name"}
		memoryUnusedLabels = append(memoryUnusedLabels, metrics.k8sLabels...)
		memoryUnusedDesc := prometheus.NewDesc(
			"kubevirt_vmi_memory_unused_bytes",
			"amount of unused memory as seen by the domain.",
			memoryUnusedLabels,
			nil,
		)

		memoryUnusedLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name}
		memoryUnusedLabelValues = append(memoryUnusedLabelValues, metrics.k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			memoryUnusedDesc, prometheus.GaugeValue,
			// the libvirt value is in KiB
			float64(vmStats.Memory.Unused)*1024,
			memoryUnusedLabelValues...,
		)
		tryToPushMetric(memoryUnusedDesc, mv, err, metrics.ch)
	}

	if vmStats.Memory.SwapInSet || vmStats.Memory.SwapOutSet {
		swapTrafficLabels := []string{"node", "namespace", "name", "type"}
		swapTrafficLabels = append(swapTrafficLabels, metrics.k8sLabels...)
		swapTrafficDesc := prometheus.NewDesc(
			"kubevirt_vmi_memory_swap_traffic_bytes_total",
			"swap memory traffic.",
			swapTrafficLabels,
			nil,
		)

		if vmStats.Memory.SwapInSet {
			swapTrafficInLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, "in"}
			swapTrafficInLabelValues = append(swapTrafficInLabelValues, metrics.k8sLabelValues...)

			mv, err := prometheus.NewConstMetric(
				swapTrafficDesc, prometheus.GaugeValue,
				// the libvirt value is in KiB
				float64(vmStats.Memory.SwapIn)*1024,
				swapTrafficInLabelValues...,
			)
			tryToPushMetric(swapTrafficDesc, mv, err, metrics.ch)
		}
		if vmStats.Memory.SwapOutSet {
			swapTrafficOutLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, "out"}
			swapTrafficOutLabelValues = append(swapTrafficOutLabelValues, metrics.k8sLabelValues...)

			mv, err := prometheus.NewConstMetric(
				swapTrafficDesc, prometheus.GaugeValue,
				// the libvirt value is in KiB
				float64(vmStats.Memory.SwapOut)*1024,
				swapTrafficOutLabelValues...,
			)
			tryToPushMetric(swapTrafficDesc, mv, err, metrics.ch)
		}
	}
}

func (metrics *vmiMetrics) updateCPU(vmStats *stats.DomainStats) {
	if vmStats.Cpu.SystemSet {
		cpuSystemLabels := []string{"node", "namespace", "name"}
		cpuSystemLabels = append(cpuSystemLabels, metrics.k8sLabels...)
		cpuSystemDesc := prometheus.NewDesc(
			"kubevirt_vmi_cpu_system_seconds_total",
			"system cpu time spent in seconds.",
			cpuSystemLabels,
			nil,
		)

		cpuSystemLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name}
		cpuSystemLabelValues = append(cpuSystemLabelValues, metrics.k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			cpuSystemDesc, prometheus.GaugeValue,
			float64(vmStats.Cpu.System/1000000000),
			cpuSystemLabelValues...,
		)

		tryToPushMetric(cpuSystemDesc, mv, err, metrics.ch)
	}

	if vmStats.Cpu.UserSet {
		cpuUserLabels := []string{"node", "namespace", "name"}
		cpuUserLabels = append(cpuUserLabels, metrics.k8sLabels...)
		cpuUserDesc := prometheus.NewDesc(
			"kubevirt_vmi_cpu_user_seconds_total",
			"user cpu time spent in seconds.",
			cpuUserLabels,
			nil,
		)

		cpuUserLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name}
		cpuUserLabelValues = append(cpuUserLabelValues, metrics.k8sLabelValues...)
		mv, err := prometheus.NewConstMetric(
			cpuUserDesc, prometheus.GaugeValue,
			float64(vmStats.Cpu.User/1000000000),
			cpuUserLabelValues...,
		)

		tryToPushMetric(cpuUserDesc, mv, err, metrics.ch)
	}
}

func (metrics *vmiMetrics) updateVcpu(vmStats *stats.DomainStats) {
	for vcpuId, vcpu := range vmStats.Vcpu {
		// Initial vcpu metrics labels
		if !vcpu.StateSet || !vcpu.TimeSet {
			log.Log.V(4).Warningf("State or time not set for vcpu#%d", vcpuId)
		} else {
			vcpuUsageLabels := []string{"node", "namespace", "name", "id", "state"}
			vcpuUsageLabels = append(vcpuUsageLabels, metrics.k8sLabels...)
			vcpuUsageDesc := prometheus.NewDesc(
				"kubevirt_vmi_vcpu_seconds",
				"Vcpu elapsed time.",
				vcpuUsageLabels,
				nil,
			)

			vcpuUsageLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, fmt.Sprintf("%v", vcpuId), humanReadableState(vcpu.State)}
			vcpuUsageLabelValues = append(vcpuUsageLabelValues, metrics.k8sLabelValues...)
			mv, err := prometheus.NewConstMetric(
				vcpuUsageDesc, prometheus.GaugeValue,
				float64(vcpu.Time/1000000000),
				vcpuUsageLabelValues...,
			)
			tryToPushMetric(vcpuUsageDesc, mv, err, metrics.ch)
		}

		if !vcpu.WaitSet {
			log.Log.V(4).Warningf("Wait not set for vcpu#%d", vcpuId)
			continue
		}

		vcpuWaitLabels := []string{"node", "namespace", "name", "id"}
		vcpuWaitLabels = append(vcpuWaitLabels, metrics.k8sLabels...)

		vcpuWaitLabelsValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name,
			fmt.Sprintf("%v", vcpuId),
		}
		vcpuWaitLabelsValues = append(vcpuWaitLabelsValues, metrics.k8sLabelValues...)

		vcpuWaitDesc := prometheus.NewDesc(
			"kubevirt_vmi_vcpu_wait_seconds",
			"vcpu time spent by waiting on I/O",
			vcpuWaitLabels,
			nil,
		)

		mv, err := prometheus.NewConstMetric(
			vcpuWaitDesc, prometheus.GaugeValue,
			float64(vcpu.Wait/1000000),
			vcpuWaitLabelsValues...,
		)
		tryToPushMetric(vcpuWaitDesc, mv, err, metrics.ch)

	}
}

func (metrics *vmiMetrics) updateBlock(vmStats *stats.DomainStats) {
	for blockId, block := range vmStats.Block {
		if !block.NameSet {
			log.Log.V(4).Warningf("Name not set for block device#%d", blockId)
			continue
		}

		if block.RdReqsSet || block.WrReqsSet {
			// Initial label set for a given metric
			storageIopsLabels := []string{"node", "namespace", "name", "drive", "type"}
			// Kubernetes labels added afterwards
			storageIopsLabels = append(storageIopsLabels, metrics.k8sLabels...)
			storageIopsDesc := prometheus.NewDesc(
				"kubevirt_vmi_storage_iops_total",
				"I/O operation performed.",
				storageIopsLabels,
				nil,
			)

			if block.RdReqsSet {
				storageIopsReadLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, block.Name, "read"}
				storageIopsReadLabelValues = append(storageIopsReadLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					storageIopsDesc, prometheus.CounterValue,
					float64(block.RdReqs),
					storageIopsReadLabelValues...,
				)
				tryToPushMetric(storageIopsDesc, mv, err, metrics.ch)
			}
			if block.WrReqsSet {
				storageIopsWriteLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, block.Name, "write"}
				storageIopsWriteLabelValues = append(storageIopsWriteLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					storageIopsDesc, prometheus.CounterValue,
					float64(block.WrReqs),
					storageIopsWriteLabelValues...,
				)
				tryToPushMetric(storageIopsDesc, mv, err, metrics.ch)
			}
		}

		if block.RdBytesSet || block.WrBytesSet {
			storageTrafficLabels := []string{"node", "namespace", "name", "drive", "type"}
			storageTrafficLabels = append(storageTrafficLabels, metrics.k8sLabels...)
			storageTrafficDesc := prometheus.NewDesc(
				"kubevirt_vmi_storage_traffic_bytes_total",
				"storage traffic.",
				storageTrafficLabels,
				nil,
			)

			if block.RdBytesSet {
				storageTrafficReadLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, block.Name, "read"}
				storageTrafficReadLabelValues = append(storageTrafficReadLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					storageTrafficDesc, prometheus.CounterValue,
					float64(block.RdBytes),
					storageTrafficReadLabelValues...,
				)
				tryToPushMetric(storageTrafficDesc, mv, err, metrics.ch)
			}
			if block.WrBytesSet {
				storageTrafficWriteLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, block.Name, "write"}
				storageTrafficWriteLabelValues = append(storageTrafficWriteLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					storageTrafficDesc, prometheus.CounterValue,
					float64(block.WrBytes),
					storageTrafficWriteLabelValues...,
				)
				tryToPushMetric(storageTrafficDesc, mv, err, metrics.ch)
			}
		}

		if block.RdTimesSet || block.WrTimesSet {
			storageTimesLabels := []string{"node", "namespace", "name", "drive", "type"}
			storageTimesLabels = append(storageTimesLabels, metrics.k8sLabels...)
			storageTimesDesc := prometheus.NewDesc(
				"kubevirt_vmi_storage_times_ms_total",
				"storage operation time.",
				storageTimesLabels,
				nil,
			)

			if block.RdTimesSet {
				storageTimesReadLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, block.Name, "read"}
				storageTimesReadLabelValues = append(storageTimesReadLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					storageTimesDesc, prometheus.CounterValue,
					float64(block.RdTimes),
					storageTimesReadLabelValues...,
				)
				tryToPushMetric(storageTimesDesc, mv, err, metrics.ch)
			}
			if block.WrTimesSet {
				storageTimesWriteLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, block.Name, "write"}
				storageTimesWriteLabelValues = append(storageTimesWriteLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					storageTimesDesc, prometheus.CounterValue,
					float64(block.WrTimes),
					storageTimesWriteLabelValues...,
				)
				tryToPushMetric(storageTimesDesc, mv, err, metrics.ch)
			}
		}

		if block.FlReqsSet {
			storageFlushLabels := []string{"node", "namespace", "name", "drive", "type"}
			storageFlushLabels = append(storageFlushLabels, metrics.k8sLabels...)
			storageFlushDesc := prometheus.NewDesc(
				"kubevirt_vmi_storage_requests_total",
				"storage flush requests.",
				storageFlushLabels,
				nil,
			)

			storageFlushLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, block.Name, "flush"}
			storageFlushLabelValues = append(storageFlushLabelValues, metrics.k8sLabelValues...)

			mv, err := prometheus.NewConstMetric(
				storageFlushDesc, prometheus.CounterValue,
				float64(block.FlReqs),
				storageFlushLabelValues...,
			)
			tryToPushMetric(storageFlushDesc, mv, err, metrics.ch)
		}
	}
}

func (metrics *vmiMetrics) updateNetwork(vmStats *stats.DomainStats) {
	for _, net := range vmStats.Net {
		if !net.NameSet {
			continue
		}

		if net.RxBytesSet || net.TxBytesSet {
			// Initial label set for a given metric
			networkTrafficBytesLabels := []string{"node", "namespace", "name", "interface", "type"}
			// Kubernetes labels added afterwards
			networkTrafficBytesLabels = append(networkTrafficBytesLabels, metrics.k8sLabels...)
			networkTrafficBytesDesc := prometheus.NewDesc(
				"kubevirt_vmi_network_traffic_bytes_total",
				"network traffic.",
				networkTrafficBytesLabels,
				nil,
			)

			if net.RxBytesSet {
				networkTrafficBytesRxLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, net.Name, "rx"}
				networkTrafficBytesRxLabelValues = append(networkTrafficBytesRxLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					networkTrafficBytesDesc, prometheus.CounterValue,
					float64(net.RxBytes),
					networkTrafficBytesRxLabelValues...,
				)
				tryToPushMetric(networkTrafficBytesDesc, mv, err, metrics.ch)
			}
			if net.TxBytesSet {
				networkTrafficBytesTxLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, net.Name, "tx"}
				networkTrafficBytesTxLabelValues = append(networkTrafficBytesTxLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					networkTrafficBytesDesc, prometheus.CounterValue,
					float64(net.TxBytes),
					networkTrafficBytesTxLabelValues...,
				)
				tryToPushMetric(networkTrafficBytesDesc, mv, err, metrics.ch)
			}
		}

		if net.RxPktsSet || net.TxPktsSet {
			networkTrafficPktsLabels := []string{"node", "namespace", "name", "interface", "type"}
			networkTrafficPktsLabels = append(networkTrafficPktsLabels, metrics.k8sLabels...)
			networkTrafficPktsDesc := prometheus.NewDesc(
				"kubevirt_vmi_network_traffic_packets_total",
				"network traffic.",
				networkTrafficPktsLabels,
				nil,
			)

			if net.RxPktsSet {
				networkTrafficPktsRxLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, net.Name, "rx"}
				networkTrafficPktsRxLabelValues = append(networkTrafficPktsRxLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					networkTrafficPktsDesc, prometheus.CounterValue,
					float64(net.RxPkts),
					networkTrafficPktsRxLabelValues...,
				)
				tryToPushMetric(networkTrafficPktsDesc, mv, err, metrics.ch)
			}
			if net.TxPktsSet {
				networkTrafficPktsTxLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, net.Name, "tx"}
				networkTrafficPktsTxLabelValues = append(networkTrafficPktsTxLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					networkTrafficPktsDesc, prometheus.CounterValue,
					float64(net.TxPkts),
					networkTrafficPktsTxLabelValues...,
				)
				tryToPushMetric(networkTrafficPktsDesc, mv, err, metrics.ch)
			}
		}

		if net.RxErrsSet || net.TxErrsSet {
			networkErrorsLabels := []string{"node", "namespace", "name", "interface", "type"}
			networkErrorsLabels = append(networkErrorsLabels, metrics.k8sLabels...)
			networkErrorsDesc := prometheus.NewDesc(
				"kubevirt_vmi_network_errors_total",
				"network errors.",
				networkErrorsLabels,
				nil,
			)

			if net.RxErrsSet {
				networkErrorsRxLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, net.Name, "rx"}
				networkErrorsRxLabelValues = append(networkErrorsRxLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					networkErrorsDesc, prometheus.CounterValue,
					float64(net.RxErrs),
					networkErrorsRxLabelValues...,
				)
				tryToPushMetric(networkErrorsDesc, mv, err, metrics.ch)
			}
			if net.TxErrsSet {
				networkErrorsTxLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, net.Name, "tx"}
				networkErrorsTxLabelValues = append(networkErrorsTxLabelValues, metrics.k8sLabelValues...)

				mv, err := prometheus.NewConstMetric(
					networkErrorsDesc, prometheus.CounterValue,
					float64(net.TxErrs),
					networkErrorsTxLabelValues...,
				)
				tryToPushMetric(networkErrorsDesc, mv, err, metrics.ch)
			}
		}

		if net.RxDropSet {
			networkRxDropLabels := []string{"node", "namespace", "name", "interface", "type"}
			networkRxDropLabels = append(networkRxDropLabels, metrics.k8sLabels...)
			networkRxDropDesc := prometheus.NewDesc(
				"kubevirt_vmi_network_receive_packets_dropped_total",
				"network rx packet drops.",
				networkRxDropLabels,
				nil,
			)

			networkRxDropLabelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name, net.Name, "rxdrop"}
			networkRxDropLabelValues = append(networkRxDropLabelValues, metrics.k8sLabelValues...)

			mv, err := prometheus.NewConstMetric(
				networkRxDropDesc, prometheus.CounterValue,
				float64(net.RxDrop),
				networkRxDropLabelValues...,
			)
			tryToPushMetric(networkRxDropDesc, mv, err, metrics.ch)
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
	// Since this is a known failure condition, let's handle it explicitly.
	defer func() {
		if err := recover(); err != nil {
			log.Log.V(2).Warningf("collector goroutine panicked for VM %s: %s", socketFile, err)
		}
	}()

	vmiMetrics := newVmiMetrics(vmi, ps.ch)
	vmiMetrics.updateMetrics(vmStats)

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

type vmiMetrics struct {
	k8sLabels      []string
	k8sLabelValues []string
	vmi            *k6tv1.VirtualMachineInstance
	ch             chan<- prometheus.Metric
}

func (metrics *vmiMetrics) updateMetrics(vmStats *stats.DomainStats) {
	metrics.updateKubernetesLabels()

	metrics.updateMemory(vmStats)
	metrics.updateCPU(vmStats)
	metrics.updateVcpu(vmStats)
	metrics.updateBlock(vmStats)
	metrics.updateNetwork(vmStats)
}

func (metrics *vmiMetrics) updateKubernetesLabels() {
	for label, val := range metrics.vmi.Labels {
		metrics.k8sLabels = append(metrics.k8sLabels, labelPrefix+labelFormatter.Replace(label))
		metrics.k8sLabelValues = append(metrics.k8sLabelValues, val)
	}
}

func newVmiMetrics(vmi *k6tv1.VirtualMachineInstance, ch chan<- prometheus.Metric) *vmiMetrics {
	return &vmiMetrics{
		vmi:            vmi,
		k8sLabels:      []string{},
		k8sLabelValues: []string{},
		ch:             ch,
	}
}

func humanReadableState(state int) string {
	switch state {
	case int(libvirt.VCPU_OFFLINE):
		return "offline"
	case int(libvirt.VCPU_BLOCKED):
		return "blocked"
	case int(libvirt.VCPU_RUNNING):
		return "running"
	default:
		return "unknown"
	}
}
