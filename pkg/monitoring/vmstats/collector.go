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
	corev1 "k8s.io/api/core/v1"
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

	pvInfoMetric = "kubevirt_vm_persistentvolume_info"
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

		pvInfoMetric: prometheus.NewDesc(
			pvInfoMetric,
			"Virtual Machine PV info.",
			[]string{"name", "namespace", "volumename", "volumeMode", "volumeAttributes"},
			nil,
		),
	}

	labels = []string{"name", "namespace"}
)

type VMCollector struct {
	vmInformer  cache.SharedIndexInformer
	pvcInformer cache.SharedIndexInformer
	pvInformer  cache.SharedIndexInformer
}

func (co *VMCollector) Describe(_ chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

func SetupVMCollector(
	vmInformer cache.SharedIndexInformer,
	pvcInformer cache.SharedIndexInformer,
	pvInformer cache.SharedIndexInformer,
) *VMCollector {

	log.Log.Infof("Starting vm collector")
	co := &VMCollector{
		vmInformer:  vmInformer,
		pvcInformer: pvcInformer,
		pvInformer:  pvInformer,
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
	scraper.Report(co, vms)
}

func NewPrometheusScraper(ch chan<- prometheus.Metric) *prometheusScraper {
	return &prometheusScraper{ch: ch}
}

type prometheusScraper struct {
	ch chan<- prometheus.Metric
}

func (ps *prometheusScraper) Report(co *VMCollector, vms []*k6tv1.VirtualMachine) {
	for _, vm := range vms {
		ps.updateVMStatusMetrics(vm)
		ps.updatePVInfoMetrics(co, vm)
	}
}

func (ps *prometheusScraper) updatePVInfoMetrics(co *VMCollector, vm *k6tv1.VirtualMachine) {
	if vm.Spec.Template.Spec.Volumes == nil {
		return
	}

	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.PersistentVolumeClaim == nil {
			continue
		}
		log.DefaultLogger().V(4).Infof("VM %s/%s has PVC: '%s'", vm.Namespace, vm.Name, volume.PersistentVolumeClaim.ClaimName)

		pvc, err := getPVC(co, vm, volume.PersistentVolumeClaim.ClaimName)
		if err != nil {
			continue
		}
		pv, err := getPV(co, vm, pvc)
		if err != nil {
			continue
		}

		ps.pushMetric(metrics[pvInfoMetric], prometheus.GaugeValue, 1,
			vm.Name, vm.Namespace, pv.Name, string(*pvc.Spec.VolumeMode), volumeAttributesToString(pv),
		)
	}
}

func getPVC(co *VMCollector, vm *k6tv1.VirtualMachine, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc, exists, err := co.pvcInformer.GetIndexer().GetByKey(vm.Namespace + "/" + name)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to get PVC '%s' for VM %s/%s", name, vm.Namespace, vm.Name)
		return nil, err
	}
	if !exists {
		log.DefaultLogger().Reason(err).Errorf("PVC '%s' for VM %s/%s does not exist", name, vm.Namespace, vm.Name)
		return nil, err
	}

	return pvc.(*corev1.PersistentVolumeClaim), nil
}

func getPV(co *VMCollector, vm *k6tv1.VirtualMachine, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolume, error) {
	pv, exists, err := co.pvInformer.GetIndexer().GetByKey(pvc.Spec.VolumeName)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to get PV '%s' for VM %s/%s", pvc.Spec.VolumeName, vm.Namespace, vm.Name)
		return nil, err
	}
	if !exists {
		log.DefaultLogger().Reason(err).Errorf("PV '%s' for VM %s/%s does not exist", pvc.Spec.VolumeName, vm.Namespace, vm.Name)
		return nil, err
	}

	return pv.(*corev1.PersistentVolume), nil
}

func volumeAttributesToString(pv *corev1.PersistentVolume) string {
	if pv.Spec.CSI == nil {
		return ""
	}

	var volumeAttributesString string
	for key, value := range pv.Spec.CSI.VolumeAttributes {
		volumeAttributesString += key + "=" + value + ";"
	}
	return volumeAttributesString
}

func (ps *prometheusScraper) updateVMStatusMetrics(vm *k6tv1.VirtualMachine) {
	status := vm.Status.PrintableStatus
	currentStateMetric := getMetricDesc(status)

	lastTransitionTime := getLastConditionDetails(vm)

	statusMetricsNames := []string{startingMetric, runningMetric, migratingMetric, nonRunningMetric, errorMetric}
	for _, statusMetricsName := range statusMetricsNames {
		if metrics[statusMetricsName] == currentStateMetric {
			ps.pushMetric(currentStateMetric, prometheus.CounterValue, float64(lastTransitionTime), vm.Name, vm.Namespace)
		} else {
			ps.pushMetric(metrics[statusMetricsName], prometheus.CounterValue, 0, vm.Name, vm.Namespace)
		}
	}
}

func (ps *prometheusScraper) pushMetric(desc *prometheus.Desc, metricType prometheus.ValueType, value float64, labelValues ...string) {
	mv, err := prometheus.NewConstMetric(desc, metricType, value, labelValues...)
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
