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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Migration Stats Collector", func() {
	getVMIM := func(phase k6tv1.VirtualMachineInstanceMigrationPhase) *k6tv1.VirtualMachineInstanceMigration {
		return &k6tv1.VirtualMachineInstanceMigration{
			Status: k6tv1.VirtualMachineInstanceMigrationStatus{
				Phase: phase,
			},
		}
	}

	It("should set all metrics to 0 when no migrations", func() {
		var vmims []*k6tv1.VirtualMachineInstanceMigration

		cr := reportMigrationStats(vmims)

		for _, result := range cr {
			Expect(result.Value).Should(BeZero())
		}
	})

	DescribeTable("should set correct metric for each phase", func(phase k6tv1.VirtualMachineInstanceMigrationPhase, metric operatormetrics.Metric) {
		vmims := []*k6tv1.VirtualMachineInstanceMigration{
			getVMIM(phase),
		}

		cr := reportMigrationStats(vmims)

		containsMetric := false

		for _, result := range cr {
			if strings.Contains(result.Metric.GetOpts().Name, metric.GetOpts().Name) {
				containsMetric = true
				Expect(result.Value).To(Equal(1.0))
			} else {
				Expect(result.Value).To(BeZero())
			}
		}

		Expect(containsMetric).To(BeTrue())
	},
		Entry("Failed migration", k6tv1.MigrationFailed, failedMigration),
		Entry("Pending migration", k6tv1.MigrationPending, pendingMigrations),
		Entry("Running migration", k6tv1.MigrationRunning, runningMigrations),
		Entry("Scheduling migration", k6tv1.MigrationScheduling, schedulingMigrations),
		Entry("Succeeded migration", k6tv1.MigrationSucceeded, succeededMigration),
		Entry("Undefined migration", k6tv1.MigrationPhaseUnset, unsetMigration),
	)

	It("should set succeeded and pending to 1 and others to 0 with 1 successful and 1 pending", func() {
		vmims := []*k6tv1.VirtualMachineInstanceMigration{
			getVMIM(k6tv1.MigrationSucceeded),
			getVMIM(k6tv1.MigrationPending),
		}

		cr := reportMigrationStats(vmims)

		for _, result := range cr {
			if strings.Contains(result.Metric.GetOpts().Name, "succeeded") {
				Expect(result.Value).To(Equal(1.0))
			} else if strings.Contains(result.Metric.GetOpts().Name, "pending") {
				Expect(result.Value).To(Equal(1.0))
			} else {
				Expect(result.Value).To(BeZero())
			}
		}
	})
})
