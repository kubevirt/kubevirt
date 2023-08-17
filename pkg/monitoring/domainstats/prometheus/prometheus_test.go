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
	"kubevirt.io/kubevirt/pkg/pointer"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"libvirt.org/go/libvirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("Prometheus", func() {
	newVmStats := func(domainStats *stats.DomainStats, fsStats *k6tv1.VirtualMachineInstanceFileSystemList) *VirtualMachineInstanceStats {
		if fsStats == nil {
			fsStats = &k6tv1.VirtualMachineInstanceFileSystemList{
				Items: []k6tv1.VirtualMachineInstanceFileSystem{},
			}
		}

		return &VirtualMachineInstanceStats{
			DomainStats: domainStats,
			FsStats:     *fsStats,
		}
	}

	Context("on blocked source", func() {
		It("should handle closed reporting socket", func() {
			ch := make(chan prometheus.Metric)
			close(ch)

			ps := prometheusScraper{ch: ch}

			testReportPanic := func() {
				domainStats := &stats.DomainStats{
					Cpu: &stats.DomainStatsCPU{},
					Memory: &stats.DomainStatsMemory{
						// trigger write on a socket. We need a value set - any value
						RSS:    1024,
						RSSSet: true,
					},
				}
				vmi := k6tv1.VirtualMachineInstance{}
				ps.Report("test", &vmi, newVmStats(domainStats, nil))
			}
			Expect(testReportPanic).ToNot(Panic())
		})
	})

	Context("on handling push", func() {
		It("should send rss", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					// trigger write on a socket. We need a value set - any value
					RSS:    1024,
					RSSSet: true,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_resident_bytes"))
		})

		It("should send available memory", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					AvailableSet: true,
					Available:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_available_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should send unused memory", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					UnusedSet: true,
					Unused:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_unused_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should send cached memory", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					CachedSet: true,
					Cached:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_cached_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should handle swapin", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					SwapInSet: true,
					SwapIn:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_swap_in_traffic_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should handle swapout", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					SwapOutSet: true,
					SwapOut:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_swap_out_traffic_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should handle major page faults metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					MajorFaultSet: true,
					MajorFault:    1024,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_pgmajfault_total"))
			Expect(dto.Counter.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should handle minor page faults metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					MinorFaultSet: true,
					MinorFault:    1024,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_pgminfault_total"))
			Expect(dto.Counter.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should handle actual balloon metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					ActualBalloonSet: true,
					ActualBalloon:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_actual_balloon_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should handle the usable metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					UsableSet: true,
					Usable:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_usable_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		It("should handle the total memory metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					TotalSet: true,
					Total:    1,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_domain_bytes"))
			Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1024)))
		})

		DescribeTable("Assert vmi migration metrics",
			func(metricName string, migrateDomainJobInfoStats *stats.DomainJobInfo) {
				ch := make(chan prometheus.Metric, 1)
				defer close(ch)

				ps := prometheusScraper{ch: ch}

				domainStats := &stats.DomainStats{
					Cpu:                  &stats.DomainStatsCPU{},
					Memory:               &stats.DomainStatsMemory{},
					MigrateDomainJobInfo: migrateDomainJobInfoStats,
				}
				vmi := k6tv1.VirtualMachineInstance{}
				ps.Report("test", &vmi, newVmStats(domainStats, nil))

				result := <-ch
				dto := &io_prometheus_client.Metric{}
				result.Write(dto)

				Expect(result).ToNot(BeNil())
				Expect(result.Desc().String()).To(ContainSubstring(metricName))
				Expect(dto.Gauge.GetValue()).To(BeEquivalentTo(float64(1)))
			},
			Entry("should handle Data_Processed metrics for VMs",
				MigrateVmiDataProcessedMetricName,
				&stats.DomainJobInfo{
					DataProcessedSet: true,
					DataProcessed:    1,
				}),
			Entry("should handle Data_Remaining metrics for VMs",
				MigrateVmiDataRemainingMetricName,
				&stats.DomainJobInfo{
					DataRemainingSet: true,
					DataRemaining:    1,
				}),
			Entry("should handle MemDirtyRate metrics for VMs",
				MigrateVmiDirtyMemoryRateMetricName,
				&stats.DomainJobInfo{
					MemDirtyRateSet: true,
					MemDirtyRate:    1,
				}),
			Entry("should handle MemoryBps metrics for VMs",
				MigrateVmiMemoryTransferRateMetricName,
				&stats.DomainJobInfo{
					MemoryBpsSet: true,
					MemoryBps:    1,
				}),
			Entry("should handle DiskBps metrics for VMs",
				MigrateVmiDiskTransferRateMetricName,
				&stats.DomainJobInfo{
					DiskBpsSet: true,
					DiskBps:    1,
				}),
		)

		It("should handle vcpu metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Vcpu: []stats.DomainStatsVcpu{
					{
						StateSet: true,
						State:    1,
						TimeSet:  true,
						Time:     2000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_vcpu_seconds_total"))
		})

		It("should not expose vcpu metrics for invalid DomainStats", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Vcpu: []stats.DomainStatsVcpu{
					{
						// vcpu State is not set!
						TimeSet: true,
						Time:    2000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			// metrics about invalid stats never get pushed into the channel
			Eventually(ch).Should(BeEmpty())
		})

		It("should expose vcpu state as a human readable string", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Vcpu: []stats.DomainStatsVcpu{
					{
						StateSet: true,
						State:    int(libvirt.VCPU_RUNNING),
						TimeSet:  true,
						Time:     2000,
					},
				},
			}

			metric := &io_prometheus_client.Metric{}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			result.Write(metric)

			Expect(result).ToNot(BeNil())
			for _, label := range metric.GetLabel() {
				if label.GetName() == "state" {
					Expect(label.GetValue()).To(BeEquivalentTo("running"))
				}
			}

			domainStats = &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Vcpu: []stats.DomainStatsVcpu{
					{
						StateSet: true,
						State:    int(libvirt.VCPU_BLOCKED),
						TimeSet:  true,
						Time:     2000,
					},
				},
			}

			metric = &io_prometheus_client.Metric{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result = <-ch
			result.Write(metric)

			Expect(result).ToNot(BeNil())
			for _, label := range metric.GetLabel() {
				if label.GetName() == "state" {
					Expect(label.GetValue()).To(BeEquivalentTo("blocked"))
				}
			}

			domainStats = &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Vcpu: []stats.DomainStatsVcpu{
					{
						StateSet: true,
						State:    int(libvirt.VCPU_OFFLINE),
						TimeSet:  true,
						Time:     2000,
					},
				},
			}

			metric = &io_prometheus_client.Metric{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result = <-ch
			result.Write(metric)

			Expect(result).ToNot(BeNil())
			for _, label := range metric.GetLabel() {
				if label.GetName() == "state" {
					Expect(label.GetValue()).To(BeEquivalentTo("offline"))
				}
			}
		})

		It("should handle block read iops metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:   true,
						Name:      "vda",
						RdReqsSet: true,
						RdReqs:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_iops_read_total"))
		})

		It("should handle block write iops metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:   true,
						Name:      "vda",
						WrReqsSet: true,
						WrReqs:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_iops_write_total"))
		})

		It("should handle block read bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						RdBytesSet: true,
						RdBytes:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_read_traffic_bytes_total"))
		})

		It("should handle block write bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						WrBytesSet: true,
						WrBytes:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_write_traffic_bytes_total"))
		})

		It("should handle block read time metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						RdTimesSet: true,
						RdTimes:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_read_times_seconds_total"))
		})

		It("should handle block write time metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						WrTimesSet: true,
						WrTimes:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_write_times_seconds_total"))
		})

		It("should handle block flush requests metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:   true,
						Name:      "vda",
						FlReqsSet: true,
						FlReqs:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_flush_requests_total"))
		})

		It("should handle block flush times metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						FlTimesSet: true,
						FlTimes:    1000000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_flush_times_seconds_total"))
		})

		It("should use alias when alias is not empty", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						Alias:      "disk0",
						FlTimesSet: true,
						FlTimes:    1000000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())

			dto := &io_prometheus_client.Metric{}
			err := result.Write(dto)
			Expect(err).ShouldNot(HaveOccurred())
			expectedLabelPair := &io_prometheus_client.LabelPair{
				Name:  pointer.P("drive"),
				Value: pointer.P("disk0"),
			}
			Expect(dto.GetLabel()).To(ContainElement(expectedLabelPair))
		})

		It("should use the name when alias is empty", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						NameSet:    true,
						Name:       "vda",
						Alias:      "",
						FlTimesSet: true,
						FlTimes:    1000000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())

			dto := &io_prometheus_client.Metric{}
			err := result.Write(dto)
			Expect(err).ShouldNot(HaveOccurred())
			expectedLabelPair := &io_prometheus_client.LabelPair{
				Name:  pointer.P("drive"),
				Value: pointer.P("vda"),
			}
			Expect(dto.GetLabel()).To(ContainElement(expectedLabelPair))
		})

		It("should not expose nameless block metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Block: []stats.DomainStatsBlock{
					{
						RdReqsSet: true,
						RdReqs:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			Eventually(ch).Should(BeEmpty())
		})

		It("should handle network rx traffic bytes metrics", func() {
			ch := make(chan prometheus.Metric, 2)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:    true,
						Name:       "vnet0",
						RxBytesSet: true,
						RxBytes:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_traffic_bytes_total"))

			result = <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_receive_bytes_total"))
		})

		It("should handle network tx traffic bytes metrics", func() {
			ch := make(chan prometheus.Metric, 2)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:    true,
						Name:       "vnet0",
						TxBytesSet: true,
						TxBytes:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_traffic_bytes_total"))

			result = <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_transmit_bytes_total"))
		})

		It("should handle network rx packets metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:   true,
						Name:      "vnet0",
						RxPktsSet: true,
						RxPkts:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_receive_packets_total"))
		})

		It("should handle network tx traffic packets metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:   true,
						Name:      "vnet0",
						TxPktsSet: true,
						TxPkts:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_transmit_packets_total"))
		})

		It("should handle network rx errors metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:   true,
						Name:      "vnet0",
						RxErrsSet: true,
						RxErrs:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_receive_errors_total"))
		})

		It("should handle network tx traffic error metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:   true,
						Name:      "vnet0",
						TxErrsSet: true,
						TxErrs:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_transmit_errors_total"))
		})

		It("should handle network rx drop metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:   true,
						Name:      "vnet0",
						RxDropSet: true,
						RxDrop:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_receive_packets_dropped_total"))
		})

		It("should handle network tx drop metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						NameSet:   true,
						Name:      "vnet0",
						TxDropSet: true,
						TxDrop:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_transmit_packets_dropped_total"))
		})

		It("should not expose nameless network interface metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net: []stats.DomainStatsNet{
					{
						TxErrsSet: true,
						TxErrs:    1000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			Eventually(ch).Should(BeEmpty())
		})

		It("should add kubernetes metadata labels", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					RSS:    1024,
					RSSSet: true,
				},
			}

			vmi := k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/nodeName": "node01",
					},
				},
			}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubernetes_vmi_label_kubevirt_io_nodeName"))
		})

		It("should expose vcpu wait metric", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net:    []stats.DomainStatsNet{},
				Vcpu: []stats.DomainStatsVcpu{
					{
						WaitSet: true,
						Wait:    6,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_vcpu_wait_seconds_total"))
		})

		It("should expose vcpu delay metric", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{},
				Net:    []stats.DomainStatsNet{},
				Vcpu: []stats.DomainStatsVcpu{
					{
						DelaySet: true,
						Delay:    800000000,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_vcpu_delay_seconds_total"))
		})

		It("should expose vcpu to cpu pinning metric", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:       &stats.DomainStatsCPU{},
				Memory:    &stats.DomainStatsMemory{},
				Net:       []stats.DomainStatsNet{},
				Vcpu:      []stats.DomainStatsVcpu{},
				CPUMapSet: true,
				CPUMap:    [][]bool{{true, false, true}},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_node_cpu_affinity"))
			Expect(dto.GetGauge().GetValue()).To(Equal(float64(2)))
		})

		It("should expose filesystem metrics", func() {
			ch := make(chan prometheus.Metric, 2)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:                  &stats.DomainStatsCPU{},
				Memory:               &stats.DomainStatsMemory{},
				Net:                  []stats.DomainStatsNet{},
				MigrateDomainJobInfo: &stats.DomainJobInfo{},
			}

			fsStats := &k6tv1.VirtualMachineInstanceFileSystemList{
				Items: []k6tv1.VirtualMachineInstanceFileSystem{
					{
						DiskName:   "disk1",
						TotalBytes: 1000,
						UsedBytes:  10,
					},
				},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, fsStats))
			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_filesystem_capacity_bytes"))
			result = <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_filesystem_used_bytes"))
			Expect(ch).To(BeEmpty())
		})

		DescribeTable("CPU metrics", func(metricName string, MetricValue int, cpuStats *stats.DomainStatsCPU) {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			domainStats := &stats.DomainStats{
				Cpu:    cpuStats,
				Memory: &stats.DomainStatsMemory{},
				Net:    []stats.DomainStatsNet{},
				Vcpu:   []stats.DomainStatsVcpu{},
			}

			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, newVmStats(domainStats, nil))

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring(metricName))

			dto := &io_prometheus_client.Metric{}
			result.Write(dto)

			Expect(dto.GetCounter().GetValue()).To(Equal(float64(MetricValue)))

		},
			Entry("Total CPU time spent in all modes (sum of both vcpu and hypervisor usage)", "kubevirt_vmi_cpu_usage_seconds_total", 123, &stats.DomainStatsCPU{
				TimeSet: true,
				Time:    123000000000},
			),
			Entry("Total CPU time spent in user mode", "kubevirt_vmi_cpu_user_usage_seconds_total", 456, &stats.DomainStatsCPU{
				UserSet: true,
				User:    456000000000},
			),
			Entry("Total CPU time spent in system mode", "kubevirt_vmi_cpu_system_usage_seconds_total", 789, &stats.DomainStatsCPU{
				SystemSet: true,
				System:    789000000000},
			))
	})
})
