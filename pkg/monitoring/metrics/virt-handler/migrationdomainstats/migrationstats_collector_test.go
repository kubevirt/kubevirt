/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package migrationdomainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("migration metrics", func() {
	Context("on Collect", func() {
		vmiStats := &result{
			vmi: "test-vmi-1",
			domainJobInfo: stats.DomainJobInfo{
				DataTotalSet:     true,
				DataTotal:        3,
				DataRemainingSet: true,
				DataRemaining:    1,
				DataProcessedSet: true,
				DataProcessed:    2,
				MemDirtyRateSet:  true,
				MemDirtyRate:     3,
				MemoryBpsSet:     true,
				MemoryBps:        4,
			},
		}

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := parse(vmiStats)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_migration_data_bytes_total", migrateVMIDataTotal, 3.0),
			Entry("kubevirt_vmi_migration_data_remaining_bytes", migrateVMIDataRemaining, 1.0),
			Entry("kubevirt_vmi_migration_data_processed_bytes", migrateVMIDataProcessed, 2.0),
			Entry("kubevirt_vmi_migration_dirty_memory_rate_bytes", migrateVmiDirtyMemoryRate, 3.0),
			Entry("kubevirt_vmi_migration_memory_transfer_rate_bytes", migrateVmiMemoryTransferRate, 4.0),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.domainJobInfo = stats.DomainJobInfo{
				DataRemainingSet: false,
			}
			crs := parse(vmiStats)
			Expect(crs).To(BeEmpty())
		})
	})
})
