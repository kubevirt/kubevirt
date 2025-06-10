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
			Entry("kubevirt_vmi_migration_data_total_bytes", migrateVMIDataTotal, 3.0),
			Entry("kubevirt_vmi_migration_data_remaining_bytes", migrateVMIDataRemaining, 1.0),
			Entry("kubevirt_vmi_migration_data_processed_bytes", migrateVMIDataProcessed, 2.0),
			Entry("kubevirt_vmi_migration_dirty_memory_rate_bytes", migrateVmiDirtyMemoryRate, 3.0),
			Entry("kubevirt_vmi_migration_disk_transfer_rate_bytes", migrateVmiMemoryTransferRate, 4.0),
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
