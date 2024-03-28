package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("cpu metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DomainStats: &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{
					TimeSet:   true,
					Time:      1,
					UserSet:   true,
					User:      2,
					SystemSet: true,
					System:    3,
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := cpuMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(gomegaContainsMetricMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_cpu_usage_seconds_total", cpuUsageSeconds, nanosecondsToSeconds(1)),
			Entry("kubevirt_vmi_cpu_user_usage_seconds_total", cpuUserUsageSeconds, nanosecondsToSeconds(2)),
			Entry("kubevirt_vmi_cpu_system_usage_seconds_total", cpuSystemUsageSeconds, nanosecondsToSeconds(3)),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.Cpu = &stats.DomainStatsCPU{
				TimeSet: false,
			}
			crs := cpuMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
