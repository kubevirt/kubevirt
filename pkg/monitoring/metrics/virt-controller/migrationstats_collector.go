/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtcontroller

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
)

var (
	migrationStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			pendingMigrations,
			schedulingMigrations,
			unsetMigration,
			runningMigrations,
			succeededMigration,
			failedMigration,
		},
		CollectCallback: migrationStatsCollectorCallback,
	}

	pendingMigrations = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_pending_phase",
			Help: "Number of current pending migrations.",
		},
	)

	schedulingMigrations = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_scheduling_phase",
			Help: "Number of current scheduling migrations.",
		},
	)

	unsetMigration = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_unset_phase",
			Help: "Number of current unset migrations. These are pending items the virt-controller hasn’t processed yet from the queue.",
		},
	)

	runningMigrations = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_running_phase",
			Help: "Number of current running migrations.",
		},
	)

	succeededMigration = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_succeeded",
			Help: "Indicates if the VMI migration succeeded.",
		},
		[]string{"vmi", "vmim", "namespace"},
	)

	failedMigration = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_failed",
			Help: "Indicates if the VMI migration failed.",
		},
		[]string{"vmi", "vmim", "namespace"},
	)
)

func migrationStatsCollectorCallback() []operatormetrics.CollectorResult {
	cachedObjs := indexers.VMIMigration.List()
	vmims := make([]*k6tv1.VirtualMachineInstanceMigration, len(cachedObjs))
	for i, obj := range cachedObjs {
		vmims[i] = obj.(*k6tv1.VirtualMachineInstanceMigration)
	}

	return reportMigrationStats(vmims)
}

func reportMigrationStats(vmims []*k6tv1.VirtualMachineInstanceMigration) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	pendingCount := 0
	schedulingCount := 0
	unsetCount := 0
	runningCount := 0

	for _, vmim := range vmims {
		switch vmim.Status.Phase {
		case k6tv1.MigrationPending:
			pendingCount++
		case k6tv1.MigrationScheduling:
			schedulingCount++
		case k6tv1.MigrationPhaseUnset:
			unsetCount++
		case k6tv1.MigrationRunning, k6tv1.MigrationScheduled, k6tv1.MigrationPreparingTarget,
			k6tv1.MigrationTargetReady, k6tv1.MigrationWaitingForSync, k6tv1.MigrationSynchronizing:
			runningCount++
		case k6tv1.MigrationSucceeded:
			cr = append(cr, operatormetrics.CollectorResult{
				Metric: succeededMigration, Value: 1,
				Labels: []string{vmim.Spec.VMIName, vmim.Name, vmim.Namespace},
			})
		case k6tv1.MigrationFailed:
			cr = append(cr, operatormetrics.CollectorResult{
				Metric: failedMigration, Value: 1,
				Labels: []string{vmim.Spec.VMIName, vmim.Name, vmim.Namespace},
			})
		}
	}

	return append(cr,
		operatormetrics.CollectorResult{Metric: pendingMigrations, Value: float64(pendingCount)},
		operatormetrics.CollectorResult{Metric: schedulingMigrations, Value: float64(schedulingCount)},
		operatormetrics.CollectorResult{Metric: unsetMigration, Value: float64(unsetCount)},
		operatormetrics.CollectorResult{Metric: runningMigrations, Value: float64(runningCount)},
	)
}
