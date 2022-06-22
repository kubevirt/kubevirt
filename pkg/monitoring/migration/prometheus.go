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

package migration

import (
	"github.com/prometheus/client_golang/prometheus"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

type gaugeAction bool

const increase gaugeAction = true
const decrease gaugeAction = false

var (
	migrationsLabels = []string{"vmi", "source", "target"}

	CurrentPendingMigrations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_migrate_vmi_pending_count",
			Help: "Number of current pending migrations.",
		},
		migrationsLabels,
	)

	CurrentSchedulingMigrations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_migrate_vmi_scheduling_count",
			Help: "Number of current scheduling migrations.",
		},
		migrationsLabels,
	)

	CurrentRunningMigrations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_migrate_vmi_running_count",
			Help: "Number of current running migrations.",
		},
		migrationsLabels,
	)

	MigrationsSucceededTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubevirt_migrate_vmi_succeeded_total",
			Help: "Number of migrations successfully executed.",
		},
		migrationsLabels,
	)

	MigrationsFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubevirt_migrate_vmi_failed_total",
			Help: "Number of failed migrations.",
		},
		migrationsLabels,
	)
)

func RegisterMigrationMetrics(vmiMigrationInformer cache.SharedIndexInformer) {
	log.Log.Infof("Starting migration's counter metrics")
	prometheus.MustRegister(CurrentPendingMigrations)
	prometheus.MustRegister(CurrentSchedulingMigrations)
	prometheus.MustRegister(CurrentRunningMigrations)
	prometheus.MustRegister(MigrationsSucceededTotal)
	prometheus.MustRegister(MigrationsFailedTotal)
	log.Log.Infof("Starting migration's performance and scale metrics")
	prometheus.MustRegister(newVMIMigrationPhaseTransitionTimeFromCreationHistogramVec(vmiMigrationInformer))
}

func IncPendingMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	updateMigrationGauge(vmi, targetPod, CurrentPendingMigrations, increase)
}

func IncSchedulingMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	updateMigrationGauge(vmi, targetPod, CurrentSchedulingMigrations, increase)
}

func IncRunningMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	updateMigrationGauge(vmi, targetPod, CurrentRunningMigrations, increase)
}

func DecPendingMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	updateMigrationGauge(vmi, targetPod, CurrentPendingMigrations, decrease)
}

func DecSchedulingMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	updateMigrationGauge(vmi, targetPod, CurrentSchedulingMigrations, decrease)
}

func DecRunningMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	updateMigrationGauge(vmi, targetPod, CurrentRunningMigrations, decrease)
}

func IncSucceededMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	incMigrationCounter(vmi, targetPod, MigrationsSucceededTotal)
}

func IncFailedMigrations(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) {
	incMigrationCounter(vmi, targetPod, MigrationsFailedTotal)
}

func getMigrationSourceAndTarget(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod) (source, target string) {
	source = vmi.Status.NodeName
	if targetPod != nil {
		target = targetPod.Spec.NodeName
	}

	if vmi.Status.MigrationState != nil {
		source = vmi.Status.MigrationState.SourceNode
		target = vmi.Status.MigrationState.TargetNode
	}

	return
}

func updateMigrationGauge(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod, gauge *prometheus.GaugeVec, action gaugeAction) {
	source, target := getMigrationSourceAndTarget(vmi, targetPod)
	labelValues := []string{vmi.Name, source, target}

	if action == increase {
		gauge.WithLabelValues(labelValues...).Inc()
	} else {
		gauge.WithLabelValues(labelValues...).Dec()
	}
}

func incMigrationCounter(vmi *virtv1.VirtualMachineInstance, targetPod *k8sv1.Pod, counter *prometheus.CounterVec) {
	source, target := getMigrationSourceAndTarget(vmi, targetPod)
	labelValues := []string{vmi.Name, source, target}

	counter.WithLabelValues(labelValues...).Inc()
}
