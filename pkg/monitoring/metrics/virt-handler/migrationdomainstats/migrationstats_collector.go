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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/domainstats"
)

var (
	migrationdomainstatsHandler *handler

	MigrationMetrics = []operatormetrics.Metric{
		migrationDowntime,
	}

	MigrationStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			migrateVMIDataTotal,
			migrateVMIDataRemaining,
			migrateVMIDataProcessed,
			migrateVmiDirtyMemoryRate,
			migrateVmiMemoryTransferRate,
			migrateVmiLastDowntime,
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

	migrationDowntime = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_downtime_seconds",
			Help: "Histogram of the time, in seconds, a guest was paused during successful live migration cut-over.",
		},
		prometheus.HistogramOpts{
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 4, 8, 16, 32},
		},
		[]string{"node"},
	)

	migrateVmiLastDowntime = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_last_downtime_seconds",
			Help: "Time, in seconds, the guest was paused during the cut-over of its last successful live migration.",
		},
	)
)

func SetupMigrationStatsCollector(
	nodeName string,
	sourceVMIInformer, globalVMIInformer cache.SharedIndexInformer,
	domainInformer cache.SharedInformer,
) error {
	if sourceVMIInformer == nil {
		return nil
	}

	var err error
	migrationdomainstatsHandler, err = newHandler(nodeName, sourceVMIInformer, globalVMIInformer, domainInformer)
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

	if r.completed && jobInfo.DowntimeSet {
		crs = append(crs, newCR(r, migrateVmiLastDowntime, domainstats.MillisecondsToSeconds(jobInfo.Downtime)))
	}

	return crs
}

func observeMigrationDowntime(node string, milliseconds uint64) error {
	histogram, err := migrationDowntime.GetMetricWithLabelValues(node)
	if err != nil {
		return err
	}
	histogram.Observe(domainstats.MillisecondsToSeconds(milliseconds))
	return nil
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
