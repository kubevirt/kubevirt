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

	"k8s.io/client-go/tools/cache"

	vms "kubevirt.io/kubevirt/pkg/monitoring/domainstats"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/version"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

const (
	PrometheusCollectionTimeout            = vms.CollectionTimeout
	MigrateVmiDataRemainingMetricName      = "kubevirt_vmi_migration_data_remaining_bytes"
	MigrateVmiDataProcessedMetricName      = "kubevirt_vmi_migration_data_processed_bytes"
	MigrateVmiDirtyMemoryRateMetricName    = "kubevirt_vmi_migration_dirty_memory_rate_bytes"
	MigrateVmiMemoryTransferRateMetricName = "kubevirt_vmi_migration_disk_transfer_rate_bytes"
	MigrateVmiDiskTransferRateMetricName   = "kubevirt_vmi_migration_memory_transfer_rate_bytes"
)

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
)

func tryToPushMetric(desc *prometheus.Desc, mv prometheus.Metric, err error, ch chan<- prometheus.Metric) {
	if err != nil {
		log.Log.Warningf("Error creating the new const metric for %s: %s", desc, err)
		return
	}
	ch <- mv
}

func (metrics *vmiMetrics) updateMigrateInfo(jobInfo *stats.DomainJobInfo) {
	if jobInfo.DataRemainingSet {
		metrics.pushCommonMetric(
			MigrateVmiDataRemainingMetricName,
			"Number of bytes that still need to be transferred",
			prometheus.GaugeValue,
			float64(jobInfo.DataRemaining),
		)
	}

	if jobInfo.DataProcessedSet {
		metrics.pushCommonMetric(
			MigrateVmiDataProcessedMetricName,
			"Number of bytes transferred from the beginning of the job.",
			prometheus.GaugeValue,
			float64(jobInfo.DataProcessed),
		)
	}

	if jobInfo.MemDirtyRateSet {
		metrics.pushCommonMetric(
			MigrateVmiDirtyMemoryRateMetricName,
			"Number of memory pages dirtied by the guest per second.",
			prometheus.GaugeValue,
			float64(jobInfo.MemDirtyRate),
		)
	}

	if jobInfo.MemoryBpsSet {
		metrics.pushCommonMetric(
			MigrateVmiMemoryTransferRateMetricName,
			"Network throughput used while migrating memory in Bytes per second.",
			prometheus.GaugeValue,
			float64(jobInfo.MemoryBps),
		)
	}

	if jobInfo.DiskBpsSet {
		metrics.pushCommonMetric(
			MigrateVmiDiskTransferRateMetricName,
			"Network throughput used while migrating disks in Bytes per second.",
			prometheus.GaugeValue,
			float64(jobInfo.DiskBps),
		)
	}
}

func (metrics *vmiMetrics) updateMemory(mem *stats.DomainStatsMemory) {
	if mem.RSSSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_resident_bytes",
			"Resident set size of the process running the domain.",
			prometheus.GaugeValue,
			float64(mem.RSS)*1024,
		)
	}

	if mem.AvailableSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_available_bytes",
			"amount of usable memory as seen by the domain. This value may not be accurate if a balloon driver is in use or if the guest OS does not initialize all assigned pages",
			prometheus.GaugeValue,
			float64(mem.Available)*1024,
		)
	}

	if mem.UnusedSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_unused_bytes",
			"The amount of memory left completely unused by the system. Memory that is available but used for reclaimable caches should NOT be reported as free.",
			prometheus.GaugeValue,
			float64(mem.Unused)*1024,
		)
	}

	if mem.CachedSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_cached_bytes",
			"The amount of memory that is being used to cache I/O and is available to be reclaimed, corresponds to the sum of `Buffers` + `Cached` + `SwapCached` in `/proc/meminfo`.",
			prometheus.GaugeValue,
			float64(mem.Cached)*1024,
		)
	}

	if mem.SwapInSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_swap_in_traffic_bytes",
			"The total amount of data read from swap space of the guest in bytes.",
			prometheus.GaugeValue,
			float64(mem.SwapIn)*1024,
		)
	}

	if mem.SwapOutSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_swap_out_traffic_bytes",
			"The total amount of memory written out to swap space of the guest in bytes.",
			prometheus.GaugeValue,
			float64(mem.SwapOut)*1024,
		)
	}

	if mem.MajorFaultSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_pgmajfault_total",
			"The number of page faults when disk IO was required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is required, it is considered as major fault.",
			prometheus.CounterValue,
			float64(mem.MajorFault),
		)
	}

	if mem.MinorFaultSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_pgminfault_total",
			"The number of other page faults, when disk IO was not required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is NOT required, it is considered as minor fault.",
			prometheus.CounterValue,
			float64(mem.MinorFault),
		)
	}

	if mem.ActualBalloonSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_actual_balloon_bytes",
			"Current balloon size in bytes.",
			prometheus.GaugeValue,
			float64(mem.ActualBalloon)*1024,
		)
	}

	if mem.UsableSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_usable_bytes",
			"The amount of memory which can be reclaimed by balloon without pushing the guest system to swap, corresponds to 'Available' in /proc/meminfo",
			prometheus.GaugeValue,
			float64(mem.Usable)*1024,
		)
	}

	if mem.TotalSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_memory_domain_bytes",
			"The amount of memory in bytes allocated to the domain. The `memory` value in domain xml file.",
			prometheus.GaugeValue,
			float64(mem.Total)*1024,
		)
	}
}

