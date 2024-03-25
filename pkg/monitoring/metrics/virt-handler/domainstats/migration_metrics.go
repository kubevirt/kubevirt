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
 * Copyright the KubeVirt Authors.
 *
 */

package domainstats

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

var (
	migrateVMIDataRemaining = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_data_remaining_bytes",
			Help: "The remaining guest OS data to be migrated to the new VM.",
		},
	)

	migrateVMIDataProcessed = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_data_processed_bytes",
			Help: "The total Guest OS data processed and migrated to the new VM.",
		},
	)

	migrateVmiDirtyMemoryRate = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_dirty_memory_rate_bytes",
			Help: "The rate of memory being dirty in the Guest OS.",
		},
	)

	migrateVmiMemoryTransferRate = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_disk_transfer_rate_bytes",
			Help: "The rate at which the memory is being transferred.",
		},
	)
)

type migrationMetrics struct{}

func (migrationMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		migrateVMIDataRemaining,
		migrateVMIDataProcessed,
		migrateVmiDirtyMemoryRate,
		migrateVmiMemoryTransferRate,
	}
}

func (migrationMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil || vmiReport.vmiStats.DomainStats.MigrateDomainJobInfo == nil {
		return crs
	}

	jobInfo := vmiReport.vmiStats.DomainStats.MigrateDomainJobInfo

	if jobInfo.DataRemainingSet {
		crs = append(crs, vmiReport.newCollectorResult(migrateVMIDataRemaining, float64(jobInfo.DataRemaining)))
	}

	if jobInfo.DataProcessedSet {
		crs = append(crs, vmiReport.newCollectorResult(migrateVMIDataProcessed, float64(jobInfo.DataProcessed)))
	}

	if jobInfo.MemDirtyRateSet {
		crs = append(crs, vmiReport.newCollectorResult(migrateVmiDirtyMemoryRate, float64(jobInfo.MemDirtyRate)))
	}

	if jobInfo.MemoryBpsSet {
		crs = append(crs, vmiReport.newCollectorResult(migrateVmiMemoryTransferRate, float64(jobInfo.MemoryBps)))
	}

	return crs
}
