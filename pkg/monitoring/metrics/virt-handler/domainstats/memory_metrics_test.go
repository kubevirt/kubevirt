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

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("memory metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DomainStats: &stats.DomainStats{
				Memory: &stats.DomainStatsMemory{
					RSSSet:           true,
					RSS:              1,
					AvailableSet:     true,
					Available:        2,
					UnusedSet:        true,
					Unused:           3,
					CachedSet:        true,
					Cached:           4,
					SwapInSet:        true,
					SwapIn:           5,
					SwapOutSet:       true,
					SwapOut:          6,
					MajorFaultSet:    true,
					MajorFault:       7,
					MinorFaultSet:    true,
					MinorFault:       8,
					ActualBalloonSet: true,
					ActualBalloon:    9,
					UsableSet:        true,
					Usable:           10,
					TotalSet:         true,
					Total:            11,
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := memoryMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_memory_resident_bytes", memoryResident, kibibytesToBytes(1)),
			Entry("kubevirt_vmi_memory_available_bytes", memoryAvailable, kibibytesToBytes(2)),
			Entry("kubevirt_vmi_memory_unused_bytes", memoryUnused, kibibytesToBytes(3)),
			Entry("kubevirt_vmi_memory_cached_bytes", memoryCached, kibibytesToBytes(4)),
			Entry("kubevirt_vmi_memory_swap_in_traffic_bytes", memorySwapInTrafficBytes, kibibytesToBytes(5)),
			Entry("kubevirt_vmi_memory_swap_out_traffic_bytes", memorySwapOutTrafficBytes, kibibytesToBytes(6)),
			Entry("kubevirt_vmi_memory_pgmajfault_total", memoryPgmajfaultTotal, 7.0),
			Entry("kubevirt_vmi_memory_pgminfault_total", memoryPgminfaultTotal, 8.0),
			Entry("kubevirt_vmi_memory_actual_ballon_bytes", memoryActualBallon, kibibytesToBytes(9)),
			Entry("kubevirt_vmi_memory_usable_bytes", memoryUsableBytes, kibibytesToBytes(10)),
			Entry("kubevirt_vmi_memory_domain_bytes", memoryDomainBytes, kibibytesToBytes(11)),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.Memory = &stats.DomainStatsMemory{
				RSSSet:        false,
				MinorFaultSet: false,
				TotalSet:      false,
			}
			crs := memoryMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
