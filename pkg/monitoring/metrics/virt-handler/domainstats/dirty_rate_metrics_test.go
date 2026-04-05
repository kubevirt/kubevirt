/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
