package downwardmetrics

import (
	"context"
	"time"

	"k8s.io/client-go/tools/cache"

	k6sv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd"
	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	metricspkg "kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/metrics"
	vms "kubevirt.io/kubevirt/pkg/monitoring/domainstats"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

const DownwardmetricsRefreshDuration = 5 * time.Second
const DownwardmetricsCollectionTimeout = vms.CollectionTimeout
const qemuVersionUnknown = "qemu-unknown"

type StaticHostMetrics struct {
	HostName             string
	HostSystemInfo       string
	VirtualizationVendor string
}

type Scraper struct {
	isolation            isolation.PodIsolationDetector
	staticHostInfo       *StaticHostMetrics
	hostMetricsCollector *hostMetricsCollector
}

func (s *Scraper) Scrape(socketFile string, vmi *k6sv1.VirtualMachineInstance) {
	if !vmi.IsRunning() || !downwardmetrics.HasDownwardMetricDisk(vmi) {
		return
	}

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

	version, err := cli.GetQemuVersion()
	if err != nil {
		if cmdclient.IsUnimplemented(err) {
			log.Log.Reason(err).Warning("getQemuVersion not implemented, consider to upgrade kubevirt")
			version = qemuVersionUnknown
		} else {
			log.Log.Reason(err).Errorf("failed to update qemu stats from socket %s", socketFile)
			return
		}
	}

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
	if elapsed > vms.StatsMaxAge {
		log.Log.Infof("took too long (%v) to collect stats from %s: ignored", elapsed, socketFile)
		return
	}
	s.Report(vmi, vmStats, version)
}
func (s *Scraper) Report(vmi *k6sv1.VirtualMachineInstance, vmStats *stats.DomainStats, qemuVersion string) {
	res, err := s.isolation.Detect(vmi)
	if err != nil {
		log.Log.Reason(err).Infof("failed to detect root directory of the vmi pod")
		return
	}
	metricsUpdater := vhostmd.NewMetricsIODisk(downwardmetrics.FormatDownwardMetricPath(res.Pid()))
	metrics := &api.Metrics{
		Metrics: []api.Metric{
			metricspkg.MustToUnitlessHostMetric(s.staticHostInfo.HostName, "HostName"),
			metricspkg.MustToUnitlessHostMetric(s.staticHostInfo.HostSystemInfo, "HostSystemInfo"),
			metricspkg.MustToUnitlessHostMetric(s.staticHostInfo.VirtualizationVendor, "VirtualizationVendor"),
			metricspkg.MustToUnitlessHostMetric(qemuVersion, "VirtProductInfo"),
		},
	}
	metrics.Metrics = append(metrics.Metrics, guestCPUMetrics(vmStats)...)
	metrics.Metrics = append(metrics.Metrics, guestMemoryMetrics(vmStats)...)
	metrics.Metrics = append(metrics.Metrics, s.hostMetricsCollector.Collect()...)
	err = metricsUpdater.Write(metrics)
	if err != nil {
		log.Log.Reason(err).Infof("failed to write metrics to disk")
		return
	}
}

func guestCPUMetrics(vmStats *stats.DomainStats) []api.Metric {
	var cpuTimeTotal uint64
	for _, vcpu := range vmStats.Vcpu {
		cpuTimeTotal += vcpu.Time
	}

	return []api.Metric{
		metricspkg.MustToVMMetric(float64(cpuTimeTotal)/float64(1000000000), "TotalCPUTime", "s"),
		metricspkg.MustToVMMetric(vmStats.NrVirtCpu, "ResourceProcessorLimit", ""),
	}
}

func guestMemoryMetrics(vmStats *stats.DomainStats) []api.Metric {

	return []api.Metric{
		metricspkg.MustToVMMetric(vmStats.Memory.ActualBalloon, "PhysicalMemoryAllocatedToVirtualSystem", "KiB"),
		// Since we don't do active ballooning, ActualBalloon is the same as the memory limit
		metricspkg.MustToVMMetric(vmStats.Memory.ActualBalloon, "ResourceMemoryLimit", "KiB"),
	}
}

type Collector struct {
	concCollector *vms.ConcurrentCollector
}

func RunDownwardMetricsCollector(context context.Context, nodeName string, vmiInformer cache.SharedIndexInformer, isolation isolation.PodIsolationDetector) error {

	scraper := &Scraper{
		isolation: isolation,
		staticHostInfo: &StaticHostMetrics{
			HostName:             nodeName,
			HostSystemInfo:       "linux",
			VirtualizationVendor: "kubevirt.io",
		},
		hostMetricsCollector: defaultHostMetricsCollector(),
	}
	collector := vms.NewConcurrentCollector(1)

	go func() {
		ticker := time.NewTicker(DownwardmetricsRefreshDuration)
		for {
			select {
			case <-ticker.C:
				cachedObjs := vmiInformer.GetIndexer().List()
				if len(cachedObjs) == 0 {
					log.Log.V(4).Infof("No VMIs detected")
					continue
				}

				vmis := []*k6sv1.VirtualMachineInstance{}

				for _, obj := range cachedObjs {
					vmis = append(vmis, obj.(*k6sv1.VirtualMachineInstance))
				}
				collector.Collect(vmis, scraper, DownwardmetricsCollectionTimeout)
			case <-context.Done():
				return
			}
		}
	}()
	return nil
}
