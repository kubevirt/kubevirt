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
 * Copyright the KubeVirt Authors.
 *
 */

package main

import (
	"github.com/prometheus/client_golang/prometheus"

	"kubevirt.io/kubevirt/pkg/monitoring/migrationstats"

	k6tv1 "kubevirt.io/api/core/v1"
)

type fakeMigrationsCollector struct {
}

func (fc fakeMigrationsCollector) Describe(_ chan<- *prometheus.Desc) {
}

// Collect needs to report all metrics to see it in docs
func (fc fakeMigrationsCollector) Collect(ch chan<- prometheus.Metric) {
	ps := migrationstats.NewPrometheusScraper(ch)

	vmims := []*k6tv1.VirtualMachineInstanceMigration{
		{
			Status: k6tv1.VirtualMachineInstanceMigrationStatus{
				Phase: k6tv1.MigrationSucceeded,
			},
		},
		{
			Status: k6tv1.VirtualMachineInstanceMigrationStatus{
				Phase: k6tv1.MigrationFailed,
			},
		},
	}

	ps.Report(vmims)
}

func RegisterFakeMigrationsCollector() {
	prometheus.MustRegister(fakeMigrationsCollector{})
}
