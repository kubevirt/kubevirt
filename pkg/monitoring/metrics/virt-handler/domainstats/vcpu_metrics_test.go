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

var _ = Describe("vcpu metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DomainStats: &stats.DomainStats{
				Vcpu: []stats.DomainStatsVcpu{
					{
						TimeSet:  true,
						Time:     1,
						WaitSet:  true,
						Wait:     2,
						DelaySet: true,
						Delay:    3,
					},
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := vcpuMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_vcpu_seconds_total", vcpuSeconds, nanosecondsToSeconds(1)),
			Entry("kubevirt_vmi_vcpu_wait_seconds_total", vcpuWaitSeconds, nanosecondsToSeconds(2)),
			Entry("kubevirt_vmi_vcpu_delay_seconds_total", vcpuDelaySeconds, nanosecondsToSeconds(3)),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.Vcpu = []stats.DomainStatsVcpu{
				{
					TimeSet: false,
				},
			}
			crs := vcpuMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
