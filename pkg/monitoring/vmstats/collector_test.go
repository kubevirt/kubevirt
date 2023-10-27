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

package vmstats

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("VM Stats Collector", func() {
	Context("VM status collector", func() {
		var ch chan prometheus.Metric
		var scrapper *prometheusScraper

		createVM := func(status k6tv1.VirtualMachinePrintableStatus, vmLastTransitionsTime time.Time) *k6tv1.VirtualMachine {
			return &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-vm"},
				Status: k6tv1.VirtualMachineStatus{
					PrintableStatus: status,
					Conditions: []k6tv1.VirtualMachineCondition{
						{
							Type:               k6tv1.VirtualMachineFailure,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime.Add(-20 * time.Second)),
						},
						{
							Type:               k6tv1.VirtualMachineReady,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime),
						},
						{
							Type:               k6tv1.VirtualMachinePaused,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime.Add(-40 * time.Second)),
						},
					},
				},
			}
		}

		BeforeEach(func() {
			ch = make(chan prometheus.Metric, 5)
			scrapper = &prometheusScraper{ch: ch}
		})

		DescribeTable("Add VM status metrics", func(status k6tv1.VirtualMachinePrintableStatus, metric string) {
			t := time.Now()
			vms := []*k6tv1.VirtualMachine{
				createVM(status, t),
			}

			scrapper.Report(vms)
			close(ch)

			containsStateMetric := false

			for m := range ch {
				dto := &io_prometheus_client.Metric{}
				m.Write(dto)

				if strings.Contains(m.Desc().String(), metric) {
					containsStateMetric = true
					Expect(*dto.Counter.Value).To(Equal(float64(t.Unix())))
				} else {
					Expect(*dto.Counter.Value).To(BeZero())
				}
			}

			Expect(containsStateMetric).To(BeTrue())
		},
			Entry("Starting VM", k6tv1.VirtualMachineStatusProvisioning, startingMetric),
			Entry("Running VM", k6tv1.VirtualMachineStatusRunning, runningMetric),
			Entry("Migrating VM", k6tv1.VirtualMachineStatusMigrating, migratingMetric),
			Entry("Non running VM", k6tv1.VirtualMachineStatusStopped, nonRunningMetric),
			Entry("Errored VM", k6tv1.VirtualMachineStatusCrashLoopBackOff, errorMetric),
		)
	})
})
