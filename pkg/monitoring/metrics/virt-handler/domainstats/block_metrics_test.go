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
 * Copyright The KubeVirt Authors.
 *
 */

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("block metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DomainStats: &stats.DomainStats{
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						RdReqsSet:  true,
						RdReqs:     1,
						WrReqsSet:  true,
						WrReqs:     2,
						RdBytesSet: true,
						RdBytes:    3,
						WrBytesSet: true,
						WrBytes:    4,
						RdTimesSet: true,
						RdTimes:    5,
						WrTimesSet: true,
						WrTimes:    6,
						FlReqsSet:  true,
						FlReqs:     7,
						FlTimesSet: true,
						FlTimes:    8,
					},
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := blockMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_storage_iops_read_total", storageIopsRead, 1.0),
			Entry("kubevirt_vmi_storage_iops_write_total", storageIopsWrite, 2.0),
			Entry("kubevirt_vmi_storage_read_traffic_bytes_total", storageReadTrafficBytes, 3.0),
			Entry("kubevirt_vmi_storage_write_traffic_bytes_total", storageWriteTrafficBytes, 4.0),
			Entry("kubevirt_vmi_storage_read_times_seconds_total", storageReadTimesSeconds, nanosecondsToSeconds(5)),
			Entry("kubevirt_vmi_storage_write_times_seconds_total", storageWriteTimesSeconds, nanosecondsToSeconds(6)),
			Entry("kubevirt_vmi_storage_flush_requests_total", storageFlushRequests, 7.0),
			Entry("kubevirt_vmi_storage_flush_times_seconds_total", storageFlushTimesSeconds, nanosecondsToSeconds(8)),
		)

		It("should convert libvirt latency histogram bins to Prometheus buckets", func() {
			histogram := &stats.DomainStatsBlockLatencyHistogram{
				Bins: []stats.DomainStatsBlockLatencyHistogramBin{
					{
						StartSet: true,
						Start:    0,
						ValueSet: true,
						Value:    5,
					},
					{
						StartSet: true,
						Start:    1_000_000,
						ValueSet: true,
						Value:    7,
					},
					{
						StartSet: true,
						Start:    10_000_000,
						ValueSet: true,
						Value:    2,
					},
				},
			}

			buckets, count, ok := convertLatencyHistogram(histogram)

			Expect(ok).To(BeTrue())
			Expect(count).To(Equal(uint64(14)))

			Expect(buckets).To(HaveLen(2))
			Expect(buckets).To(HaveKeyWithValue(0.001, uint64(5)))
			Expect(buckets).To(HaveKeyWithValue(0.01, uint64(12)))
		})

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.Block[0].NameSet = false
			crs := blockMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})

	Context("Describe", func() {
		It("should include histogram descriptor with correct label names", func() {
			ch := make(chan *prometheus.Desc, 64)
			collector := domainStatsPrometheusCollector{}
			collector.Describe(ch)
			close(ch)

			var found bool
			for desc := range ch {
				if desc.String() == histogramDesc.String() {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "histogramDesc not found in Describe output")
		})
	})

	Context("collectBlockLatencyHistograms", func() {
		It("should emit histogram metrics with correct labels and values", func() {
			histVmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-1",
					Namespace: "test-ns-1",
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "test-node-1",
				},
			}

			histStats := &VirtualMachineInstanceStats{
				DomainStats: &stats.DomainStats{
					Block: []stats.DomainStatsBlock{
						{
							NameSet:    true,
							Name:       "vda",
							Alias:      "rootdisk",
							RdTimesSet: true,
							RdTimes:    500_000_000,
							WrTimesSet: true,
							WrTimes:    300_000_000,
							FlTimesSet: true,
							FlTimes:    100_000_000,
							LatencyHistograms: stats.DomainStatsBlockLatencyHistograms{
								Read: &stats.DomainStatsBlockLatencyHistogram{
									Bins: []stats.DomainStatsBlockLatencyHistogramBin{
										{StartSet: true, Start: 0, ValueSet: true, Value: 10},
										{StartSet: true, Start: 1_000_000, ValueSet: true, Value: 5},
										{StartSet: true, Start: 10_000_000, ValueSet: true, Value: 2},
									},
								},
								Write: &stats.DomainStatsBlockLatencyHistogram{
									Bins: []stats.DomainStatsBlockLatencyHistogramBin{
										{StartSet: true, Start: 0, ValueSet: true, Value: 3},
										{StartSet: true, Start: 1_000_000, ValueSet: true, Value: 7},
										{StartSet: true, Start: 10_000_000, ValueSet: true, Value: 1},
									},
								},
								Flush: &stats.DomainStatsBlockLatencyHistogram{
									Bins: []stats.DomainStatsBlockLatencyHistogramBin{
										{StartSet: true, Start: 0, ValueSet: true, Value: 1},
										{StartSet: true, Start: 1_000_000, ValueSet: true, Value: 0},
										{StartSet: true, Start: 10_000_000, ValueSet: true, Value: 0},
									},
								},
							},
						},
					},
				},
			}

			report := newVirtualMachineInstanceReport(histVmi, histStats)

			ch := make(chan prometheus.Metric, 64)
			collectBlockLatencyHistograms(report, ch)
			close(ch)

			var metrics []prometheus.Metric
			for m := range ch {
				metrics = append(metrics, m)
			}

			Expect(metrics).To(HaveLen(3), "expected one histogram per operation (read, write, flush)")

			for _, m := range metrics {
				Expect(m.Desc()).To(Equal(histogramDesc),
					"every emitted metric must use the shared histogramDesc")
			}

			dto := &io_prometheus_client.Metric{}
			Expect(metrics[0].Write(dto)).To(Succeed())

			labelMap := map[string]string{}
			for _, lp := range dto.Label {
				labelMap[lp.GetName()] = lp.GetValue()
			}
			Expect(labelMap).To(HaveKeyWithValue("node", "test-node-1"))
			Expect(labelMap).To(HaveKeyWithValue("namespace", "test-ns-1"))
			Expect(labelMap).To(HaveKeyWithValue("name", "test-vmi-1"))
			Expect(labelMap).To(HaveKeyWithValue("drive", "rootdisk"))
			Expect(labelMap).To(HaveKeyWithValue("operation", "read"))

			Expect(dto.Histogram).NotTo(BeNil())
			Expect(dto.Histogram.GetSampleCount()).To(Equal(uint64(17)))
			Expect(dto.Histogram.GetSampleSum()).To(BeNumerically("~", 0.5, 1e-9))
			Expect(dto.Histogram.GetBucket()).To(HaveLen(2))
		})

		It("should skip operations without time data", func() {
			histVmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-2",
					Namespace: "test-ns-2",
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "test-node-2",
				},
			}

			histStats := &VirtualMachineInstanceStats{
				DomainStats: &stats.DomainStats{
					Block: []stats.DomainStatsBlock{
						{
							NameSet:    true,
							Name:       "vdb",
							RdTimesSet: true,
							RdTimes:    100_000_000,
							WrTimesSet: false,
							FlTimesSet: false,
							LatencyHistograms: stats.DomainStatsBlockLatencyHistograms{
								Read: &stats.DomainStatsBlockLatencyHistogram{
									Bins: []stats.DomainStatsBlockLatencyHistogramBin{
										{StartSet: true, Start: 0, ValueSet: true, Value: 1},
										{StartSet: true, Start: 1_000_000, ValueSet: true, Value: 0},
									},
								},
							},
						},
					},
				},
			}

			report := newVirtualMachineInstanceReport(histVmi, histStats)

			ch := make(chan prometheus.Metric, 64)
			collectBlockLatencyHistograms(report, ch)
			close(ch)

			var metrics []prometheus.Metric
			for m := range ch {
				metrics = append(metrics, m)
			}

			Expect(metrics).To(HaveLen(1), "only read should emit (write/flush have no time data)")

			dto := &io_prometheus_client.Metric{}
			Expect(metrics[0].Write(dto)).To(Succeed())

			labelMap := map[string]string{}
			for _, lp := range dto.Label {
				labelMap[lp.GetName()] = lp.GetValue()
			}
			Expect(labelMap).To(HaveKeyWithValue("drive", "vdb"))
			Expect(labelMap).To(HaveKeyWithValue("operation", "read"))
		})
	})
})
