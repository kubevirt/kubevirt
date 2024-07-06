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

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
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
			Expect(crs).To(ContainElement(gomegaContainsMetricMatcher(metric, expectedValue)))
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
