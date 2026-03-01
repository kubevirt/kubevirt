package vhostmd

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	metricspkg "kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/metrics"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/util"
)

var _ = Describe("vhostmd", func() {

	Context("given real data from a real vhostmd", func() {

		It("should properly read and verify a real vhostmd0", func() {
			p, err := filepath.Abs("testdata/vhostmd0")
			Expect(err).ToNot(HaveOccurred())

			sPath, err := safepath.JoinAndResolveWithRelativeRoot(p)
			Expect(err).ToNot(HaveOccurred())

			v := NewMetricsIODisk(sPath)
			Expect(err).ToNot(HaveOccurred())

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
			path := filepath.Join(targetDir, "vhostmd0")
			Expect(CreateDisk(path)).To(Succeed())

			sPath, err := safepath.NewPathNoFollow(path)
			Expect(err).ToNot(HaveOccurred())
			metricsIO := NewMetricsIODisk(sPath)

			metrics, err := metricsIO.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(metrics.Metrics).To(BeEmpty())
		})

		It("should be able to repreatedly read and write metrics without modifying the result", func() {
			path := filepath.Join(targetDir, "vhostmd0")
			Expect(CreateDisk(path)).To(Succeed())

			sPath, err := safepath.NewPathNoFollow(path)
			Expect(err).ToNot(HaveOccurred())
			metricsIO := NewMetricsIODisk(sPath)
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

func (d *Disk) Metrics() (*api.Metrics, error) {
	m := &api.Metrics{}
	if err := xml.Unmarshal(d.Raw, m); err != nil {
		return nil, err
	}
	m.Text = strings.TrimSpace(m.Text)
	for i, metric := range m.Metrics {
		m.Metrics[i].Name = strings.TrimSpace(metric.Name)
		m.Metrics[i].Type = api.MetricType(strings.TrimSpace(string(metric.Type)))
		m.Metrics[i].Context = api.MetricContext(strings.TrimSpace(string(metric.Context)))
		m.Metrics[i].Value = strings.TrimSpace(metric.Value)
		m.Metrics[i].Text = strings.TrimSpace(metric.Text)
	}
	return m, nil
}

func (d *Disk) Verify() error {
	var checksum int32
	for _, b := range d.Raw {
		checksum = checksum + int32(b)
	}
	if d.Header.Flag > 0 {
		return fmt.Errorf("file is locked")
	}
	if checksum != d.Header.Checksum {
		return fmt.Errorf("checksum is %v, but expected %v", checksum, d.Header.Checksum)
	}
	return nil
}

func (v *vhostmd) Read() (*api.Metrics, error) {
	disk, err := readDisk(unsafepath.UnsafeAbsolute(v.filePath.Raw()))
	if err != nil {
		return nil, fmt.Errorf("failed to load vhostmd file: %v", err)
	}
	if err := disk.Verify(); err != nil {
		return nil, fmt.Errorf("failed to verify vhostmd file: %v", err)
	}
	return disk.Metrics()
}

func readDisk(filePath string) (*Disk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	// If the read operation succeeds, but close fails, we have already read the data,
	// so it is ok to not return the error.
	defer util.CloseIOAndCheckErr(f, nil)

	d := &Disk{
		Header: &Header{},
	}
	if err = binary.Read(f, binary.BigEndian, d.Header); err != nil {
		return nil, err
	}

	if d.Header.Flag == 0 {
		if d.Header.Length > maxBodyLength {
			return nil, fmt.Errorf("Invalid metrics file. Expected a maximum body length of %v, got %v", maxBodyLength, d.Header.Length)
		}

		d.Raw = make([]byte, d.Header.Length)

		if _, err = io.ReadFull(f, d.Raw); err != nil {
			return nil, err
		}
	}
	return d, err
}