func (metrics *vmiMetrics) updateCPUAffinity(cpuMap [][]bool) {
	affinityCount := 0.0

	for vidx := 0; vidx < len(cpuMap); vidx++ {
		for cidx := 0; cidx < len(cpuMap[vidx]); cidx++ {
			if cpuMap[vidx][cidx] {
				affinityCount++
			}
		}
	}

	metrics.pushCustomMetric(
		"kubevirt_vmi_node_cpu_affinity",
		"Number of VMI CPU affinities to node physical cores.",
		prometheus.GaugeValue, affinityCount,
		nil, nil,
	)
}

func nanosecondsToSeconds(ns uint64) float64 {
	return float64(ns) / 1000000000
}

func (metrics *vmiMetrics) updateCPU(vmi *k6tv1.VirtualMachineInstance, domainCPUStats *stats.DomainStatsCPU) {
	if !domainCPUStats.TimeSet && !domainCPUStats.UserSet && !domainCPUStats.SystemSet {
		log.Log.Warningf("No domain CPU stats is set for %s VMI.", vmi.Name)
	}

	if domainCPUStats.TimeSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_cpu_usage_seconds_total",
			"Total CPU time spent in all modes (sum of both vcpu and hypervisor usage).",
			prometheus.CounterValue,
			nanosecondsToSeconds(domainCPUStats.Time),
		)
	}

	if domainCPUStats.UserSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_cpu_user_usage_seconds_total",
			"Total CPU time spent in user mode.",
			prometheus.CounterValue,
			nanosecondsToSeconds(domainCPUStats.User),
		)
	}

	if domainCPUStats.SystemSet {
		metrics.pushCommonMetric(
			"kubevirt_vmi_cpu_system_usage_seconds_total",
			"Total CPU time spent in system mode.",
			prometheus.CounterValue,
			nanosecondsToSeconds(domainCPUStats.System),
		)
	}
}

func (metrics *vmiMetrics) updateVcpu(vcpuStats []stats.DomainStatsVcpu) {
	for vcpuIdx, vcpu := range vcpuStats {
		stringVcpuIdx := fmt.Sprintf("%d", vcpuIdx)

		if vcpu.StateSet && vcpu.TimeSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_vcpu_seconds_total",
				"Total amount of time spent in each state by each vcpu (cpu_time excluding hypervisor time). Where `id` is the vcpu identifier and `state` can be one of the following: [`OFFLINE`, `RUNNING`, `BLOCKED`].",
				prometheus.CounterValue,
				float64(vcpu.Time/1000000000),
				[]string{"id", "state"},
				[]string{stringVcpuIdx, humanReadableState(vcpu.State)},
			)
		}

		if vcpu.WaitSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_vcpu_wait_seconds_total",
				"Amount of time spent by each vcpu while waiting on I/O.",
				prometheus.CounterValue,
				float64(vcpu.Wait)/float64(1000000000),
				[]string{"id"},
				[]string{stringVcpuIdx},
			)
		}
		if vcpu.DelaySet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_vcpu_delay_seconds_total",
				"Amount of time spent by each vcpu waiting in the queue instead of running.",
				prometheus.CounterValue,
				float64(vcpu.Delay)/float64(1000000000),
				[]string{"id"},
				[]string{stringVcpuIdx},
			)
		}
	}
}

