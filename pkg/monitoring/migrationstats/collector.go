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

package migrationstats

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/cache"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	PendingMigrations    = "kubevirt_vmi_migrations_in_pending_phase"
	SchedulingMigrations = "kubevirt_vmi_migrations_in_scheduling_phase"
	RunningMigrations    = "kubevirt_vmi_migrations_in_running_phase"
	SucceededMigration   = "kubevirt_vmi_migration_succeeded"
	FailedMigration      = "kubevirt_vmi_migration_failed"
)

var (
	migrationMetrics = map[string]*prometheus.Desc{
		PendingMigrations: prometheus.NewDesc(
			PendingMigrations,
			"Number of current pending migrations.",
			nil,
			nil,
		),
		SchedulingMigrations: prometheus.NewDesc(
			SchedulingMigrations,
			"Number of current scheduling migrations.",
			nil,
			nil,
		),
		RunningMigrations: prometheus.NewDesc(
			RunningMigrations,
			"Number of current running migrations.",
			nil,
			nil,
		),
		SucceededMigration: prometheus.NewDesc(
			SucceededMigration,
			"Indicates if the VMI migration succeeded.",
			[]string{"vmi", "vmim"},
			nil,
		),
		FailedMigration: prometheus.NewDesc(
			FailedMigration,
			"Indicates if the VMI migration failed.",
			[]string{"vmi", "vmim"},
			nil,
		),
	}
)

type MigrationCollector struct {
	migrationInformer cache.SharedIndexInformer
}

func (co *MigrationCollector) Describe(_ chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

func SetupMigrationsCollector(migrationInformer cache.SharedIndexInformer) *MigrationCollector {
	log.Log.Infof("Starting migration collector")
	co := &MigrationCollector{
		migrationInformer: migrationInformer,
	}

	prometheus.MustRegister(co)
	return co
}

func (co *MigrationCollector) Collect(ch chan<- prometheus.Metric) {
	cachedObjs := co.migrationInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		return
	}

	vmims := make([]*k6tv1.VirtualMachineInstanceMigration, len(cachedObjs))
	for i, obj := range cachedObjs {
		vmims[i] = obj.(*k6tv1.VirtualMachineInstanceMigration)
	}

	scraper := NewPrometheusScraper(ch)
	scraper.Report(vmims)
}

func NewPrometheusScraper(ch chan<- prometheus.Metric) *prometheusScraper {
	return &prometheusScraper{ch: ch}
}

type prometheusScraper struct {
	ch chan<- prometheus.Metric
}

func (ps *prometheusScraper) Report(vmims []*k6tv1.VirtualMachineInstanceMigration) {
	pendingCount := 0
	schedulingCount := 0
	runningCount := 0

	for _, vmim := range vmims {
		switch vmim.Status.Phase {
		case k6tv1.MigrationPending:
			pendingCount++
		case k6tv1.MigrationScheduling:
			schedulingCount++
		case k6tv1.MigrationRunning, k6tv1.MigrationScheduled, k6tv1.MigrationPreparingTarget, k6tv1.MigrationTargetReady:
			runningCount++
		case k6tv1.MigrationSucceeded:
			ps.pushMetric(migrationMetrics[SucceededMigration], 1, vmim.Spec.VMIName, vmim.Name)
		default:
			ps.pushMetric(migrationMetrics[FailedMigration], 1, vmim.Spec.VMIName, vmim.Name)
		}
	}

	ps.pushMetric(migrationMetrics[PendingMigrations], float64(pendingCount))
	ps.pushMetric(migrationMetrics[SchedulingMigrations], float64(schedulingCount))
	ps.pushMetric(migrationMetrics[RunningMigrations], float64(runningCount))
}

func (ps *prometheusScraper) pushMetric(desc *prometheus.Desc, value float64, labelValues ...string) {
	mv, err := prometheus.NewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		labelValues...,
	)
	if err != nil {
		log.Log.Warningf("Error creating the new const metric for %s: %s", desc, err)
		return
	}

	ps.ch <- mv
}
