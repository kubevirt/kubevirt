/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
)

var _ = Describe("filesystem metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			FsStats: k6tv1.VirtualMachineInstanceFileSystemList{
				Items: []k6tv1.VirtualMachineInstanceFileSystem{
					{
						DiskName:       "test-disk-1",
						MountPoint:     "/",
						FileSystemType: "ext4",
						TotalBytes:     1,
						UsedBytes:      2,
					},
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		DescribeTable("should collect metrics values", func(metric operatormetrics.Metric, expectedValue float64) {
			crs := filesystemMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(metric, expectedValue)))
		},
			Entry("kubevirt_vmi_filesystem_capacity_bytes", filesystemCapacityBytes, 1.0),
			Entry("kubevirt_vmi_filesystem_used_bytes", filesystemUsedBytes, 2.0),
		)

		It("result should be empty if stat not populated or set is false", func() {
			vmiStats.FsStats.Items = []k6tv1.VirtualMachineInstanceFileSystem{}
			crs := filesystemMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
