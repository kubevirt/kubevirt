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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/prometheus/client_golang/prometheus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k6tv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = BeforeSuite(func() {
	log.Log.SetIOWriter(GinkgoWriter)
})

var _ = Describe("Prometheus", func() {
	Context("on blocked source", func() {
		It("should handle closed reporting socket", func() {
			ch := make(chan prometheus.Metric)
			close(ch)

			ps := prometheusScraper{ch: ch}

			testReportPanic := func() {
				vmStats := &stats.DomainStats{
					Cpu: &stats.DomainStatsCPU{},
					Memory: &stats.DomainStatsMemory{
						// trigger write on a socket. We need a value set - any value
						RSS:    1024,
						RSSSet: true,
					},
				}
				vmi := k6tv1.VirtualMachineInstance{}
				ps.Report("test", &vmi, vmStats)
			}
			Expect(testReportPanic).ToNot(Panic())
		})
	})

	Context("on handling push", func() {
		It("should send rss", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					// trigger write on a socket. We need a value set - any value
					RSS:    1024,
					RSSSet: true,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_resident_bytes"))
		})

		It("should send available memory", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					AvailableSet: true,
					Available:    2048,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_available_bytes"))
		})

		It("should handle swapin", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					SwapInSet: true,
					SwapIn:    4096,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_swap_traffic_bytes_total"))
		})

		It("should handle swapout", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{},
				Memory: &stats.DomainStatsMemory{
					SwapOutSet: true,
					SwapOut:    4096,
				},
			}
			vmi := k6tv1.VirtualMachineInstance{}
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_memory_swap_traffic_bytes_total"))
		})

		It("should handle vcpu metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_vcpu_seconds"))
		})

		It("should not expose vcpu metrics for invalid DomainStats", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			// metrics about invalid stats never get pushed into the channel
			Eventually(ch).Should(BeEmpty())
		})

		It("should handle block read iops metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_iops_total"))
		})

		It("should handle block write iops metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_iops_total"))
		})

		It("should handle block read bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_traffic_bytes_total"))
		})

		It("should handle block write bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_traffic_bytes_total"))
		})

		It("should handle block read time metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_times_ms_total"))
		})

		It("should handle block write time metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_storage_times_ms_total"))
		})

		It("should not expose nameless block metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			Eventually(ch).Should(BeEmpty())
		})

		It("should handle network rx traffic bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_traffic_bytes_total"))
		})

		It("should handle network tx traffic bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_traffic_bytes_total"))
		})

		It("should handle network rx packets metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_traffic_packets_total"))
		})

		It("should handle network tx traffic bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_traffic_packets_total"))
		})

		It("should handle network rx errors metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_errors_total"))
		})

		It("should handle network tx traffic bytes metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_network_errors_total"))
		})

		It("should not expose nameless network interface metrics", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			Eventually(ch).Should(BeEmpty())
		})

		It("should add kubernetes metadata labels", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubernetes_vmi_label_kubevirt_io_nodeName"))
		})

		It("should expose vcpu wait metric", func() {
			ch := make(chan prometheus.Metric, 1)
			defer close(ch)

			ps := prometheusScraper{ch: ch}

			vmStats := &stats.DomainStats{
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
			ps.Report("test", &vmi, vmStats)

			result := <-ch
			Expect(result).ToNot(BeNil())
			Expect(result.Desc().String()).To(ContainSubstring("kubevirt_vmi_vcpu_wait_seconds"))
		})
	})
})

var _ = Describe("Utility functions", func() {
	Context("VMI Phases map reporting", func() {
		It("should handle missing VMs", func() {
			var phasesMap map[string]uint64

			phasesMap = makeVMIsPhasesMap(nil)
			Expect(phasesMap).NotTo(BeNil())
			Expect(len(phasesMap)).To(Equal(0))

			vmis := []*k6tv1.VirtualMachineInstance{}
			phasesMap = makeVMIsPhasesMap(vmis)
			Expect(phasesMap).NotTo(BeNil())
			Expect(len(phasesMap)).To(Equal(0))
		})

		It("should handle different VMI phases", func() {
			vmis := []*k6tv1.VirtualMachineInstance{
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "running#0",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Running",
					},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pending#0",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Pending",
					},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "scheduling#0",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Scheduling",
					},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "running#1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Phase: "Running",
					},
				},
			}

			phasesMap := makeVMIsPhasesMap(vmis)
			Expect(phasesMap).NotTo(BeNil())
			Expect(len(phasesMap)).To(Equal(3))
			Expect(phasesMap["running"]).To(Equal(uint64(2)))
			Expect(phasesMap["pending"]).To(Equal(uint64(1)))
			Expect(phasesMap["scheduling"]).To(Equal(uint64(1)))
			Expect(phasesMap["bogus"]).To(Equal(uint64(0))) // intentionally bogus key
		})
	})
})
