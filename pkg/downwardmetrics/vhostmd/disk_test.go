package vhostmd

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	metricspkg "kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/metrics"
)

var _ = Describe("vhostmd", func() {

	Context("given real data from a real vhostmd", func() {

		It("should properly read and verify a real vhostmd0", func() {
			v := NewMetricsIODisk("testdata/vhostmd0")
			metrics, err := v.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(metrics.Metrics).To(HaveLen(14))
			Expect(metrics.Metrics[0].Name).To(Equal("TotalCPUTime"))
			Expect(metrics.Metrics[0].Value).To(Equal("1292869.190000"))
			Expect(metrics.Metrics[0].Context).To(Equal(api.MetricContextHost))
			Expect(metrics.Metrics[0].Type).To(Equal(api.MetricTypeReal64))
			Expect(metrics.Metrics[1].Name).To(Equal("PagedOutMemory"))
			Expect(metrics.Metrics[1].Value).To(Equal("34433"))
			Expect(metrics.Metrics[1].Context).To(Equal(api.MetricContextHost))
			Expect(metrics.Metrics[1].Type).To(Equal(api.MetricTypeUInt64))
			Expect(metrics.Metrics[6].Name).To(Equal("MemoryAllocatedToVirtualServers"))
			Expect(metrics.Metrics[6].Value).To(Equal("13377"))
			Expect(metrics.Metrics[6].Context).To(Equal(api.MetricContextHost))
			Expect(metrics.Metrics[6].Type).To(Equal(api.MetricTypeUInt64))
			Expect(metrics.Metrics[12].Name).To(Equal("HostName"))
			Expect(metrics.Metrics[12].Value).To(Equal("linux.fritz.box"))
			Expect(metrics.Metrics[12].Context).To(Equal(api.MetricContextHost))
			Expect(metrics.Metrics[12].Type).To(Equal(api.MetricTypeString))
			Expect(metrics.Metrics[13].Name).To(Equal("ResourceProcessorLimit"))
			Expect(metrics.Metrics[13].Value).To(Equal("2"))
			Expect(metrics.Metrics[13].Context).To(Equal(api.MetricContextVM))
			Expect(metrics.Metrics[13].Type).To(Equal(api.MetricTypeUInt64))
		})
	})

	Context("operating on selfcreated files", func() {
		var targetDir string
		var err error

		BeforeEach(func() {
			targetDir, err = os.MkdirTemp("", "vhostmd")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			_ = os.RemoveAll(targetDir)
		})

		It("should create a properly formatted empty vhostmd disk", func() {
			metricsIO := NewMetricsIODisk(filepath.Join(targetDir, "vhostmd0"))
			Expect(metricsIO.Create()).To(Succeed())
			metrics, err := metricsIO.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(metrics.Metrics).To(BeEmpty())
		})

		It("should be able to repreatedly read and write metrics without modifying the result", func() {
			metricsIO := NewMetricsIODisk(filepath.Join(targetDir, "vhostmd0"))
			Expect(metricsIO.Create()).To(Succeed())
			metrics := &api.Metrics{
				Metrics: []api.Metric{
					metricspkg.MustToMetric(1292869.190000, "TotalCPUTime", "s", api.MetricContextHost),
					metricspkg.MustToMetric(3443, "PagedOutMemory", "KiB", api.MetricContextHost),
					metricspkg.MustToMetric("linux.fritz.box", "HostName", "", api.MetricContextHost),
					metricspkg.MustToMetric(3, "TotalCPU", "", api.MetricContextVM),
					metricspkg.MustToVMMetric(2, "ResourceProcessorLimit", ""),
				},
			}
			for x := 0; x < 5; x++ {
				Expect(metricsIO.Write(metrics)).To(Succeed())
				readMetrics, err := metricsIO.Read()
				Expect(err).ToNot(HaveOccurred())
				Expect(readMetrics.Metrics).To(ConsistOf(metrics.Metrics))
			}
		})
	})

})