func (metrics *vmiMetrics) updateBlock(blkStats []stats.DomainStatsBlock) {
	for blockIdx, block := range blkStats {
		if !block.NameSet {
			log.Log.Warningf("Name not set for block device#%d", blockIdx)
			continue
		}

		blkLabels := []string{"drive"}
		blkLabelValues := []string{block.Name}

		if block.Alias != "" {
			blkLabelValues[0] = block.Alias
		}

		if block.RdReqsSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_iops_read_total",
				"Total number of I/O read operations.",
				prometheus.CounterValue,
				float64(block.RdReqs),
				blkLabels,
				blkLabelValues,
			)
		}

		if block.WrReqsSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_iops_write_total",
				"Total number of I/O write operations.",
				prometheus.CounterValue,
				float64(block.WrReqs),
				blkLabels,
				blkLabelValues,
			)
		}

		if block.RdBytesSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_read_traffic_bytes_total",
				"Total number of bytes read from storage.",
				prometheus.CounterValue,
				float64(block.RdBytes),
				blkLabels,
				blkLabelValues,
			)
		}

		if block.WrBytesSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_write_traffic_bytes_total",
				"Total number of written bytes.",
				prometheus.CounterValue,
				float64(block.WrBytes),
				blkLabels,
				blkLabelValues,
			)
		}

		if block.RdTimesSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_read_times_seconds_total",
				"Total time spent on read operations.",
				prometheus.CounterValue,
				float64(block.RdTimes)/1000000000,
				blkLabels,
				blkLabelValues,
			)
		}

		if block.WrTimesSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_write_times_seconds_total",
				"Total time spent on write operations.",
				prometheus.CounterValue,
				float64(block.WrTimes)/1000000000,
				blkLabels,
				blkLabelValues,
			)
		}

		if block.FlReqsSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_flush_requests_total",
				"Total storage flush requests.",
				prometheus.CounterValue,
				float64(block.FlReqs),
				blkLabels,
				blkLabelValues,
			)
		}

		if block.FlTimesSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_storage_flush_times_seconds_total",
				"Total time spent on cache flushing.",
				prometheus.CounterValue,
				float64(block.FlTimes)/1000000000,
				blkLabels,
				blkLabelValues,
			)
		}
	}
}

func (metrics *vmiMetrics) updateNetwork(netStats []stats.DomainStatsNet) {
	for _, net := range netStats {
		if !net.NameSet {
			continue
		}

		ifaceLabel := net.Name
		if net.AliasSet {
			ifaceLabel = net.Alias
		}

		netLabels := []string{"interface"}
		netLabelValues := []string{ifaceLabel}

		if net.RxBytesSet || net.TxBytesSet {
			desc := metrics.newPrometheusDesc(
				"kubevirt_vmi_network_traffic_bytes_total",
				"deprecated.",
				[]string{"interface", "type"},
			)

			if net.RxBytesSet {
				metrics.pushPrometheusMetric(desc, prometheus.CounterValue, float64(net.RxBytes), []string{net.Name, "rx"})
				metrics.pushCustomMetric(
					"kubevirt_vmi_network_receive_bytes_total",
					"Total network traffic received in bytes.",
					prometheus.CounterValue,
					float64(net.RxBytes),
					netLabels,
					netLabelValues,
				)
			}

			if net.TxBytesSet {
				metrics.pushPrometheusMetric(desc, prometheus.CounterValue, float64(net.TxBytes), []string{net.Name, "tx"})
				metrics.pushCustomMetric(
					"kubevirt_vmi_network_transmit_bytes_total",
					"Total network traffic transmitted in bytes.",
					prometheus.CounterValue,
					float64(net.TxBytes),
					netLabels,
					netLabelValues,
				)
			}
		}

		if net.RxPktsSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_network_receive_packets_total",
				"Total network traffic received packets.",
				prometheus.CounterValue,
				float64(net.RxPkts),
				netLabels,
				netLabelValues,
			)
		}

		if net.TxPktsSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_network_transmit_packets_total",
				"Total network traffic transmitted packets.",
				prometheus.CounterValue,
				float64(net.TxPkts),
				netLabels,
				netLabelValues,
			)
		}

		if net.RxErrsSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_network_receive_errors_total",
				"Total network received error packets.",
				prometheus.CounterValue,
				float64(net.RxErrs),
				netLabels,
				netLabelValues,
			)
		}

		if net.TxErrsSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_network_transmit_errors_total",
				"Total network transmitted error packets.",
				prometheus.CounterValue,
				float64(net.TxErrs),
				netLabels,
				netLabelValues,
			)
		}

		if net.RxDropSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_network_receive_packets_dropped_total",
				"The total number of rx packets dropped on vNIC interfaces.",
				prometheus.CounterValue,
				float64(net.RxDrop),
				netLabels,
				netLabelValues,
			)
		}

		if net.TxDropSet {
			metrics.pushCustomMetric(
				"kubevirt_vmi_network_transmit_packets_dropped_total",
				"The total number of tx packets dropped on vNIC interfaces.",
				prometheus.CounterValue,
				float64(net.TxDrop),
				netLabels,
				netLabelValues,
			)
		}
	}
}

