package domainstats

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

var (
	migrateVMIDataRemaining = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_data_remaining_bytes",
			Help: "Number of bytes that still need to be transferred",
		},
	)

	migrateVMIDataProcessed = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_data_processed_bytes",
			Help: "Number of bytes transferred from the beginning of the job.",
		},
	)

	migrateVmiDirtyMemoryRate = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_dirty_memory_rate_bytes",
			Help: "Number of memory pages dirtied by the guest per second.",
		},
	)

	migrateVmiMemoryTransferRate = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_disk_transfer_rate_bytes",
			Help: "Network throughput used while migrating memory in Bytes per second.",
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
