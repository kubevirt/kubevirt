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

var _ = Describe("network metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DomainStats: &stats.DomainStats{
				Net: []stats.DomainStatsNet{
					{
						NameSet:    true,
						Name:       "vnet0",
						RxBytesSet: true,
						RxBytes:    1,
						TxBytesSet: true,
						TxBytes:    2,
						RxPktsSet:  true,
						RxPkts:     3,
						TxPktsSet:  true,
						TxPkts:     4,
						RxErrsSet:  true,
						RxErrs:     5,
						TxErrsSet:  true,
						TxErrs:     6,
						RxDropSet:  true,
						RxDrop:     7,
						TxDropSet:  true,
						TxDrop:     8,
					},
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := networkMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_network_receive_bytes_total", networkReceiveBytes, 1.0),
			Entry("kubevirt_vmi_network_transmit_bytes_total", networkTransmitBytes, 2.0),
			Entry("kubevirt_vmi_network_receive_packets_total", networkReceivePackets, 3.0),
			Entry("kubevirt_vmi_network_transmit_packets_total", networkTransmitPackets, 4.0),
			Entry("kubevirt_vmi_network_receive_errors_total", networkReceiveErrors, 5.0),
			Entry("kubevirt_vmi_network_transmit_errors_total", networkTransmitErrors, 6.0),
			Entry("kubevirt_vmi_network_receive_packets_dropped_total", networkReceivePacketsDropped, 7.0),
			Entry("kubevirt_vmi_network_transmit_packets_dropped_total", networkTransmitPacketsDropped, 8.0),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.Net[0].NameSet = false
			crs := networkMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
