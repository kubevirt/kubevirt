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

package virtcontroller

import (
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

const (
	migrationTriggerUser           = "user"
	migrationTriggerEvacuation     = "evacuation"
	migrationTriggerWorkloadUpdate = "workload_update"

	migrationResultSucceeded  = "succeeded"
	migrationResultFailed     = "failed"
	migrationResultInProgress = "in_progress"

	migrationReasonNone          = "none"
	migrationReasonTimeout       = "timeout"
	migrationReasonCanceled      = "canceled"
	migrationReasonUnschedulable = "unschedulable"
	migrationReasonFailed        = "failed"

	migrationPhaseUnset = "unset"
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
			migrationInfo,
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

	migrationInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_info",
			Help: "Information about VirtualMachineInstanceMigrations. Includes name (VMI name), " +
				"namespace, migration_name (VMIM name), source_node, target_node, phase (VMIM phase in lowercase), " +
				"trigger (user, evacuation, workload_update), result (succeeded, failed, in_progress), " +
				"and reason (none, timeout, canceled, unschedulable, failed).",
		},
		[]string{
			"namespace", "name", "migration_name",
			"source_node", "target_node",
			"phase", "trigger", "result", "reason",
		},
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
		cr = append(cr, collectMigrationInfo(vmim))

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

func collectMigrationInfo(vmim *k6tv1.VirtualMachineInstanceMigration) operatormetrics.CollectorResult {
	sourceNode := none
	targetNode := none
	if vmim.Status.MigrationState != nil {
		sourceNode = vmim.Status.MigrationState.SourceNode
		targetNode = vmim.Status.MigrationState.TargetNode
	}

	result := getMigrationResult(vmim.Status.Phase)

	return operatormetrics.CollectorResult{
		Metric: migrationInfo,
		Value:  1,
		Labels: []string{
			vmim.Namespace,
			vmim.Spec.VMIName,
			vmim.Name,
			sourceNode,
			targetNode,
			getMigrationPhaseLabel(vmim.Status.Phase),
			getMigrationTrigger(vmim),
			result,
			getMigrationReason(vmim, result),
		},
	}
}

func getMigrationPhaseLabel(phase k6tv1.VirtualMachineInstanceMigrationPhase) string {
	if phase == k6tv1.MigrationPhaseUnset {
		return migrationPhaseUnset
	}
	return strings.ToLower(string(phase))
}

func getMigrationResult(phase k6tv1.VirtualMachineInstanceMigrationPhase) string {
	switch phase {
	case k6tv1.MigrationSucceeded:
		return migrationResultSucceeded
	case k6tv1.MigrationFailed:
		return migrationResultFailed
	default:
		return migrationResultInProgress
	}
}

func getMigrationTrigger(vmim *k6tv1.VirtualMachineInstanceMigration) string {
	if metav1.HasAnnotation(vmim.ObjectMeta, k6tv1.EvacuationMigrationAnnotation) {
		return migrationTriggerEvacuation
	}
	if metav1.HasAnnotation(vmim.ObjectMeta, k6tv1.WorkloadUpdateMigrationAnnotation) {
		return migrationTriggerWorkloadUpdate
	}
	return migrationTriggerUser
}

func getMigrationReason(vmim *k6tv1.VirtualMachineInstanceMigration, result string) string {
	if result != migrationResultFailed {
		return migrationReasonNone
	}
	if migrationWasCanceled(vmim) {
		return migrationReasonCanceled
	}
	if migrationWasUnschedulable(vmim) {
		return migrationReasonUnschedulable
	}
	if migrationTimedOut(vmim) {
		return migrationReasonTimeout
	}
	return migrationReasonFailed
}

func migrationWasCanceled(vmim *k6tv1.VirtualMachineInstanceMigration) bool {
	if migrationHasCondition(vmim, k6tv1.VirtualMachineInstanceMigrationAbortRequested) {
		return true
	}
	if state := vmim.Status.MigrationState; state != nil {
		if state.AbortRequested ||
			state.AbortStatus == k6tv1.MigrationAbortSucceeded ||
			state.AbortStatus == k6tv1.MigrationAbortInProgress {
			return true
		}
	}
	return strings.Contains(migrationFailureReason(vmim), "abort")
}

func migrationWasUnschedulable(vmim *k6tv1.VirtualMachineInstanceMigration) bool {
	if migrationHasCondition(vmim, k6tv1.VirtualMachineInstanceMigrationRejectedByResourceQuota) {
		return true
	}
	return strings.Contains(migrationFailureReason(vmim), "unschedulable")
}

func migrationTimedOut(vmim *k6tv1.VirtualMachineInstanceMigration) bool {
	return strings.Contains(migrationFailureReason(vmim), "timeout")
}

func migrationFailureReason(vmim *k6tv1.VirtualMachineInstanceMigration) string {
	if vmim.Status.MigrationState == nil {
		return ""
	}
	return strings.ToLower(vmim.Status.MigrationState.FailureReason)
}

func migrationHasCondition(vmim *k6tv1.VirtualMachineInstanceMigration, condType k6tv1.VirtualMachineInstanceMigrationConditionType) bool {
	for _, condition := range vmim.Status.Conditions {
		if condition.Type == condType && condition.Status == k8sv1.ConditionTrue {
			return true
		}
	}
	return false
}
