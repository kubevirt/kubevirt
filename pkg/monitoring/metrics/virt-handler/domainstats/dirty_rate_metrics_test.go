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

	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("dirty rate metrics", func() {
	Context("on Collect", func() {
		var (
			vmiStats  *VirtualMachineInstanceStats
			vmiReport *VirtualMachineInstanceReport
		)

		BeforeEach(func() {
			vmiStats = &VirtualMachineInstanceStats{
				DomainStats: &stats.DomainStats{
					DirtyRate: &stats.DomainStatsDirtyRate{
						MegabytesPerSecondSet: true,
						MegabytesPerSecond:    54,
					},
				},
			}

			vmiReport = newVirtualMachineInstanceReport(api.NewMinimalVMI("test-vmi"), vmiStats)
		})

		It("should collect kubevirt_vmi_dirty_rate_bytes_per_second metrics values", func() {
			const expectedValue = 54.0 * 1024 * 1024
			crs := dirtyRateMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(dirtyRateBytesPerSecond, expectedValue)))
		})

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.DomainStats.DirtyRate = &stats.DomainStatsDirtyRate{
				MegabytesPerSecondSet: false,
			}
			crs := dirtyRateMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