func (metrics *vmiMetrics) updateFilesystem(vmFSStats k6tv1.VirtualMachineInstanceFileSystemList) {
	if len(vmFSStats.Items) == 0 {
		return
	}

	fsLabels := []string{"disk_name", "mount_point", "file_system_type"}

	for _, fsStat := range vmFSStats.Items {
		fsLabelValues := []string{fsStat.DiskName, fsStat.MountPoint, fsStat.FileSystemType}

		metrics.pushCustomMetric(
			"kubevirt_vmi_filesystem_capacity_bytes",
			"Total VM filesystem capacity in bytes.",
			prometheus.GaugeValue,
			float64(fsStat.TotalBytes),
			fsLabels,
			fsLabelValues,
		)

		metrics.pushCustomMetric(
			"kubevirt_vmi_filesystem_used_bytes",
			"Used VM filesystem capacity in bytes.",
			prometheus.GaugeValue,
			float64(fsStat.UsedBytes),
			fsLabels,
			fsLabelValues,
		)
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

type DomainStatsCollector struct {
	virtShareDir  string
	nodeName      string
	concCollector *vms.ConcurrentCollector
	vmiInformer   cache.SharedIndexInformer
}

// aggregates to virt-launcher
func SetupDomainStatsCollector(virtCli kubecli.KubevirtClient, virtShareDir, nodeName string, MaxRequestsInFlight int, vmiInformer cache.SharedIndexInformer) *DomainStatsCollector {
	log.Log.Infof("Starting domain stats collector: node name=%v", nodeName)
	co := &DomainStatsCollector{
		virtShareDir:  virtShareDir,
		nodeName:      nodeName,
		concCollector: vms.NewConcurrentCollector(MaxRequestsInFlight),
		vmiInformer:   vmiInformer,
	}

	prometheus.MustRegister(co)
	return co
}

func (co *DomainStatsCollector) Describe(_ chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

// Note that Collect could be called concurrently
func (co *DomainStatsCollector) Collect(ch chan<- prometheus.Metric) {
	updateVersion(ch)

	cachedObjs := co.vmiInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		log.Log.V(4).Infof("No VMIs detected")
		return
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	scraper := &prometheusScraper{ch: ch}
	co.concCollector.Collect(vmis, scraper, PrometheusCollectionTimeout)
	return
}

func NewPrometheusScraper(ch chan<- prometheus.Metric) *prometheusScraper {
	return &prometheusScraper{ch: ch}
}

type prometheusScraper struct {
	ch chan<- prometheus.Metric
}

type VirtualMachineInstanceStats struct {
	DomainStats *stats.DomainStats
	FsStats     k6tv1.VirtualMachineInstanceFileSystemList
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

	vmStats := &VirtualMachineInstanceStats{}
	var exists bool

	vmStats.DomainStats, exists, err = cli.GetDomainStats()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to update domain stats from socket %s", socketFile)
		return
	}
	if !exists || vmStats.DomainStats.Name == "" {
		log.Log.V(2).Infof("disappearing VM on %s, ignored", socketFile) // VM may be shutting down
		return
	}

	vmStats.FsStats, err = cli.GetFilesystems()
	if err != nil {
		log.Log.Reason(err).Errorf("failed to update filesystem stats from socket %s", socketFile)
		return
	}

	// GetDomainStats() may hang for a long time.
	// If it wakes up past the timeout, there is no point in send back any metric.
	// In the best case the information is stale, in the worst case the information is stale *and*
	// the reporting channel is already closed, leading to a possible panic - see below
	elapsed := time.Now().Sub(ts)
	if elapsed > vms.StatsMaxAge {
		log.Log.Infof("took too long (%v) to collect stats from %s: ignored", elapsed, socketFile)
		return
	}

	ps.Report(socketFile, vmi, vmStats)
}

func (ps *prometheusScraper) Report(socketFile string, vmi *k6tv1.VirtualMachineInstance, vmStats *VirtualMachineInstanceStats) {
	// statsMaxAge is an estimation - and there is no better way to do that. So it is possible that
	// GetDomainStats() takes enough time to lag behind, but not enough to trigger the statsMaxAge check.
	// In this case the next functions will end up writing on a closed channel. This will panic.
	// It is actually OK in this case to abort the goroutine that panicked -that's what we want anyway,
	// and the very reason we collect in throwaway goroutines. We need however to avoid dump stacktraces in the logs.
	// Since this is a known failure condition, let's handle it explicitly.
	defer func() {
		if err := recover(); err != nil {
			log.Log.Warningf("collector goroutine panicked for VM %s: %s", socketFile, err)
		}
	}()

	vmiMetrics := newVmiMetrics(vmi, ps.ch)
	vmiMetrics.updateMetrics(vmi, vmStats)
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

func (metrics *vmiMetrics) updateMetrics(vmi *k6tv1.VirtualMachineInstance, vmStats *VirtualMachineInstanceStats) {
	metrics.updateKubernetesLabels()

	metrics.updateMemory(vmStats.DomainStats.Memory)
	metrics.updateCPU(vmi, vmStats.DomainStats.Cpu)
	metrics.updateVcpu(vmStats.DomainStats.Vcpu)
	metrics.updateBlock(vmStats.DomainStats.Block)
	metrics.updateNetwork(vmStats.DomainStats.Net)

	if vmStats.DomainStats.CPUMapSet {
		metrics.updateCPUAffinity(vmStats.DomainStats.CPUMap)
	}
	metrics.updateMigrateInfo(vmStats.DomainStats.MigrateDomainJobInfo)
	metrics.updateFilesystem(vmStats.FsStats)
}

func (metrics *vmiMetrics) newPrometheusDesc(name string, help string, customLabels []string) *prometheus.Desc {
	labels := []string{"node", "namespace", "name"} // Common labels
	labels = append(labels, customLabels...)
	labels = append(labels, metrics.k8sLabels...)
	return prometheus.NewDesc(name, help, labels, nil)
}

func (metrics *vmiMetrics) pushPrometheusMetric(desc *prometheus.Desc, valueType prometheus.ValueType, value float64, customLabelValues []string) {
	labelValues := []string{metrics.vmi.Status.NodeName, metrics.vmi.Namespace, metrics.vmi.Name}
	labelValues = append(labelValues, customLabelValues...)
	labelValues = append(labelValues, metrics.k8sLabelValues...)
	mv, err := prometheus.NewConstMetric(desc, valueType, value, labelValues...)
	tryToPushMetric(desc, mv, err, metrics.ch)
}

func (metrics *vmiMetrics) pushCommonMetric(name string, help string, valueType prometheus.ValueType, value float64) {
	metrics.pushCustomMetric(name, help, valueType, value, nil, nil)
}

func (metrics *vmiMetrics) pushCustomMetric(name string, help string, valueType prometheus.ValueType, value float64, customLabels []string, customLabelValues []string) {
	desc := metrics.newPrometheusDesc(name, help, customLabels)
	metrics.pushPrometheusMetric(desc, valueType, value, customLabelValues)
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
	case stats.VCPUOffline:
		return "offline"
	case stats.VCPUBlocked:
		return "blocked"
	case stats.VCPURunning:
		return "running"
	default:
		return "unknown"
	}
}
