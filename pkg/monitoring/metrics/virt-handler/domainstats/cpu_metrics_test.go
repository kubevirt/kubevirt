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
 * Copyright The KubeVirt Authors.
 *
 */

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("cpu metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DomainStats: &stats.DomainStats{
				Cpu: &stats.DomainStatsCPU{
					TimeSet:   true,
					Time:      1,
					UserSet:   true,
					User:      2,
					SystemSet: true,
					System:    3,
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := cpuMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_cpu_usage_seconds_total", cpuUsageSeconds, nanosecondsToSeconds(1)),
			Entry("kubevirt_vmi_cpu_user_usage_seconds_total", cpuUserUsageSeconds, nanosecondsToSeconds(2)),
			Entry("kubevirt_vmi_cpu_system_usage_seconds_total", cpuSystemUsageSeconds, nanosecondsToSeconds(3)),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.Cpu = &stats.DomainStatsCPU{
				TimeSet: false,
			}
			crs := cpuMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
