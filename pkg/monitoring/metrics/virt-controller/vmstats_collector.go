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
	"k8s.io/apimachinery/pkg/api/resource"
	"kubevirt.io/client-go/log"

	k6tv1 "kubevirt.io/api/core/v1"
)

var (
	vmStatsCollector = operatormetrics.Collector{
		Metrics:         append(timestampMetrics, vmResourceRequests),
		CollectCallback: vmStatsCollectorCallback,
	}

	vmResourceRequests = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_resource_requests",
			Help: "Resources requested by Virtual Machine. Reports memory and CPU requests.",
		},
		[]string{"name", "namespace", "resource", "unit", "source"},
	)

	vmResourceLimits = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_resource_limits",
			Help: "Resources limits by Virtual Machine. Reports memory and CPU limits.",
		},
		[]string{"name", "namespace", "resource", "unit"},
	)

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
		cr = append(cr, ReportVmStats(vm)...)
	}

	cr = append(cr, CollectResourceRequestsAndLimits(vms)...)

	return cr
}

func CollectResourceRequestsAndLimits(vms []*k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult

	for _, vm := range vms {
		// Apply any instance type and preference to a copy of the VM before proceeding
		vmCopy := vm.DeepCopy()
		_ = instancetypeMethods.ApplyToVM(vmCopy)

		// Memory requests and limits from domain resources
		results = append(results, collectMemoryResourceRequestsFromDomainResources(vmCopy)...)
		results = append(results, collectMemoryResourceLimitsFromDomainResources(vmCopy)...)

		// CPU requests from domain CPU
		results = append(results, collectCpuResourceRequestsFromDomainCpu(vmCopy)...)

		// CPU requests and limits from domain resources
		results = append(results, collectCpuResourceRequestsFromDomainResources(vmCopy)...)
		results = append(results, collectCpuResourceLimitsFromDomainResources(vmCopy)...)
	}

	return results
}

func collectMemoryResourceRequestsFromDomainResources(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	if vm.Spec.Template == nil {
		return cr
	}

	memoryRequested := vm.Spec.Template.Spec.Domain.Resources.Requests.Memory()
	if !memoryRequested.IsZero() {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(memoryRequested.Value()),
			Labels: []string{vm.Name, vm.Namespace, "memory", "bytes", "domain"},
		})
	}

	if vm.Spec.Template.Spec.Domain.Memory == nil {
		return cr
	}

	guestMemory := vm.Spec.Template.Spec.Domain.Memory.Guest
	if guestMemory != nil && !guestMemory.IsZero() {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(guestMemory.Value()),
			Labels: []string{vm.Name, vm.Namespace, "memory", "bytes", "guest"},
		})
	}

	hugepagesMemory := vm.Spec.Template.Spec.Domain.Memory.Hugepages
	if hugepagesMemory != nil {
		quantity, err := resource.ParseQuantity(hugepagesMemory.PageSize)
		if err == nil {
			cr = append(cr, operatormetrics.CollectorResult{
				Metric: vmResourceRequests,
				Value:  float64(quantity.Value()),
				Labels: []string{vm.Name, vm.Namespace, "memory", "bytes", "hugepages"},
			})
		}
	}

	return cr
}

func collectMemoryResourceLimitsFromDomainResources(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	if vm.Spec.Template == nil {
		return []operatormetrics.CollectorResult{}
	}

	memoryLimit := vm.Spec.Template.Spec.Domain.Resources.Limits.Memory()
	if memoryLimit.IsZero() {
		return []operatormetrics.CollectorResult{}
	}

	return []operatormetrics.CollectorResult{{
		Metric: vmResourceLimits,
		Value:  float64(memoryLimit.Value()),
		Labels: []string{vm.Name, vm.Namespace, "memory", "bytes"},
	}}
}

func collectCpuResourceRequestsFromDomainCpu(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	if vm.Spec.Template.Spec.Domain.CPU == nil {
		return cr
	}

	if vm.Spec.Template.Spec.Domain.CPU.Cores != 0 {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(vm.Spec.Template.Spec.Domain.CPU.Cores),
			Labels: []string{vm.Name, vm.Namespace, "cpu", "cores", "domain"},
		})
	}

	if vm.Spec.Template.Spec.Domain.CPU.Threads != 0 {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(vm.Spec.Template.Spec.Domain.CPU.Threads),
			Labels: []string{vm.Name, vm.Namespace, "cpu", "threads", "domain"},
		})
	}

	if vm.Spec.Template.Spec.Domain.CPU.Sockets != 0 {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(vm.Spec.Template.Spec.Domain.CPU.Sockets),
			Labels: []string{vm.Name, vm.Namespace, "cpu", "sockets", "domain"},
		})
	}

	return cr
}

func collectCpuResourceRequestsFromDomainResources(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	cpuRequests := vm.Spec.Template.Spec.Domain.Resources.Requests.Cpu()

	if cpuRequests == nil || cpuRequests.IsZero() {
		// If no CPU requests and no Domain CPU are set, the VMI will default to 1 thread with 1 core and 1 socket
		if vm.Spec.Template.Spec.Domain.CPU == nil {
			return append(cr,
				operatormetrics.CollectorResult{
					Metric: vmResourceRequests,
					Value:  1.0,
					Labels: []string{vm.Name, vm.Namespace, "cpu", "cores", "default"},
				},
				operatormetrics.CollectorResult{
					Metric: vmResourceRequests,
					Value:  1.0,
					Labels: []string{vm.Name, vm.Namespace, "cpu", "threads", "default"},
				},
				operatormetrics.CollectorResult{
					Metric: vmResourceRequests,
					Value:  1.0,
					Labels: []string{vm.Name, vm.Namespace, "cpu", "sockets", "default"},
				},
			)
		}

		return cr
	}

	cr = append(cr, operatormetrics.CollectorResult{
		Metric: vmResourceRequests,
		Value:  float64(cpuRequests.ScaledValue(resource.Milli)) / 1000,
		Labels: []string{vm.Name, vm.Namespace, "cpu", "cores", "requests"},
	})

	return cr
}

func collectCpuResourceLimitsFromDomainResources(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	cpuLimits := vm.Spec.Template.Spec.Domain.Resources.Limits.Cpu()

	if cpuLimits == nil || cpuLimits.IsZero() {
		return cr
	}

	cr = append(cr, operatormetrics.CollectorResult{
		Metric: vmResourceLimits,
		Value:  float64(cpuLimits.ScaledValue(resource.Milli)) / 1000,
		Labels: []string{vm.Name, vm.Namespace, "cpu", "cores"},
	})

	return cr
}

func ReportVmStats(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
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
