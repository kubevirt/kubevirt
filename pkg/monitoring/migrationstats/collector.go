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

	"kubevirt.io/kubevirt/pkg/monitoring/scraper"

	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	PendingMigrations    = "kubevirt_migrate_vmi_pending_count"
	SchedulingMigrations = "kubevirt_migrate_vmi_scheduling_count"
	RunningMigrations    = "kubevirt_migrate_vmi_running_count"
	SucceededMigrations  = "kubevirt_migrate_vmi_succeeded_total"
	FailedMigrations     = "kubevirt_migrate_vmi_failed_total"
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
		SucceededMigrations: prometheus.NewDesc(
			SucceededMigrations,
			"Number of migrations successfully executed.",
			[]string{"vmi", "vmim"},
			nil,
		),
		FailedMigrations: prometheus.NewDesc(
			FailedMigrations,
			"Number of failed migrations.",
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

	scraper := VmimPrometheusScraper(ch, vmims)
	scraper.Scrape()
}

func VmimPrometheusScraper(ch chan<- prometheus.Metric, vmims []*k6tv1.VirtualMachineInstanceMigration) *vmimPrometheusScraper {
	return &vmimPrometheusScraper{
		PrometheusScraper: &scraper.PrometheusScraper{
			Ch: ch,
		},
		vmims: vmims,
	}
}

type vmimPrometheusScraper struct {
	*scraper.PrometheusScraper
	vmims []*k6tv1.VirtualMachineInstanceMigration
}

func (ps *vmimPrometheusScraper) Scrape() {
	pendingCount := 0
	schedulingCount := 0
	runningCount := 0

	for _, vmim := range ps.vmims {
		switch vmim.Status.Phase {
		case k6tv1.MigrationPending:
			pendingCount++
		case k6tv1.MigrationScheduling:
			schedulingCount++
		case k6tv1.MigrationRunning, k6tv1.MigrationScheduled, k6tv1.MigrationPreparingTarget, k6tv1.MigrationTargetReady:
			runningCount++
		case k6tv1.MigrationSucceeded:
			ps.PushConstMetric(migrationMetrics[SucceededMigrations], prometheus.GaugeValue, 1, vmim.Spec.VMIName, vmim.Name)
		default:
			ps.PushConstMetric(migrationMetrics[FailedMigrations], prometheus.GaugeValue, 1, vmim.Spec.VMIName, vmim.Name)
		}
	}

	ps.PushConstMetric(migrationMetrics[PendingMigrations], prometheus.GaugeValue, float64(pendingCount))
	ps.PushConstMetric(migrationMetrics[SchedulingMigrations], prometheus.GaugeValue, float64(schedulingCount))
	ps.PushConstMetric(migrationMetrics[RunningMigrations], prometheus.GaugeValue, float64(runningCount))
}
