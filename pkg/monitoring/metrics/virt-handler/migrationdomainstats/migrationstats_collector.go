/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package migrationdomainstats

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"
)

var (
	migrationdomainstatsHandler *handler

	MigrationStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			migrateVMIDataTotal,
			migrateVMIDataRemaining,
			migrateVMIDataProcessed,
			migrateVmiDirtyMemoryRate,
			migrateVmiMemoryTransferRate,
		},
		CollectCallback: migrationStatsCollectorCallback,
	}

	migrateVMIDataTotal = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_data_bytes_total",
			Help: "The total Guest OS data to be migrated to the new VM.",
		},
	)

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
			Name: "kubevirt_vmi_migration_memory_transfer_rate_bytes",
			Help: "The rate at which the memory is being transferred.",
		},
	)
)

func SetupMigrationStatsCollector(vmiInformer cache.SharedIndexInformer) error {
	if vmiInformer == nil {
		return nil
	}

	var err error
	migrationdomainstatsHandler, err = newHandler(vmiInformer)
	return err
}

func migrationStatsCollectorCallback() []operatormetrics.CollectorResult {
	results := migrationdomainstatsHandler.Collect()

	var crs []operatormetrics.CollectorResult
	for _, r := range results {
		crs = append(crs, parse(&r)...)
	}

	return crs
}

func parse(r *result) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	jobInfo := r.domainJobInfo

	if jobInfo.DataTotalSet {
		crs = append(crs, newCR(r, migrateVMIDataTotal, float64(jobInfo.DataTotal)))
	}

	if jobInfo.DataRemainingSet {
		crs = append(crs, newCR(r, migrateVMIDataRemaining, float64(jobInfo.DataRemaining)))
	}

	if jobInfo.DataProcessedSet {
		crs = append(crs, newCR(r, migrateVMIDataProcessed, float64(jobInfo.DataProcessed)))
	}

	if jobInfo.MemDirtyRateSet {
		crs = append(crs, newCR(r, migrateVmiDirtyMemoryRate, float64(jobInfo.MemDirtyRate)))
	}

	if jobInfo.MemoryBpsSet {
		crs = append(crs, newCR(r, migrateVmiMemoryTransferRate, float64(jobInfo.MemoryBps)))
	}

	return crs
}

func newCR(r *result, metric operatormetrics.Metric, value float64) operatormetrics.CollectorResult {
	vmiLabels := map[string]string{
		"namespace": r.namespace,
		"name":      r.vmi,
		"node":      r.node,
	}

	return operatormetrics.CollectorResult{
		Metric:      metric,
		ConstLabels: vmiLabels,
		Value:       value,
		Timestamp:   r.timestamp,
	}
}
