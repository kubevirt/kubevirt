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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package virt_controller

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"kubevirt.io/client-go/log"

	k6tv1 "kubevirt.io/api/core/v1"
)

var (
	vmStatsCollector = operatormetrics.Collector{
		Metrics:         timestampMetrics,
		CollectCallback: vmStatsCollectorCallback,
	}

	timestampMetrics = []operatormetrics.Metric{
		startingTimestamp,
		runningTimestamp,
		migratingTimestamp,
		nonRunningTimestamp,
		errorTimestamp,
	}

	startingTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_starting_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to starting status.",
		},
		labels,
	)

	runningTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_running_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to running status.",
		},
		labels,
	)

	migratingTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_migrating_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to migrating status.",
		},
		labels,
	)

	nonRunningTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_non_running_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to paused/stopped status.",
		},
		labels,
	)

	errorTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_error_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to error status.",
		},
		labels,
	)

	labels = []string{"name", "namespace"}

	startingStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusProvisioning,
		k6tv1.VirtualMachineStatusStarting,
		k6tv1.VirtualMachineStatusWaitingForVolumeBinding,
	}

	runningStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusRunning,
	}

	migratingStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusMigrating,
	}

	nonRunningStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusStopped,
		k6tv1.VirtualMachineStatusPaused,
		k6tv1.VirtualMachineStatusStopping,
		k6tv1.VirtualMachineStatusTerminating,
	}

	errorStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusCrashLoopBackOff,
		k6tv1.VirtualMachineStatusUnknown,
		k6tv1.VirtualMachineStatusUnschedulable,
		k6tv1.VirtualMachineStatusErrImagePull,
		k6tv1.VirtualMachineStatusImagePullBackOff,
		k6tv1.VirtualMachineStatusPvcNotFound,
		k6tv1.VirtualMachineStatusDataVolumeError,
	}
)

func vmStatsCollectorCallback() []operatormetrics.CollectorResult {
	cachedObjs := vmInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		log.Log.V(4).Infof("No VMs detected")
		return []operatormetrics.CollectorResult{}
	}

	vms := make([]*k6tv1.VirtualMachine, len(cachedObjs))

	for i, obj := range cachedObjs {
		vms[i] = obj.(*k6tv1.VirtualMachine)
	}

	return reportVmsStats(vms)
}

func reportVmsStats(vms []*k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	for _, vm := range vms {
		cr = append(cr, reportVmStats(vm)...)
	}

	return cr
}

func reportVmStats(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	status := vm.Status.PrintableStatus
	currentStateMetric := getMetricDesc(status)

	lastTransitionTime := getLastConditionDetails(vm)

	for _, metric := range timestampMetrics {
		value := float64(0)
		if metric == currentStateMetric {
			value = float64(lastTransitionTime)
		}

		cr = append(cr, operatormetrics.CollectorResult{
			Metric: metric,
			Labels: []string{vm.Name, vm.Namespace},
			Value:  value,
		})
	}

	return cr
}

func getMetricDesc(status k6tv1.VirtualMachinePrintableStatus) *operatormetrics.CounterVec {
	switch {
	case containsStatus(status, startingStatuses):
		return startingTimestamp
	case containsStatus(status, runningStatuses):
		return runningTimestamp
	case containsStatus(status, migratingStatuses):
		return migratingTimestamp
	case containsStatus(status, nonRunningStatuses):
		return nonRunningTimestamp
	case containsStatus(status, errorStatuses):
		return errorTimestamp
	}

	return errorTimestamp
}

func containsStatus(target k6tv1.VirtualMachinePrintableStatus, elems []k6tv1.VirtualMachinePrintableStatus) bool {
	for _, elem := range elems {
		if elem == target {
			return true
		}
	}
	return false
}

func getLastConditionDetails(vm *k6tv1.VirtualMachine) int64 {
	conditions := []k6tv1.VirtualMachineConditionType{
		k6tv1.VirtualMachineReady,
		k6tv1.VirtualMachineFailure,
		k6tv1.VirtualMachinePaused,
	}

	latestTransitionTime := int64(-1)

	for _, c := range vm.Status.Conditions {
		if containsCondition(c.Type, conditions) && c.LastTransitionTime.Unix() > latestTransitionTime {
			latestTransitionTime = c.LastTransitionTime.Unix()
		}
	}

	return latestTransitionTime
}

func containsCondition(target k6tv1.VirtualMachineConditionType, elems []k6tv1.VirtualMachineConditionType) bool {
	for _, elem := range elems {
		if elem == target {
			return true
		}
	}
	return false
}
