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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Migration Stats Collector", func() {
	var ch chan prometheus.Metric
	var scrapper *prometheusScraper

	BeforeEach(func() {
		ch = make(chan prometheus.Metric, 5)
		scrapper = &prometheusScraper{ch: ch}
	})

	getVMIM := func(phase k6tv1.VirtualMachineInstanceMigrationPhase) *k6tv1.VirtualMachineInstanceMigration {
		return &k6tv1.VirtualMachineInstanceMigration{
			Status: k6tv1.VirtualMachineInstanceMigrationStatus{
				Phase: phase,
			},
		}
	}

	It("should set all metrics to 0 when no migrations", func() {
		var vmims []*k6tv1.VirtualMachineInstanceMigration
		scrapper.Report(vmims)
		close(ch)

		for m := range ch {
			dto := &io_prometheus_client.Metric{}
			m.Write(dto)
			Expect(*dto.Gauge.Value).Should(BeZero())
		}
	})

	DescribeTable("should set correct metric for each phase", func(phase k6tv1.VirtualMachineInstanceMigrationPhase, metric string) {
		vmims := []*k6tv1.VirtualMachineInstanceMigration{
			getVMIM(phase),
		}

		scrapper.Report(vmims)
		close(ch)

		containsMetric := false

		for m := range ch {
			dto := &io_prometheus_client.Metric{}
			m.Write(dto)

			if strings.Contains(m.Desc().String(), metric) {
				containsMetric = true
				Expect(*dto.Gauge.Value).To(Equal(1.0))
			} else {
				Expect(*dto.Gauge.Value).To(BeZero())
			}
		}

		Expect(containsMetric).To(BeTrue())
	},
		Entry("Failed migration", k6tv1.MigrationFailed, FailedMigration),
		Entry("Pending migration", k6tv1.MigrationPending, PendingMigrations),
		Entry("Running migration", k6tv1.MigrationRunning, RunningMigrations),
		Entry("Scheduling migration", k6tv1.MigrationScheduling, SchedulingMigrations),
		Entry("Succeeded migration", k6tv1.MigrationSucceeded, SucceededMigration),
	)

	It("should set succeeded and pending to 1 and others to 0 with 1 successful and 1 pending", func() {
		vmims := []*k6tv1.VirtualMachineInstanceMigration{
			getVMIM(k6tv1.MigrationSucceeded),
			getVMIM(k6tv1.MigrationPending),
		}

		scrapper.Report(vmims)
		close(ch)

		for m := range ch {
			dto := &io_prometheus_client.Metric{}
			m.Write(dto)

			if strings.Contains(m.Desc().String(), "succeeded") {
				Expect(*dto.Gauge.Value).To(Equal(1.0))
			} else if strings.Contains(m.Desc().String(), "pending") {
				Expect(*dto.Gauge.Value).To(Equal(1.0))
			} else {
				Expect(*dto.Gauge.Value).To(BeZero())
			}
		}
	})
})
