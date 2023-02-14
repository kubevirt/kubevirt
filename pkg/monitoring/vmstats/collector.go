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

package vmstats

import (
	"k8s.io/client-go/tools/cache"

	k6tv1 "kubevirt.io/api/core/v1"

	"github.com/prometheus/client_golang/prometheus"

	"kubevirt.io/client-go/log"
)

const (
	startingMetric   = "kubevirt_vm_starting_status_last_transition_timestamp_seconds"
	runningMetric    = "kubevirt_vm_running_status_last_transition_timestamp_seconds"
	migratingMetric  = "kubevirt_vm_migrating_status_last_transition_timestamp_seconds"
	nonRunningMetric = "kubevirt_vm_non_running_status_last_transition_timestamp_seconds"
	errorMetric      = "kubevirt_vm_error_status_last_transition_timestamp_seconds"
)

var (
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

	metrics = map[string]*prometheus.Desc{
		startingMetric: prometheus.NewDesc(
			startingMetric,
			"Virtual Machine last transition timestamp to starting status.",
			labels,
			nil,
		),
		runningMetric: prometheus.NewDesc(
			runningMetric,
			"Virtual Machine last transition timestamp to running status.",
			labels,
			nil,
		),
		migratingMetric: prometheus.NewDesc(
			migratingMetric,
			"Virtual Machine last transition timestamp to migrating status.",
			labels,
			nil,
		),
		nonRunningMetric: prometheus.NewDesc(
			nonRunningMetric,
			"Virtual Machine last transition timestamp to paused/stopped status.",
			labels,
			nil,
		),
		errorMetric: prometheus.NewDesc(
			errorMetric,
			"Virtual Machine last transition timestamp to error status.",
			labels,
			nil,
		),
	}

	labels = []string{"name", "namespace"}
)

type VMCollector struct {
	vmInformer cache.SharedIndexInformer
}

func (co *VMCollector) Describe(_ chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

func SetupVMCollector(vmInformer cache.SharedIndexInformer) *VMCollector {
	log.Log.Infof("Starting vm collector")
	co := &VMCollector{
		vmInformer: vmInformer,
	}

	prometheus.MustRegister(co)
	return co
}

func (co *VMCollector) Collect(ch chan<- prometheus.Metric) {
	cachedObjs := co.vmInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		return
	}

	vms := make([]*k6tv1.VirtualMachine, len(cachedObjs))
	for i, obj := range cachedObjs {
		vms[i] = obj.(*k6tv1.VirtualMachine)
	}

	scraper := NewPrometheusScraper(ch)
	scraper.Report(vms)
}

func NewPrometheusScraper(ch chan<- prometheus.Metric) *prometheusScraper {
	return &prometheusScraper{ch: ch}
}

type prometheusScraper struct {
	ch chan<- prometheus.Metric
}

func (ps *prometheusScraper) Report(vms []*k6tv1.VirtualMachine) {
	for _, vm := range vms {
		ps.updateVMMetrics(vm)
	}
}

func (ps *prometheusScraper) updateVMMetrics(vm *k6tv1.VirtualMachine) {
	status := vm.Status.PrintableStatus
	currentStateMetric := getMetricDesc(status)

	lastTransitionTime := getLastConditionDetails(vm)

	for _, metric := range metrics {
		if metric == currentStateMetric {
			ps.pushMetric(currentStateMetric, float64(lastTransitionTime), vm.Name, vm.Namespace)
		} else {
			ps.pushMetric(metric, 0, vm.Name, vm.Namespace)
		}
	}
}

func (ps *prometheusScraper) pushMetric(desc *prometheus.Desc, value float64, labelValues ...string) {
	mv, err := prometheus.NewConstMetric(
		desc,
		prometheus.CounterValue,
		value,
		labelValues...,
	)
	if err != nil {
		log.Log.Warningf("Error creating the new const metric for %s: %s", desc, err)
		return
	}
	ps.ch <- mv
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

func getMetricDesc(status k6tv1.VirtualMachinePrintableStatus) *prometheus.Desc {
	switch {
	case containsStatus(status, startingStatuses):
		return metrics[startingMetric]
	case containsStatus(status, runningStatuses):
		return metrics[runningMetric]
	case containsStatus(status, migratingStatuses):
		return metrics[migratingMetric]
	case containsStatus(status, nonRunningStatuses):
		return metrics[nonRunningMetric]
	case containsStatus(status, errorStatuses):
		return metrics[errorMetric]
	}

	return metrics[errorMetric]
}

func containsStatus(target k6tv1.VirtualMachinePrintableStatus, elems []k6tv1.VirtualMachinePrintableStatus) bool {
	for _, elem := range elems {
		if elem == target {
			return true
		}
	}
	return false
}
