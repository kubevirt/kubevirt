/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.Block[0].NameSet = false
			crs := blockMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
