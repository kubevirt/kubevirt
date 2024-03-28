package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("migration metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DomainStats: &stats.DomainStats{
				MigrateDomainJobInfo: &stats.DomainJobInfo{
					DataRemainingSet: true,
					DataRemaining:    1,
					DataProcessedSet: true,
					DataProcessed:    2,
					MemDirtyRateSet:  true,
					MemDirtyRate:     3,
					MemoryBpsSet:     true,
					MemoryBps:        4,
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := migrationMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(gomegaContainsMetricMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_migration_data_remaining_bytes", migrateVMIDataRemaining, 1.0),
			Entry("kubevirt_vmi_migration_data_processed_bytes", migrateVMIDataProcessed, 2.0),
			Entry("kubevirt_vmi_migration_dirty_memory_rate_bytes", migrateVmiDirtyMemoryRate, 3.0),
			Entry("kubevirt_vmi_migration_disk_transfer_rate_bytes", migrateVmiMemoryTransferRate, 4.0),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.MigrateDomainJobInfo = &stats.DomainJobInfo{
				DataRemainingSet: false,
			}
			crs := migrationMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
